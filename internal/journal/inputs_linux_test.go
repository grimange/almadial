//go:build linux

package journal

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallationInputOwnership(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state")
	if _, e := InspectInput(dir, "S01.input.json"); !errors.Is(e, os.ErrNotExist) {
		t.Fatal(e)
	}
	if _, e := os.Stat(dir); !errors.Is(e, os.ErrNotExist) {
		t.Fatal("inspection created state")
	}
	s, e := Acquire(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if _, e := Acquire(dir); !errors.Is(e, ErrBusy) {
		t.Fatal("installation ownership bypass", e)
	}
	data := []byte(`{"approved":"first"}`)
	if e = s.PreserveInput("S01.input.json", data); e != nil {
		t.Fatal(e)
	}
	st, _ := os.Stat(filepath.Join(dir, "S01.input.json"))
	if e = s.PreserveInput("S01.input.json", data); e != nil {
		t.Fatal(e)
	}
	st2, _ := os.Stat(filepath.Join(dir, "S01.input.json"))
	if !os.SameFile(st, st2) {
		t.Fatal("identical input rewritten")
	}
	if e = s.PreserveInput("S01.input.json", []byte(`{"approved":"changed"}`)); e == nil {
		t.Fatal("overwrote approved input")
	}
	got, e := InspectInput(dir, "S01.input.json")
	if e != nil || string(got) != string(data) {
		t.Fatal("input changed", e)
	}
	for _, name := range []string{"../S01.input.json", "journal.json", "apply.lock", "a/b.input.json"} {
		if s.PreserveInput(name, data) == nil {
			t.Fatal("unsafe name", name)
		}
	}
	target := filepath.Join(t.TempDir(), "private")
	if e = os.WriteFile(target, data, 0600); e != nil {
		t.Fatal(e)
	}
	if e = os.Symlink(target, filepath.Join(dir, "evil.input.json")); e != nil {
		t.Fatal(e)
	}
	if s.PreserveInput("evil.input.json", data) == nil {
		t.Fatal("followed symlink")
	}
}
