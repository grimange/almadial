//go:build linux

package journal

import (
	"bytes"
	"context"

	"github.com/grimange/almadial/internal/execution"
)

type Observed string

const (
	Satisfied Observed = "satisfied"
	Absent    Observed = "absent"
	Unknown   Observed = "unknown"
)

// Observation must establish the complete action postcondition, not just exit 0.
// Absent means a known safe pre-action state, not merely one missing side effect.
// Evidence is a SHA-256 of safe observed checks. No free-form output is persisted.
type Observation struct {
	State    Observed
	Evidence string
}

// Observer is trusted bounded read-only code, supplied by a future stage. It
// must honor ctx and independently check every postcondition in the record.
type Observer interface {
	Observe(context.Context, Record) (Observation, error)
}

func index(d Document, id string) int {
	for n, a := range d.Actions {
		if a.ID == id {
			return n
		}
	}
	return -1
}
func dependencies(d Document, a Record) bool {
	for _, id := range a.Dependencies {
		n := index(d, id)
		if n < 0 || d.Actions[n].State != Succeeded {
			return false
		}
	}
	return true
}
func observe(ctx context.Context, o Observer, a Record) Observation {
	if ctx == nil || ctx.Err() != nil || o == nil {
		return Observation{State: Unknown}
	}
	// Callbacks receive a defensive copy. Ignore their raw errors and invalid data.
	c := clone(Document{Actions: []Record{a}}).Actions[0]
	v, e := o.Observe(ctx, c)
	if e != nil || ctx.Err() != nil || !digest(v.Evidence) {
		return Observation{State: Unknown}
	}
	if v.State != Satisfied && v.State != Absent {
		return Observation{State: Unknown}
	}
	return v
}

// Begin must return nil before the caller starts the first mutating side effect.
// It commits running intent durably; it neither executes nor approves an action.
func (s *Session) Begin(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e := s.owned(); e != nil {
		return e
	}
	if s.data == nil {
		return ErrTransition
	}
	d := clone(*s.data)
	for _, a := range d.Actions {
		if a.State == ReconciliationRequired {
			return ErrReconcile
		}
		if a.State == Running {
			return ErrTransition
		}
	}
	n := index(d, id)
	if n < 0 || d.Actions[n].State != Pending || !dependencies(d, d.Actions[n]) {
		return ErrTransition
	}
	d.Actions[n].State = Running
	return s.persist(d)
}

// Finish accepts a single ALM-13 trace, discards diagnostic strings, and requires
// fresh sufficient postconditions before committing success. It never reruns a
// command. A process failure remains failed even if a postcondition might pass.
func (s *Session) Finish(ctx context.Context, t execution.Trace, o Observer) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e := s.owned(); e != nil {
		return e
	}
	if s.data == nil {
		return ErrTransition
	}
	d := clone(*s.data)
	n := index(d, t.ActionID)
	if n < 0 || d.Actions[n].State != Running || d.Actions[n].Stage != t.StageID {
		return ErrTransition
	}
	a := &d.Actions[n]
	if t.Outcome == "failed" {
		if t.Failure == nil || !category(t.Failure.Category) || t.Failure.ExitCode < -1 || t.Failure.ExitCode > 255 {
			return ErrInvalid
		}
		a.State = Failed
		a.Failure = t.Failure.Category
		a.ExitCode = t.Failure.ExitCode
		return s.persist(d)
	}
	if t.Outcome != "succeeded" || t.Failure != nil || !t.Result.Started || t.Result.ExitCode != 0 {
		return ErrInvalid
	}
	v := observe(ctx, o, *a)
	if v.State != Satisfied {
		a.State = ReconciliationRequired
		if e := s.persist(d); e != nil {
			return e
		}
		return ErrReconcile
	}
	a.State = Succeeded
	a.Evidence = v.Evidence
	a.ExitCode = 0
	return s.persist(d)
}

// Resume rejects changed identity and metadata before invoking observations.
// The expected document must be freshly derived with New from the current graph
// and explicit retry policy. All previously started actions are re-inspected,
// including succeeded ones. Unknown/drift/non-idempotent absent states block
// automatic replay. Retryable absent states become pending, never execute here.
// Callers must inspect the result and independently revalidate approval/preflight.
func (s *Session) Resume(ctx context.Context, expected Document, o Observer) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e := s.owned(); e != nil {
		return e
	}
	if s.data != nil {
		return ErrTransition
	}
	if ctx == nil || ctx.Err() != nil || expected.Validate() != nil {
		return ErrInvalid
	}
	for _, a := range expected.Actions {
		if a.State != Pending {
			return ErrInvalid
		}
	}
	d, e := read(s.dir)
	if e != nil {
		return e
	}
	for _, pair := range []struct{ name, actual, want string }{
		{"host", d.Identity.Host, expected.Identity.Host}, {"host_state", d.Identity.HostState, expected.Identity.HostState},
		{"config", d.Identity.Config, expected.Identity.Config}, {"manifest", d.Identity.Manifest, expected.Identity.Manifest},
		{"plan", d.Identity.Plan, expected.Identity.Plan}, {"policy", d.Identity.Policy, expected.Identity.Policy}, {"graph", d.Identity.Graph, expected.Identity.Graph},
	} {
		if pair.actual != pair.want {
			return &IdentityError{Field: pair.name}
		}
	}
	normalized := clone(d)
	for n := range normalized.Actions {
		a := &normalized.Actions[n]
		a.State = Pending
		a.Evidence = ""
		a.Failure = ""
		a.ExitCode = -1
	}
	b, _ := normalized.JSON()
	want, _ := expected.JSON()
	if !bytes.Equal(b, want) {
		return &IdentityError{Field: "action_metadata"}
	}
	blocked := false
	for n := range d.Actions {
		a := &d.Actions[n]
		if a.State == Pending {
			continue
		}
		prior := a.State
		v := observe(ctx, o, *a)
		a.State = ReconciliationRequired
		a.Evidence = ""
		a.Failure = ""
		a.ExitCode = -1
		switch v.State {
		case Satisfied:
			a.State = Succeeded
			a.Evidence = v.Evidence
			a.ExitCode = 0
		case Absent:
			if (prior == Running || prior == Failed) && a.Retry == Idempotent {
				a.State = Pending
			}
		}
	}
	// A descendant cannot remain verified if a prerequisite drifted. Iterate to a
	// fixed point because document order is lexical, not necessarily topological.
	for changed := true; changed; {
		changed = false
		for n, a := range d.Actions {
			if a.State == Succeeded && !dependencies(d, a) {
				d.Actions[n].State = ReconciliationRequired
				d.Actions[n].Evidence = ""
				d.Actions[n].ExitCode = -1
				changed = true
			}
		}
	}
	for _, a := range d.Actions {
		if a.State == ReconciliationRequired {
			blocked = true
		}
	}
	if e = s.persist(d); e != nil {
		return e
	}
	if blocked {
		return ErrReconcile
	}
	return nil
}
