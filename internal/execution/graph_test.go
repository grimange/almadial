package execution

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

type fakeChecks struct {
	fail   string
	known  bool
	calls  []string
	cancel context.CancelFunc
}

func (f *fakeChecks) Supports(string) bool { return f.known }
func (f *fakeChecks) Check(ctx context.Context, id string) error {
	f.calls = append(f.calls, id)
	if f.cancel != nil {
		f.cancel()
	}
	if id == f.fail {
		return errors.New("artificial-private-diagnostic")
	}
	return nil
}

type fakeRunner struct {
	calls     int
	err       error
	badResult bool
}

func (f *fakeRunner) Validate(Command) error { return nil }
func (f *fakeRunner) Run(context.Context, Command) (Result, error) {
	f.calls++
	if f.badResult {
		return Result{}, nil
	}
	return Result{Started: true, ExitCode: 0}, f.err
}
func action(t *testing.T, id string, deps ...string) Action {
	t.Helper()
	return Action{ID: id, StageID: "S01", Description: "Controlled test action", Dependencies: deps, Preconditions: []string{"eligible"}, Command: helper(t, "empty"), Postconditions: []string{"healthy"}}
}
func ids(g Graph) []string {
	out := []string{}
	for _, a := range g.Actions() {
		out = append(out, a.ID)
	}
	return out
}
func TestGraphValidation(t *testing.T) {
	for _, tt := range []struct {
		name    string
		actions []Action
		want    []string
		bad     bool
	}{
		{"empty", nil, []string{}, false}, {"independent", []Action{action(t, "b"), action(t, "a")}, []string{"a", "b"}, false},
		{"chain", []Action{action(t, "a", "b"), action(t, "b", "c"), action(t, "c")}, []string{"c", "b", "a"}, false},
		{"branch", []Action{action(t, "d", "a", "c"), action(t, "c", "b"), action(t, "b"), action(t, "a")}, []string{"a", "b", "c", "d"}, false},
		{"duplicate", []Action{action(t, "a"), action(t, "a")}, nil, true}, {"unknown", []Action{action(t, "a", "b")}, nil, true},
		{"self", []Action{action(t, "a", "a")}, nil, true}, {"cycle", []Action{action(t, "a", "b"), action(t, "b", "a")}, nil, true},
		{"duplicate-dependency", []Action{action(t, "a"), action(t, "b", "a", "a")}, nil, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g, err := NewGraph(tt.actions)
			if tt.bad {
				category(t, err, Contract)
				return
			}
			if err != nil || !reflect.DeepEqual(ids(g), tt.want) {
				t.Fatalf("wrong graph: %v %v", ids(g), err)
			}
		})
	}
	for _, field := range []string{"id", "stage", "description", "pre", "post", "command"} {
		t.Run(field, func(t *testing.T) {
			a := action(t, "a")
			switch field {
			case "id":
				a.ID = "invalid\nsecret"
			case "stage":
				a.StageID = ""
			case "description":
				a.Description = ""
			case "pre":
				a.Preconditions = nil
			case "post":
				a.Postconditions = nil
			case "command":
				a.Command.Timeout = 0
			}
			_, err := NewGraph([]Action{a})
			category(t, err, Contract)
		})
	}
}
func TestGraphIdentityAndImmutability(t *testing.T) {
	input := []Action{action(t, "b", "a"), action(t, "a")}
	g, err := NewGraph(input)
	if err != nil {
		t.Fatal(err)
	}
	before := string(g.JSON())
	digest := g.Digest()
	input[0].Dependencies[0] = "changed"
	input[0].Command.Args[0] = "changed"
	copy := g.Actions()
	copy[0].ID = "changed"
	copy[0].Command.Args[0] = "changed"
	raw := g.JSON()
	raw[0] = '!'
	if string(g.JSON()) != before || g.Digest() != digest {
		t.Fatal("graph is mutable")
	}
	original := []Action{action(t, "a"), action(t, "b", "a")}
	g2, err := NewGraph(original)
	if err != nil || g2.Digest() != digest {
		t.Fatal("declaration order changes identity")
	}
	a := action(t, "a")
	a.Preconditions = []string{"z", "a"}
	g1, _ := NewGraph([]Action{a})
	a.Preconditions = []string{"a", "z"}
	g2, _ = NewGraph([]Action{a})
	if g1.Digest() != g2.Digest() {
		t.Fatal("assertion order not normalized")
	}
}
func TestGraphExecution(t *testing.T) {
	for _, tt := range []struct {
		name, fail string
		runnerErr  error
		kind       Category
		calls      int
	}{
		{"success", "", nil, "", 2}, {"pre", "eligible", nil, Precondition, 1}, {"post", "healthy", nil, Postcondition, 2},
		{"process", "", failure(Exit), Exit, 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			a := action(t, "a")
			b := action(t, "b", "a")
			c := action(t, "c")
			c.Preconditions = []string{"independent-pre"}
			c.Postconditions = []string{"independent-post"}
			g, _ := NewGraph([]Action{c, b, a})
			r := &fakeRunner{err: tt.runnerErr}
			checks := &fakeChecks{known: true, fail: tt.fail}
			traces, err := g.Execute(context.Background(), r, checks)
			if tt.kind == "" {
				if err != nil {
					t.Fatal(err)
				}
				if traces[1].Outcome != "succeeded" {
					t.Fatal("dependent not run")
				}
				if r.calls != 3 {
					t.Fatal("actions missing")
				}
			} else {
				e := category(t, err, tt.kind)
				if e.ActionID != "a" || e.StageID != "S01" {
					t.Fatal("error identity lost")
				}
				if traces[1].Outcome != "blocked" || traces[1].Failure.Category != Dependency {
					t.Fatal("dependent ran")
				}
				if r.calls != tt.calls {
					t.Fatal("wrong runner count")
				}
				if strings.Contains(err.Error(), "artificial-private") {
					t.Fatal("predicate diagnostic leaked")
				}
			}
			if traces[0].ActionID != "a" || traces[1].ActionID != "b" || traces[2].ActionID != "c" {
				t.Fatal("unstable trace")
			}
		})
	}
}
func TestGraphRejectsUnboundAndUnexpectedResults(t *testing.T) {
	g, _ := NewGraph([]Action{action(t, "a")})
	r := &fakeRunner{}
	traces, err := g.Execute(context.Background(), r, &fakeChecks{})
	category(t, err, Contract)
	if r.calls != 0 || len(traces) != 0 {
		t.Fatal("ran before assertions bound")
	}
	r = &fakeRunner{badResult: true}
	_, err = g.Execute(context.Background(), r, &fakeChecks{known: true})
	category(t, err, Internal)
	r = &fakeRunner{err: errors.New("unsafe-diagnostic")}
	_, err = g.Execute(context.Background(), r, &fakeChecks{known: true})
	category(t, err, Internal)
	if strings.Contains(err.Error(), "unsafe") {
		t.Fatal("untrusted runner error leaked")
	}
	_, err = (Graph{}).Execute(context.Background(), r, &fakeChecks{known: true})
	category(t, err, Contract)
}
func TestGraphCancellation(t *testing.T) {
	g, _ := NewGraph([]Action{action(t, "a"), action(t, "b", "a"), action(t, "c")})
	ctx, cancel := context.WithCancel(context.Background())
	r := &fakeRunner{}
	traces, err := g.Execute(ctx, r, &fakeChecks{known: true, cancel: cancel})
	category(t, err, Canceled)
	if r.calls != 0 || traces[1].Failure.Category != Dependency || traces[2].Failure.Category != Canceled {
		t.Fatal("canceled graph executed")
	}
}
func TestGraphWithRealRunner(t *testing.T) {
	a := action(t, "a")
	a.Command = helper(t, "ok")
	g, _ := NewGraph([]Action{a})
	r, _ := NewRunner(nil)
	traces, err := g.Execute(context.Background(), r, &fakeChecks{known: true})
	if err != nil || traces[0].Outcome != "succeeded" || traces[0].Result.Stdout != "hello" {
		t.Fatal("real integration failed")
	}
}

func TestAllCommandsValidatedBeforeExecution(t *testing.T) {
	a, b := action(t, "a"), action(t, "b")
	b.Command.Args = append(b.Command.Args, "registered-sensitive-value")
	g, _ := NewGraph([]Action{b, a})
	r, _ := NewRunner([]string{"registered-sensitive-value"})
	checks := &fakeChecks{known: true}
	trace, err := g.Execute(context.Background(), r, checks)
	e := category(t, err, Contract)
	if e.ActionID != "b" || len(trace) != 0 || len(checks.calls) != 0 {
		t.Fatal("validation was deferred until after earlier execution")
	}
}
func TestDeterministicTraceEvidence(t *testing.T) {
	a, b := action(t, "a"), action(t, "b", "a")
	a.Command.Executable = "/usr/bin/true"
	a.Command.Args = nil
	b.Command = a.Command
	g, _ := NewGraph([]Action{b, a})
	one, err := g.Execute(context.Background(), &fakeRunner{}, &fakeChecks{known: true})
	if err != nil {
		t.Fatal(err)
	}
	two, err := g.Execute(context.Background(), &fakeRunner{}, &fakeChecks{known: true})
	if err != nil || !reflect.DeepEqual(one, two) {
		t.Fatal("trace nondeterminism")
	}
	t.Logf("synthetic graph SHA-256: %s; canonical metadata: %s", g.Digest(), g.JSON())
}

type deadlineChecks struct{}

func (deadlineChecks) Supports(string) bool                      { return true }
func (deadlineChecks) Check(ctx context.Context, _ string) error { <-ctx.Done(); return ctx.Err() }
func TestAssertionDeadline(t *testing.T) {
	a := action(t, "a")
	a.Command.Timeout = 10 * time.Millisecond
	g, _ := NewGraph([]Action{a})
	r := &fakeRunner{}
	_, err := g.Execute(context.Background(), r, deadlineChecks{})
	category(t, err, Timeout)
	if r.calls != 0 {
		t.Fatal("command started after assertion deadline")
	}
}
