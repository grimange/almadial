//go:build linux

package provision

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/grimange/almadial/internal/artifacts"
	"github.com/grimange/almadial/internal/execution"
	"github.com/grimange/almadial/internal/journal"
	"github.com/grimange/almadial/internal/operatorlog"
	"github.com/grimange/almadial/internal/operatorui"
	"github.com/grimange/almadial/internal/plan"
)

type SystemInputs struct {
	Parent   Inputs          `json:"parent"`
	Snapshot SystemSnapshot  `json:"snapshot"`
	Budget   RecordingBudget `json:"budget"`
}
type SystemDirectoryRequest struct {
	Before    DirectoryObservation
	HostState string
}
type SystemStage struct {
	data  string
	graph execution.Graph
}
type systemStageData struct {
	Parent, Config, Manifest, Policy, Graph string
	Before                                  SystemSnapshot
	Budget                                  RecordingBudget
}

func (i SystemInputs) Prepare(m artifacts.Manifest) (SystemStage, error) {
	i.Snapshot = canonicalSystem(i.Snapshot)
	p, e := plan.New(i.Parent.Config, i.Parent.Preflight, i.Parent.Requirements, m.Digest())
	if e != nil {
		return SystemStage{}, e
	}
	id, e := hex.DecodeString(i.Parent.Preflight.HostID)
	h := sha256.Sum256([]byte(i.Parent.Preflight.HostID))
	if e != nil || len(id) != 16 || hex.EncodeToString(h[:]) != i.Snapshot.RPM.Host || i.Parent.Inventory.Digest() != i.Snapshot.RPM.Digest() || i.Parent.Transaction != nil || i.Parent.PrepareRepositoryTrust || !systemReady(i.Snapshot, i.Budget) {
		return SystemStage{}, errSystem
	}
	actions := []execution.Action{}
	add := func(id, description string, cmd execution.Command) {
		deps := []string{}
		if len(actions) > 0 {
			deps = append(deps, actions[len(actions)-1].ID)
		}
		actions = append(actions, execution.Action{ID: id, StageID: "S02", Description: description, Dependencies: deps, Preconditions: []string{"system.fresh"}, Command: cmd, Postconditions: []string{id + ".verified"}})
	}
	if _, ok := group(i.Snapshot, "asterisk"); !ok {
		add("system.00.group", "Create the approved dynamic system group", systemCommand("/usr/sbin/groupadd", "--system", "asterisk"))
	}
	if _, ok := account(i.Snapshot, "asterisk"); !ok {
		add("system.01.user", "Create the M0 noninteractive service account", systemCommand("/usr/sbin/useradd", "--system", "--gid", "asterisk", "--home-dir", "/var/lib/asterisk", "--shell", "/sbin/nologin", "--no-create-home", "--no-log-init", "asterisk"))
	}
	for j, p := range ManagedDirectories() {
		d, _ := directory(i.Snapshot, p.Path)
		if !directorySatisfied(i.Snapshot, p, d) {
			add(fmt.Sprintf("system.02.directory.%02d", j), "Provision only the approved directory inode and vendor default label", systemCommand("/proc/self/exe", "__directory", fmt.Sprint(j), jsonString(SystemDirectoryRequest{d, i.Snapshot.Digest()})))
		}
	}
	g, e := execution.NewGraph(actions)
	if e != nil {
		return SystemStage{}, e
	}
	d := systemStageData{p.Digest(), p.ConfigSHA256(), m.Digest(), SystemPolicyDigest(), g.Digest(), i.Snapshot, i.Budget}
	return SystemStage{jsonString(d), g}, nil
}
func (s SystemStage) JSON() []byte { return []byte(s.data) }
func (s SystemStage) Digest() string {
	if s.data == "" {
		return ""
	}
	return hash(json.RawMessage(s.data))
}
func (s SystemStage) Graph() execution.Graph { return s.graph }
func (s SystemStage) doc() systemStageData {
	var d systemStageData
	_ = json.Unmarshal([]byte(s.data), &d)
	return d
}

// SystemLabAuthorization is deliberately separate from ALM-16 and normal
// production approval. These bindings record prior operator permission, never
// manufacture it; a candidate manifest remains unauthorized for installation.
type SystemLabAuthorization struct {
	Scope, Host, StageSHA256, ParentSHA256, ManifestSHA256 string
	SnapshotReference, OperatorEvidenceSHA256              string
	CandidateAcknowledged                                  bool
}

func (a SystemLabAuthorization) validate(s SystemStage, m artifacts.Manifest) error {
	d := s.doc()
	if s.Digest() == "" || a.Scope != "ALM-17/disposable-lab" || a.Host != d.Before.RPM.Host || a.StageSHA256 != s.Digest() || a.ParentSHA256 != d.Parent || a.ManifestSHA256 != d.Manifest || m.Digest() != d.Manifest || d.Policy != SystemPolicyDigest() || !shaPattern.MatchString(a.SnapshotReference) || !shaPattern.MatchString(a.OperatorEvidenceSHA256) || !a.CandidateAcknowledged {
		return &execution.Error{Category: execution.Precondition, StageID: "S02", ExitCode: -1, Detail: "exact ALM-17 disposable laboratory authorization required"}
	}
	return nil
}

// SystemHost is the narrow observation seam used by deterministic tests. All
// actual commands still pass through the accepted ALM-13 CommandRunner.
type SystemHost interface {
	InspectSystem(context.Context) (SystemSnapshot, error)
}
type systemChecks struct {
	host    SystemHost
	d       systemStageData
	current SystemSnapshot
	action  execution.Action
}

func directoryIndex(id string) int {
	for j := range ManagedDirectories() {
		if id == fmt.Sprintf("system.02.directory.%02d", j) {
			return j
		}
	}
	return -1
}
func mergeSystemAction(before, after SystemSnapshot, id string) (SystemSnapshot, error) {
	// Copy collections before changing the expected snapshot.
	before.Accounts = append([]AccountObservation{}, before.Accounts...)
	before.Groups = append([]GroupObservation{}, before.Groups...)
	before.Directories = append([]DirectoryObservation{}, before.Directories...)
	switch id {
	case "system.00.group":
		g, ok := group(after, "asterisk")
		if !ok || g.GID == 0 || g.GID > after.SystemGIDMax {
			return before, errSystem
		}
		for _, v := range after.Groups {
			if v.Name != g.Name && v.GID == g.GID {
				return before, errSystem
			}
		}
		if _, ok = group(before, g.Name); !ok {
			before.Groups = append(before.Groups, g)
		}
	case "system.01.user":
		a, ok := account(after, "asterisk")
		g, gok := group(after, "asterisk")
		if !ok || !gok || !identityValid(after, ServiceIdentities()[2], a, g) {
			return before, errSystem
		}
		if _, ok = account(before, a.Name); !ok {
			before.Accounts = append(before.Accounts, a)
		}
	default:
		j := directoryIndex(id)
		if j < 0 {
			return before, errSystem
		}
		p := ManagedDirectories()[j]
		v, ok := directory(after, p.Path)
		if !ok || !directorySatisfied(after, p, v) {
			return before, errSystem
		}
		for k, d := range before.Directories {
			if d.Path == p.Path {
				before.Directories[k] = v
				return before, nil
			}
		}
		return before, errSystem
	}
	return before, nil
}
func (c *systemChecks) Supports(id string) bool {
	return id == "system.fresh" || id == c.action.ID+".verified"
}
func (c *systemChecks) Check(ctx context.Context, id string) error {
	s, e := c.host.InspectSystem(ctx)
	if e != nil {
		return e
	}
	if !systemReady(s, c.d.Budget) || s.RPM.Digest() != c.d.Before.RPM.Digest() {
		return errSystem
	}
	if id == "system.fresh" {
		if s.Digest() != c.current.Digest() {
			return errSystem
		}
		return nil
	}
	if id != c.action.ID+".verified" {
		return errSystem
	}
	want, e := mergeSystemAction(c.current, s, c.action.ID)
	if e != nil || want.Digest() != s.Digest() {
		return errSystem
	}
	c.current = s
	return nil
}
func actionEvidence(s SystemSnapshot, id string) string {
	switch id {
	case "system.00.group":
		g, _ := group(s, "asterisk")
		return hash(g)
	case "system.01.user":
		a, _ := account(s, "asterisk")
		return hash(a)
	default:
		j := directoryIndex(id)
		if j < 0 {
			return ""
		}
		d, _ := directory(s, ManagedDirectories()[j].Path)
		return hash(d)
	}
}
func (c *systemChecks) Observe(ctx context.Context, r journal.Record) (journal.Observation, error) {
	s, e := c.host.InspectSystem(ctx)
	if e != nil || !systemReady(s, c.d.Budget) || s.RPM.Digest() != c.d.Before.RPM.Digest() {
		return journal.Observation{State: journal.Unknown}, e
	}
	_, e = mergeSystemAction(c.d.Before, s, r.ID)
	if e == nil {
		ev := actionEvidence(s, r.ID)
		if r.State == journal.Succeeded && r.Evidence != ev {
			return journal.Observation{State: journal.Unknown}, nil
		}
		return journal.Observation{State: journal.Satisfied, Evidence: ev}, nil
	}
	// Only the exact known pre-action state is retryable. A partially created
	// directory with incorrect metadata is unknown, not an invitation to chown.
	absent := false
	switch r.ID {
	case "system.00.group":
		_, ok := group(s, "asterisk")
		absent = !ok
	case "system.01.user":
		_, ok := account(s, "asterisk")
		absent = !ok
	default:
		if j := directoryIndex(r.ID); j >= 0 {
			p := ManagedDirectories()[j]
			a, _ := directory(c.d.Before, p.Path)
			b, _ := directory(s, p.Path)
			absent = a == b
		}
	}
	if absent {
		return journal.Observation{State: journal.Absent, Evidence: hash(struct{ ID, Before string }{r.ID, c.d.Before.Digest()})}, nil
	}
	return journal.Observation{State: journal.Unknown}, nil
}
func (s SystemStage) journalDocument(a SystemLabAuthorization) (journal.Document, error) {
	d := s.doc()
	retry := map[string]journal.Retry{}
	for _, v := range s.graph.Actions() {
		retry[v.ID] = journal.Idempotent
	}
	return journal.New(journal.Identity{Host: d.Before.RPM.Host, HostState: d.Before.Digest(), Config: d.Config, Manifest: d.Manifest, Plan: d.Parent, Policy: hash(struct{ Policy, Stage, Authorization string }{d.Policy, s.Digest(), hash(a)}), Graph: s.graph.Digest()}, s.graph, retry)
}

// ExecuteSystemLab orchestrates only S02's fixed graph. ALM-14 owns locking,
// durable intent and reconciliation; ALM-13 owns assertions, bounded processes,
// cancellation and failure taxonomy. No mutation is attempted before Begin.
func ExecuteSystemLab(ctx context.Context, s SystemStage, m artifacts.Manifest, a SystemLabAuthorization, stateDir string, host SystemHost, runner execution.CommandRunner, resume bool, log *operatorlog.Logger) (out Outcome, err error) {
	out.Traces = []execution.Trace{}
	if ctx == nil || host == nil || runner == nil {
		return out, errSystem
	}
	if e := a.validate(s, m); e != nil {
		return out, e
	}
	session, e := journal.Acquire(stateDir)
	if e != nil {
		return out, e
	}
	defer session.Close()
	defer func() {
		if doc, e := session.Snapshot(); e == nil {
			out.Journal = &doc
		}
	}()
	d := s.doc()
	fresh, e := host.InspectSystem(ctx)
	if e != nil {
		return out, e
	}
	if fresh.RPM.Host != d.Before.RPM.Host || fresh.RPM.Digest() != d.Before.RPM.Digest() || !systemReady(fresh, d.Budget) {
		return out, errSystem
	}
	c := &systemChecks{host: host, d: d, current: d.Before}
	doc, e := s.journalDocument(a)
	if e != nil {
		return out, e
	}
	if resume {
		if e = session.Resume(ctx, doc, c); e != nil {
			return out, e
		}
		current, _ := session.Snapshot()
		for _, action := range s.graph.Actions() {
			for _, r := range current.Actions {
				if action.ID == r.ID && r.State == journal.Succeeded {
					c.current, e = mergeSystemAction(c.current, fresh, r.ID)
					if e != nil {
						return out, e
					}
				}
			}
		}
		if c.current.Digest() != fresh.Digest() {
			return out, errSystem
		}
	} else {
		if fresh.Digest() != d.Before.Digest() {
			return out, &execution.Error{Category: execution.Precondition, StageID: "S02", ExitCode: -1, Detail: "service/storage plan stale; inspect and review again"}
		}
		if e = session.Initialize(doc); e != nil {
			return out, e
		}
	}
	for _, action := range s.graph.Actions() {
		current, _ := session.Snapshot()
		done := false
		for _, r := range current.Actions {
			if r.ID == action.ID && r.State == journal.Succeeded {
				done = true
			}
		}
		if done {
			continue
		}
		c.action = action
		if e = c.Check(ctx, "system.fresh"); e != nil {
			return out, e
		}
		if e = session.Begin(action.ID); e != nil {
			return out, e
		}
		if log != nil {
			_ = operatorui.RenderEvent(log, operatorui.Event{Type: operatorui.ActionStarted, StageID: "S02", ActionID: action.ID})
		}
		// ALM-14 Begin validates the original graph dependencies. The single
		// action primitive must have no references outside its own graph.
		one := action
		one.Dependencies = []string{}
		g, e := execution.NewGraph([]execution.Action{one})
		if e != nil {
			return out, e
		}
		traces, runErr := g.Execute(ctx, runner, c)
		out.Traces = append(out.Traces, traces...)
		if len(traces) != 1 {
			return out, errSystem
		}
		t := traces[0]
		if t.Outcome == "blocked" {
			t.Outcome = "failed"
		}
		if e = session.Finish(ctx, t, c); e != nil {
			return out, e
		}
		if log != nil {
			ev, ee := operatorui.EventFromTrace(traces[0])
			if ee == nil {
				_ = operatorui.RenderEvent(log, ev)
			}
		}
		if runErr != nil {
			return out, runErr
		}
	}
	fresh, e = host.InspectSystem(ctx)
	if e != nil {
		return out, e
	}
	if !systemSatisfied(fresh, d.Budget) || fresh.Digest() != c.current.Digest() {
		return out, &execution.Error{Category: execution.Postcondition, StageID: "S02", ExitCode: -1, Detail: "independent service/storage postconditions failed"}
	}
	out.NoOp = len(out.Traces) == 0
	return out, nil
}
