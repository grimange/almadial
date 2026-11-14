package artifacts

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// FetchOptions names an existing private staging directory. RootCAs is an
// explicit enterprise/test CA pool; nil uses OS trust. TLS validation is mandatory.
type FetchOptions struct {
	Directory string
	RootCAs   *x509.CertPool
}

// Fetch revalidates a private cache before trying approved retention origins,
// then the original pinned upstream URL. Every candidate uses identical content
// verification; changing location cannot change source identity or policy.
func Fetch(ctx context.Context, s Spec, o FetchOptions) (Result, error) {
	return fetch(ctx, s, o, exportSVN)
}

func fetch(ctx context.Context, s Spec, o FetchOptions, exporter revisionExporter) (Result, error) {
	r := initial(s)
	if !s.valid() {
		return r, errors.New("artifact: uninitialized specification")
	}
	origins := append(append([]string{}, s.origins...), s.source)
	for _, origin := range origins {
		if !approvedHTTPS(origin) {
			return r, errors.New("fetch: approved HTTPS source required")
		}
	}
	root, err := privateRoot(o.Directory)
	if err != nil {
		return r, err
	}
	defer root.Close()
	final := s.component + ".artifact"
	if st, e := root.Lstat(final); e == nil {
		owner, ok := st.Sys().(*syscall.Stat_t)
		if !st.Mode().IsRegular() || st.Mode().Perm()&0077 != 0 || !ok || owner.Uid != uint32(os.Geteuid()) {
			return r, errors.New("cache: expected owned private regular file")
		}
		f, e := root.OpenFile(final, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
		if e != nil {
			return r, errors.New("cache: cannot inspect existing artifact")
		}
		r, e = verifyInstallFile(ctx, s, f)
		f.Close()
		if e == nil {
			return r, nil
		}
		if ctx.Err() != nil {
			return r, ctx.Err()
		}
		// Keep rejected bytes for diagnosis, but permit verified replacement. A
		// private root and random destination avoid overwriting earlier evidence.
		nonce := make([]byte, 16)
		if _, err = rand.Read(nonce); err != nil {
			return r, err
		}
		if err = root.Rename(final, final+".rejected-"+hex.EncodeToString(nonce)); err != nil {
			return r, errors.New("cache: cannot quarantine rejected artifact")
		}
	} else if !os.IsNotExist(e) {
		return r, errors.New("cache: cannot inspect existing artifact")
	}
	failures := []string{}
	for _, origin := range origins {
		if ctx.Err() != nil {
			return r, ctx.Err()
		}
		r, err = fetchFrom(ctx, s, o, root, final, origin)
		if err == nil {
			return r, nil
		}
		failures = append(failures, origin+": "+err.Error())
	}
	if revisionSpec(s) {
		r, err = fetchRevision(ctx, s, root, final, exporter)
		if err == nil {
			return r, nil
		}
		failures = append(failures, vicidialSVN+"@4037: "+err.Error())
	}
	return r, fmt.Errorf("fetch: all approved acquisition origins exhausted: %s; restore the exact pinned object in an approved store or retry connectivity; source repinning requires review", strings.Join(failures, "; "))
}

func approvedHTTPS(source string) bool {
	u, err := url.Parse(source)
	return err == nil && u.Scheme == "https" && u.Host != "" && u.User == nil && u.RawQuery == "" && u.Fragment == "" && u.Opaque == "" && u.RawPath == "" && !strings.ContainsAny(source, "\\\r\n\t")
}

func fetchFrom(ctx context.Context, s Spec, o FetchOptions, root *os.Root, final, origin string) (Result, error) {
	r := initial(s)
	var err error
	nonce := make([]byte, 16)
	if _, err = rand.Read(nonce); err != nil {
		return r, errors.New("fetch: cannot allocate staging name")
	}
	temp := ".download-" + hex.EncodeToString(nonce)
	f, err := root.OpenFile(temp, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		return r, errors.New("fetch: cannot create private staging file")
	}
	defer f.Close()
	defer root.Remove(temp)
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	transport := &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 15 * time.Second}).DialContext, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: o.RootCAs}, TLSHandshakeTimeout: 15 * time.Second, ResponseHeaderTimeout: 20 * time.Second, DisableCompression: true}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirect rejected") }}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, origin, nil)
	if err != nil {
		return r, errors.New("fetch: invalid request")
	}
	resp, err := client.Do(req)
	if err != nil {
		r.Transport = Rejected
		return r, errors.New("fetch: authenticated HTTPS request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return r, fmt.Errorf("fetch: HTTP status %d rejected", resp.StatusCode)
	}
	if resp.ContentLength > maxArchiveBytes {
		return r, errors.New("fetch: download exceeds limit")
	}
	n, err := io.Copy(f, io.LimitReader(resp.Body, maxArchiveBytes+1))
	if err != nil || n > maxArchiveBytes {
		return r, errors.New("fetch: incomplete or oversized download")
	}
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return r, errors.New("fetch: cannot inspect staged file")
	}
	r, err = verifyFile(ctx, s, f)
	if err != nil {
		return r, err
	}
	r.Source = origin
	r.Transport = Verified
	r.Provenance = "approved pinned HTTPS origin and exact content identity; no detached signature claim"
	if s.component != "asterisk" {
		r.Publisher = Verified
		r.Disposition = Verified
		r.Limitations = []string{"publisher evidence is approved HTTPS origin, not a cryptographic release signature", "candidate policy is not installation authorization"}
	}
	if err = f.Sync(); err != nil {
		return r, errors.New("fetch: cannot sync verified cache file")
	}
	// Link is atomic and refuses to replace an existing destination.
	if err = root.Link(temp, final); err != nil {
		return r, errors.New("fetch: cache publication collision; reverify existing artifact")
	}
	return r, nil
}

func privateRoot(dir string) (*os.Root, error) {
	if !filepath.IsAbs(dir) {
		return nil, errors.New("cache: existing absolute private directory required")
	}
	st, err := os.Lstat(dir)
	if err != nil || !st.IsDir() || st.Mode().Perm()&0077 != 0 {
		return nil, errors.New("cache: expected existing private directory (0700)")
	}
	stat, ok := st.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Geteuid()) {
		return nil, errors.New("cache: directory must belong to current user")
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, errors.New("cache: cannot open directory")
	}
	opened, e := root.Stat(".")
	if e != nil || !os.SameFile(st, opened) {
		root.Close()
		return nil, errors.New("cache: staging directory changed")
	}
	return root, nil
}
