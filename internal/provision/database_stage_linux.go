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

type DatabaseInputs struct {
	Parent       SystemInputs
	Snapshot     DatabaseSnapshot
	SourceSHA256 string
}
type DatabaseStage struct {
	data  string
	graph execution.Graph
}
type databaseStageData struct {
	Parent, Config, Manifest, Policy, Source, Graph string
	Before                                          DatabaseSnapshot
}

func (s DatabaseStage) JSON() []byte { return []byte(s.data) }
func (s DatabaseStage) Digest() string {
	if s.data == "" {
		return ""
	}
	return hash(json.RawMessage(s.data))
}
func (s DatabaseStage) Graph() execution.Graph { return s.graph }
func (s DatabaseStage) doc() databaseStageData {
	var d databaseStageData
	_ = json.Unmarshal(s.JSON(), &d)
	return d
}

func (i DatabaseInputs) Prepare(m artifacts.Manifest, source DatabaseSource) (DatabaseStage, error) {
	p, e := plan.New(i.Parent.Parent.Config, i.Parent.Parent.Preflight, i.Parent.Parent.Requirements, m.Digest())
	if e != nil {
		return DatabaseStage{}, e
	}
	id, e := hex.DecodeString(i.Parent.Parent.Preflight.HostID)
	h := sha256.Sum256([]byte(i.Parent.Parent.Preflight.HostID))
	if e != nil || len(id) != 16 || hex.EncodeToString(h[:]) != i.Snapshot.Host || i.Parent.Snapshot.RPM.Host != i.Snapshot.Host || i.Parent.Snapshot.Digest() != i.Snapshot.Foundation || i.Parent.Parent.Inventory.Digest() != i.Parent.Snapshot.RPM.Digest() || i.Parent.Parent.Transaction != nil || i.Parent.Parent.PrepareRepositoryTrust || !systemSatisfied(i.Parent.Snapshot, i.Parent.Budget) || !databaseReferences(i.Parent.Parent.Config) || !databaseReady(i.Snapshot) || i.SourceSHA256 == "" || i.SourceSHA256 != source.Digest() {
		return DatabaseStage{}, errDatabase
	}
	actions := []execution.Action{}
	add := func(id, desc string, c execution.Command) {
		deps := []string{}
		if len(actions) > 0 {
			deps = append(deps, actions[len(actions)-1].ID)
		}
		actions = append(actions, execution.Action{ID: id, StageID: "S03", Description: desc, Dependencies: deps, Preconditions: []string{"database.fresh"}, Command: c, Postconditions: []string{id + ".verified"}})
	}
	s := i.Snapshot
	if s.Config == "absent" {
		add("database.00.config", "Create only the absent AlmaDial-owned local MariaDB configuration", dbFileCommand("config"))
	}
	if s.Secrets == "absent" {
		add("database.01.credentials", "Create unique protected file references without replacing credentials", dbFileCommand("credentials"))
	}
	if !s.Active || !s.Enabled {
		add("database.02.service", "Initialize and enable/start only the local MariaDB service", dbCommand("/usr/bin/systemctl", "enable", "--now", "mariadb"))
	}
	if s.State() == DatabaseEmpty {
		add("database.03.bootstrap", "S03 sole writer: import verified r4037 SQL once into an empty target", dbClient("--comments"))
	}
	if !s.AccountExists {
		add("database.04.grants", "Create only the missing local application account with M0 database privileges", dbClient("--database=mysql"))
	}
	if len(actions) > 0 {
		add("database.05.restart", "Restart MariaDB and independently verify persistence and transport", dbCommand("/usr/bin/systemctl", "restart", "mariadb"))
	}
	g, e := execution.NewGraph(actions)
	if e != nil {
		return DatabaseStage{}, e
	}
	d := databaseStageData{p.Digest(), p.ConfigSHA256(), m.Digest(), DatabasePolicyDigest(), source.Digest(), g.Digest(), s}
	return DatabaseStage{jsonString(d), g}, nil
}

type DatabaseLabAuthorization struct {
	Scope, Host, StageSHA256, ParentSHA256, ManifestSHA256, SourceSHA256 string
	SnapshotReference, OperatorEvidenceSHA256                            string
	CandidateAcknowledged                                                bool
}

func (a DatabaseLabAuthorization) validate(s DatabaseStage, m artifacts.Manifest) error {
	d := s.doc()
	if s.Digest() == "" || a.Scope != "ALM-18/disposable-lab" || a.Host != d.Before.Host || a.StageSHA256 != s.Digest() || a.ParentSHA256 != d.Parent || a.ManifestSHA256 != d.Manifest || a.SourceSHA256 != d.Source || m.Digest() != d.Manifest || d.Policy != DatabasePolicyDigest() || !shaPattern.MatchString(a.SnapshotReference) || !shaPattern.MatchString(a.OperatorEvidenceSHA256) || !a.CandidateAcknowledged {
		return &execution.Error{Category: execution.Precondition, StageID: "S03", ExitCode: -1, Detail: "exact ALM-18 laboratory authorization, verified source and fresh recovery snapshot required"}
	}
	return nil
}

type DatabaseHost interface {
	InspectDatabase(context.Context) (DatabaseSnapshot, error)
}
type databaseChecks struct {
	host    DatabaseHost
	d       databaseStageData
	current DatabaseSnapshot
	action  execution.Action
}

func mergeDatabase(before, after DatabaseSnapshot, id string) (DatabaseSnapshot, error) {
	if !databaseReady(after) || before.Host != after.Host || before.Foundation != after.Foundation || before.ConfigInputs != after.ConfigInputs {
		return before, errDatabase
	}
	w := before
	switch id {
	case "database.00.config":
		if after.Config != "valid" {
			return before, errDatabase
		}
		w.Config = after.Config
	case "database.01.credentials":
		if after.Secrets != "valid" {
			return before, errDatabase
		}
		w.Secrets, w.CredentialID = after.Secrets, after.CredentialID
	case "database.02.service":
		if !after.Active || !after.Enabled || !after.Initialized || !after.RuntimeOK || !after.SchemaVerified {
			return before, errDatabase
		}
		w.Active, w.Enabled, w.Initialized, w.RuntimeOK, w.SchemaVerified, w.ServiceID = after.Active, after.Enabled, after.Initialized, after.RuntimeOK, after.SchemaVerified, after.ServiceID
	case "database.03.bootstrap":
		if after.State() != DatabaseComplete {
			return before, errDatabase
		}
		w.Exists, w.Version, w.CatalogSHA, w.Tables, w.Indexes, w.OtherObjects, w.SeedsOK, w.AdminCredentialOK = after.Exists, after.Version, after.CatalogSHA, after.Tables, after.Indexes, after.OtherObjects, after.SeedsOK, after.AdminCredentialOK
	case "database.04.grants":
		if !after.AccountExists || !after.GrantsOK || !after.ApplicationAuthOK {
			return before, errDatabase
		}
		w.AccountExists, w.GrantsOK, w.ApplicationAuthOK = after.AccountExists, after.GrantsOK, after.ApplicationAuthOK
	case "database.05.restart":
		if !databaseSatisfied(after) || after.ServiceID == "" || before.ServiceID == after.ServiceID {
			return before, errDatabase
		}
		w.ServiceID = after.ServiceID
	default:
		return before, errDatabase
	}
	if w.Digest() != after.Digest() {
		return before, errDatabase
	}
	return w, nil
}
func (c *databaseChecks) Supports(id string) bool {
	return id == "database.fresh" || id == c.action.ID+".verified"
}
func (c *databaseChecks) Check(ctx context.Context, id string) error {
	s, e := c.host.InspectDatabase(ctx)
	if e != nil {
		return e
	}
	if id == "database.fresh" {
		if !databaseReady(s) || s.Digest() != c.current.Digest() {
			return errDatabase
		}
		return nil
	}
	if id != c.action.ID+".verified" {
		return errDatabase
	}
	w, e := mergeDatabase(c.current, s, c.action.ID)
	if e == nil {
		c.current = w
	}
	return e
}
func databaseActionSatisfied(s DatabaseSnapshot, id string) bool {
	if !databaseReady(s) {
		return false
	}
	switch id {
	case "database.00.config":
		return s.Config == "valid"
	case "database.01.credentials":
		return s.Secrets == "valid"
	case "database.02.service":
		return s.Active && s.Enabled && s.Initialized && s.RuntimeOK
	case "database.03.bootstrap":
		return s.State() == DatabaseComplete
	case "database.04.grants":
		return s.AccountExists && s.GrantsOK && s.ApplicationAuthOK
	case "database.05.restart":
		return databaseSatisfied(s)
	}
	return false
}
func databaseActionEvidence(s DatabaseSnapshot, id, source string) string {
	value := ""
	switch id {
	case "database.00.config":
		value = hash(databaseConfig)
	case "database.01.credentials":
		value = s.CredentialID
	case "database.02.service":
		value = hash(struct{ Config, Inputs string }{s.Config, s.ConfigInputs})
	case "database.03.bootstrap":
		value = hash(struct{ Catalog, Version, Credential string }{s.CatalogSHA, s.Version, s.CredentialID})
	case "database.04.grants":
		value = hash(struct{ User, Credential string }{DatabaseUser, s.CredentialID})
	case "database.05.restart":
		value = s.ServiceID
	}
	return hash(struct{ Policy, ID, Source, Value string }{DatabasePolicyDigest(), id, source, value})
}
func (c *databaseChecks) Observe(ctx context.Context, r journal.Record) (journal.Observation, error) {
	s, e := c.host.InspectDatabase(ctx)
	if e != nil || s.Host != c.d.Before.Host || s.Foundation != c.d.Before.Foundation || s.ConfigInputs != c.d.Before.ConfigInputs {
		return journal.Observation{State: journal.Unknown}, e
	}
	if databaseActionSatisfied(s, r.ID) {
		evidence := databaseActionEvidence(s, r.ID, c.d.Source)
		if r.State == journal.Succeeded && r.Evidence != evidence {
			return journal.Observation{State: journal.Unknown}, nil
		}
		// A started restart with no durable postcondition cannot establish that the
		// service actually restarted. An explicit review is required in that window.
		if r.ID == "database.05.restart" && r.State != journal.Succeeded && c.action.ID != r.ID {
			return journal.Observation{State: journal.Unknown}, nil
		}
		return journal.Observation{State: journal.Satisfied, Evidence: evidence}, nil
	}
	// No absent mutation is automatically replayed, including an empty database
	// after an interrupted import. Actual partial/unknown state blocks reconciliation.
	return journal.Observation{State: journal.Unknown}, nil
}

func ExecuteDatabaseLab(ctx context.Context, s DatabaseStage, m artifacts.Manifest, a DatabaseLabAuthorization, stateDir string, host DatabaseHost, runner execution.CommandRunner, resume bool, log *operatorlog.Logger) (out Outcome, err error) {
	out.Traces = []execution.Trace{}
	if ctx == nil || host == nil || runner == nil {
		return out, errDatabase
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
	fresh, e := host.InspectDatabase(ctx)
	if e != nil {
		return out, e
	}
	if fresh.Host != d.Before.Host || fresh.Foundation != d.Before.Foundation || fresh.ConfigInputs != d.Before.ConfigInputs {
		return out, errDatabase
	}
	c := &databaseChecks{host: host, d: d, current: d.Before}
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
		if !databaseReady(fresh) {
			return out, errDatabase
		}
		c.current = fresh
	} else {
		if !databaseReady(fresh) || fresh.Digest() != d.Before.Digest() {
			return out, &execution.Error{Category: execution.Precondition, StageID: "S03", ExitCode: -1, Detail: "stale database plan; inspect and review again"}
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
		if e = c.Check(ctx, "database.fresh"); e != nil {
			return out, e
		}
		if e = session.Begin(action.ID); e != nil {
			return out, e
		}
		if log != nil {
			_ = operatorui.RenderEvent(log, operatorui.Event{Type: operatorui.ActionStarted, StageID: "S03", ActionID: action.ID})
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
			return out, errDatabase
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
	fresh, e = host.InspectDatabase(ctx)
	if e != nil {
		return out, e
	}
	if !databaseSatisfied(fresh) || fresh.Digest() != c.current.Digest() {
		return out, &execution.Error{Category: execution.Postcondition, StageID: "S03", ExitCode: -1, Detail: "independent database postconditions failed"}
	}
	out.NoOp = len(out.Traces) == 0
	return out, nil
}
func (n DatabaseNative) Runner(s DatabaseStage) execution.CommandRunner {
	actions := map[string]string{}
	for _, a := range s.graph.Actions() {
		actions[jsonString(a.Command)] = a.ID
	}
	return databaseRunner{native: n, actions: actions, source: s.doc().Source}
}
