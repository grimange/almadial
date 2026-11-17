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

type PhoneSnapshot struct{ Host, Foundation, Foreign, Files, Services, Health, Current string }

func (s PhoneSnapshot) Digest() string { return hash(s) }
func phoneReady(s PhoneSnapshot) bool {
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
func phoneSatisfied(s PhoneSnapshot) bool {
	return phoneReady(s) && shaPattern.MatchString(s.Files) && shaPattern.MatchString(s.Services) && shaPattern.MatchString(s.Health)
}
func PhonePolicyDigest() string {
	return hash(struct{ Adapter, Revision string }{phoneAdapterSource, PhoneRevision})
}

type PhoneInputs struct {
	Parent            NetworkInputs
	Request           PhoneRequest
	Snapshot          PhoneSnapshot
	PhoneSourceSHA256 string
}
type PhoneStage struct {
	data  string
	graph execution.Graph
}
type phoneStageData struct {
	Parent, Config, Manifest, Source, Policy string
	Before                                   PhoneSnapshot
	Request                                  PhoneRequest
	PhoneSourceSHA256                        string
}

func (s PhoneStage) JSON() []byte { return []byte(s.data) }
func (s PhoneStage) Digest() string {
	if s.data == "" {
		return ""
	}
	return hash(json.RawMessage(s.data))
}
func (s PhoneStage) Graph() execution.Graph { return s.graph }
func (s PhoneStage) doc() phoneStageData {
	var d phoneStageData
	_ = json.Unmarshal(s.JSON(), &d)
	return d
}
func phoneCommand(id string) execution.Command { return dbCommand("/proc/self/exe", "__phone", id) }

func (i PhoneInputs) Prepare(m artifacts.Manifest, source VICIdialSource, perl VICIdialPerlSource, ast AsteriskSource, db DatabaseSource, phone PhoneSource) (PhoneStage, error) {
	parent, e := i.Parent.Prepare(m, source, perl, ast, db)
	if e != nil || !phoneRequestValid(i.Request) || phone.Digest() == "" || i.PhoneSourceSHA256 != phone.Digest() || !networkSatisfied(i.Parent.Snapshot) || !phoneReady(i.Snapshot) || i.Snapshot.Host != i.Parent.Snapshot.Host || i.Snapshot.Foundation != i.Parent.Snapshot.Digest() {
		return PhoneStage{}, errPhone
	}
	actions := []execution.Action{}
	for _, p := range []struct{ id, description, state string }{
		{"phone.00.activate", "Deploy pinned phone assets, protected synthetic identity and local PJSIP registration configuration", i.Snapshot.Health},
	} {
		if shaPattern.MatchString(p.state) {
			continue
		}
		deps := []string{}
		if len(actions) > 0 {
			deps = append(deps, actions[len(actions)-1].ID)
		}
		actions = append(actions, execution.Action{ID: p.id, StageID: "S10", Description: p.description, Dependencies: deps, Preconditions: []string{"phone.fresh"}, Command: phoneCommand(p.id), Postconditions: []string{p.id + ".verified"}})
	}
	g, e := execution.NewGraph(actions)
	if e != nil {
		return PhoneStage{}, e
	}
	d := phoneStageData{parent.Digest(), parent.doc().Config, m.Digest(), source.Digest(), PhonePolicyDigest(), i.Snapshot, i.Request, i.PhoneSourceSHA256}
	return PhoneStage{jsonString(d), g}, nil
}

type PhoneLabAuthorization struct {
	Scope, Host, StageSHA256, ParentSHA256, ManifestSHA256, PolicySHA256, SnapshotReference, OperatorEvidenceSHA256 string
	CandidateAcknowledged, DialingInactive, ManagementRecoveryVerified                                              bool
}

func (a PhoneLabAuthorization) validate(s PhoneStage, m artifacts.Manifest) error {
	d := s.doc()
	if s.Digest() == "" || !stageAuthorization(a.Scope, "ALM-26/disposable-lab", a.SnapshotReference, a.OperatorEvidenceSHA256, a.CandidateAcknowledged, a.ManagementRecoveryVerified, true) || a.Host != d.Before.Host || a.StageSHA256 != s.Digest() || a.ParentSHA256 != d.Parent || a.ManifestSHA256 != d.Manifest || m.Digest() != d.Manifest || a.PolicySHA256 != PhonePolicyDigest() || d.Policy != a.PolicySHA256 || !a.DialingInactive {
		return errPhone
	}
	return nil
}

type PhoneHost interface {
	InspectPhone(context.Context) (PhoneSnapshot, error)
}
type phoneChecks struct {
	host            PhoneHost
	before, current PhoneSnapshot
	action          execution.Action
}

func phoneSameFoundation(a, b PhoneSnapshot) bool {
	return a.Host == b.Host && a.Foundation == b.Foundation && a.Foreign == b.Foreign
}
func phoneValue(s PhoneSnapshot, id string) string {
	if id == "phone.00.activate" {
		return s.Health
	}

	return ""
}
func (c *phoneChecks) Supports(id string) bool {
	return id == "phone.fresh" || id == c.action.ID+".verified"
}
func (c *phoneChecks) Check(ctx context.Context, id string) error {
	s, e := c.host.InspectPhone(ctx)
	if e != nil {
		return e
	}
	if id == "phone.fresh" {
		if !phoneReady(s) || s.Digest() != c.current.Digest() {
			return errPhone
		}
		return nil
	}
	if id != c.action.ID+".verified" || !phoneReady(s) || !phoneSameFoundation(c.current, s) || !shaPattern.MatchString(phoneValue(s, c.action.ID)) {
		return errPhone
	}
	if c.action.ID != "phone.00.activate" || !phoneSatisfied(s) {
		return errPhone
	}
	c.current = s
	return nil
}
func (c *phoneChecks) Observe(ctx context.Context, r journal.Record) (journal.Observation, error) {
	s, e := c.host.InspectPhone(ctx)
	if e != nil || !phoneReady(s) || !phoneSameFoundation(c.before, s) {
		return journal.Observation{State: journal.Unknown}, e
	}
	if value := phoneValue(s, r.ID); shaPattern.MatchString(value) {
		evidence := hash(struct{ Policy, ID, Value string }{PhonePolicyDigest(), r.ID, value})
		if r.State == journal.Succeeded && r.Evidence != evidence {
			return journal.Observation{State: journal.Unknown}, nil
		}
		return journal.Observation{State: journal.Satisfied, Evidence: evidence}, nil
	}
	// Only absent/matching files can be replayed automatically. An incomplete
	// service transition requires a new assessment and reviewed execution.

	return journal.Observation{State: journal.Unknown}, nil
}

func ExecutePhoneLab(ctx context.Context, s PhoneStage, m artifacts.Manifest, a PhoneLabAuthorization, stateDir string, host PhoneHost, runner execution.CommandRunner, resume bool, log *operatorlog.Logger) (out Outcome, err error) {
	out.Traces = []execution.Trace{}
	if ctx == nil || host == nil || runner == nil || a.validate(s, m) != nil {
		return out, errPhone
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
	fresh, e := host.InspectPhone(ctx)
	if e != nil {
		return out, e
	}
	if !phoneReady(fresh) || !phoneSameFoundation(d.Before, fresh) {
		return out, errPhone
	}
	c := &phoneChecks{host: host, before: d.Before, current: d.Before}
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
			return out, errPhone
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
		if e = c.Check(ctx, "phone.fresh"); e != nil {
			return out, e
		}
		if e = session.Begin(action.ID); e != nil {
			return out, e
		}
		if log != nil {
			_ = operatorui.RenderEvent(log, operatorui.Event{Type: operatorui.ActionStarted, StageID: "S10", ActionID: action.ID})
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
			return out, errPhone
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
	fresh, e = host.InspectPhone(ctx)
	if e != nil {
		return out, e
	}
	if !phoneSatisfied(fresh) || fresh.Digest() != c.current.Digest() {
		return out, errPhone
	}
	out.NoOp = len(out.Traces) == 0
	return out, nil
}
