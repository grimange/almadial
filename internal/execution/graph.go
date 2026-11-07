package execution

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"sort"
)

type Action struct {
	ID             string   `json:"id"`
	StageID        string   `json:"stage_id"`
	Description    string   `json:"description"`
	Dependencies   []string `json:"dependencies"`
	Preconditions  []string `json:"preconditions"`
	Command        Command  `json:"command"`
	Postconditions []string `json:"postconditions"`
}

// Graph owns serialized definitions, not caller-owned collections or callbacks.
// Its digest identifies metadata only; it establishes neither consent nor code trust.
type Graph struct{ data string }

var identifier = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)

func NewGraph(actions []Action) (Graph, error) {
	b, err := json.Marshal(actions)
	if err != nil {
		return Graph{}, failure(Contract)
	}
	var copied []Action
	if err = json.Unmarshal(b, &copied); err != nil {
		return Graph{}, failure(Contract)
	}
	sort.Slice(copied, func(i, j int) bool { return copied[i].ID < copied[j].ID })
	byID := map[string]Action{}
	for _, a := range copied {
		if !identifier.MatchString(a.ID) || !identifier.MatchString(a.StageID) || a.Description == "" || len(a.Description) > 4096 || len(a.Preconditions) == 0 || len(a.Postconditions) == 0 {
			return Graph{}, failure(Contract)
		}
		if _, ok := byID[a.ID]; ok {
			return Graph{}, graphContract(a, "duplicate action ID")
		}
		if err := a.Command.validate(); err != nil {
			return Graph{}, err
		}
		for _, list := range [][]string{a.Dependencies, a.Preconditions, a.Postconditions} {
			seen := map[string]bool{}
			for _, id := range list {
				if !identifier.MatchString(id) || seen[id] {
					return Graph{}, graphContract(a, "invalid or duplicate dependency/assertion ID")
				}
				seen[id] = true
			}
		}
		sort.Strings(a.Dependencies)
		sort.Strings(a.Preconditions)
		sort.Strings(a.Postconditions)
		byID[a.ID] = a
	}
	for _, a := range copied {
		for _, id := range a.Dependencies {
			if _, ok := byID[id]; !ok || id == a.ID {
				return Graph{}, graphContract(a, "dependency must name another action in this graph")
			}
		}
	}
	ordered := make([]Action, 0, len(copied))
	done := map[string]bool{}
	for len(ordered) < len(copied) {
		found := false
		for _, a := range copied {
			if done[a.ID] {
				continue
			}
			a = byID[a.ID]
			ready := true
			for _, id := range a.Dependencies {
				if !done[id] {
					ready = false
				}
			}
			if ready {
				ordered = append(ordered, a)
				done[a.ID] = true
				found = true
				break
			}
		}
		if !found {
			return Graph{}, contract("dependency graph contains a cycle")
		}
	}
	b, err = json.Marshal(ordered)
	if err != nil {
		return Graph{}, failure(Internal)
	}
	return Graph{string(b)}, nil
}
func (g Graph) Actions() []Action { var a []Action; _ = json.Unmarshal([]byte(g.data), &a); return a }
func (g Graph) JSON() []byte      { return []byte(g.data) }
func (g Graph) Digest() string {
	if g.data == "" {
		return ""
	}
	s := sha256.Sum256([]byte(g.data))
	return hex.EncodeToString(s[:])
}

// Checker implements trusted, bounded, read-only assertions, not dynamic plugins.
// IDs have to be bound before any command can run. Context must be honored.
type Checker interface {
	Supports(id string) bool
	Check(context.Context, string) error
}
type CommandRunner interface {
	Validate(Command) error
	Run(context.Context, Command) (Result, error)
}

type Trace struct {
	ActionID string `json:"action_id"`
	StageID  string `json:"stage_id"`
	Outcome  string `json:"outcome"` // succeeded, failed, blocked
	Result   Result `json:"result"`
	Failure  *Error `json:"failure,omitempty"`
}

// Execute is a sequential primitive for an already-authorized caller. All
// assertion bindings are checked before effects; independent branches continue
// after ordinary failure. Cancellation prevents further commands. There are no retries.
func (g Graph) Execute(ctx context.Context, r CommandRunner, c Checker) ([]Trace, error) {
	if ctx == nil || r == nil || c == nil || g.data == "" {
		return nil, failure(Contract)
	}
	actions := g.Actions()
	for _, a := range actions {
		if r.Validate(a.Command) != nil {
			e := failure(Contract)
			e.ActionID = a.ID
			e.StageID = a.StageID
			return nil, e
		}
		for _, ids := range [][]string{a.Preconditions, a.Postconditions} {
			for _, id := range ids {
				if !c.Supports(id) {
					e := failure(Contract)
					e.ActionID = a.ID
					e.StageID = a.StageID
					e.AssertionID = id
					return nil, e
				}
			}
		}
	}
	traces := make([]Trace, 0, len(actions))
	success := map[string]bool{}
	var first error
	for _, a := range actions {
		t := Trace{ActionID: a.ID, StageID: a.StageID, Outcome: "failed", Result: Result{ExitCode: -1}}
		var e *Error
		for _, id := range a.Dependencies {
			if !success[id] {
				e = failure(Dependency)
				e.DependencyID = id
				t.Outcome = "blocked"
				break
			}
		}
		if e == nil && ctx.Err() != nil {
			e = contextFailure(ctx.Err())
			t.Outcome = "blocked"
		}
		actionCtx, cancel := context.WithTimeout(ctx, a.Command.Timeout)
		if e == nil {
			e = assert(actionCtx, c, a.Preconditions, Precondition)
		}
		if e == nil {
			var err error
			t.Result, err = r.Run(actionCtx, a.Command)
			if err != nil {
				e = failure(Internal)
				if known, ok := err.(*Error); ok {
					clone := *known
					e = &clone
				}
			} else if !t.Result.Started || t.Result.ExitCode != 0 || t.Result.StdoutTruncated || t.Result.StderrTruncated {
				e = failure(Internal)
			}
		}
		if e == nil {
			e = assert(actionCtx, c, a.Postconditions, Postcondition)
		}
		cancel()
		if e != nil {
			e.ActionID = a.ID
			e.StageID = a.StageID
			t.Failure = e
			if first == nil {
				first = e
			}
		} else {
			t.Outcome = "succeeded"
			success[a.ID] = true
		}
		traces = append(traces, t)
	}
	return traces, first
}
func assert(ctx context.Context, c Checker, ids []string, kind Category) *Error {
	for _, id := range ids {
		if ctx.Err() != nil {
			return contextFailure(ctx.Err())
		}
		if c.Check(ctx, id) != nil {
			if ctx.Err() != nil {
				return contextFailure(ctx.Err())
			}
			e := failure(kind)
			e.AssertionID = id
			return e
		}
	}
	if ctx.Err() != nil {
		return contextFailure(ctx.Err())
	}
	return nil
}

func graphContract(a Action, detail string) *Error {
	e := contract(detail)
	if identifier.MatchString(a.ID) {
		e.ActionID = a.ID
	}
	if identifier.MatchString(a.StageID) {
		e.StageID = a.StageID
	}
	return e
}
