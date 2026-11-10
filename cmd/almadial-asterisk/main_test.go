package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestQualificationCLI(t *testing.T) {
	for _, args := range [][]string{nil, {"--help"}, {"help"}} {
		var b bytes.Buffer
		if e := run(context.Background(), args, &b); e != nil || !strings.Contains(b.String(), "CANDIDATE") {
			t.Fatal(e)
		}
	}
	for _, args := range [][]string{{"install"}, {"plan"}, {"lab-run"}, {"inspect", "--manifest", "missing", "--source", "missing"}, {"inspect", "--unknown"}, {"lab-resume", "extra"}} {
		var b bytes.Buffer
		if run(context.Background(), args, &b) == nil {
			t.Fatal("unsafe incomplete CLI", args)
		}
	}
}
func TestQualificationInputSafety(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "input")
	if e := os.WriteFile(p, []byte(`{"Scope":"test"}`), 0600); e != nil {
		t.Fatal(e)
	}
	var v struct{ Scope string }
	if load(p, &v) != nil {
		t.Fatal("valid fixture")
	}
	for _, b := range []string{`{"Unknown":true}`, `{} {}`, `{"Scope":1}`} {
		_ = os.WriteFile(p, []byte(b), 0600)
		if load(p, &v) == nil {
			t.Fatal("accepted invalid JSON")
		}
	}
	q := filepath.Join(d, "link")
	_ = os.Symlink(p, q)
	if load(q, &v) == nil {
		t.Fatal("accepted symlink")
	}
}
