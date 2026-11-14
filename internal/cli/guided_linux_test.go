//go:build linux

package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grimange/almadial/internal/operatorui"
)

func TestGuidedApproval(t *testing.T) {
	for _, tc := range []struct {
		input string
		ok    bool
	}{{"yes\n", true}, {"no\n", false}, {"\n", false}, {"cancel\n", false}, {"yes", false}} {
		var out bytes.Buffer
		p := operatorui.NewPrompts(context.Background(), strings.NewReader(tc.input), &out)
		err := approveInstall(p, &out, strings.Repeat("a", 64))
		if (err == nil) != tc.ok {
			t.Fatalf("approval %q: %v", tc.input, err)
		}
		if !strings.Contains(out.String(), strings.Repeat("a", 64)) {
			t.Fatal("digest hidden")
		}
	}
}
func TestGuidedTLSAndPreservation(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	r, err := guidedLabTLS(dir)
	if err != nil {
		t.Fatal(err)
	}
	first, err := guidedTLSIdentity(r)
	if err != nil {
		t.Fatal(err)
	}
	_, err = guidedLabTLS(dir)
	if err != nil {
		t.Fatal(err)
	}
	next, err := guidedTLSIdentity(r)
	if err != nil || first != next {
		t.Fatal("TLS rotated on retry", err)
	}
	st, err := os.Stat(r.PrivateKeyFile)
	if err != nil || st.Mode().Perm() != 0600 {
		t.Fatal("unsafe private key")
	}
	if err = guidedSave(r.PrivateKeyFile, []byte("replacement")); err == nil {
		t.Fatal("private key overwritten")
	}
	link := filepath.Join(dir, "foreign")
	if err = os.Symlink(filepath.Dir(dir), link); err != nil {
		t.Fatal(err)
	}
	if guidedDirectory(link) == nil {
		t.Fatal("symlink directory accepted")
	}
}
func TestGuidedNoninteractive(t *testing.T) {
	var out, errout bytes.Buffer
	if RunContext(context.Background(), []string{"install"}, strings.NewReader("yes\n"), &out, &errout) == Success {
		t.Fatal("nonterminal accepted")
	}
	if !strings.Contains(errout.String(), "interactive terminal") {
		t.Fatal(errout.String())
	}
}

func TestGuidedSavedInputRejectsLinkAndPublicFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "input")
	if err := guidedSave(file, []byte("original")); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(file, link); err != nil {
		t.Fatal(err)
	}
	if guidedSave(link, []byte("original")) == nil {
		t.Fatal("accepted symlink input")
	}
	if err := os.Chmod(file, 0644); err != nil {
		t.Fatal(err)
	}
	if guidedSave(file, []byte("original")) == nil {
		t.Fatal("accepted public saved input")
	}
}
