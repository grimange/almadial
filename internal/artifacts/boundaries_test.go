package artifacts

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestCanonicalPathOrdering(t *testing.T) {
	// pathlib sorts components: directory 'a' precedes file 'a.txt'.
	if !pathLess("a/z", "a.txt") || pathLess("a.txt", "a/z") {
		t.Fatal("wrong canonical path ordering")
	}
	data := tgz(t, entry{"root/a.txt", "a", 0}, entry{"root/a/z", "z", 0})
	s := fixtureSpec(data)
	s.root = "root"
	r, e := VerifyLocal(context.Background(), s, writeFixture(t, data))
	expected := digestBytes([]byte(digestBytes([]byte("z")) + "  a/z\n" + digestBytes([]byte("a")) + "  a.txt\n"))
	if e != nil || r.TreeSHA256 != expected {
		t.Fatal(r, e)
	}
	s.embeddedPath = "a/z"
	s.embeddedSHA = digestBytes([]byte("z"))
	s.files = 2
	if _, e = VerifyLocal(context.Background(), s, writeFixture(t, data)); e != nil {
		t.Fatal(e)
	}
	s.embeddedSHA = strings.Repeat("0", 64)
	if _, e = VerifyLocal(context.Background(), s, writeFixture(t, data)); e == nil {
		t.Fatal("substituted embedded asset accepted")
	}
}
func TestPinnedAsteriskLinks(t *testing.T) {
	for _, target := range []string{"file", "../../outside", "missing", "/absolute"} {
		t.Run(target, func(t *testing.T) {
			var b bytes.Buffer
			g := gzip.NewWriter(&b)
			w := tar.NewWriter(g)
			for _, h := range []*tar.Header{{Name: "asterisk-22.12.0/file", Mode: 0600, Typeflag: tar.TypeReg, Size: 1}, {Name: "asterisk-22.12.0/link", Mode: 0777, Typeflag: tar.TypeSymlink, Linkname: target}} {
				if e := w.WriteHeader(h); e != nil {
					t.Fatal(e)
				}
				if h.Size == 1 {
					w.Write([]byte("a"))
				}
			}
			w.Close()
			g.Close()
			s := fixtureSpec(b.Bytes())
			s.component = "asterisk"
			_, e := VerifyLocal(context.Background(), s, writeFixture(t, b.Bytes()))
			if (e == nil) != (target == "file") {
				t.Fatal("incorrect link decision", e)
			}
		})
	}
}
func TestZIPDirectoryBounds(t *testing.T) {
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	w, _ := z.Create("file")
	w.Write([]byte("a"))
	z.Close()
	for _, field := range []string{"entries", "directory", "spanned"} {
		data := append([]byte{}, b.Bytes()...)
		end := len(data) - 22
		switch field {
		case "entries":
			binary.LittleEndian.PutUint16(data[end+8:], 30001)
			binary.LittleEndian.PutUint16(data[end+10:], 30001)
		case "directory":
			binary.LittleEndian.PutUint32(data[end+12:], 17<<20)
		case "spanned":
			binary.LittleEndian.PutUint16(data[end+4:], 1)
		}
		s := fixtureSpec(data)
		s.format = "zip"
		if _, e := VerifyLocal(context.Background(), s, writeFixture(t, data)); e == nil {
			t.Fatal("unbounded ZIP accepted", field)
		}
	}
}
func TestDerivationIdentity(t *testing.T) {
	results, e := VerifyDerivations(context.Background(), "../..")
	if e != nil || len(results) != 6 {
		t.Fatal(results, e)
	}
	dir := t.TempDir()
	for _, r := range results {
		p := filepath.Join(dir, r.Path)
		os.MkdirAll(filepath.Dir(p), 0700)
		b, e := os.ReadFile(filepath.Join("../..", r.Path))
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(p, b, 0600); e != nil {
			t.Fatal(e)
		}
	}
	if _, e = VerifyDerivations(context.Background(), dir); e != nil {
		t.Fatal(e)
	}
	os.WriteFile(filepath.Join(dir, results[0].Path), []byte("substituted patch"), 0600)
	if _, e = VerifyDerivations(context.Background(), dir); e == nil {
		t.Fatal("changed patch accepted")
	}
}
func TestLocalSpecialFilesAndCache(t *testing.T) {
	data := tgz(t, entry{"a", "b", 0})
	s := fixtureSpec(data)
	fifo := filepath.Join(t.TempDir(), "fifo")
	if e := syscall.Mkfifo(fifo, 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := VerifyLocal(context.Background(), s, fifo); e == nil {
		t.Fatal("FIFO accepted")
	}
	link := filepath.Join(t.TempDir(), "link")
	os.Symlink(writeFixture(t, data), link)
	if _, e := VerifyLocal(context.Background(), s, link); e == nil {
		t.Fatal("symlink accepted")
	}
	for _, kind := range []string{"symlink", "public file", "public directory"} {
		dir := privateDir(t)
		file := filepath.Join(dir, "fixture.artifact")
		switch kind {
		case "symlink":
			os.Symlink(writeFixture(t, data), file)
		case "public file":
			os.WriteFile(file, data, 0644)
		case "public directory":
			os.Chmod(dir, 0755)
		}
		if _, e := Fetch(context.Background(), s, FetchOptions{Directory: dir}); e == nil {
			t.Fatal("unsafe cache accepted", kind)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	dir := privateDir(t)
	if _, e := Fetch(ctx, s, FetchOptions{Directory: dir}); e == nil {
		t.Fatal("canceled fetch accepted")
	}
	files, _ := os.ReadDir(dir)
	if len(files) != 0 {
		t.Fatal("canceled download retained")
	}
}
