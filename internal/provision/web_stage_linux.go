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

type WebSnapshot struct{ Host, Foundation, Foreign, Files, Services, Health string }

func (s WebSnapshot) Digest() string { return hash(s) }
func webReady(s WebSnapshot) bool {
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
func webSatisfied(s WebSnapshot) bool {
	return webReady(s) && shaPattern.MatchString(s.Files) && shaPattern.MatchString(s.Services) && shaPattern.MatchString(s.Health)
}
func WebPolicyDigest() string {
	return hash(struct{ Vhost, Loader, Adapter string }{WebVhost(), webLoader, webAdapter()})
}

type WebInputs struct {
	Parent   RuntimeInputs
	Snapshot WebSnapshot
}
type WebStage struct {
	data  string
	graph execution.Graph
}
type webStageData struct {
	Parent, Config, Manifest, Source, Policy string
	Before                                   WebSnapshot
}

func (s WebStage) JSON() []byte { return []byte(s.data) }
func (s WebStage) Digest() string {
	if s.data == "" {
		return ""
	}
	return hash(json.RawMessage(s.data))
}
func (s WebStage) Graph() execution.Graph { return s.graph }
func (s WebStage) doc() webStageData {
	var d webStageData
	_ = json.Unmarshal(s.JSON(), &d)
	return d
}
func webCommand(id string) execution.Command { return dbCommand("/proc/self/exe", "__web", id) }

func (i WebInputs) Prepare(m artifacts.Manifest, source VICIdialSource, perl VICIdialPerlSource, ast AsteriskSource, db DatabaseSource) (WebStage, error) {
	parent, e := i.Parent.Prepare(m, source, perl, ast, db)
	if e != nil || !runtimeSatisfied(i.Parent.Snapshot) || !webReady(i.Snapshot) || i.Snapshot.Host != i.Parent.Snapshot.Host || i.Snapshot.Foundation != i.Parent.Snapshot.Digest() {
		return WebStage{}, errWeb
	}
	actions := []execution.Action{}
	for _, p := range []struct{ id, description, state string }{
		{"web.00.files", "Install missing owned S07 vhost and loader; preserve foreign Apache configuration", i.Snapshot.Files},
		{"web.01.services", "Validate Apache/PHP configuration before graceful web activation", i.Snapshot.Services},
		{"web.02.verify", "Complete guarded first-login setup and verify authenticated local application", i.Snapshot.Health},
	} {
		if shaPattern.MatchString(p.state) && (p.id != "web.02.verify" || len(actions) == 0) {
			continue
		}
		deps := []string{}
		if len(actions) > 0 {
			deps = append(deps, actions[len(actions)-1].ID)
		}
		actions = append(actions, execution.Action{ID: p.id, StageID: "S07", Description: p.description, Dependencies: deps, Preconditions: []string{"web.fresh"}, Command: webCommand(p.id), Postconditions: []string{p.id + ".verified"}})
	}
	g, e := execution.NewGraph(actions)
	if e != nil {
		return WebStage{}, e
	}
	d := webStageData{parent.Digest(), parent.doc().Config, m.Digest(), source.Digest(), WebPolicyDigest(), i.Snapshot}
	return WebStage{jsonString(d), g}, nil
}

type WebLabAuthorization struct {
	Scope, Host, StageSHA256, ParentSHA256, ManifestSHA256, PolicySHA256, SnapshotReference, OperatorEvidenceSHA256 string
	CandidateAcknowledged, DialingInactive                                                                          bool
}

func (a WebLabAuthorization) validate(s WebStage, m artifacts.Manifest) error {
	d := s.doc()
	if s.Digest() == "" || a.Scope != "ALM-23/disposable-lab" || a.Host != d.Before.Host || a.StageSHA256 != s.Digest() || a.ParentSHA256 != d.Parent || a.ManifestSHA256 != d.Manifest || m.Digest() != d.Manifest || a.PolicySHA256 != WebPolicyDigest() || d.Policy != a.PolicySHA256 || !shaPattern.MatchString(a.SnapshotReference) || !shaPattern.MatchString(a.OperatorEvidenceSHA256) || !a.CandidateAcknowledged || !a.DialingInactive {
		return errWeb
	}
	return nil
}

type WebHost interface {
	InspectWeb(context.Context) (WebSnapshot, error)
}
type webChecks struct {
	host            WebHost
	before, current WebSnapshot
	action          execution.Action
}

func webSameFoundation(a, b WebSnapshot) bool {
	return a.Host == b.Host && a.Foundation == b.Foundation && a.Foreign == b.Foreign
}
func webValue(s WebSnapshot, id string) string {
	switch id {
	case "web.00.files":
		return s.Files
	case "web.01.services":
		return s.Services
	case "web.02.verify":
		return s.Health
	}
	return ""
}
func (c *webChecks) Supports(id string) bool {
	return id == "web.fresh" || id == c.action.ID+".verified"
}
func (c *webChecks) Check(ctx context.Context, id string) error {
	s, e := c.host.InspectWeb(ctx)
	if e != nil {
		return e
	}
	if id == "web.fresh" {
		if !webReady(s) || s.Digest() != c.current.Digest() {
			return errWeb
		}
		return nil
	}
	if id != c.action.ID+".verified" || !webReady(s) || !webSameFoundation(c.current, s) || !shaPattern.MatchString(webValue(s, c.action.ID)) {
		return errWeb
	}
	w := c.current
	switch c.action.ID {
	case "web.00.files":
		w.Files = s.Files
	case "web.01.services":
		w.Services = s.Services
		w.Health = s.Health
	case "web.02.verify":
		w.Health = s.Health
	default:
		return errWeb
	}
	if w.Digest() != s.Digest() {
		return errWeb
	}
	c.current = s
	return nil
}
func (c *webChecks) Observe(ctx context.Context, r journal.Record) (journal.Observation, error) {
	s, e := c.host.InspectWeb(ctx)
	if e != nil || !webReady(s) || !webSameFoundation(c.before, s) {
		return journal.Observation{State: journal.Unknown}, e
	}
	if value := webValue(s, r.ID); shaPattern.MatchString(value) {
		evidence := hash(struct{ Policy, ID, Value string }{WebPolicyDigest(), r.ID, value})
		if r.State == journal.Succeeded && r.Evidence != evidence {
			return journal.Observation{State: journal.Unknown}, nil
		}
		return journal.Observation{State: journal.Satisfied, Evidence: evidence}, nil
	}
	// Only absent/matching files can be replayed automatically. An incomplete
	// service transition requires a new assessment and reviewed execution.
	if r.ID == "web.00.files" {
		return journal.Observation{State: journal.Absent, Evidence: s.Digest()}, nil
	}
	return journal.Observation{State: journal.Unknown}, nil
}

func ExecuteWebLab(ctx context.Context, s WebStage, m artifacts.Manifest, a WebLabAuthorization, stateDir string, host WebHost, runner execution.CommandRunner, resume bool, log *operatorlog.Logger) (out Outcome, err error) {
	out.Traces = []execution.Trace{}
	if ctx == nil || host == nil || runner == nil || a.validate(s, m) != nil {
		return out, errWeb
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
	fresh, e := host.InspectWeb(ctx)
	if e != nil {
		return out, e
	}
	if !webReady(fresh) || !webSameFoundation(d.Before, fresh) {
		return out, errWeb
	}
	c := &webChecks{host: host, before: d.Before, current: d.Before}
	retries := map[string]journal.Retry{}
	for _, action := range s.graph.Actions() {
		retries[action.ID] = journal.Never
		if action.ID == "web.00.files" {
			retries[action.ID] = journal.Idempotent
		}
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
			return out, errWeb
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
		if e = c.Check(ctx, "web.fresh"); e != nil {
			return out, e
		}
		if e = session.Begin(action.ID); e != nil {
			return out, e
		}
		if log != nil {
			_ = operatorui.RenderEvent(log, operatorui.Event{Type: operatorui.ActionStarted, StageID: "S07", ActionID: action.ID})
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
			return out, errWeb
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
	fresh, e = host.InspectWeb(ctx)
	if e != nil {
		return out, e
	}
	if !webSatisfied(fresh) || fresh.Digest() != c.current.Digest() {
		return out, errWeb
	}
	out.NoOp = len(out.Traces) == 0
	return out, nil
}
