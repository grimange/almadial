package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestReadOnlyCLI(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		fail bool
	}{
		{"help", []string{"--help"}, false},
		{"no arguments", nil, false},
		{"unknown operation", []string{"private-value"}, true},
		{"no inventory flags", []string{"inventory", "--input", "private-value"}, true},
		{"plan needs saved input", []string{"plan"}, true},
		{"missing manifest", []string{"plan", "--input", "private-value"}, true},
		{"no arbitrary executable", []string{"lab-run", "--command", "private-value"}, true},
		{"reject positional tail", []string{"assess", "private-value"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			e := run(context.Background(), tc.args, &out)
			if (e != nil) != tc.fail {
				t.Fatal(e)
			}
			if e != nil && strings.Contains(e.Error(), "private-value") {
				t.Fatal("private argument leaked")
			}
		})
	}
	file := filepath.Join(t.TempDir(), "inventory.json")
	if e := os.WriteFile(file, []byte(`{}`), 0o600); e != nil {
		t.Fatal(e)
	}
	var out bytes.Buffer
	if e := run(context.Background(), []string{"assess", "--input", file}, &out); e != nil {
		t.Fatal(e)
	}
	var a []struct{ Status string }
	if e := json.Unmarshal(out.Bytes(), &a); e != nil || len(a) != 44 {
		t.Fatal(e, len(a))
	}
	for _, v := range a {
		if v.Status != "NOT_VERIFIED" {
			t.Fatal(v)
		}
	}
}

func TestInputBoundary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "input")
	for _, text := range []string{`{"unknown":"private-value"}`, `{} {}`, `{"x":`, strings.Repeat(" ", (4<<20)+1)} {
		if e := os.WriteFile(path, []byte(text), 0o600); e != nil {
			t.Fatal(e)
		}
		var value struct{}
		if e := input(path, &value); e == nil || strings.Contains(e.Error(), "private-value") {
			t.Fatal(e)
		}
	}
	link := filepath.Join(dir, "link")
	if e := os.Symlink(path, link); e != nil {
		t.Fatal(e)
	}
	if f, e := regularInput(link); e == nil {
		f.Close()
		t.Fatal("followed symlink")
	}
	if f, e := regularInput(dir); e == nil {
		f.Close()
		t.Fatal("opened directory")
	}
	fifo := filepath.Join(dir, "fifo")
	if e := syscall.Mkfifo(fifo, 0o600); e != nil {
		t.Fatal(e)
	}
	if f, e := regularInput(fifo); e == nil {
		f.Close()
		t.Fatal("opened FIFO")
	}
}
