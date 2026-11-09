package operatorlog

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
)

func TestRedactionBoundaries(t *testing.T) {
	r, _ := NewRedactor([]string{"FAKE-credential-123", "abcdef", "cdefgh"})
	for _, tc := range []struct {
		name      string
		fragments []string
		want      string
	}{
		{"stdout", []string{"FAKE-credential-123"}, "[REDACTED]"},
		{"stderr", []string{"error: FAKE-credential-123"}, "error: [REDACTED]"},
		{"chunks", []string{"FAKE-", "credential-", "123"}, "[REDACTED]"},
		{"cross-stream", []string{"FAKE-cred", "ential-123"}, "[REDACTED]"},
		{"capture-boundary", []string{"prefix FAKE-cred"}, "prefix [REDACTED]"},
		{"overlap", []string{"abcdefgh"}, "[REDACTED]"},
		{"repeat", []string{"FAKE-credential-123 FAKE-credential-123"}, "[REDACTED] [REDACTED]"},
		{"controls", []string{"hello\x1b\nworld"}, "hello  world"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := r.Text(tc.fragments...); got != tc.want {
				t.Fatalf("redaction mismatch: %q", got)
			}
		})
	}
	for _, offset := range []int{MaxMessage - 18, MaxMessage - 10, MaxMessage - 1, MaxMessage} {
		prefix := strings.Repeat("z", offset)
		got := r.Text(prefix, "FAKE-credential-123", "end")
		if strings.Contains(got, "FAKE") || len(got) > MaxMessage+len(" [TRUNCATED]") {
			t.Fatal("limit leaked token or exceeded bounds")
		}
	}
}
func TestRegistrationLimits(t *testing.T) {
	for _, s := range [][]string{{""}, {strings.Repeat("x", 4097)}, make([]string, 65)} {
		if _, e := NewRedactor(s); e == nil {
			t.Fatal("invalid registration accepted")
		}
	}
}

type failSink struct{ short bool }

func (w failSink) Write(b []byte) (int, error) {
	if w.short {
		return len(b) - 1, nil
	}
	return 0, syscall.ENOSPC
}
func TestRoutingBoundsAndFailures(t *testing.T) {
	for _, level := range []Level{Info, Warning, Error} {
		t.Run(string(level), func(t *testing.T) {
			var out, err, file bytes.Buffer
			r, _ := NewRedactor([]string{"FAKE-secret"})
			l, _ := New(&out, &err, &file, r)
			if e := l.Write(Record{Level: level, Stage: "S03", Action: "db", Code: "failure", Message: "FAKE-", Next: "FAKE-secret"}, "secret"); e != nil {
				t.Fatal(e)
			}
			var rec Record
			if e := json.Unmarshal(bytes.TrimSpace(file.Bytes()), &rec); e != nil {
				t.Fatal(e)
			}
			if strings.Contains(file.String(), "FAKE") || rec.Message != "[REDACTED]" {
				t.Fatal("raw diagnostics retained")
			}
			if level == Info {
				if out.String() != file.String() || err.Len() != 0 {
					t.Fatal("routing")
				}
			} else {
				if err.String() != file.String() || out.Len() != 0 {
					t.Fatal("routing")
				}
			}
		})
	}
	for _, sink := range []io.Writer{failSink{}, failSink{short: true}} {
		r, _ := NewRedactor(nil)
		l, _ := New(io.Discard, io.Discard, sink, r)
		if !errors.Is(l.Write(Record{Level: Info}), ErrWrite) || !errors.Is(l.Write(Record{Level: Info}), ErrWrite) {
			t.Fatal("write failure not sticky")
		}
	}
	r, _ := NewRedactor(nil)
	var file bytes.Buffer
	l, _ := New(io.Discard, io.Discard, &file, r)
	for i := 0; i < 1000; i++ {
		e := l.Write(Record{Level: Info, Message: strings.Repeat("q", MaxMessage)})
		if e != nil {
			if e != ErrLimit {
				t.Fatal(e)
			}
			break
		}
		if i == 999 {
			t.Fatal("no total bound")
		}
	}
	if file.Len() > MaxOutput {
		t.Fatal("file size bound")
	}
	if l.Write(Record{Level: Info}) != ErrLimit {
		t.Fatal("limit not sticky")
	}
}
func TestStableAndConcurrentRecords(t *testing.T) {
	r, _ := NewRedactor(nil)
	var out bytes.Buffer
	l, _ := New(&out, io.Discard, nil, r)
	rec := Record{Level: Info, Stage: "S01", Message: "pending"}
	l.Write(rec)
	first := out.String()
	l.Write(rec)
	if out.String() != first+first {
		t.Fatal("nondeterministic")
	}
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			if e := l.Write(rec); e != nil {
				t.Error(e)
			}
		})
	}
	wg.Wait()
	if strings.Count(out.String(), "\n") != 12 {
		t.Fatal("interleaved record")
	}
}
func TestPrivateFilePolicy(t *testing.T) {
	dir := t.TempDir()
	os.Chmod(dir, 0700)
	path := filepath.Join(dir, "run.jsonl")
	f, e := OpenFile(path)
	if e != nil {
		t.Fatal(e)
	}
	s, _ := f.Stat()
	if s.Mode().Perm() != 0600 {
		t.Fatal("file mode")
	}
	f.Close()
	if _, e = OpenFile(path); e == nil {
		t.Fatal("overwrote file")
	}
	link := filepath.Join(dir, "link")
	os.Symlink(path, link)
	if _, e = OpenFile(link); e == nil {
		t.Fatal("symlink followed")
	}
	parent := filepath.Join(t.TempDir(), "alias")
	os.Symlink(dir, parent)
	if _, e = OpenFile(filepath.Join(parent, "new")); e == nil {
		t.Fatal("parent symlink followed")
	}
	os.Chmod(dir, 0755)
	if _, e = OpenFile(filepath.Join(dir, "public")); e == nil {
		t.Fatal("public dir accepted")
	}
	for _, p := range []string{"relative.log", filepath.Join(dir, "missing", "new"), "/"} {
		if _, e = OpenFile(p); e == nil {
			t.Fatal("unsafe path accepted")
		}
	}
	b, _ := os.ReadFile(path)
	if len(b) != 0 {
		t.Fatal("unrelated file changed")
	}
}
