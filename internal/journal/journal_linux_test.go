//go:build linux

package journal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/grimange/almadial/internal/config"
	"github.com/grimange/almadial/internal/plan"
	"github.com/grimange/almadial/internal/preflight"
	"io"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/grimange/almadial/internal/execution"
)

func hash(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func fixture(t *testing.T) Document {
	t.Helper()
	g, e := execution.NewGraph([]execution.Action{{ID: "a", StageID: "S01", Description: "controlled fixture", Preconditions: []string{"eligible"}, Postconditions: []string{"installed"}, Command: execution.Command{Executable: "/test/helper", Timeout: time.Second, StdoutLimit: 128, StderrLimit: 128}}})
	if e != nil {
		t.Fatal(e)
	}
	d, e := New(Identity{hash("host"), hash("host-state"), hash("config"), hash("manifest"), hash("plan"), hash("policy"), g.Digest()}, g, map[string]Retry{"a": Never})
	if e != nil {
		t.Fatal(e)
	}
	return d
}
func stateDir(t *testing.T) string { t.Helper(); return filepath.Join(t.TempDir(), "state") }
func initialized(t *testing.T) (string, *Session, Document) {
	t.Helper()
	p := stateDir(t)
	s, e := Acquire(p)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	d := fixture(t)
	if e = s.Initialize(d); e != nil {
		t.Fatal(e)
	}
	return p, s, d
}

type observerFunc func(context.Context, Record) (Observation, error)

func (f observerFunc) Observe(ctx context.Context, r Record) (Observation, error) { return f(ctx, r) }
func obs(v Observed) Observer {
	return observerFunc(func(context.Context, Record) (Observation, error) { return Observation{v, hash("observed")}, nil })
}
func trace() execution.Trace {
	return execution.Trace{ActionID: "a", StageID: "S01", Outcome: "succeeded", Result: execution.Result{Started: true, ExitCode: 0}}
}
func mustRead(t *testing.T, p string) Document {
	t.Helper()
	d, e := Inspect(p)
	if e != nil {
		t.Fatal(e)
	}
	return d
}
func TestSerialization(t *testing.T) {
	d := fixture(t)
	b, e := d.JSON()
	if e != nil {
		t.Fatal(e)
	}
	loaded, e := Decode(bytes.NewReader(b))
	if e != nil {
		t.Fatal(e)
	}
	other, _ := loaded.JSON()
	if !bytes.Equal(b, other) {
		t.Fatal("nondeterministic serialization")
	}
	if strings.Contains(string(b), "/test/helper") || strings.Contains(string(b), "controlled fixture") {
		t.Fatal("unneeded command data persisted")
	}
}
func TestStrictDecode(t *testing.T) {
	d := fixture(t)
	b, _ := d.JSON()
	s := string(b)
	cases := map[string]string{
		"malformed": "{", "truncated": s[:len(s)-2], "trailing": s + " {}", "schema": strings.Replace(s, `"schema_version":1`, `"schema_version":2`, 1),
		"duplicate": strings.Replace(s, `"schema_version":1`, `"schema_version":1,"schema_version":1`, 1), "case": strings.Replace(s, `"identity"`, `"Identity"`, 1),
		"unknown": strings.Replace(s, `"schema_version":1`, `"password":"ARTIFICIAL_DO_NOT_ECHO","schema_version":1`, 1),
		"missing": strings.Replace(s, `"policy":"`+d.Identity.Policy+`",`, "", 1), "bad-digest": strings.Replace(s, d.Identity.Config, "bad", 1),
		"null": strings.Replace(s, `"dependencies":[]`, `"dependencies":null`, 1), "bad-state": strings.Replace(s, `"pending"`, `"completed"`, 1),
		"empty-post": strings.Replace(s, `["installed"]`, `[]`, 1), "bad-id": strings.Replace(s, `"id":"a"`, `"id":"../a"`, 1),
		"invalid-utf8": s + string([]byte{0xff}), "oversized": strings.Repeat(" ", MaxBytes+1),
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			_, e := Decode(strings.NewReader(input))
			if !errors.Is(e, ErrInvalid) {
				t.Fatal("accepted invalid document", e)
			}
			if strings.Contains(e.Error(), "ARTIFICIAL") {
				t.Fatal("leaked input")
			}
		})
	}
}
func TestInvalidModels(t *testing.T) {
	for _, name := range []string{"unsorted", "duplicate", "self", "unknown-dependency", "cycle", "retry", "success-without-evidence", "failed-category", "pending-exit", "nil-actions"} {
		t.Run(name, func(t *testing.T) {
			d := fixture(t)
			switch name {
			case "unsorted":
				b := d.Actions[0]
				b.ID = "0"
				d.Actions = append(d.Actions, b)
			case "duplicate":
				d.Actions = append(d.Actions, d.Actions[0])
			case "self":
				d.Actions[0].Dependencies = []string{"a"}
			case "unknown-dependency":
				d.Actions[0].Dependencies = []string{"b"}
			case "cycle":
				b := d.Actions[0]
				b.ID = "b"
				b.Dependencies = []string{"a"}
				d.Actions[0].Dependencies = []string{"b"}
				d.Actions = append(d.Actions, b)
			case "retry":
				d.Actions[0].Retry = "unknown"
			case "success-without-evidence":
				d.Actions[0].State = Succeeded
			case "failed-category":
				d.Actions[0].State = Failed
				d.Actions[0].Failure = "unsafe"
			case "pending-exit":
				d.Actions[0].ExitCode = 0
			case "nil-actions":
				d.Actions = nil
			}
			if d.Validate() == nil {
				t.Fatal("accepted invalid model")
			}
		})
	}
}
func TestTransitions(t *testing.T) {
	p, s, _ := initialized(t)
	if e := s.Finish(context.Background(), trace(), obs(Satisfied)); !errors.Is(e, ErrTransition) {
		t.Fatal(e)
	}
	if e := s.Begin("unknown"); !errors.Is(e, ErrTransition) {
		t.Fatal(e)
	}
	if e := s.Begin("a"); e != nil {
		t.Fatal(e)
	}
	if mustRead(t, p).Actions[0].State != Running {
		t.Fatal("intent not durable")
	}
	if e := s.Begin("a"); !errors.Is(e, ErrTransition) {
		t.Fatal(e)
	}
	if e := s.Finish(context.Background(), trace(), obs(Satisfied)); e != nil {
		t.Fatal(e)
	}
	if mustRead(t, p).Actions[0].State != Succeeded {
		t.Fatal("success not durable")
	}
	if e := s.Begin("a"); !errors.Is(e, ErrTransition) {
		t.Fatal("completed action replayed")
	}
	snap, _ := s.Snapshot()
	snap.Actions[0].State = Pending
	snap.Actions[0].Postconditions[0] = "changed"
	again, _ := s.Snapshot()
	if again.Actions[0].State != Succeeded || again.Actions[0].Postconditions[0] != "installed" {
		t.Fatal("snapshot aliases internal state")
	}
}
func TestFinishRequiresObservation(t *testing.T) {
	for _, v := range []Observed{Absent, Unknown} {
		t.Run(string(v), func(t *testing.T) {
			p, s, _ := initialized(t)
			if e := s.Begin("a"); e != nil {
				t.Fatal(e)
			}
			if e := s.Finish(context.Background(), trace(), obs(v)); !errors.Is(e, ErrReconcile) {
				t.Fatal(e)
			}
			if mustRead(t, p).Actions[0].State != ReconciliationRequired {
				t.Fatal("unverified success")
			}
			if !errors.Is(s.Begin("a"), ErrReconcile) {
				t.Fatal("recovery bypassed")
			}
		})
	}
}
func TestFailureExcludesDiagnostics(t *testing.T) {
	p, s, _ := initialized(t)
	if e := s.Begin("a"); e != nil {
		t.Fatal(e)
	}
	tr := trace()
	tr.Outcome = "failed"
	tr.Result.Stdout = "ARTIFICIAL_SECRET"
	tr.Result.Stderr = "ARTIFICIAL_SECRET"
	tr.Failure = &execution.Error{Category: execution.Exit, Detail: "ARTIFICIAL_SECRET", ExitCode: 9}
	if e := s.Finish(context.Background(), tr, nil); e != nil {
		t.Fatal(e)
	}
	b, _ := mustRead(t, p).JSON()
	if bytes.Contains(b, []byte("ARTIFICIAL_SECRET")) {
		t.Fatal("raw diagnostics persisted")
	}
}
func TestResumeIdentity(t *testing.T) {
	for _, field := range []string{"host", "host-state", "config", "manifest", "plan", "policy", "graph", "metadata"} {
		t.Run(field, func(t *testing.T) {
			p, s, d := initialized(t)
			s.Close()
			switch field {
			case "host":
				d.Identity.Host = hash("changed")
			case "host-state":
				d.Identity.HostState = hash("changed")
			case "config":
				d.Identity.Config = hash("changed")
			case "manifest":
				d.Identity.Manifest = hash("changed")
			case "plan":
				d.Identity.Plan = hash("changed")
			case "policy":
				d.Identity.Policy = hash("changed")
			case "graph":
				d.Identity.Graph = hash("changed")
			case "metadata":
				d.Actions[0].Retry = Idempotent
			}
			r, e := Acquire(p)
			if e != nil {
				t.Fatal(e)
			}
			defer r.Close()
			called := false
			o := observerFunc(func(context.Context, Record) (Observation, error) { called = true; return Observation{}, nil })
			before, _ := os.ReadFile(filepath.Join(p, journalName))
			if e = r.Resume(context.Background(), d, o); !errors.Is(e, ErrIdentity) {
				t.Fatal(e)
			}
			after, _ := os.ReadFile(filepath.Join(p, journalName))
			if called || !bytes.Equal(before, after) {
				t.Fatal("identity mismatch performed reconciliation or journal mutation")
			}
		})
	}
}
func TestReconciliation(t *testing.T) {
	cases := []struct {
		name     string
		prior    State
		retry    Retry
		observed Observed
		want     State
		blocked  bool
	}{
		{"completed-valid", Succeeded, Never, Satisfied, Succeeded, false}, {"completed-drift", Succeeded, Idempotent, Absent, ReconciliationRequired, true},
		{"interrupted-complete", Running, Never, Satisfied, Succeeded, false}, {"interrupted-nonidempotent", Running, Never, Absent, ReconciliationRequired, true},
		{"interrupted-safe-retry", Running, Idempotent, Absent, Pending, false}, {"interrupted-unknown", Running, Idempotent, Unknown, ReconciliationRequired, true},
		{"failed-complete", Failed, Never, Satisfied, Succeeded, false}, {"failed-unsafe", Failed, Never, Absent, ReconciliationRequired, true},
		{"recovery-unknown", ReconciliationRequired, Idempotent, Unknown, ReconciliationRequired, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := stateDir(t)
			s, e := Acquire(p)
			if e != nil {
				t.Fatal(e)
			}
			d := fixture(t)
			d.Actions[0].Retry = tc.retry
			if e = s.Initialize(d); e != nil {
				t.Fatal(e)
			}
			current := clone(d)
			a := &current.Actions[0]
			a.State = tc.prior
			if tc.prior == Succeeded {
				a.Evidence = hash("prior")
				a.ExitCode = 0
			}
			if tc.prior == Failed {
				a.Failure = execution.Timeout
			}
			if e = s.persist(current); e != nil {
				t.Fatal(e)
			}
			s.Close()
			s, e = Acquire(p)
			if e != nil {
				t.Fatal(e)
			}
			defer s.Close()
			e = s.Resume(context.Background(), d, obs(tc.observed))
			if tc.blocked != errors.Is(e, ErrReconcile) || (e != nil && !tc.blocked) {
				t.Fatal(e)
			}
			if got := mustRead(t, p).Actions[0].State; got != tc.want {
				t.Fatalf("got %s want %s", got, tc.want)
			}
		})
	}
}
func TestObserverFailureSafe(t *testing.T) {
	for _, mode := range []string{"error", "invalid-evidence", "invalid-state", "canceled", "mutates-record"} {
		t.Run(mode, func(t *testing.T) {
			p, s, d := initialized(t)
			if e := s.Begin("a"); e != nil {
				t.Fatal(e)
			}
			s.Close()
			s, e := Acquire(p)
			if e != nil {
				t.Fatal(e)
			}
			defer s.Close()
			o := observerFunc(func(ctx context.Context, r Record) (Observation, error) {
				switch mode {
				case "error":
					return Observation{}, errors.New("ARTIFICIAL_SECRET")
				case "invalid-evidence":
					return Observation{Satisfied, "bad"}, nil
				case "invalid-state":
					return Observation{"bogus", hash("x")}, nil
				case "mutates-record":
					r.Postconditions[0] = "modified"
				}
				return Observation{Unknown, hash("x")}, nil
			})
			ctx := context.Background()
			if mode == "canceled" {
				c, cancel := context.WithCancel(ctx)
				cancel()
				ctx = c
			}
			e = s.Resume(ctx, d, o)
			if e == nil || strings.Contains(e.Error(), "ARTIFICIAL") {
				t.Fatal(e)
			}
			if mode != "canceled" && mustRead(t, p).Actions[0].Postconditions[0] != "installed" {
				t.Fatal("callback mutated record")
			}
		})
	}
}
func TestFaultBoundaries(t *testing.T) {
	for _, op := range []string{"create", "write", "file_sync", "close", "rename", "directory_sync", "committed"} {
		t.Run(op, func(t *testing.T) {
			p, s, _ := initialized(t)
			before, _ := os.ReadFile(filepath.Join(p, journalName))
			s.hook = func(at string) error {
				if at == op {
					return syscall.EIO
				}
				return nil
			}
			e := s.Begin("a")
			var pe *PersistenceError
			if !errors.As(e, &pe) || pe.Operation != op {
				t.Fatal(e)
			}
			uncertain := op == "directory_sync" || op == "committed"
			if pe.Uncertain != uncertain {
				t.Fatal("wrong durability classification")
			}
			if !errors.Is(s.Begin("a"), ErrClosed) {
				t.Fatal("failed session reused")
			}
			after, _ := os.ReadFile(filepath.Join(p, journalName))
			if !uncertain && !bytes.Equal(before, after) {
				t.Fatal("last valid journal lost")
			}
			if _, e = Decode(bytes.NewReader(after)); e != nil {
				t.Fatal("partial journal visible")
			}
			entries, _ := os.ReadDir(p)
			for _, v := range entries {
				if strings.HasSuffix(v.Name(), ".tmp") {
					t.Fatal("failed staging file retained")
				}
			}
		})
	}
}
func TestWriteFailures(t *testing.T) {
	for _, mode := range []string{"short", "disk-full"} {
		t.Run(mode, func(t *testing.T) {
			p, s, _ := initialized(t)
			before, _ := os.ReadFile(filepath.Join(p, journalName))
			s.write = func(f *os.File, b []byte) (int, error) {
				n, _ := f.Write(b[:8])
				if mode == "disk-full" {
					return n, syscall.ENOSPC
				}
				return n, nil
			}
			var pe *PersistenceError
			if e := s.Begin("a"); !errors.As(e, &pe) || pe.Uncertain {
				t.Fatal(e)
			}
			after, _ := os.ReadFile(filepath.Join(p, journalName))
			if !bytes.Equal(before, after) {
				t.Fatal("partial write replaced journal")
			}
		})
	}
}
func TestCorruptionNeverReset(t *testing.T) {
	p, s, d := initialized(t)
	s.Close()
	bad := []byte(`{"schema_version":1`)
	if e := os.WriteFile(filepath.Join(p, journalName), bad, 0600); e != nil {
		t.Fatal(e)
	}
	s, e := Acquire(p)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if !errors.Is(s.Initialize(d), ErrInvalid) || !errors.Is(s.Resume(context.Background(), d, obs(Satisfied)), ErrInvalid) {
		t.Fatal("corrupt journal accepted")
	}
	got, _ := os.ReadFile(filepath.Join(p, journalName))
	if !bytes.Equal(got, bad) {
		t.Fatal("corrupt journal reset")
	}
}
func TestReadOnlyInspection(t *testing.T) {
	p := stateDir(t)
	if _, e := Inspect(p); !errors.Is(e, os.ErrNotExist) {
		t.Fatal(e)
	}
	if _, e := os.Stat(p); !errors.Is(e, os.ErrNotExist) {
		t.Fatal("inspection created directory")
	}
	if e := os.Mkdir(p, 0700); e != nil {
		t.Fatal(e)
	}
	if _, e := Inspect(p); !errors.Is(e, os.ErrNotExist) {
		t.Fatal(e)
	}
	entries, _ := os.ReadDir(p)
	if len(entries) != 0 {
		t.Fatal("read-only created state")
	}
}
func TestStableLock(t *testing.T) {
	p, s, _ := initialized(t)
	a, e := os.Stat(filepath.Join(p, lockName))
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Begin("a"); e != nil {
		t.Fatal(e)
	}
	b, _ := os.Stat(filepath.Join(p, lockName))
	if !os.SameFile(a, b) {
		t.Fatal("lock replaced")
	}
	s.Close()
	r, e := Acquire(p)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	c, _ := os.Stat(filepath.Join(p, lockName))
	if !os.SameFile(a, c) {
		t.Fatal("lock recreated")
	}
}
func TestUnsafeFiles(t *testing.T) {
	for _, mode := range []string{"relative", "directory-symlink", "ancestor-symlink", "directory-mode", "lock-symlink", "lock-hardlink", "lock-directory", "lock-fifo", "lock-mode", "journal-symlink", "journal-mode"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			p := filepath.Join(root, "state")
			if e := os.Mkdir(p, 0700); e != nil {
				t.Fatal(e)
			}
			switch mode {
			case "relative":
				p = "relative"
			case "directory-symlink":
				q := filepath.Join(root, "alias")
				os.Symlink(p, q)
				p = q
			case "ancestor-symlink":
				q := filepath.Join(root, "alias")
				os.Symlink(root, q)
				p = filepath.Join(q, "state")
			case "directory-mode":
				os.Chmod(p, 0755)
			case "lock-symlink":
				os.Symlink(filepath.Join(root, "other"), filepath.Join(p, lockName))
			case "lock-hardlink":
				q := filepath.Join(root, "other")
				os.WriteFile(q, nil, 0600)
				os.Link(q, filepath.Join(p, lockName))
			case "lock-directory":
				os.Mkdir(filepath.Join(p, lockName), 0600)
			case "lock-fifo":
				syscall.Mkfifo(filepath.Join(p, lockName), 0600)
			case "lock-mode":
				os.WriteFile(filepath.Join(p, lockName), nil, 0644)
			case "journal-symlink":
				os.Symlink(filepath.Join(root, "other"), filepath.Join(p, journalName))
			case "journal-mode":
				os.WriteFile(filepath.Join(p, journalName), nil, 0644)
			}
			s, e := Acquire(p)
			if strings.HasPrefix(mode, "journal-") {
				if e != nil {
					t.Fatal(e)
				}
				defer s.Close()
				e = s.Initialize(fixture(t))
			} else if s != nil {
				s.Close()
			}
			if !errors.Is(e, ErrUnsafe) {
				t.Fatal("unsafe path accepted", e)
			}
		})
	}
}
func TestLockReplacementDetected(t *testing.T) {
	p, s, _ := initialized(t)
	if e := os.Rename(filepath.Join(p, lockName), filepath.Join(p, "displaced")); e != nil {
		t.Fatal(e)
	}
	os.WriteFile(filepath.Join(p, lockName), nil, 0600)
	if !errors.Is(s.Begin("a"), ErrUnsafe) {
		t.Fatal("replaced lock accepted")
	}
}

// Helper processes communicate through inherited pipes; only isolated test state
// is written. EOF on stdin releases a healthy owner. os.Exit simulates a crash.
func TestProcessHelper(t *testing.T) {
	mode := os.Getenv("ALMADIAL_JOURNAL_HELPER")
	if mode == "" {
		return
	}
	p := os.Getenv("ALMADIAL_JOURNAL_PATH")
	s, e := Acquire(p)
	if errors.Is(e, ErrBusy) {
		os.Exit(23)
	}
	if e != nil {
		os.Exit(24)
	}
	defer s.Close()
	if mode == "contend" {
		os.Exit(25)
	}
	if mode == "crash-update" {
		d := fixture(t)
		if e = s.Resume(context.Background(), d, obs(Satisfied)); e != nil {
			os.Exit(26)
		}
		boundary := os.Getenv("ALMADIAL_JOURNAL_BOUNDARY")
		if boundary == "partial-write" {
			s.write = func(f *os.File, b []byte) (int, error) { _, _ = f.Write(b[:len(b)/2]); os.Exit(42); return 0, nil }
		}
		s.hook = func(at string) error {
			if at == boundary {
				os.Exit(42)
			}
			return nil
		}
		if e = s.Begin("a"); e != nil {
			os.Exit(27)
		}
		os.Exit(28)
	}
	fmt.Println("locked")
	var b [1]byte
	_, _ = os.Stdin.Read(b[:])
	if mode == "crash" {
		os.Exit(41)
	}
}
func helper(t *testing.T, p, mode string) (*exec.Cmd, io.WriteCloser, io.ReadCloser) {
	t.Helper()
	cmd := boundedHelper(t, os.Args[0], "-test.run=^TestProcessHelper$")
	cmd.Env = append(os.Environ(), "ALMADIAL_JOURNAL_HELPER="+mode, "ALMADIAL_JOURNAL_PATH="+p)
	in, e := cmd.StdinPipe()
	if e != nil {
		t.Fatal(e)
	}
	out, e := cmd.StdoutPipe()
	if e != nil {
		t.Fatal(e)
	}
	if e = cmd.Start(); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { in.Close(); cmd.Process.Kill(); cmd.Wait() })
	return cmd, in, out
}
func boundedHelper(t *testing.T, binary string, args ...string) *exec.Cmd {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.WaitDelay = time.Second
	return cmd
}
func waitCode(cmd *exec.Cmd) int {
	e := cmd.Wait()
	if e == nil {
		return 0
	}
	var x *exec.ExitError
	if errors.As(e, &x) {
		return x.ExitCode()
	}
	return -1
}
func TestCrossProcessLock(t *testing.T) {
	for _, mode := range []string{"normal", "crash", "kill"} {
		t.Run(mode, func(t *testing.T) {
			p, s, _ := initialized(t)
			s.Close()
			before, _ := os.ReadFile(filepath.Join(p, journalName))
			cmdMode := mode
			if mode == "kill" {
				cmdMode = "normal"
			}
			cmd, in, out := helper(t, p, cmdMode)
			ready := make(chan bool, 1)
			go func() {
				b := make([]byte, len("locked\n"))
				_, e := io.ReadFull(out, b)
				ready <- e == nil && string(b) == "locked\n"
			}()
			select {
			case ok := <-ready:
				if !ok {
					t.Fatal("owner failed")
				}
			case <-time.After(10 * time.Second):
				t.Fatal("owner timed out")
			}
			competitor := boundedHelper(t, os.Args[0], "-test.run=^TestProcessHelper$")
			competitor.Env = append(os.Environ(), "ALMADIAL_JOURNAL_HELPER=contend", "ALMADIAL_JOURNAL_PATH="+p)
			if e := competitor.Start(); e != nil {
				t.Fatal(e)
			}
			if code := waitCode(competitor); code != 23 {
				t.Fatal("competitor acquired lock", code)
			}
			after, _ := os.ReadFile(filepath.Join(p, journalName))
			if !bytes.Equal(before, after) {
				t.Fatal("loser mutated journal")
			}
			if mode == "kill" {
				cmd.Process.Kill()
			} else {
				in.Close()
			}
			code := waitCode(cmd)
			if mode == "normal" && code != 0 || mode == "crash" && code != 41 {
				t.Fatal("owner outcome", code)
			}
			next, e := Acquire(p)
			if e != nil {
				t.Fatal("kernel retained lock", e)
			}
			next.Close()
			if _, e = Inspect(p); e != nil {
				t.Fatal(e)
			}
		})
	}
}
func TestProcessCrashPersistence(t *testing.T) {
	for _, boundary := range []string{"create", "partial-write", "write", "file_sync", "close", "rename", "directory_sync", "committed"} {
		t.Run(boundary, func(t *testing.T) {
			p, s, _ := initialized(t)
			s.Close()
			cmd := boundedHelper(t, os.Args[0], "-test.run=^TestProcessHelper$")
			cmd.Env = append(os.Environ(), "ALMADIAL_JOURNAL_HELPER=crash-update", "ALMADIAL_JOURNAL_PATH="+p, "ALMADIAL_JOURNAL_BOUNDARY="+boundary)
			if e := cmd.Start(); e != nil {
				t.Fatal(e)
			}
			if code := waitCode(cmd); code != 42 {
				t.Fatal("fault not reached", code)
			}
			d := mustRead(t, p)
			want := Pending
			if boundary == "directory_sync" || boundary == "committed" {
				want = Running
			}
			if d.Actions[0].State != want {
				t.Fatal("unexpected crash state", d.Actions[0].State)
			}
			// Leftover staging files are ignored; only journal.json can supply progress.
			next, e := Acquire(p)
			if e != nil {
				t.Fatal(e)
			}
			defer next.Close()
			expected := fixture(t)
			e = next.Resume(context.Background(), expected, obs(Unknown))
			if want == Running && !errors.Is(e, ErrReconcile) {
				t.Fatal("interrupted action assumed complete", e)
			}
			if want == Pending && e != nil {
				t.Fatal(e)
			}
		})
	}
}
func TestEmptyGraph(t *testing.T) {
	g, e := execution.NewGraph(nil)
	if e != nil {
		t.Fatal(e)
	}
	d := fixture(t)
	d.Identity.Graph = g.Digest()
	d, e = New(d.Identity, g, map[string]Retry{})
	if e != nil || d.Validate() != nil {
		t.Fatal(e)
	}
}
func TestFailedDependencyBlocksBegin(t *testing.T) {
	p, s, d := initialized(t)
	_ = p
	b := d.Actions[0]
	b.ID = "b"
	b.Dependencies = []string{"a"}
	d.Actions = append(d.Actions, b)
	if e := s.persist(d); e != nil {
		t.Fatal(e)
	}
	if !errors.Is(s.Begin("b"), ErrTransition) {
		t.Fatal("dependency ignored")
	}
}
func TestReadErrorRedaction(t *testing.T) {
	_, e := Decode(errorReader{})
	if !errors.Is(e, ErrInvalid) || strings.Contains(e.Error(), "ARTIFICIAL") {
		t.Fatal(e)
	}
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, errors.New("ARTIFICIAL_SECRET") }
func TestNewRequiresGraphPolicy(t *testing.T) {
	d := fixture(t)
	if _, e := New(d.Identity, execution.Graph{}, nil); e == nil {
		t.Fatal("empty graph identity accepted")
	}
}
func TestJSONShapeTypes(t *testing.T) {
	d := fixture(t)
	b, _ := d.JSON()
	for _, v := range []any{nil, []any{}, "x", 1} {
		raw, _ := json.Marshal(v)
		if _, e := Decode(bytes.NewReader(raw)); e == nil {
			t.Fatal("non-object accepted")
		}
	}
	if _, e := Decode(bytes.NewReader(b)); e != nil {
		t.Fatal(e)
	}
}

func TestReadOnlyFoundationCreatesNoState(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	ref := config.SecretRef{Type: config.Environment, Source: "FIXTURE_SOURCE"}
	c := config.Defaults(config.Config{SchemaVersion: 1, Network: config.Network{BindAddress: netip.MustParseAddr("192.0.2.10")}, Credentials: config.Credentials{Database: ref, AMI: ref, SIP: ref, Admin: ref}})
	if _, e := config.Digest(c); e != nil {
		t.Fatal(e)
	}
	s := preflight.Snapshot{HostID: strings.Repeat("a", 32), OS: "almalinux", Release: "10.2", Kernel: "fixture", Architecture: "amd64", CPUs: 4, CPUFlags: []string{"sse2"}, TotalMemory: 8192, AvailableMemory: 4096, Disks: []preflight.Disk{{Path: "/", Total: 100000, Available: 90000}}, Addresses: []netip.Addr{c.Network.BindAddress}, SELinux: "Enforcing", Firewall: "empty nftables ruleset", TimeSync: "yes", Known: map[string]bool{}}
	for _, k := range preflight.RequiredProbes() {
		s.Known[k] = true
	}
	r := preflight.Requirements{Profile: "fixture-budget", MinCPUs: 1, CPUFlags: []string{"sse2"}, MinTotalMemory: 1024, MinAvailableMemory: 512, MinDiskAvailable: 4096}
	if _, e := preflight.Assess(c, s, r); e != nil {
		t.Fatal(e)
	}
	p, e := plan.New(c, s, r, hash("manifest"))
	if e != nil || !p.Eligible() {
		t.Fatal("fixture plan blocked", e)
	}
	if e = p.Revalidate(c, s, r, hash("manifest")); e != nil {
		t.Fatal(e)
	}
	if _, e = Inspect(filepath.Join(root, "state")); !errors.Is(e, os.ErrNotExist) {
		t.Fatal(e)
	}
	entries, e := os.ReadDir(root)
	if e != nil || len(entries) != 0 {
		t.Fatal("read-only foundation wrote state", entries, e)
	}
}
func TestDependencyDriftPropagates(t *testing.T) {
	p, s, d := initialized(t)
	b := d.Actions[0]
	b.ID = "b"
	b.Dependencies = []string{"a"}
	d.Actions = append(d.Actions, b)
	current := clone(d)
	for n := range current.Actions {
		current.Actions[n].State = Succeeded
		current.Actions[n].Evidence = hash("prior")
		current.Actions[n].ExitCode = 0
	}
	if e := s.persist(current); e != nil {
		t.Fatal(e)
	}
	s.Close()
	s, e := Acquire(p)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	o := observerFunc(func(_ context.Context, a Record) (Observation, error) {
		v := Satisfied
		if a.ID == "a" {
			v = Absent
		}
		return Observation{v, hash("fresh")}, nil
	})
	if e = s.Resume(context.Background(), d, o); !errors.Is(e, ErrReconcile) {
		t.Fatal(e)
	}
	for _, a := range mustRead(t, p).Actions {
		if a.State != ReconciliationRequired {
			t.Fatal("dependent success survived prerequisite drift")
		}
	}
}
func TestInvalidTraceAndSession(t *testing.T) {
	p, s, d := initialized(t)
	if !errors.Is(s.Initialize(d), ErrTransition) {
		t.Fatal("duplicate init accepted")
	}
	if e := s.Begin("a"); e != nil {
		t.Fatal(e)
	}
	for _, mode := range []string{"wrong-stage", "wrong-action", "no-process", "nonzero-success", "bad-outcome", "invalid-failure"} {
		t.Run(mode, func(t *testing.T) {
			tr := trace()
			switch mode {
			case "wrong-stage":
				tr.StageID = "S02"
			case "wrong-action":
				tr.ActionID = "other"
			case "no-process":
				tr.Result.Started = false
			case "nonzero-success":
				tr.Result.ExitCode = 1
			case "bad-outcome":
				tr.Outcome = "bogus"
			case "invalid-failure":
				tr.Outcome = "failed"
				tr.Failure = &execution.Error{Category: "unsafe"}
			}
			if e := s.Finish(context.Background(), tr, obs(Satisfied)); e == nil {
				t.Fatal("invalid trace accepted")
			}
			if mustRead(t, p).Actions[0].State != Running {
				t.Fatal("invalid trace changed progress")
			}
		})
	}
	s.Close()
	if !errors.Is(s.Begin("a"), ErrClosed) {
		t.Fatal("closed session accepted")
	}
	if _, e := s.Snapshot(); !errors.Is(e, ErrClosed) {
		t.Fatal(e)
	}
}
func TestUninitializedAndInvalidDirectory(t *testing.T) {
	p := stateDir(t)
	s, e := Acquire(p)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if !errors.Is(s.Begin("a"), ErrTransition) {
		t.Fatal("began without journal")
	}
	if _, e = s.Snapshot(); !errors.Is(e, ErrInvalid) {
		t.Fatal(e)
	}
	if _, e = Acquire(filepath.Join(t.TempDir(), "missing-parent", "state")); e == nil {
		t.Fatal("created undeclared parents")
	}
}
func TestInitialPersistenceFailure(t *testing.T) {
	p := stateDir(t)
	s, e := Acquire(p)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	s.hook = func(string) error { return syscall.EACCES }
	if s.Initialize(fixture(t)) == nil {
		t.Fatal("failure ignored")
	}
	if _, e = Inspect(p); !errors.Is(e, os.ErrNotExist) {
		t.Fatal("invalid initial state published", e)
	}
}
func TestInspectIgnoresStagingFiles(t *testing.T) {
	p, s, _ := initialized(t)
	s.Close()
	os.WriteFile(filepath.Join(p, ".journal-orphan.tmp"), []byte("truncated"), 0600)
	if _, e := Inspect(p); e != nil {
		t.Fatal("staging used as journal", e)
	}
}

func TestRepeatedResumeCannotEraseDrift(t *testing.T) {
	p, s, d := initialized(t)
	d.Actions[0].Retry = Idempotent
	current := clone(d)
	current.Actions[0].State = Succeeded
	current.Actions[0].Evidence = hash("prior")
	current.Actions[0].ExitCode = 0
	if e := s.persist(current); e != nil {
		t.Fatal(e)
	}
	s.Close()
	for n := 0; n < 2; n++ {
		r, e := Acquire(p)
		if e != nil {
			t.Fatal(e)
		}
		if e = r.Resume(context.Background(), d, obs(Absent)); !errors.Is(e, ErrReconcile) {
			t.Fatal("drift became retryable", e)
		}
		if !errors.Is(r.Begin("a"), ErrReconcile) {
			t.Fatal("drift replayed")
		}
		r.Close()
	}
}

func TestAtomicReadersAndSequence(t *testing.T) {
	p, s, d := initialized(t)
	var events []string
	s.hook = func(op string) error { events = append(events, op); return nil }
	if e := s.Begin("a"); e != nil {
		t.Fatal(e)
	}
	if strings.Join(events, ",") != "create,write,file_sync,close,rename,directory_sync,committed" {
		t.Fatal("durability operation order", events)
	}
	for _, name := range []string{lockName, journalName} {
		st, e := os.Stat(filepath.Join(p, name))
		if e != nil || st.Mode().Perm() != 0600 {
			t.Fatal("private mode missing", name, e)
		}
	}
	s.hook = nil
	stop := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		for {
			select {
			case <-stop:
				done <- nil
				return
			default:
				if _, e := Inspect(p); e != nil {
					done <- e
					return
				}
			}
		}
	}()
	for n := 0; n < 30; n++ {
		if e := s.persist(d); e != nil {
			close(stop)
			<-done
			t.Fatal(e)
		}
	}
	close(stop)
	if e := <-done; e != nil {
		t.Fatal("reader saw partial replacement", e)
	}
}
