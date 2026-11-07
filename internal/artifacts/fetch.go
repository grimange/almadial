package artifacts

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// FetchOptions names an existing private staging directory. RootCAs is an
// explicit enterprise/test CA pool; nil uses OS trust. TLS validation is mandatory.
type FetchOptions struct {
	Directory string
	RootCAs   *x509.CertPool
}

// Fetch is an explicit network operation, never called by preflight or planning.
// It only uses Spec's approved URL; every redirect is rejected. Cache publication
// means identity-verified storage, not permission to execute/extract/install.
func Fetch(ctx context.Context, s Spec, o FetchOptions) (Result, error) {
	r := initial(s)
	if !s.valid() {
		return r, errors.New("artifact: uninitialized specification")
	}
	u, err := url.Parse(s.source)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return r, errors.New("fetch: approved HTTPS source required")
	}
	root, err := privateRoot(o.Directory)
	if err != nil {
		return r, err
	}
	defer root.Close()
	final := s.component + ".artifact"
	if st, e := root.Lstat(final); e == nil && (!st.Mode().IsRegular() || st.Mode().Perm()&0077 != 0) {
		return r, errors.New("cache: expected private regular file")
	}
	if f, err := root.Open(final); err == nil {
		defer f.Close()
		st, e := f.Stat()
		if e != nil || !st.Mode().IsRegular() {
			return r, errors.New("cache: expected regular file")
		}
		return verifyFile(ctx, s, f)
	} else if !os.IsNotExist(err) {
		return r, errors.New("cache: cannot inspect existing artifact")
	}
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.source, nil)
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
		return r, errors.New("fetch: HTTP status rejected")
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
