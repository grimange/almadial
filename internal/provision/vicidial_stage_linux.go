//go:build linux

package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grimange/almadial/internal/artifacts"
	"github.com/grimange/almadial/internal/execution"
	"github.com/grimange/almadial/internal/journal"
	"github.com/grimange/almadial/internal/operatorlog"
	"github.com/grimange/almadial/internal/operatorui"
)

// Component identities come from fresh native inspection, not journal success.
// A partial component is admissible only when its native inspector has proved
// that every existing managed file matches and missing files can be completed.
type VICIdialSnapshot struct {
	Host, Foundation, Database, Asterisk, Source, Dependency, GeneratorInputs string
	Files, Perl, Configuration, Registration, Generated, Web, Runtime         string
}

func (s VICIdialSnapshot) Digest() string { return hash(s) }
func vicidialReady(s VICIdialSnapshot) bool {
	for _, x := range []string{s.Host, s.Foundation, s.Database, s.Asterisk, s.Source, s.Dependency, s.GeneratorInputs} {
		if !shaPattern.MatchString(x) {
			return false
		}
	}
	for _, x := range []string{s.Files, s.Perl, s.Configuration, s.Registration, s.Generated, s.Web, s.Runtime} {
		if x != "absent" && x != "partial" && !shaPattern.MatchString(x) {
			return false
		}
	}
	return true
}
func vicidialSatisfied(s VICIdialSnapshot) bool {
	if !vicidialReady(s) {
		return false
	}
	for _, x := range []string{s.Files, s.Perl, s.Configuration, s.Registration, s.Generated, s.Web, s.Runtime} {
		if !shaPattern.MatchString(x) {
			return false
		}
	}
	return true
}

type VICIdialInputs struct {
	Parent                         AsteriskInputs
	Snapshot                       VICIdialSnapshot
	SourceSHA256, DependencySHA256 string
}
type VICIdialStage struct {
	data  string
	graph execution.Graph
}
type vicidialStageData struct {
	Parent, Config, Manifest, Policy, Source, Dependency, Deployment, Graph string
	Before                                                                  VICIdialSnapshot
}

func (s VICIdialStage) JSON() []byte { return []byte(s.data) }
func (s VICIdialStage) Digest() string {
	if s.data == "" {
		return ""
	}
	return hash(json.RawMessage(s.data))
}
func (s VICIdialStage) Graph() execution.Graph { return s.graph }
func (s VICIdialStage) doc() vicidialStageData {
	var d vicidialStageData
	_ = json.Unmarshal(s.JSON(), &d)
	return d
}

func VICIdialPolicyDigest() string {
	// Bind configuration derivation with public placeholders, never live secrets.
	files, _ := vicidialConfiguration(strings.Repeat("0", 48), strings.Repeat("0", 20))
	templates := [][]string{}
	for _, f := range files {
		templates = append(templates, []string{f.path, f.owner, f.group, f.label, fmt.Sprint(f.mode), string(f.content)})
	}
	sql, _ := vicidialRegistrationSQL(strings.Repeat("0", 20))
	guarded, _ := guardedVICIdialRegistrationSQL(vicidialSeedRegistration(), strings.Repeat("0", 20))
	return hash(struct {
		Source, Generator, FileAdapter, Web, Service, DatabaseOwner, Registration string
		Templates                                                                 [][]string
		Wrappers                                                                  map[string][2]string
		Prerequisites                                                             string
		GeneratorInputs, Runtime                                                  string
	}{vicidialTreeSHA, vicidialGeneratorSourceSHA + vicidialGeneratorBlockSHA + vicidialGeneratorDerivedSHA, vicidialFilesAdapter, vicidialHTTPD, vicidialHTTPDDropIn, "S03-only; S05 guarded TESTast registration and upstream generator; no initial import/v1", sql + vicidialRegistrationQuery + guarded, templates, vicidialAsteriskWrappers(), hash(struct {
		PerlProbe, PHPProbe, PerlVersion, PHPVersion string
		Modules                                      map[string]string
	}{vicidialPerlProbe, vicidialPHPProbe, "5.40.2", "8.3.36", vicidialPerlVersions}), hash(struct {
		Tables                   []string
		RegistrationColumns      map[string]bool
		Policy, Bridges, Rebuild string
		Reloads                  []string
	}{vicidialGeneratorTables, vicidialRegistrationColumns, "HEX(CAST AS BINARY)-JSON/sorted-multiset/v2; system_settings.webroot_writable and pass_key owned; blank pass_key cryptographically initialized under registration intent; valid nonblank keys preserved; local bridge slots only 300 inactive 9600000..9600299; preserve seed phones; retain NULL and nonlocal bridge rows; five exact S05 wrapper transitions; AMI local login ping status deny command", vicidialBridgesQuery, vicidialRebuildSQL, vicidialReloadCommands()}), hash(struct {
		Harness      string
		Syntax       []string
		Web, Runtime string
	}{vicidialPhoneFixtureHarness, vicidialSyntaxScripts(), "anonymous local HTTP; no redirects/cookies; 401 admin, 200 agent form, 403 options; exact loopback listener and unit selection/v1", "fresh native components; non-root Asterisk::AGI load from M0 versioned runtime path; repeat isolated upstream PJSIP phone generation normal/reserved/empty; no persistent phone writes; deferred jobs inactive; private component receipt; accepted historical restart receipt plus fresh process verification; normalize only DB/Asterisk systemd invocation across VM snapshot/v3"})})
}
func vicidialActionCommand(id string) execution.Command {
	return dbCommand("/proc/self/exe", "__vicidial", id)
}
func (i VICIdialInputs) Prepare(m artifacts.Manifest, source VICIdialSource, dependency VICIdialPerlSource, asterisk AsteriskSource, database DatabaseSource) (VICIdialStage, error) {
	p, e := i.Parent.Prepare(m, asterisk)
	if e != nil || !asteriskSatisfied(i.Parent.Snapshot) || i.Parent.Parent.SourceSHA256 != database.Digest() || VerifyDatabaseForApplication(i.Parent.Parent.Snapshot, database) != nil {
		return VICIdialStage{}, errVICIdial
	}
	s := i.Snapshot
	if !vicidialReady(s) || source.Digest() == "" || dependency.Digest() == "" || i.SourceSHA256 != source.Digest() || i.DependencySHA256 != dependency.Digest() || s.Source != source.Digest() || s.Dependency != dependency.Digest() || s.Host != i.Parent.Snapshot.Host || s.Foundation != i.Parent.Snapshot.Foundation || s.Database != i.Parent.Snapshot.Database || s.Asterisk != i.Parent.Snapshot.Digest() {
		return VICIdialStage{}, errVICIdial
	}
	deployment, e := source.Deployment()
	if e != nil || len(deployment) == 0 {
		return VICIdialStage{}, errVICIdial
	}
	actions := []execution.Action{}
	for _, v := range []struct{ id, description, state string }{
		{"vicidial.00.files", "Install only absent verified r4037 files; preserve existing matching files and unrelated content", s.Files},
		{"vicidial.01.perl", "Install only the reviewed pure-Perl Net::Telnet module and attribution", s.Perl},
		{"vicidial.02.configuration", "Provision protected application settings and owned local AMI include", s.Configuration},
		{"vicidial.03.registration", "Guard the exact application server preimage; preserve the S03 schema and credentials", s.Registration},
		{"vicidial.04.generate", "Run the isolated M0 generator; preserve preimages and publish only modern configuration includes", s.Generated},
		{"vicidial.05.web", "Select the owned local Apache configuration and verify PHP-FPM application entry points", s.Web},
		{"vicidial.06.verify", "Independently verify integration, reload persistence, deferred scheduling and accepted foundation", s.Runtime},
	} {
		if shaPattern.MatchString(v.state) && (v.id != "vicidial.06.verify" || len(actions) == 0) {
			continue
		}
		deps := []string{}
		if len(actions) > 0 {
			deps = append(deps, actions[len(actions)-1].ID)
		}
		actions = append(actions, execution.Action{ID: v.id, StageID: "S05", Description: v.description, Dependencies: deps, Preconditions: []string{"vicidial.fresh"}, Command: vicidialActionCommand(v.id), Postconditions: []string{v.id + ".verified"}})
	}
	g, e := execution.NewGraph(actions)
	if e != nil {
		return VICIdialStage{}, e
	}
	d := vicidialStageData{p.Digest(), p.doc().Config, m.Digest(), VICIdialPolicyDigest(), s.Source, s.Dependency, hash(deployment), g.Digest(), s}
	return VICIdialStage{jsonString(d), g}, nil
}

type VICIdialLabAuthorization struct {
	Scope, Host, StageSHA256, ParentSHA256, ManifestSHA256, SourceSHA256, DependencySHA256 string
	SnapshotReference, OperatorEvidenceSHA256                                              string
	CandidateAcknowledged                                                                  bool
}

func (a VICIdialLabAuthorization) validate(s VICIdialStage, m artifacts.Manifest) error {
	d := s.doc()
	if s.Digest() == "" || a.Scope != "ALM-20/disposable-lab" || a.Host != d.Before.Host || a.StageSHA256 != s.Digest() || a.ParentSHA256 != d.Parent || a.ManifestSHA256 != d.Manifest || m.Digest() != d.Manifest || a.SourceSHA256 != d.Source || a.DependencySHA256 != d.Dependency || d.Policy != VICIdialPolicyDigest() || !shaPattern.MatchString(a.SnapshotReference) || !shaPattern.MatchString(a.OperatorEvidenceSHA256) || !a.CandidateAcknowledged {
		return &execution.Error{Category: execution.Precondition, StageID: "S05", ExitCode: -1, Detail: "exact ALM-20 laboratory authorization, pinned sources and fresh recovery snapshot required"}
	}
	return nil
}

type VICIdialHost interface {
	InspectVICIdial(context.Context) (VICIdialSnapshot, error)
}
type vicidialChecks struct {
	host    VICIdialHost
	d       vicidialStageData
	current VICIdialSnapshot
	action  execution.Action
}

func vicidialSameFoundation(a, b VICIdialSnapshot) bool {
	return a.Host == b.Host && a.Foundation == b.Foundation && a.Database == b.Database && a.Asterisk == b.Asterisk && a.Source == b.Source && a.Dependency == b.Dependency && a.GeneratorInputs == b.GeneratorInputs
}
func vicidialActionValue(s VICIdialSnapshot, id string) string {
	switch id {
	case "vicidial.00.files":
		return s.Files
	case "vicidial.01.perl":
		return s.Perl
	case "vicidial.02.configuration":
		return s.Configuration
	case "vicidial.03.registration":
		return s.Registration
	case "vicidial.04.generate":
		return s.Generated
	case "vicidial.05.web":
		return s.Web
	case "vicidial.06.verify":
		return s.Runtime
	}
	return ""
}
func mergeVICIdial(before, after VICIdialSnapshot, id string) (VICIdialSnapshot, error) {
	if !vicidialReady(after) || !vicidialSameFoundation(before, after) || !shaPattern.MatchString(vicidialActionValue(after, id)) {
		return before, errVICIdial
	}
	w := before
	switch id {
	case "vicidial.00.files":
		w.Files = after.Files
	case "vicidial.01.perl":
		w.Perl = after.Perl
	case "vicidial.02.configuration":
		w.Configuration = after.Configuration
	case "vicidial.03.registration":
		w.Registration = after.Registration
	case "vicidial.04.generate":
		w.Generated = after.Generated
	case "vicidial.05.web":
		w.Web = after.Web
	case "vicidial.06.verify":
		w.Runtime = after.Runtime
	default:
		return before, errVICIdial
	}
	// A later verification receipt is invalidated by an intentional preceding
	// action. No other component may change as a hidden side effect.
	if id != "vicidial.06.verify" && after.Runtime == "absent" {
		w.Runtime = after.Runtime
	}
	if w.Digest() != after.Digest() {
		return before, errVICIdial
	}
	return w, nil
}
func (c *vicidialChecks) Supports(id string) bool {
	return id == "vicidial.fresh" || id == c.action.ID+".verified"
}
func (c *vicidialChecks) Check(ctx context.Context, id string) error {
	s, e := c.host.InspectVICIdial(ctx)
	if e != nil {
		return e
	}
	if id == "vicidial.fresh" {
		if !vicidialReady(s) || s.Digest() != c.current.Digest() {
			return errVICIdial
		}
		return nil
	}
	if id != c.action.ID+".verified" {
		return errVICIdial
	}
	w, e := mergeVICIdial(c.current, s, c.action.ID)
	if e == nil {
		c.current = w
	}
	return e
}
func (c *vicidialChecks) Observe(ctx context.Context, r journal.Record) (journal.Observation, error) {
	s, e := c.host.InspectVICIdial(ctx)
	if e != nil || !vicidialReady(s) || !vicidialSameFoundation(c.d.Before, s) {
		return journal.Observation{State: journal.Unknown}, e
	}
	value := vicidialActionValue(s, r.ID)
	if shaPattern.MatchString(value) {
		evidence := hash(struct{ Policy, ID, Source, Dependency, Value string }{c.d.Policy, r.ID, c.d.Source, c.d.Dependency, value})
		if r.State == journal.Succeeded && r.Evidence != evidence {
			return journal.Observation{State: journal.Unknown}, nil
		}
		return journal.Observation{State: journal.Satisfied, Evidence: evidence}, nil
	}
	// Only exact absent/matching file sets are safely completable. Registration,
	// generation and service transitions need explicit reconciliation if their
	// complete postconditions cannot be independently demonstrated.
	if r.ID == "vicidial.00.files" && (value == "absent" || value == "partial") {
		return journal.Observation{State: journal.Absent, Evidence: hash(s)}, nil
	}
	return journal.Observation{State: journal.Unknown}, nil
}

// ExecuteVICIdialLab uses the existing ALM-13/14 runner, lock and durable journal.
// It is not a production authorization path. Native runtime qualification is
// required in addition to these controlled execution semantics.
func ExecuteVICIdialLab(ctx context.Context, s VICIdialStage, m artifacts.Manifest, a VICIdialLabAuthorization, stateDir string, host VICIdialHost, runner execution.CommandRunner, resume bool, log *operatorlog.Logger) (out Outcome, err error) {
	out.Traces = []execution.Trace{}
	if ctx == nil || host == nil || runner == nil {
		return out, errVICIdial
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
	fresh, e := host.InspectVICIdial(ctx)
	if e != nil {
		return out, e
	}
	if !vicidialReady(fresh) || !vicidialSameFoundation(d.Before, fresh) {
		return out, errVICIdial
	}
	c := &vicidialChecks{host: host, d: d, current: d.Before}
	retries := map[string]journal.Retry{}
	for _, action := range s.graph.Actions() {
		retries[action.ID] = journal.Never
		if action.ID == "vicidial.00.files" {
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
			return out, &execution.Error{Category: execution.Precondition, StageID: "S05", ExitCode: -1, Detail: "stale VICIdial plan; inspect and review again"}
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
		if e = c.Check(ctx, "vicidial.fresh"); e != nil {
			return out, e
		}
		if e = session.Begin(action.ID); e != nil {
			return out, e
		}
		if log != nil {
			_ = operatorui.RenderEvent(log, operatorui.Event{Type: operatorui.ActionStarted, StageID: "S05", ActionID: action.ID})
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
			return out, errVICIdial
		}
		t := traces[0]
		if t.Outcome == "blocked" {
			t.Outcome = "failed"
		}
		if e = session.Finish(ctx, t, c); e != nil {
			return out, e
		}
		if log != nil {
			if ev, ee := operatorui.EventFromTrace(traces[0]); ee == nil {
				_ = operatorui.RenderEvent(log, ev)
			}
		}
		if runErr != nil {
			return out, runErr
		}
	}
	fresh, e = host.InspectVICIdial(ctx)
	if e != nil {
		return out, e
	}
	if !vicidialSatisfied(fresh) || fresh.Digest() != c.current.Digest() {
		return out, &execution.Error{Category: execution.Postcondition, StageID: "S05", ExitCode: -1, Detail: "independent VICIdial application integration postconditions failed"}
	}
	out.NoOp = len(out.Traces) == 0
	return out, nil
}
