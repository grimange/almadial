//go:build linux

package provision

import (
	"context"
	"encoding/json"

	"github.com/grimange/almadial/internal/artifacts"
	"github.com/grimange/almadial/internal/execution"
	"github.com/grimange/almadial/internal/journal"
	"github.com/grimange/almadial/internal/operatorlog"
	"github.com/grimange/almadial/internal/operatorui"
)

type RuntimeSnapshot struct {
	Host, Foundation, Foreign, Files, Services, Health string
	SSHPort                                            uint16 `json:",omitempty"`
}

func (s RuntimeSnapshot) Digest() string { return hash(s) }
func runtimeReady(s RuntimeSnapshot) bool {
	for _, v := range []string{s.Host, s.Foundation, s.Foreign} {
		if !shaPattern.MatchString(v) {
			return false
		}
	}
	for _, v := range []string{s.Files, s.Services, s.Health} {
		if v != "absent" && v != "partial" && !shaPattern.MatchString(v) {
			return false
		}
	}
	return true
}
func runtimeSatisfied(s RuntimeSnapshot) bool {
	return runtimeReady(s) && shaPattern.MatchString(s.Files) && shaPattern.MatchString(s.Services) && shaPattern.MatchString(s.Health)
}
func RuntimePolicyDigest() string {
	return runtimePolicyDigestForSSH(0)
}
func runtimePolicyDigestForSSH(port uint16) string {
	return hash(struct {
		Inventory                                                        []RuntimeComponent
		Cron, FastAGI, Updater, Listener, Readiness, Adapter, Derivation string
	}{RuntimeInventory(), RuntimeCron(), runtimeUnit("fastagi", "FastAGI_log.pl", true), runtimeUnit("updater", "AST_update.pl", false), runtimeUnit("listener", "AST_manager_listen.pl", false), runtimeReadiness, runtimeAdapterForSSH(port), runtimeDerivationSource})
}

type RuntimeInputs struct {
	Parent   VICIdialInputs
	Snapshot RuntimeSnapshot
}
type RuntimeStage struct {
	data  string
	graph execution.Graph
}
type runtimeStageData struct {
	Parent, Config, Manifest, Source, Policy string
	Before                                   RuntimeSnapshot
}

func (s RuntimeStage) JSON() []byte { return []byte(s.data) }
func (s RuntimeStage) Digest() string {
	if s.data == "" {
		return ""
	}
	return hash(json.RawMessage(s.data))
}
func (s RuntimeStage) Graph() execution.Graph { return s.graph }
func (s RuntimeStage) doc() runtimeStageData {
	var d runtimeStageData
	_ = json.Unmarshal(s.JSON(), &d)
	return d
}
func runtimeCommand(id string) execution.Command { return dbCommand("/proc/self/exe", "__runtime", id) }

func (i RuntimeInputs) Prepare(m artifacts.Manifest, source VICIdialSource, perl VICIdialPerlSource, ast AsteriskSource, db DatabaseSource) (RuntimeStage, error) {
	parent, e := i.Parent.Prepare(m, source, perl, ast, db)
	if e != nil || !vicidialSatisfied(i.Parent.Snapshot) || !runtimeReady(i.Snapshot) || i.Snapshot.Host != i.Parent.Snapshot.Host || i.Snapshot.Foundation != i.Parent.Snapshot.Digest() {
		return RuntimeStage{}, errRuntime
	}
	actions := []execution.Action{}
	for _, p := range []struct{ id, description, state string }{
		{"runtime.00.files", "Install only absent exact S06 files; preserve all foreign schedules", i.Snapshot.Files},
		{"runtime.01.services", "Enable existing web units and systemd FastAGI only; readiness before startup", i.Snapshot.Services},
		{"runtime.02.verify", "Verify current process ownership, local listeners, deferred activation and foundation", i.Snapshot.Health},
	} {
		if shaPattern.MatchString(p.state) && (p.id != "runtime.02.verify" || len(actions) == 0) {
			continue
		}
		deps := []string{}
		if len(actions) > 0 {
			deps = append(deps, actions[len(actions)-1].ID)
		}
		actions = append(actions, execution.Action{ID: p.id, StageID: "S06", Description: p.description, Dependencies: deps, Preconditions: []string{"runtime.fresh"}, Command: runtimeCommand(p.id), Postconditions: []string{p.id + ".verified"}})
	}
	g, e := execution.NewGraph(actions)
	if e != nil {
		return RuntimeStage{}, e
	}
	d := runtimeStageData{parent.Digest(), parent.doc().Config, m.Digest(), source.Digest(), runtimePolicyDigestForSSH(i.Snapshot.SSHPort), i.Snapshot}
	return RuntimeStage{jsonString(d), g}, nil
}

type RuntimeLabAuthorization struct {
	Scope, Host, StageSHA256, ParentSHA256, ManifestSHA256, PolicySHA256, SnapshotReference, OperatorEvidenceSHA256 string
	CandidateAcknowledged, DialingInactive                                                                          bool
}

func (a RuntimeLabAuthorization) validate(s RuntimeStage, m artifacts.Manifest) error {
	d := s.doc()
	if s.Digest() == "" || !stageAuthorization(a.Scope, "ALM-21/disposable-lab", a.SnapshotReference, a.OperatorEvidenceSHA256, a.CandidateAcknowledged, false, false) || a.Host != d.Before.Host || a.StageSHA256 != s.Digest() || a.ParentSHA256 != d.Parent || a.ManifestSHA256 != d.Manifest || m.Digest() != d.Manifest || a.PolicySHA256 != runtimePolicyDigestForSSH(d.Before.SSHPort) || d.Policy != a.PolicySHA256 || !a.DialingInactive {
		return errRuntime
	}
	return nil
}

type RuntimeHost interface {
	InspectRuntime(context.Context) (RuntimeSnapshot, error)
}
type runtimeChecks struct {
	host            RuntimeHost
	before, current RuntimeSnapshot
	action          execution.Action
	policy          string
}

func runtimeSameFoundation(a, b RuntimeSnapshot) bool {
	return a.Host == b.Host && a.Foundation == b.Foundation && a.Foreign == b.Foreign && a.SSHPort == b.SSHPort
}
func runtimeValue(s RuntimeSnapshot, id string) string {
	switch id {
	case "runtime.00.files":
		return s.Files
	case "runtime.01.services":
		return s.Services
	case "runtime.02.verify":
		return s.Health
	}
	return ""
}
func (c *runtimeChecks) Supports(id string) bool {
	return id == "runtime.fresh" || id == c.action.ID+".verified"
}
func (c *runtimeChecks) Check(ctx context.Context, id string) error {
	s, e := c.host.InspectRuntime(ctx)
	if e != nil {
		return e
	}
	if id == "runtime.fresh" {
		if !runtimeReady(s) || s.Digest() != c.current.Digest() {
			return errRuntime
		}
		return nil
	}
	if id != c.action.ID+".verified" || !runtimeReady(s) || !runtimeSameFoundation(c.current, s) || !shaPattern.MatchString(runtimeValue(s, c.action.ID)) {
		return errRuntime
	}
	w := c.current
	switch c.action.ID {
	case "runtime.00.files":
		// Native readiness can become observable after restoring a missing
		// generated prerequisite while already-running workers remain unchanged.
		w.Files = s.Files
		if c.policy == NativeRuntimePolicyDigest() {
			w.Services, w.Health = s.Services, s.Health
		}
	case "runtime.01.services":
		w.Services = s.Services
		w.Health = s.Health
	case "runtime.02.verify":
		w.Health = s.Health
	default:
		return errRuntime
	}
	if w.Digest() != s.Digest() {
		return errRuntime
	}
	c.current = s
	return nil
}
func (c *runtimeChecks) Observe(ctx context.Context, r journal.Record) (journal.Observation, error) {
	s, e := c.host.InspectRuntime(ctx)
	if e != nil || !runtimeReady(s) || !runtimeSameFoundation(c.before, s) {
		return journal.Observation{State: journal.Unknown}, e
	}
	if value := runtimeValue(s, r.ID); shaPattern.MatchString(value) {
		policy := c.policy
		if policy == "" {
			policy = RuntimePolicyDigest()
		}
		evidence := hash(struct{ Policy, ID, Value string }{policy, r.ID, value})
		if r.State == journal.Succeeded && r.Evidence != evidence {
			return journal.Observation{State: journal.Unknown}, nil
		}
		return journal.Observation{State: journal.Satisfied, Evidence: evidence}, nil
	}
	// Only absent/matching files can be replayed automatically. An incomplete
	// service transition requires a new assessment and reviewed execution.
	if r.ID == "runtime.00.files" || c.policy == NativeRuntimePolicyDigest() {
		return journal.Observation{State: journal.Absent, Evidence: s.Digest()}, nil
	}
	return journal.Observation{State: journal.Unknown}, nil
}

func ExecuteRuntimeLab(ctx context.Context, s RuntimeStage, m artifacts.Manifest, a RuntimeLabAuthorization, stateDir string, host RuntimeHost, runner execution.CommandRunner, resume bool, log *operatorlog.Logger) (out Outcome, err error) {
	out.Traces = []execution.Trace{}
	if ctx == nil || host == nil || runner == nil || a.validate(s, m) != nil {
		return out, errRuntime
	}
	return executeRuntimeStage(ctx, s, stateDir, host, runner, resume, log, hash(a))
}

// Shared durable executor for the isolated and remote native runtime policies.
func executeRuntimeStage(ctx context.Context, s RuntimeStage, stateDir string, host RuntimeHost, runner execution.CommandRunner, resume bool, log *operatorlog.Logger, authorization string) (out Outcome, err error) {
	out.Traces = []execution.Trace{}
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
	fresh, e := host.InspectRuntime(ctx)
	if e != nil {
		return out, e
	}
	if !runtimeReady(fresh) || !runtimeSameFoundation(d.Before, fresh) {
		return out, errRuntime
	}
	c := &runtimeChecks{host: host, before: d.Before, current: d.Before, policy: d.Policy}
	retries := map[string]journal.Retry{}
	for _, action := range s.graph.Actions() {
		retries[action.ID] = journal.Never
		if action.ID == "runtime.00.files" || d.Policy == NativeRuntimePolicyDigest() {
			retries[action.ID] = journal.Idempotent
		}
	}
	doc, e := journal.New(journal.Identity{Host: d.Before.Host, HostState: d.Before.Digest(), Config: d.Config, Manifest: d.Manifest, Plan: d.Parent, Policy: hash(struct{ Policy, Stage, Authorization string }{d.Policy, s.Digest(), authorization}), Graph: s.graph.Digest()}, s.graph, retries)
	if e != nil {
		return out, e
	}
	if resume {
		if e = session.Resume(ctx, doc, c); e != nil {
			return out, e
		}
		c.current = fresh
	} else {
		if fresh.Digest() != d.Before.Digest() {
			return out, errRuntime
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
		if e = c.Check(ctx, "runtime.fresh"); e != nil {
			return out, e
		}
		if e = session.Begin(action.ID); e != nil {
			return out, e
		}
		if log != nil {
			_ = operatorui.RenderEvent(log, operatorui.Event{Type: operatorui.ActionStarted, StageID: "S06", ActionID: action.ID})
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
			return out, errRuntime
		}
		trace := traces[0]
		if trace.Outcome == "blocked" {
			trace.Outcome = "failed"
		}
		if e = session.Finish(ctx, trace, c); e != nil {
			return out, e
		}
		if log != nil {
			if ev, ee := operatorui.EventFromTrace(trace); ee == nil {
				_ = operatorui.RenderEvent(log, ev)
			}
		}
		if runErr != nil {
			return out, runErr
		}
	}
	fresh, e = host.InspectRuntime(ctx)
	if e != nil {
		return out, e
	}
	if !runtimeSatisfied(fresh) || fresh.Digest() != c.current.Digest() {
		return out, errRuntime
	}
	out.NoOp = len(out.Traces) == 0
	return out, nil
}
