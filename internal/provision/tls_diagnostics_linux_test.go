//go:build linux

package provision

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTLSPathDiagnostics(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "fullchain.pem")
	if err := os.WriteFile(target, []byte("PUBLIC FIXTURE"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "managed.pem")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	parent := filepath.Join(dir, "live")
	if err := os.Symlink(dir, parent); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		path    string
		private bool
		reason  string
	}{{"relative.pem", false, "absolute path"}, {link, false, "symlink"}, {filepath.Join(parent, "fullchain.pem"), false, "parent directory"}, {filepath.Join(dir, "missing-key.pem"), true, "cannot open"}}
	if os.Geteuid() != 0 {
		cases = append(cases, struct {
			path    string
			private bool
			reason  string
		}{target, true, "root ownership"})
	}
	for _, tc := range cases {
		_, err := tlsRead(tc.path, tc.private)
		if err == nil {
			t.Fatal(tc.path)
		}
		for _, want := range []string{tc.path, tc.reason, "root-owned regular PEM", "0600 root:root"} {
			if !strings.Contains(err.Error(), want) {
				t.Fatal(want, err)
			}
		}
		if strings.Contains(err.Error(), "PUBLIC FIXTURE") {
			t.Fatal("file contents in diagnostic")
		}
	}
	if err := os.Chmod(target, 0666); err != nil {
		t.Fatal(err)
	}
	if _, err := tlsRead(target, false); err == nil || !strings.Contains(err.Error(), "group/world write") {
		t.Fatal(err)
	}
}
