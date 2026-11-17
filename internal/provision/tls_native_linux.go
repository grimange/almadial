//go:build linux

package provision

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/grimange/almadial/internal/execution"
)

//go:embed tls_native.py
var tlsAdapterSource string

type TLSNative struct {
	Web         WebNative
	Request     TLSRequest
	Certificate TLSCertificate
}

// Read only bounded regular files. Private keys may be root-only or readable
// by the endpoint's asterisk group; group/world writes and world reads fail.
func tlsRead(path string, private bool) ([]byte, error) {
	reject := func(reason string) ([]byte, error) {
		return nil, fmt.Errorf("TLS input %q: %s; supply an absolute root-owned regular PEM file with no symlink in its path; use 0600 root:root for private keys and 0644 root:root for certificate/CA files", path, reason)
	}
	if !filepath.IsAbs(path) {
		return reject("absolute path required")
	}
	for p := filepath.Dir(path); ; p = filepath.Dir(p) {
		st, e := os.Lstat(p)
		if e != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
			return reject("parent directory is missing, is not a directory, or is a symlink")
		}
		if p == "/" {
			break
		}
	}
	f, e := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if e != nil {
		return reject("cannot open input; missing file, symlink, or inaccessible path")
	}
	defer f.Close()
	s, e := f.Stat()
	if e != nil || !s.Mode().IsRegular() || s.Size() > 4<<20 || s.Mode().Perm()&0022 != 0 {
		return reject("input must be a regular PEM file of at most 4 MiB, without group/world write permission")
	}
	st, ok := s.Sys().(*syscall.Stat_t)
	if !ok || st.Uid != 0 || private && (s.Mode().Perm()&0007 != 0 || s.Mode().Perm()&0700 != 0600) {
		return reject("root ownership required; private keys require owner read/write and no world access")
	}
	if private && st.Gid != 0 {
		group, e := user.LookupGroup("asterisk")
		if e != nil || strconv.FormatUint(uint64(st.Gid), 10) != group.Gid {
			return reject("private key group must be root or asterisk")
		}
	}
	return io.ReadAll(io.LimitReader(f, 4<<20))
}

func (n TLSNative) Material() (TLSCertificate, error) {
	_, c, e := n.payload("")
	return c, e
}

func (n TLSNative) payload(expected string) ([]byte, TLSCertificate, error) {
	for _, input := range []struct{ name, path string }{{"certificate full-chain", n.Request.CertificateFile}, {"private key", n.Request.PrivateKeyFile}, {"CA trust", n.Request.TrustFile}} {
		if !filepath.IsAbs(input.path) || filepath.Clean(input.path) != input.path || strings.ContainsAny(input.path, "\n\r\x00") {
			return nil, TLSCertificate{}, fmt.Errorf("TLS %s input %q: supply a clean absolute path to a root-owned regular PEM file without symlinks", input.name, input.path)
		}
	}
	if !tlsRequestValid(n.Request) {
		return nil, TLSCertificate{}, fmt.Errorf("TLS request: require a valid FQDN for remote mode, supported mode, and separate certificate/trust paths from the private key: %w", errTLS)
	}
	chain, e := tlsRead(n.Request.CertificateFile, false)
	if e != nil {
		return nil, TLSCertificate{}, fmt.Errorf("certificate full-chain input: %w", e)
	}
	key, e := tlsRead(n.Request.PrivateKeyFile, true)
	if e != nil {
		return nil, TLSCertificate{}, fmt.Errorf("private-key input: %w", e)
	}
	trust, e := tlsRead(n.Request.TrustFile, false)
	if e != nil {
		return nil, TLSCertificate{}, fmt.Errorf("CA trust input: %w", e)
	}
	c, e := ValidateTLSMaterial(n.Request.Hostname, chain, key, trust, time.Now())
	if e != nil {
		return nil, c, fmt.Errorf("TLS certificate %q / private key %q / CA trust %q for hostname %q: %w", n.Request.CertificateFile, n.Request.PrivateKeyFile, n.Request.TrustFile, n.Request.Hostname, e)
	}
	if n.Certificate.LeafSHA256 != "" && n.Certificate != c {
		return nil, c, errTLS
	}
	p := map[string]any{"Hostname": n.Request.Hostname, "Chain": chain, "Key": key, "Trust": trust, "Leaf": c.LeafSHA256, "Generation": c.Digest(), "CertificateDigest": c.Digest(), "Base": tlsAsteriskBase, "HTTP": tlsAsteriskHTTP, "Loader": tlsLoader, "Vhost": tlsVhostFor(n.Request), "Remote": n.Request.Mode == "remote", "ExpectedCurrent": expected}
	b, e := json.Marshal(p)
	return b, c, e
}

func (n TLSNative) adapter(ctx context.Context, mode, expected string) (TLSSnapshot, error) {
	var s TLSSnapshot
	b, _, e := n.payload(expected)
	if e != nil {
		return s, e
	}
	r, e := n.Web.Runtime.Application.input()
	if e != nil {
		return s, e
	}
	result, e := r.RunInput(ctx, dbCommand("/usr/bin/python3", "-B", "-c", tlsAdapterSource, mode), b)
	if e != nil || !result.Started || result.ExitCode != 0 || result.StdoutTruncated || result.StderrTruncated {
		return s, errTLS
	}
	if decode(result.Stdout, &s) != nil {
		return s, errTLS
	}
	return s, nil
}

func (n TLSNative) InspectTLS(ctx context.Context) (TLSSnapshot, error) {
	// Check the closed S08 state before admitting its two narrow downstream
	// handoffs. Ordinary S04/S05/S07 inspection retains its original contract.
	s, e := n.adapter(ctx, "tls-inspect", "")
	if e != nil {
		return s, fmt.Errorf("S08 native TLS inspection: %w", e)
	}
	n.Web.TLSHandoff = true
	n.Web.Runtime.Application.Asterisk.TLSHandoff = true
	n.Web.Runtime.Application.Asterisk.RemoteTLSHandoff = n.Request.Mode == "remote"
	parent, e := n.Web.InspectWeb(ctx)
	if e != nil {
		return s, fmt.Errorf("S08 S07 prerequisite: %w", e)
	}
	if !webSatisfied(parent) {
		return s, errTLS
	}
	s.Host, s.Foundation = parent.Host, parent.Digest()
	return s, nil
}

type tlsRunner struct {
	native TLSNative
	stage  TLSStage
}

func (n TLSNative) Runner(s TLSStage) execution.CommandRunner { return tlsRunner{n, s} }
func (r tlsRunner) Validate(c execution.Command) error {
	d := r.stage.doc()
	if d.Policy != TLSPolicyDigest() || d.Request != r.native.Request || d.Certificate != r.native.Certificate || d.Source != r.native.Web.Runtime.Application.Source.Digest() {
		return errTLS
	}
	for _, a := range r.stage.Graph().Actions() {
		if jsonString(a.Command) == jsonString(c) {
			return nil
		}
	}
	return errTLS
}
func (r tlsRunner) Run(ctx context.Context, c execution.Command) (execution.Result, error) {
	if r.Validate(c) != nil || len(c.Args) != 2 || c.Args[1] != "tls.00.activate" {
		return execution.Result{}, errTLS
	}
	_, e := r.native.adapter(ctx, "tls-activate", r.stage.doc().Before.Current)
	result := execution.Result{Started: true}
	if e != nil {
		result.ExitCode = 1
	}
	return result, e
}
