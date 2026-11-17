//go:build linux

package provision

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/grimange/almadial/internal/artifacts"
	"github.com/grimange/almadial/internal/execution"
	"github.com/grimange/almadial/internal/journal"
	"github.com/grimange/almadial/internal/operatorlog"
	"github.com/grimange/almadial/internal/operatorui"
	"github.com/grimange/almadial/internal/plan"
)

type AsteriskInputs struct {
	Parent       DatabaseInputs
	Snapshot     AsteriskSnapshot
	SourceSHA256 string
}
type AsteriskStage struct {
	data  string
	graph execution.Graph
}
type asteriskStageData struct {
	Parent, Config, Manifest, Policy, Source, Graph string
	Before                                          AsteriskSnapshot
}

func (s AsteriskStage) JSON() []byte { return []byte(s.data) }
func (s AsteriskStage) Digest() string {
	if s.data == "" {
		return ""
	}
	return hash(json.RawMessage(s.data))
}
func (s AsteriskStage) Graph() execution.Graph { return s.graph }
func (s AsteriskStage) doc() asteriskStageData {
	var d asteriskStageData
	_ = json.Unmarshal(s.JSON(), &d)
	return d
}

func (i AsteriskInputs) Prepare(m artifacts.Manifest, source AsteriskSource) (AsteriskStage, error) {
	p, e := plan.New(i.Parent.Parent.Parent.Config, i.Parent.Parent.Parent.Preflight, i.Parent.Parent.Parent.Requirements, m.Digest())
	if e != nil {
		return AsteriskStage{}, e
	}
	id, e := hex.DecodeString(i.Parent.Parent.Parent.Preflight.HostID)
	h := sha256.Sum256([]byte(i.Parent.Parent.Parent.Preflight.HostID))
	if e != nil || len(id) != 16 || hex.EncodeToString(h[:]) != i.Snapshot.Host || i.Parent.Parent.Snapshot.Digest() != i.Snapshot.Foundation || i.Parent.Snapshot.Digest() != i.Snapshot.Database || i.Parent.Snapshot.Foundation != i.Snapshot.Foundation || !databaseSatisfied(i.Parent.Snapshot) || !systemSatisfied(i.Parent.Parent.Snapshot, i.Parent.Parent.Budget) || i.Parent.Parent.Parent.Inventory.Digest() != i.Parent.Parent.Snapshot.RPM.Digest() || i.Parent.Parent.Parent.Transaction != nil || i.Parent.Parent.Parent.PrepareRepositoryTrust || !databaseReferences(i.Parent.Parent.Parent.Config) || !asteriskReady(i.Snapshot) || source.Digest() == "" || i.SourceSHA256 != source.Digest() || i.Snapshot.Source != source.Digest() {
		return AsteriskStage{}, errAsterisk
	}
	actions := []execution.Action{}
	add := func(id, desc string) {
		deps := []string{}
		if len(actions) > 0 {
			deps = append(deps, actions[len(actions)-1].ID)
		}
		actions = append(actions, execution.Action{ID: id, StageID: "S04", Description: desc, Dependencies: deps, Preconditions: []string{"asterisk.fresh"}, Command: asteriskActionCommand(id), Postconditions: []string{id + ".verified"}})
	}
	s := i.Snapshot
	if s.Build == "absent" {
		add("asterisk.00.build", "Compile authenticated pinned source with bundled PJPROJECT and M0 options")
	}
	if s.Install == "absent" {
		add("asterisk.01.install", "Stage upstream installation; install only verified executable and native module closure")
	}
	if s.Install == "resources-missing" {
		add("asterisk.01.resources", "Stop failed service and install three absent pinned XML runtime resources against proven receipt")
	}
	if s.Install == "libraries-missing" {
		add("asterisk.01.libraries", "Stop the failed service and reconcile proven receipt by installing only the two absent pinned runtime libraries")
	}
	if s.Configuration == "absent" {
		add("asterisk.02.config", "Create absent S04-owned includes; preserve every existing configuration inode")
	}
	if s.Unit == "absent" {
		add("asterisk.03.unit", "Create absent approved native service unit")
	}
	if len(actions) > 0 || !asteriskSatisfied(s) {
		add("asterisk.04.service", "Start approved service and verify native runtime")
		add("asterisk.05.restart", "Stop, start and independently verify persistent native runtime")
	}
	g, e := execution.NewGraph(actions)
	if e != nil {
		return AsteriskStage{}, e
	}
	d := asteriskStageData{p.Digest(), p.ConfigSHA256(), m.Digest(), AsteriskPolicyDigest(), source.Digest(), g.Digest(), s}
	return AsteriskStage{jsonString(d), g}, nil
}

type AsteriskLabAuthorization struct {
	Scope, Host, StageSHA256, ParentSHA256, ManifestSHA256, SourceSHA256 string
	SnapshotReference, OperatorEvidenceSHA256                            string
	CandidateAcknowledged                                                bool
}

func (a AsteriskLabAuthorization) validate(s AsteriskStage, m artifacts.Manifest) error {
	d := s.doc()
	if s.Digest() == "" || !stageAuthorization(a.Scope, "ALM-19/disposable-lab", a.SnapshotReference, a.OperatorEvidenceSHA256, a.CandidateAcknowledged, false, false) || a.Host != d.Before.Host || a.StageSHA256 != s.Digest() || a.ParentSHA256 != d.Parent || a.ManifestSHA256 != d.Manifest || a.SourceSHA256 != d.Source || m.Digest() != d.Manifest || d.Policy != AsteriskPolicyDigest() {
		return &execution.Error{Category: execution.Precondition, StageID: "S04", ExitCode: -1, Detail: "valid authorization mode, exact telephony-stage bindings and verified source required"}
	}
	return nil
}

type AsteriskHost interface {
	InspectAsterisk(context.Context) (AsteriskSnapshot, error)
}
type asteriskChecks struct {
	host    AsteriskHost
	d       asteriskStageData
	current AsteriskSnapshot
	action  execution.Action
}

func mergeAsterisk(before, after AsteriskSnapshot, id string) (AsteriskSnapshot, error) {
	if !asteriskReady(after) || before.Host != after.Host || before.Foundation != after.Foundation || before.Database != after.Database || before.Source != after.Source || before.Toolchain != after.Toolchain {
		return before, errAsterisk
	}
	w := before
	switch id {
	case "asterisk.00.build":
		if after.Build == "absent" {
			return before, errAsterisk
		}
		w.Build = after.Build
		w.Modules = after.Modules
	case "asterisk.01.install", "asterisk.01.libraries", "asterisk.01.resources":
		if id == "asterisk.01.libraries" || id == "asterisk.01.resources" {
			if after.Active {
				return before, errAsterisk
			}
			w.Active, w.RuntimeOK, w.ServiceID = after.Active, after.RuntimeOK, after.ServiceID
		}
		if !shaPattern.MatchString(after.Install) {
			return before, errAsterisk
		}
		w.Install = after.Install
		w.Modules = after.Modules
	case "asterisk.02.config":
		if after.Configuration == "absent" {
			return before, errAsterisk
		}
		w.Configuration = after.Configuration
	case "asterisk.03.unit":
		if after.Unit == "absent" {
			return before, errAsterisk
		}
		w.Unit = after.Unit
	case "asterisk.04.service":
		if !after.Active || !after.Enabled || !after.RuntimeOK {
			return before, errAsterisk
		}
		w.Active, w.Enabled, w.RuntimeOK, w.ServiceID, w.Modules = after.Active, after.Enabled, after.RuntimeOK, after.ServiceID, after.Modules
	case "asterisk.05.restart":
		if !asteriskSatisfied(after) || before.ServiceID == after.ServiceID {
			return before, errAsterisk
		}
		w.ServiceID, w.RestartVerified, w.Modules = after.ServiceID, after.RestartVerified, after.Modules
	default:
		return before, errAsterisk
	}
	if w.Digest() != after.Digest() {
		return before, errAsterisk
	}
	return w, nil
}
func (c *asteriskChecks) Supports(id string) bool {
	return id == "asterisk.fresh" || id == c.action.ID+".verified"
}
func (c *asteriskChecks) Check(ctx context.Context, id string) error {
	s, e := c.host.InspectAsterisk(ctx)
	if e != nil {
		return e
	}
	if id == "asterisk.fresh" {
		if !asteriskReady(s) || s.Digest() != c.current.Digest() {
			return errAsterisk
		}
		return nil
	}
	if id != c.action.ID+".verified" {
		return errAsterisk
	}
	w, e := mergeAsterisk(c.current, s, c.action.ID)
	if e == nil {
		c.current = w
	}
	return e
}
func asteriskActionSatisfied(s AsteriskSnapshot, id string) bool {
	if !asteriskReady(s) {
		return false
	}
	switch id {
	case "asterisk.00.build":
		return s.Build != "absent"
	case "asterisk.01.install", "asterisk.01.libraries", "asterisk.01.resources":
		return shaPattern.MatchString(s.Install)
	case "asterisk.02.config":
		return s.Configuration != "absent"
	case "asterisk.03.unit":
		return s.Unit != "absent"
	case "asterisk.04.service":
		return s.Active && s.Enabled && s.RuntimeOK
	case "asterisk.05.restart":
		return asteriskSatisfied(s)
	}
	return false
}
func asteriskActionEvidence(s AsteriskSnapshot, id, source string) string {
	value := ""
	switch id {
	case "asterisk.00.build":
		value = s.Build
	case "asterisk.01.install", "asterisk.01.libraries", "asterisk.01.resources":
		value = s.Install
	case "asterisk.02.config":
		value = s.Configuration
	case "asterisk.03.unit":
		value = s.Unit
	case "asterisk.04.service":
		value = hash(struct{ Install, Config, Unit string }{s.Install, s.Configuration, s.Unit})
	case "asterisk.05.restart":
		value = s.ServiceID
	}
	return hash(struct{ Policy, ID, Source, Value string }{AsteriskPolicyDigest(), id, source, value})
}
func (c *asteriskChecks) Observe(ctx context.Context, r journal.Record) (journal.Observation, error) {
	s, e := c.host.InspectAsterisk(ctx)
	if e != nil || s.Host != c.d.Before.Host || s.Foundation != c.d.Before.Foundation || s.Database != c.d.Before.Database {
		return journal.Observation{State: journal.Unknown}, e
	}
	if asteriskActionSatisfied(s, r.ID) {
		evidence := asteriskActionEvidence(s, r.ID, c.d.Source)
		if r.State == journal.Succeeded && r.Evidence != evidence {
			return journal.Observation{State: journal.Unknown}, nil
		}
		// A started restart with no durable postcondition cannot establish that the
		// service actually restarted. An explicit review is required in that window.
		if r.ID == "asterisk.05.restart" && r.State != journal.Succeeded && c.action.ID != r.ID {
			return journal.Observation{State: journal.Unknown}, nil
		}
		return journal.Observation{State: journal.Satisfied, Evidence: evidence}, nil
	}
	// No uncertain build, partial installation or restart is blindly replayed.
	// Only independently satisfied postconditions reconcile an incomplete intent.
	return journal.Observation{State: journal.Unknown}, nil
}

func ExecuteAsteriskLab(ctx context.Context, s AsteriskStage, m artifacts.Manifest, a AsteriskLabAuthorization, stateDir string, host AsteriskHost, runner execution.CommandRunner, resume bool, log *operatorlog.Logger) (out Outcome, err error) {
	out.Traces = []execution.Trace{}
	if ctx == nil || host == nil || runner == nil {
		return out, errAsterisk
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
		if d, e := session.Snapshot(); e == nil {
			out.Journal = &d
		}
	}()
	d := s.doc()
	fresh, e := host.InspectAsterisk(ctx)
	if e != nil {
		return out, e
	}
	if fresh.Host != d.Before.Host || fresh.Foundation != d.Before.Foundation || fresh.Database != d.Before.Database {
		return out, errAsterisk
	}
	c := &asteriskChecks{host: host, d: d, current: d.Before}
	retries := map[string]journal.Retry{}
	for _, v := range s.graph.Actions() {
		retries[v.ID] = journal.Never
	}
	doc, e := journal.New(journal.Identity{Host: d.Before.Host, HostState: d.Before.Digest(), Config: d.Config, Manifest: d.Manifest, Plan: d.Parent, Policy: hash(struct{ Policy, Stage, Authorization string }{d.Policy, s.Digest(), hash(a)}), Graph: s.graph.Digest()}, s.graph, retries)
	if e != nil {
		return out, e
	}
	if resume {
		if e = session.Resume(ctx, doc, c); e != nil {
			return out, e
		}
		// Fresh actual state, rather than historical intermediate snapshots,
		// governs the remaining pending work after ALM-14 reconciles dependencies.
		if !asteriskReady(fresh) {
			return out, errAsterisk
		}
		c.current = fresh
	} else {
		if !asteriskReady(fresh) || fresh.Digest() != d.Before.Digest() {
			return out, &execution.Error{Category: execution.Precondition, StageID: "S04", ExitCode: -1, Detail: "stale asterisk plan; inspect and review again"}
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
		if e = c.Check(ctx, "asterisk.fresh"); e != nil {
			return out, e
		}
		if e = session.Begin(action.ID); e != nil {
			return out, e
		}
		if log != nil {
			_ = operatorui.RenderEvent(log, operatorui.Event{Type: operatorui.ActionStarted, StageID: "S04", ActionID: action.ID})
		}
		one := action
		one.Dependencies = []string{}
		g, e := execution.NewGraph([]execution.Action{one})
		if e != nil {
			return out, e
		}
		traces, runErr := g.Execute(ctx, runner, c)
		out.Traces = append(out.Traces, traces...)
		if len(traces) != 1 {
			return out, errAsterisk
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
	fresh, e = host.InspectAsterisk(ctx)
	if e != nil {
		return out, e
	}
	if !asteriskSatisfied(fresh) || fresh.Digest() != c.current.Digest() {
		return out, &execution.Error{Category: execution.Postcondition, StageID: "S04", ExitCode: -1, Detail: "independent Asterisk native runtime postconditions failed"}
	}
	out.NoOp = len(out.Traces) == 0
	return out, nil
}
