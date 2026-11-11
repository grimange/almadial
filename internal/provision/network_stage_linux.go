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

type NetworkSnapshot struct{ Host, Foundation, Foreign, Files, Services, Health, Current string }

func (s NetworkSnapshot) Digest() string { return hash(s) }
func networkReady(s NetworkSnapshot) bool {
	if shaPattern.MatchString(s.Health) && (!shaPattern.MatchString(s.Files) || !shaPattern.MatchString(s.Services)) {
		return false
	}
	for _, v := range []string{s.Host, s.Foundation, s.Foreign, s.Current} {
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
func networkSatisfied(s NetworkSnapshot) bool {
	return networkReady(s) && shaPattern.MatchString(s.Files) && shaPattern.MatchString(s.Services) && shaPattern.MatchString(s.Health)
}
func NetworkPolicyDigest() string {
	template, _ := NetworkPolicy(NetworkRequest{Mode: "isolated", Interface: "eth0", ManagementAddress: "10.0.0.1", SSHPort: 22})
	return hash(struct{ Adapter, Template, Dependencies string }{networkAdapterSource, template, networkDependencyCandidates})
}

type NetworkInputs struct {
	Parent       TLSInputs
	Request      NetworkRequest
	Snapshot     NetworkSnapshot
	Dependencies *NetworkDependencies
}
type NetworkStage struct {
	data  string
	graph execution.Graph
}
type networkStageData struct {
	Parent, Config, Manifest, Source, Policy string
	Before                                   NetworkSnapshot
	Request                                  NetworkRequest
	DependenciesSHA256                       string
}

func (s NetworkStage) JSON() []byte { return []byte(s.data) }
func (s NetworkStage) Digest() string {
	if s.data == "" {
		return ""
	}
	return hash(json.RawMessage(s.data))
}
func (s NetworkStage) Graph() execution.Graph { return s.graph }
func (s NetworkStage) doc() networkStageData {
	var d networkStageData
	_ = json.Unmarshal(s.JSON(), &d)
	return d
}
func networkCommand(id string) execution.Command { return dbCommand("/proc/self/exe", "__network", id) }

func (i NetworkInputs) Prepare(m artifacts.Manifest, source VICIdialSource, perl VICIdialPerlSource, ast AsteriskSource, db DatabaseSource) (NetworkStage, error) {
	if i.Dependencies != nil && (!i.Dependencies.valid() || i.Dependencies.Before.Digest() != i.Parent.Parent.Parent.Parent.Parent.Parent.Parent.Parent.Inventory.Digest()) {
		return NetworkStage{}, errNetwork
	}
	parent, e := i.Parent.Prepare(m, source, perl, ast, db)
	if e != nil || !networkRequestValid(i.Request) || !tlsSatisfied(i.Parent.Snapshot) || !networkReady(i.Snapshot) || i.Snapshot.Host != i.Parent.Snapshot.Host || i.Snapshot.Foundation != i.Parent.Snapshot.Digest() {
		return NetworkStage{}, errNetwork
	}
	actions := []execution.Action{}
	for _, p := range []struct{ id, description, state string }{
		{"network.00.activate", "Validate management access, publish owned firewalld policy and verify effective isolation", i.Snapshot.Health},
	} {
		if shaPattern.MatchString(p.state) {
			continue
		}
		deps := []string{}
		if len(actions) > 0 {
			deps = append(deps, actions[len(actions)-1].ID)
		}
		actions = append(actions, execution.Action{ID: p.id, StageID: "S09", Description: p.description, Dependencies: deps, Preconditions: []string{"network.fresh"}, Command: networkCommand(p.id), Postconditions: []string{p.id + ".verified"}})
	}
	g, e := execution.NewGraph(actions)
	if e != nil {
		return NetworkStage{}, e
	}
	d := networkStageData{parent.Digest(), parent.doc().Config, m.Digest(), source.Digest(), NetworkPolicyDigest(), i.Snapshot, i.Request, hash(i.Dependencies)}
	return NetworkStage{jsonString(d), g}, nil
}

type NetworkLabAuthorization struct {
	Scope, Host, StageSHA256, ParentSHA256, ManifestSHA256, PolicySHA256, SnapshotReference, OperatorEvidenceSHA256 string
	CandidateAcknowledged, DialingInactive, ManagementRecoveryVerified                                              bool
}

func (a NetworkLabAuthorization) validate(s NetworkStage, m artifacts.Manifest) error {
	d := s.doc()
	if s.Digest() == "" || a.Scope != "ALM-25/disposable-lab" || a.Host != d.Before.Host || a.StageSHA256 != s.Digest() || a.ParentSHA256 != d.Parent || a.ManifestSHA256 != d.Manifest || m.Digest() != d.Manifest || a.PolicySHA256 != NetworkPolicyDigest() || d.Policy != a.PolicySHA256 || !shaPattern.MatchString(a.SnapshotReference) || !shaPattern.MatchString(a.OperatorEvidenceSHA256) || !a.CandidateAcknowledged || !a.DialingInactive || !a.ManagementRecoveryVerified {
		return errNetwork
	}
	return nil
}

type NetworkHost interface {
	InspectNetwork(context.Context) (NetworkSnapshot, error)
}
type networkChecks struct {
	host            NetworkHost
	before, current NetworkSnapshot
	action          execution.Action
}

func networkSameFoundation(a, b NetworkSnapshot) bool {
	return a.Host == b.Host && a.Foundation == b.Foundation && a.Foreign == b.Foreign
}
func networkValue(s NetworkSnapshot, id string) string {
	if id == "network.00.activate" {
		return s.Health
	}

	return ""
}
func (c *networkChecks) Supports(id string) bool {
	return id == "network.fresh" || id == c.action.ID+".verified"
}
func (c *networkChecks) Check(ctx context.Context, id string) error {
	s, e := c.host.InspectNetwork(ctx)
	if e != nil {
		return e
	}
	if id == "network.fresh" {
		if !networkReady(s) || s.Digest() != c.current.Digest() {
			return errNetwork
		}
		return nil
	}
	if id != c.action.ID+".verified" || !networkReady(s) || !networkSameFoundation(c.current, s) || !shaPattern.MatchString(networkValue(s, c.action.ID)) {
		return errNetwork
	}
	if c.action.ID != "network.00.activate" || !networkSatisfied(s) {
		return errNetwork
	}
	c.current = s
	return nil
}
func (c *networkChecks) Observe(ctx context.Context, r journal.Record) (journal.Observation, error) {
	s, e := c.host.InspectNetwork(ctx)
	if e != nil || !networkReady(s) || !networkSameFoundation(c.before, s) {
		return journal.Observation{State: journal.Unknown}, e
	}
	if value := networkValue(s, r.ID); shaPattern.MatchString(value) {
		evidence := hash(struct{ Policy, ID, Value string }{NetworkPolicyDigest(), r.ID, value})
		if r.State == journal.Succeeded && r.Evidence != evidence {
			return journal.Observation{State: journal.Unknown}, nil
		}
		return journal.Observation{State: journal.Satisfied, Evidence: evidence}, nil
	}
	// Only absent/matching files can be replayed automatically. An incomplete
	// service transition requires a new assessment and reviewed execution.

	return journal.Observation{State: journal.Unknown}, nil
}

func ExecuteNetworkLab(ctx context.Context, s NetworkStage, m artifacts.Manifest, a NetworkLabAuthorization, stateDir string, host NetworkHost, runner execution.CommandRunner, resume bool, log *operatorlog.Logger) (out Outcome, err error) {
	out.Traces = []execution.Trace{}
	if ctx == nil || host == nil || runner == nil || a.validate(s, m) != nil {
		return out, errNetwork
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
	fresh, e := host.InspectNetwork(ctx)
	if e != nil {
		return out, e
	}
	if !networkReady(fresh) || !networkSameFoundation(d.Before, fresh) {
		return out, errNetwork
	}
	c := &networkChecks{host: host, before: d.Before, current: d.Before}
	retries := map[string]journal.Retry{}
	for _, action := range s.graph.Actions() {
		retries[action.ID] = journal.Never
	}
	doc, e := journal.New(journal.Identity{Host: d.Before.Host, HostState: d.Before.Digest(), Config: d.Config, Manifest: d.Manifest, Plan: d.Parent, Policy: hash(struct{ Policy, Stage, Authorization string }{d.Policy, s.Digest(), hash(a)}), Graph: s.graph.Digest()}, s.graph, retries)
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
			return out, errNetwork
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
		if e = c.Check(ctx, "network.fresh"); e != nil {
			return out, e
		}
		if e = session.Begin(action.ID); e != nil {
			return out, e
		}
		if log != nil {
			_ = operatorui.RenderEvent(log, operatorui.Event{Type: operatorui.ActionStarted, StageID: "S09", ActionID: action.ID})
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
			return out, errNetwork
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
	fresh, e = host.InspectNetwork(ctx)
	if e != nil {
		return out, e
	}
	if !networkSatisfied(fresh) || fresh.Digest() != c.current.Digest() {
		return out, errNetwork
	}
	out.NoOp = len(out.Traces) == 0
	return out, nil
}
