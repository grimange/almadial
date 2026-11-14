//go:build linux

package provision

import (
	"context"
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/grimange/almadial/internal/artifacts"
	"github.com/grimange/almadial/internal/execution"
	"github.com/grimange/almadial/internal/journal"
	"github.com/grimange/almadial/internal/operatorlog"
	"github.com/grimange/almadial/internal/operatorui"
)

type TLSSnapshot struct{ Host, Foundation, Foreign, Files, Services, Health, Current string }

func (s TLSSnapshot) Digest() string { return hash(s) }
func tlsReady(s TLSSnapshot) bool {
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
func tlsSatisfied(s TLSSnapshot) bool {
	return tlsReady(s) && shaPattern.MatchString(s.Files) && shaPattern.MatchString(s.Services) && shaPattern.MatchString(s.Health)
}
func TLSPolicyDigest() string {
	return hash(struct{ Adapter, HTTP, Loader, Vhost, RemoteVhost string }{tlsAdapterSource, tlsAsteriskHTTP, tlsLoader, TLSVhost("localhost"), RemoteTLSVhost("remote.example.test")})
}

type TLSInputs struct {
	Parent      WebInputs
	Request     TLSRequest
	Certificate TLSCertificate
	Snapshot    TLSSnapshot
}
type TLSStage struct {
	data  string
	graph execution.Graph
}
type tlsStageData struct {
	Parent, Config, Manifest, Source, Policy string
	Before                                   TLSSnapshot
	Request                                  TLSRequest
	Certificate                              TLSCertificate
}

func (s TLSStage) JSON() []byte { return []byte(s.data) }
func (s TLSStage) Digest() string {
	if s.data == "" {
		return ""
	}
	return hash(json.RawMessage(s.data))
}
func (s TLSStage) Graph() execution.Graph { return s.graph }
func (s TLSStage) doc() tlsStageData {
	var d tlsStageData
	_ = json.Unmarshal(s.JSON(), &d)
	return d
}
func tlsCommand(id string) execution.Command { return dbCommand("/proc/self/exe", "__tls", id) }

func (i TLSInputs) Prepare(m artifacts.Manifest, source VICIdialSource, perl VICIdialPerlSource, ast AsteriskSource, db DatabaseSource) (TLSStage, error) {
	parent, e := i.Parent.Prepare(m, source, perl, ast, db)
	if e != nil || !tlsRequestValid(i.Request) || !shaPattern.MatchString(i.Certificate.LeafSHA256) || !shaPattern.MatchString(i.Certificate.ChainSHA256) || !shaPattern.MatchString(i.Certificate.TrustSHA256) || !webSatisfied(i.Parent.Snapshot) || !tlsReady(i.Snapshot) || i.Snapshot.Host != i.Parent.Snapshot.Host || i.Snapshot.Foundation != i.Parent.Snapshot.Digest() {
		return TLSStage{}, errTLS
	}
	actions := []execution.Action{}
	for _, p := range []struct{ id, description, state string }{
		{"tls.00.activate", "Validate supplied certificate, atomically publish generation, reload affected TLS endpoints and verify trust", i.Snapshot.Health},
	} {
		if shaPattern.MatchString(p.state) {
			continue
		}
		deps := []string{}
		if len(actions) > 0 {
			deps = append(deps, actions[len(actions)-1].ID)
		}
		actions = append(actions, execution.Action{ID: p.id, StageID: "S08", Description: p.description, Dependencies: deps, Preconditions: []string{"tls.fresh"}, Command: tlsCommand(p.id), Postconditions: []string{p.id + ".verified"}})
	}
	g, e := execution.NewGraph(actions)
	if e != nil {
		return TLSStage{}, e
	}
	d := tlsStageData{parent.Digest(), parent.doc().Config, m.Digest(), source.Digest(), TLSPolicyDigest(), i.Snapshot, i.Request, i.Certificate}
	return TLSStage{jsonString(d), g}, nil
}

type TLSLabAuthorization struct {
	Scope, Host, StageSHA256, ParentSHA256, ManifestSHA256, PolicySHA256, SnapshotReference, OperatorEvidenceSHA256 string
	CandidateAcknowledged, DialingInactive                                                                          bool
}

func (a TLSLabAuthorization) validate(s TLSStage, m artifacts.Manifest) error {
	d := s.doc()
	if s.Digest() == "" || a.Scope != "ALM-24/disposable-lab" || a.Host != d.Before.Host || a.StageSHA256 != s.Digest() || a.ParentSHA256 != d.Parent || a.ManifestSHA256 != d.Manifest || m.Digest() != d.Manifest || a.PolicySHA256 != TLSPolicyDigest() || d.Policy != a.PolicySHA256 || !shaPattern.MatchString(a.SnapshotReference) || !shaPattern.MatchString(a.OperatorEvidenceSHA256) || !a.CandidateAcknowledged || !a.DialingInactive {
		return errTLS
	}
	return nil
}

type TLSHost interface {
	InspectTLS(context.Context) (TLSSnapshot, error)
}
type tlsChecks struct {
	host            TLSHost
	before, current TLSSnapshot
	action          execution.Action
}

func tlsSameFoundation(a, b TLSSnapshot) bool {
	return a.Host == b.Host && a.Foundation == b.Foundation && a.Foreign == b.Foreign
}
func tlsValue(s TLSSnapshot, id string) string {
	if id == "tls.00.activate" {
		return s.Health
	}

	return ""
}
func (c *tlsChecks) Supports(id string) bool {
	return id == "tls.fresh" || id == c.action.ID+".verified"
}
func (c *tlsChecks) Check(ctx context.Context, id string) error {
	s, e := c.host.InspectTLS(ctx)
	if e != nil {
		return e
	}
	if id == "tls.fresh" {
		if !tlsReady(s) || s.Digest() != c.current.Digest() {
			return errTLS
		}
		return nil
	}
	if id != c.action.ID+".verified" || !tlsReady(s) || !tlsSameFoundation(c.current, s) || !shaPattern.MatchString(tlsValue(s, c.action.ID)) {
		return errTLS
	}
	if c.action.ID != "tls.00.activate" || !tlsSatisfied(s) {
		return errTLS
	}
	c.current = s
	return nil
}
func (c *tlsChecks) Observe(ctx context.Context, r journal.Record) (journal.Observation, error) {
	s, e := c.host.InspectTLS(ctx)
	if e != nil || !tlsReady(s) || !tlsSameFoundation(c.before, s) {
		return journal.Observation{State: journal.Unknown}, e
	}
	if value := tlsValue(s, r.ID); shaPattern.MatchString(value) {
		evidence := hash(struct{ Policy, ID, Value string }{TLSPolicyDigest(), r.ID, value})
		if r.State == journal.Succeeded && r.Evidence != evidence {
			return journal.Observation{State: journal.Unknown}, nil
		}
		return journal.Observation{State: journal.Satisfied, Evidence: evidence}, nil
	}
	// Only absent/matching files can be replayed automatically. An incomplete
	// service transition requires a new assessment and reviewed execution.

	return journal.Observation{State: journal.Unknown}, nil
}

func ExecuteTLSLab(ctx context.Context, s TLSStage, m artifacts.Manifest, a TLSLabAuthorization, stateDir string, host TLSHost, runner execution.CommandRunner, resume bool, log *operatorlog.Logger) (out Outcome, err error) {
	out.Traces = []execution.Trace{}
	if ctx == nil || host == nil || runner == nil || a.validate(s, m) != nil {
		return out, errTLS
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
	fresh, e := host.InspectTLS(ctx)
	if e != nil {
		return out, e
	}
	if !tlsReady(fresh) || !tlsSameFoundation(d.Before, fresh) {
		return out, errTLS
	}
	c := &tlsChecks{host: host, before: d.Before, current: d.Before}
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
			return out, errTLS
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
		if e = c.Check(ctx, "tls.fresh"); e != nil {
			return out, e
		}
		if e = session.Begin(action.ID); e != nil {
			return out, e
		}
		if log != nil {
			_ = operatorui.RenderEvent(log, operatorui.Event{Type: operatorui.ActionStarted, StageID: "S08", ActionID: action.ID})
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
			return out, errTLS
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
	fresh, e = host.InspectTLS(ctx)
	if e != nil {
		return out, e
	}
	if !tlsSatisfied(fresh) || fresh.Digest() != c.current.Digest() {
		return out, errTLS
	}
	out.NoOp = len(out.Traces) == 0
	return out, nil
}

func tlsRequestValid(r TLSRequest) bool {
	if (r.Mode != "provided" && r.Mode != "remote") || !tlsHostname.MatchString(r.Hostname) {
		return false
	}
	if r.Mode == "remote" && (!strings.Contains(r.Hostname, ".") || r.Hostname == "localhost" || strings.Contains(r.Hostname, "..")) {
		return false
	}
	for _, p := range []string{r.CertificateFile, r.PrivateKeyFile, r.TrustFile} {
		if !filepath.IsAbs(p) || filepath.Clean(p) != p || strings.ContainsAny(p, "\n\r\x00") {
			return false
		}
	}
	return r.CertificateFile != r.PrivateKeyFile && r.TrustFile != r.PrivateKeyFile
}

var tlsHostname = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9.-]{0,252}$`)
