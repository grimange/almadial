// Package journal supplies local persistence primitives, not installation approval
// or an executor. Callers must establish approval, artifact trust and fresh host
// eligibility before acquiring a mutating session.
package journal

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/grimange/almadial/internal/execution"
)

const SchemaVersion = 1
const MaxBytes = 4 << 20

var (
	ErrInvalid    = errors.New("invalid journal contract or data")
	ErrIdentity   = errors.New("installation identity changed; regenerate and obtain approval")
	ErrTransition = errors.New("invalid action transition; reconcile before execution")
	ErrBusy       = errors.New("another process owns exclusive execution")
	ErrUnsafe     = errors.New("unsafe state directory or file; inspect ownership, modes and path")
	ErrClosed     = errors.New("session closed or persistence uncertain; close and inspect before resume")
	ErrReconcile  = errors.New("action requires fresh reconciliation or operator intervention")
)

// All identities are lowercase SHA-256. Host identifies the server; HostState is
// ALM-11's material preflight fingerprint, including its reviewed policy inputs.
// Policy identifies the caller's versioned recovery/approval policy. These hashes
// identify content; they authenticate neither an operator nor a manifest publisher.
// IdentityError names a static field, never its private value.
type IdentityError struct{ Field string }

func (e *IdentityError) Error() string {
	return "journal identity mismatch: " + e.Field + "; regenerate and obtain approval"
}
func (e *IdentityError) Is(target error) bool { return target == ErrIdentity }

type Identity struct {
	Host      string `json:"host"`
	HostState string `json:"host_state"`
	Config    string `json:"config"`
	Manifest  string `json:"manifest"`
	Plan      string `json:"plan"`
	Policy    string `json:"policy"`
	Graph     string `json:"graph"`
}
type State string

const (
	Pending                State = "pending"
	Running                State = "running"
	Succeeded              State = "succeeded"
	Failed                 State = "failed"
	ReconciliationRequired State = "reconciliation_required"
)

type Retry string

const (
	Never      Retry = "never"
	Idempotent Retry = "idempotent"
)

// Record excludes argv, output, free-form diagnostics and resolved secrets.
// Evidence is a digest of independently collected postcondition evidence.
type Record struct {
	ID             string             `json:"id"`
	Stage          string             `json:"stage"`
	Dependencies   []string           `json:"dependencies"`
	Postconditions []string           `json:"postconditions"`
	Retry          Retry              `json:"retry"`
	State          State              `json:"state"`
	Evidence       string             `json:"evidence"`
	Failure        execution.Category `json:"failure"`
	ExitCode       int                `json:"exit_code"`
}
type Document struct {
	SchemaVersion int      `json:"schema_version"`
	Identity      Identity `json:"identity"`
	Actions       []Record `json:"actions"`
}

var identifier = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)

func digest(s string) bool {
	b, e := hex.DecodeString(s)
	return e == nil && len(b) == 32 && strings.ToLower(s) == s
}
func (i Identity) validate() error {
	for _, s := range []string{i.Host, i.HostState, i.Config, i.Manifest, i.Plan, i.Policy, i.Graph} {
		if !digest(s) {
			return ErrInvalid
		}
	}
	return nil
}
func category(k execution.Category) bool {
	switch k {
	case execution.Contract, execution.Precondition, execution.Start, execution.Exit, execution.Timeout, execution.Canceled, execution.OutputLimit, execution.Postcondition, execution.Dependency, execution.Internal:
		return true
	}
	return false
}
func (d Document) Validate() error {
	if d.SchemaVersion != SchemaVersion || d.Identity.validate() != nil || len(d.Actions) > 4096 || d.Actions == nil {
		return ErrInvalid
	}
	seen := map[string]Record{}
	for n, a := range d.Actions {
		if !identifier.MatchString(a.ID) || !identifier.MatchString(a.Stage) || (n > 0 && d.Actions[n-1].ID >= a.ID) {
			return ErrInvalid
		}
		if a.Retry != Never && a.Retry != Idempotent {
			return ErrInvalid
		}
		for _, list := range [][]string{a.Dependencies, a.Postconditions} {
			for j, s := range list {
				if !identifier.MatchString(s) || (j > 0 && list[j-1] >= s) {
					return ErrInvalid
				}
			}
		}
		if len(a.Postconditions) == 0 || a.Dependencies == nil {
			return ErrInvalid
		}
		switch a.State {
		case Pending, Running, Succeeded, Failed, ReconciliationRequired:
		default:
			return ErrInvalid
		}
		if a.State == Succeeded {
			if !digest(a.Evidence) || a.Failure != "" || a.ExitCode != 0 {
				return ErrInvalid
			}
		} else if a.Evidence != "" {
			return ErrInvalid
		}
		if a.State == Failed {
			if !category(a.Failure) || a.ExitCode < -1 || a.ExitCode > 255 {
				return ErrInvalid
			}
		} else if a.Failure != "" || (a.State != Succeeded && a.ExitCode != -1) {
			return ErrInvalid
		}
		seen[a.ID] = a
	}
	// Validate dependency existence and acyclicity without relying on stored order.
	done := map[string]bool{}
	for len(done) < len(seen) {
		progress := false
		for _, a := range d.Actions {
			if done[a.ID] {
				continue
			}
			ready := true
			for _, dep := range a.Dependencies {
				if _, ok := seen[dep]; !ok || dep == a.ID {
					return ErrInvalid
				}
				if !done[dep] {
					ready = false
				}
			}
			if ready {
				done[a.ID] = true
				progress = true
			}
		}
		if !progress {
			return ErrInvalid
		}
	}
	return nil
}

// New binds the exact accepted execution graph. Retry policy must be explicit for
// every action; unrecognized or missing policies are rejected before persistence.
func New(i Identity, g execution.Graph, retry map[string]Retry) (Document, error) {
	d := Document{SchemaVersion: SchemaVersion, Identity: i, Actions: []Record{}}
	if g.Digest() == "" || i.Graph != g.Digest() || len(retry) != len(g.Actions()) {
		return Document{}, ErrInvalid
	}
	for _, a := range g.Actions() {
		p, ok := retry[a.ID]
		if !ok {
			return Document{}, ErrInvalid
		}
		deps := append([]string{}, a.Dependencies...)
		posts := append([]string{}, a.Postconditions...)
		sort.Strings(deps)
		sort.Strings(posts)
		d.Actions = append(d.Actions, Record{a.ID, a.StageID, deps, posts, p, Pending, "", "", -1})
	}
	sort.Slice(d.Actions, func(a, b int) bool { return d.Actions[a].ID < d.Actions[b].ID })
	return d, d.Validate()
}
func (d Document) JSON() ([]byte, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	b, e := json.Marshal(d)
	if len(b) > MaxBytes {
		return nil, ErrInvalid
	}
	return b, e
}
func clone(d Document) Document {
	b, _ := json.Marshal(d)
	var c Document
	_ = json.Unmarshal(b, &c)
	return c
}

// Decode is bounded, rejects duplicate/case-variant/unknown/missing keys, null,
// invalid UTF-8, trailing documents and unsupported versions. Errors never echo
// input bytes, paths or parser diagnostics which could contain secret values.
func Decode(r io.Reader) (Document, error) {
	if r == nil {
		return Document{}, ErrInvalid
	}
	b, e := io.ReadAll(io.LimitReader(r, MaxBytes+1))
	if e != nil || len(b) > MaxBytes || !utf8.Valid(b) {
		return Document{}, ErrInvalid
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	if scan(dec, 0) != nil {
		return Document{}, ErrInvalid
	}
	if _, e = dec.Token(); e != io.EOF {
		return Document{}, ErrInvalid
	}
	var raw map[string]json.RawMessage
	if json.Unmarshal(b, &raw) != nil || !keys(raw, "schema_version", "identity", "actions") {
		return Document{}, ErrInvalid
	}
	var id map[string]json.RawMessage
	if json.Unmarshal(raw["identity"], &id) != nil || !keys(id, "host", "host_state", "config", "manifest", "plan", "policy", "graph") {
		return Document{}, ErrInvalid
	}
	var acts []map[string]json.RawMessage
	if json.Unmarshal(raw["actions"], &acts) != nil {
		return Document{}, ErrInvalid
	}
	for _, a := range acts {
		if !keys(a, "id", "stage", "dependencies", "postconditions", "retry", "state", "evidence", "failure", "exit_code") {
			return Document{}, ErrInvalid
		}
	}
	var d Document
	dec = json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if dec.Decode(&d) != nil || d.Validate() != nil {
		return Document{}, ErrInvalid
	}
	return d, nil
}
func keys(m map[string]json.RawMessage, names ...string) bool {
	if len(m) != len(names) {
		return false
	}
	for _, n := range names {
		if _, ok := m[n]; !ok {
			return false
		}
	}
	return true
}
func scan(d *json.Decoder, depth int) error {
	if depth > 16 {
		return ErrInvalid
	}
	t, e := d.Token()
	if e != nil || t == nil {
		return ErrInvalid
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			k, e := d.Token()
			s, ok := k.(string)
			if e != nil || !ok || seen[s] {
				return ErrInvalid
			}
			seen[s] = true
			if scan(d, depth+1) != nil {
				return ErrInvalid
			}
		}
		t, e = d.Token()
		if e != nil || t != json.Delim('}') {
			return ErrInvalid
		}
	case '[':
		for d.More() {
			if scan(d, depth+1) != nil {
				return ErrInvalid
			}
		}
		t, e = d.Token()
		if e != nil || t != json.Delim(']') {
			return ErrInvalid
		}
	default:
		return ErrInvalid
	}
	return nil
}
