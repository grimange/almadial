package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grimange/almadial/internal/preflight"
)

func requirements(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "budget.json")
	b, _ := json.Marshal(preflight.Requirements{Profile: "synthetic-only", MinCPUs: 1, CPUFlags: []string{"sse2"}, MinTotalMemory: 1024, MinAvailableMemory: 512, MinDiskAvailable: 4096})
	if e := os.WriteFile(p, b, 0600); e != nil {
		t.Fatal(e)
	}
	return p
}
func TestWizardNoninteractive(t *testing.T) {
	budget := requirements(t)
	base := []string{"wizard", "--manifest", "../../manifests/stack.lock.yaml", "--requirements", budget, "--config", "../config/testdata/basic.json"}
	for _, tc := range []struct {
		name    string
		extra   []string
		code    int
		message string
	}{
		{"no TTY", nil, Usage, "no interactive terminal"},
		{"review", []string{"--unattended", "--review-only"}, Success, "approval NOT REQUESTED"},
		{"missing approval", []string{"--unattended"}, Failure, "explicit unattended approval"},
		{"bad digest", []string{"--unattended", "--approve", "--config-sha256", strings.Repeat("0", 64)}, Failure, "must match"},
		{"no interactive fallback", []string{"--approve"}, Usage, "no interactive terminal"},
		{"review approval conflict", []string{"--unattended", "--review-only", "--approve"}, Usage, "cannot be combined"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, err bytes.Buffer
			args := append(append([]string{}, base...), tc.extra...)
			code := RunContext(context.Background(), args, strings.NewReader("yes\n"), &out, &err)
			if code != tc.code || !strings.Contains(out.String()+err.String(), tc.message) {
				t.Fatalf("code=%d stdout=%s stderr=%s", code, &out, &err)
			}
			if strings.Contains(out.String(), "Approve this exact plan") {
				t.Fatal("prompt in unattended mode")
			}
		})
	}
}
func TestWizardInputAndOutputFailures(t *testing.T) {
	for _, args := range [][]string{{"wizard", "--invalid-secret-token"}, {"wizard"}, {"wizard", "--manifest", "missing", "--requirements", "missing", "--unattended"}, {"install"}, {"resume"}} {
		var out, err bytes.Buffer
		if RunContext(context.Background(), args, strings.NewReader(""), &out, &err) != Usage {
			t.Fatal("invalid command accepted")
		}
		if strings.Contains(err.String(), "invalid-secret-token") {
			t.Fatal("argv leaked")
		}
	}
	var out, err bytes.Buffer
	if RunContext(context.Background(), []string{"wizard", "--help"}, nil, &out, &err) != Success || !strings.Contains(out.String(), "CANDIDATE") {
		t.Fatal("wizard help")
	}
	if RunContext(context.Background(), []string{"wizard", "--help"}, nil, brokenWriter{}, &err) != Failure {
		t.Fatal("write failure")
	}
}
func TestWizardReadOnlyAndExplicitLog(t *testing.T) {
	dir := t.TempDir()
	os.Chmod(dir, 0700)
	state := filepath.Join(dir, "state")
	t.Setenv("ALMADIAL_STATE_DIR", state)
	args := []string{"wizard", "--unattended", "--review-only", "--manifest", "../../manifests/stack.lock.yaml", "--requirements", requirements(t), "--config", "../config/testdata/basic.json"}
	var out, err bytes.Buffer
	if RunContext(context.Background(), args, strings.NewReader(""), &out, &err) != Success {
		t.Fatal(err.String())
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatal("read-only wizard created state")
	}
	path := filepath.Join(dir, "operator.jsonl")
	args = append(args, "--log-file", path)
	if RunContext(context.Background(), args, nil, &out, &err) != Success {
		t.Fatal(err.String())
	}
	entries, _ = os.ReadDir(dir)
	if len(entries) != 1 || entries[0].Name() != "operator.jsonl" {
		t.Fatal("log created journal/lock")
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Fatal("log permissions")
	}
	if RunContext(context.Background(), args, nil, &out, &err) != Failure {
		t.Fatal("overwrote log")
	}
}
func TestRegularFileRejectsSpecialInputs(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "link")
	os.Symlink("../config/testdata/basic.json", link)
	if _, e := regular(link, 1<<20); e == nil {
		t.Fatal("followed symlink")
	}
	if _, e := regular(dir, 1<<20); e == nil {
		t.Fatal("opened directory")
	}
}
