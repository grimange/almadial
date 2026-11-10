package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNativeCommandRejectsIncompleteOrCrossModeFlags(t *testing.T) {
	common := []string{"--manifest", "missing", "--source", "missing", "--perl-source", "missing", "--asterisk-source", "missing", "--signature", "missing", "--keyring", "missing", "--scratch", "missing", "--pjproject", "missing"}
	for _, mode := range []string{"inspect", "plan", "lab-run", "lab-resume"} {
		for _, extra := range [][]string{nil, {"--accepted-asterisk", "missing", "--input", "missing"}, {"--input", "missing", "--authorization", "missing"}, {"--state-dir", "missing"}, {"trailing"}} {
			args := append(append([]string{mode}, common...), extra...)
			var out bytes.Buffer
			if err := runNative(context.Background(), args, &out); err != errInput || out.Len() != 0 {
				t.Fatalf("invalid flags accepted: %s: %v", mode, err)
			}
		}
	}
}

func TestNativeJSONRejectsSymlinksUnknownFieldsAndTrailingData(t *testing.T) {
	dir := t.TempDir()
	type input struct{ Value string }
	for _, raw := range []string{`{"Value":"ok","Foreign":true}`, `{"Value":"ok"} {}`, `{"Value":"ok"} trailing`, `[]`, ""} {
		path := filepath.Join(dir, "input")
		if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		var value input
		if loadNative(path, &value) == nil {
			t.Fatal("malformed input accepted")
		}
	}
	path := filepath.Join(dir, "valid")
	if err := os.WriteFile(path, []byte(`{"Value":"ok"}`), 0600); err != nil {
		t.Fatal(err)
	}
	var value input
	if loadNative(path, &value) != nil || value.Value != "ok" {
		t.Fatal("valid input rejected")
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if loadNative(link, &value) == nil {
		t.Fatal("symlink accepted")
	}
}
