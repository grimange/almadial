package provision

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"testing"
)

func perlArchive(t *testing.T, names []string, kind byte) []byte {
	t.Helper()
	var b bytes.Buffer
	z := gzip.NewWriter(&b)
	w := tar.NewWriter(z)
	for _, name := range names {
		data := []byte("synthetic Perl fixture, never executed")
		h := tar.Header{Name: name, Mode: 0644, Typeflag: kind, Size: int64(len(data))}
		if kind == tar.TypeSymlink {
			h.Linkname = "/etc/passwd"
			h.Size = 0
			data = nil
		}
		if e := w.WriteHeader(&h); e != nil {
			t.Fatal(e)
		}
		if _, e := w.Write(data); e != nil {
			t.Fatal(e)
		}
	}
	if e := w.Close(); e != nil {
		t.Fatal(e)
	}
	if e := z.Close(); e != nil {
		t.Fatal(e)
	}
	return b.Bytes()
}

func TestVICIdialPerlArchiveContract(t *testing.T) {
	for _, tc := range []struct {
		name  string
		paths []string
		kind  byte
		ok    bool
	}{
		{"regular", []string{"Net-Telnet-3.05/lib/Net/Telnet.pm", "Net-Telnet-3.05/README"}, tar.TypeReg, true},
		{"wrong version", []string{"Net-Telnet-3.06/lib/Net/Telnet.pm"}, tar.TypeReg, false},
		{"traversal", []string{"Net-Telnet-3.05/../escape"}, tar.TypeReg, false},
		{"absolute", []string{"/etc/passwd"}, tar.TypeReg, false},
		{"symlink", []string{"Net-Telnet-3.05/lib/Net/Telnet.pm"}, tar.TypeSymlink, false},
		{"duplicate", []string{"Net-Telnet-3.05/README", "Net-Telnet-3.05/README"}, tar.TypeReg, false},
		{"file parent", []string{"Net-Telnet-3.05/lib", "Net-Telnet-3.05/lib/child"}, tar.TypeReg, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, e := readVICIdialPerl(context.Background(), perlArchive(t, tc.paths, tc.kind))
			if (e == nil) != tc.ok || tc.ok && len(f) != 2 {
				t.Fatal(f, e)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := readVICIdialPerl(ctx, perlArchive(t, []string{"Net-Telnet-3.05/README"}, tar.TypeReg)); e == nil {
		t.Fatal("cancel ignored")
	}
	if _, e := readVICIdialPerl(nil, nil); e == nil {
		t.Fatal("nil context accepted")
	}
	if _, e := readVICIdialPerl(context.Background(), []byte("corrupt")); e == nil {
		t.Fatal("corrupt archive accepted")
	}
	if (VICIdialPerlSource{}).Digest() != "" {
		t.Fatal("empty dependency identified")
	}
}
