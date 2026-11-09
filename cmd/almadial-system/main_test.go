package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestQualificationCLIArguments(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		help bool
	}{
		{"help", nil, true}, {"explicit help", []string{"--help"}, true},
		{"unknown operation", []string{"install"}, false},
		{"unexpected inventory flag", []string{"inventory", "--input", "private-fake-secret"}, false},
		{"unexpected trailing token", []string{"inventory", "private-fake-secret"}, false},
		{"missing inspect prerequisites", []string{"inspect"}, false},
		{"missing assessment input", []string{"assess"}, false},
		{"missing plan input", []string{"plan"}, false},
		{"missing run input", []string{"lab-run"}, false},
		{"missing resume input", []string{"lab-resume"}, false},
		{"missing helper argument", []string{"__directory", "0"}, false},
		{"invalid helper index", []string{"__directory", "private-fake-secret", "{}"}, false},
		{"unknown helper input", []string{"__directory", "0", `{"private-fake-secret":true}`}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			e := run(context.Background(), tc.args, &out)
			if tc.help {
				if e != nil || !strings.Contains(out.String(), "CANDIDATE") {
					t.Fatal(e, out.String())
				}
				return
			}
			if !errors.Is(e, errInput) || strings.Contains(e.Error(), "private-fake-secret") || out.Len() != 0 {
				t.Fatalf("invalid input exposed diagnostics or succeeded: %v %s", e, out.String())
			}
		})
	}
}
func TestQualificationInputFiles(t *testing.T) {
	dir := t.TempDir()
	valid := filepath.Join(dir, "valid.json")
	if e := os.WriteFile(valid, []byte(`{"Value":1}`), 0600); e != nil {
		t.Fatal(e)
	}
	link := filepath.Join(dir, "symlink")
	if e := os.Symlink(valid, link); e != nil {
		t.Fatal(e)
	}
	fifo := filepath.Join(dir, "fifo")
	if e := syscall.Mkfifo(fifo, 0600); e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct {
		name, path, data string
		want             bool
	}{
		{"valid", valid, "", true}, {"symlink", link, "", false}, {"directory", dir, "", false}, {"fifo", fifo, "", false},
		{"unknown field", filepath.Join(dir, "unknown"), `{"private-fake-secret":1}`, false},
		{"trailing document", filepath.Join(dir, "trailing"), `{"Value":1}{}`, false},
		{"oversized", filepath.Join(dir, "large"), strings.Repeat(" ", (4<<20)+1), false},
		{"missing", filepath.Join(dir, "private-fake-secret"), "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.data != "" {
				if e := os.WriteFile(tc.path, []byte(tc.data), 0600); e != nil {
					t.Fatal(e)
				}
			}
			var v struct{ Value int }
			e := load(tc.path, &v)
			if (e == nil) != tc.want {
				t.Fatal(e)
			}
			if e != nil && (e != errInput || strings.Contains(e.Error(), "private-fake-secret")) {
				t.Fatal("unsafe input diagnostics", e)
			}
		})
	}
}
