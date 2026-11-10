package provision

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"path"
	"strings"

	"github.com/grimange/almadial/internal/artifacts"
)

const netTelnetArchiveSHA = "677f68ba2cd2a824fae323fa82e183bf7e3d03c3c499c91d923bd6283796a743"

// VICIdialPerlSource retains the reviewed pure-Perl dependency. Deployment must
// use the accepted site path and must not execute Makefile.PL or a CPAN client.
type VICIdialPerlSource struct{ archive, module, notice []byte }

func (s VICIdialPerlSource) Digest() string {
	if len(s.archive) == 0 || len(s.module) == 0 || len(s.notice) == 0 {
		return ""
	}
	return hash(struct{ Archive, Module, Notice string }{rawSHA(s.archive), rawSHA(s.module), rawSHA(s.notice)})
}

func LoadVICIdialPerlSource(ctx context.Context, m artifacts.Manifest, archive string) (VICIdialPerlSource, error) {
	spec, e := m.Artifact("net-telnet")
	if e != nil || ctx == nil {
		return VICIdialPerlSource{}, errVICIdial
	}
	r, e := artifacts.VerifyLocal(ctx, spec, archive)
	if e != nil || r.Identity != artifacts.Verified {
		return VICIdialPerlSource{}, errVICIdial
	}
	b, e := sourceBytes(archive, netTelnetArchiveSHA)
	if e != nil {
		return VICIdialPerlSource{}, errVICIdial
	}
	f, e := readVICIdialPerl(ctx, b)
	if e != nil || len(f["lib/Net/Telnet.pm"]) == 0 || len(f["README"]) == 0 {
		return VICIdialPerlSource{}, errVICIdial
	}
	return VICIdialPerlSource{b, f["lib/Net/Telnet.pm"], f["README"]}, nil
}

func readVICIdialPerl(ctx context.Context, b []byte) (map[string][]byte, error) {
	if ctx == nil {
		return nil, errVICIdial
	}
	z, e := gzip.NewReader(bytes.NewReader(b))
	if e != nil {
		return nil, errVICIdial
	}
	defer z.Close()
	t := tar.NewReader(z)
	files, seen := map[string][]byte{}, map[string]bool{}
	var size int64
	for count := 0; ; count++ {
		if e := ctx.Err(); e != nil {
			return nil, e
		}
		h, e := t.Next()
		if e == io.EOF {
			break
		}
		if e != nil || count >= 100 || h.Size < 0 || h.Size > 2<<20 || size+h.Size > 8<<20 {
			return nil, errVICIdial
		}
		size += h.Size
		name := strings.TrimSuffix(h.Name, "/")
		if path.Clean(name) != name || strings.ContainsAny(name, "\\\x00\r\n:") || seen[name] || (name != "Net-Telnet-3.05" && !strings.HasPrefix(name, "Net-Telnet-3.05/")) {
			return nil, errVICIdial
		}
		seen[name] = true
		if h.Typeflag == tar.TypeDir && h.Size == 0 {
			continue
		}
		if h.Typeflag != tar.TypeReg || !strings.HasPrefix(name, "Net-Telnet-3.05/") {
			return nil, errVICIdial
		}
		v, e := io.ReadAll(io.LimitReader(t, h.Size+1))
		if e != nil || int64(len(v)) != h.Size {
			return nil, errVICIdial
		}
		files[strings.TrimPrefix(name, "Net-Telnet-3.05/")] = v
	}
	for name := range files {
		for p := path.Dir(name); p != "."; p = path.Dir(p) {
			if _, exists := files[p]; exists {
				return nil, errVICIdial
			}
		}
	}
	return files, nil
}
