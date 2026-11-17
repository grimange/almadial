//go:build linux

package provision

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grimange/almadial/internal/artifacts"
	"github.com/grimange/almadial/internal/config"
	"github.com/grimange/almadial/internal/execution"
	"github.com/grimange/almadial/internal/journal"
)

func databaseInputs(t *testing.T, complete bool) (DatabaseInputs, artifacts.Manifest, DatabaseSource) {
	t.Helper()
	parent, m := systemInputs(t, true)
	parent.Parent.Config.Credentials.Database = config.SecretRef{Type: config.File, Source: DatabaseSecret}
	parent.Parent.Config.Credentials.Admin = config.SecretRef{Type: config.File, Source: DatabaseAdminSecret}
	source := DatabaseSource{"synthetic unit-test CREATE", "synthetic unit-test seed"}
	s := DatabaseSnapshot{Host: parent.Snapshot.RPM.Host, Foundation: parent.Snapshot.Digest(), ConfigInputs: hash("vendor-config"), EnvironmentOK: true, Config: "absent", Secrets: "absent"}
	if complete {
		s = completeDatabase(s)
	}
	return DatabaseInputs{Parent: parent, Snapshot: s, SourceSHA256: source.Digest()}, m, source
}
func completeDatabase(s DatabaseSnapshot) DatabaseSnapshot {
	s.Config = "valid"
	s.Secrets = "valid"
	s.CredentialID = hash("synthetic unique files")
	s.ServiceID = "invocation-1"
	s.Initialized = true
	s.Active = true
	s.Enabled = true
	s.RuntimeOK = true
	s.Exists = true
	s.SchemaVerified = true
	s.Version = "1748"
	s.CatalogSHA = databaseCatalogSHA
	s.Tables = 345
	s.Indexes = 1011
	s.OtherObjects = 0
	s.SeedsOK = true
	s.AdminCredentialOK = true
	s.AccountExists = true
	s.GrantsOK = true
	s.ApplicationAuthOK = true
	return s
}
func dbAuth(s DatabaseStage) DatabaseLabAuthorization {
	d := s.doc()
	return DatabaseLabAuthorization{"ALM-18/disposable-lab", d.Before.Host, s.Digest(), d.Parent, d.Manifest, d.Source, hash("fresh synthetic snapshot"), hash("explicit synthetic permission"), true}
}

type dbFake struct {
	s        DatabaseSnapshot
	calls    int
	failAt   int
	failure  execution.Category
	noChange bool
	partial  bool
	crashAt  int
}

func (f *dbFake) InspectDatabase(context.Context) (DatabaseSnapshot, error) { return f.s, nil }
func (f *dbFake) Validate(execution.Command) error                          { return nil }
func (f *dbFake) Run(_ context.Context, c execution.Command) (execution.Result, error) {
	f.calls++
	id := ""
	switch c.Executable {
	case "/usr/bin/python3":
		if c.Args[len(c.Args)-1] == "config" {
			id = "database.00.config"
		} else {
			id = "database.01.credentials"
		}
	case "/usr/bin/systemctl":
		if c.Args[0] == "restart" {
			id = "database.05.restart"
		} else {
			id = "database.02.service"
		}
	default:
		if c.Args[len(c.Args)-1] == "--comments" {
			id = "database.03.bootstrap"
		} else {
			id = "database.04.grants"
		}
	}
	if f.failAt == f.calls {
		if f.partial {
			f.s.Exists = true
			f.s.Tables = 1
			f.s.SchemaVerified = true
		}
		return execution.Result{Started: true, ExitCode: 7}, &execution.Error{Category: f.failure, ExitCode: 7, Detail: "bounded failure"}
	}
	if !f.noChange {
		switch id {
		case "database.00.config":
			f.s.Config = "valid"
		case "database.01.credentials":
			f.s.Secrets = "valid"
			f.s.CredentialID = hash("synthetic unique files")
		case "database.02.service":
			f.s.Initialized = true
			f.s.Active = true
			f.s.Enabled = true
			f.s.RuntimeOK = true
			f.s.SchemaVerified = true
			f.s.ServiceID = "invocation-1"
		case "database.03.bootstrap":
			v := completeDatabase(f.s)
			f.s.Exists = v.Exists
			f.s.Version = v.Version
			f.s.CatalogSHA = v.CatalogSHA
			f.s.Tables = v.Tables
			f.s.Indexes = v.Indexes
			f.s.SeedsOK = v.SeedsOK
			f.s.AdminCredentialOK = v.AdminCredentialOK
		case "database.04.grants":
			f.s.AccountExists = true
			f.s.GrantsOK = true
			f.s.ApplicationAuthOK = true
		case "database.05.restart":
			f.s.ServiceID = "invocation-2"
		}
	}
	if f.calls == f.crashAt {
		panic("synthetic process loss after effect, before durable outcome")
	}
	return execution.Result{Started: true, ExitCode: 0}, nil
}
func TestDatabaseStates(t *testing.T) {
	for _, tc := range []struct {
		name     string
		complete bool
		edit     func(*DatabaseSnapshot)
		want     DatabaseState
		ready    bool
	}{
		{"empty", false, func(*DatabaseSnapshot) {}, DatabaseEmpty, true},
		{"complete", true, func(*DatabaseSnapshot) {}, DatabaseComplete, true},
		{"missing table", true, func(s *DatabaseSnapshot) { s.Tables-- }, DatabasePartial, false},
		{"missing column or index", true, func(s *DatabaseSnapshot) { s.CatalogSHA = hash("different metadata") }, DatabasePartial, false},
		{"missing index", true, func(s *DatabaseSnapshot) { s.Indexes-- }, DatabasePartial, false},
		{"missing seed", true, func(s *DatabaseSnapshot) { s.SeedsOK = false }, DatabasePartial, false},
		{"wrong schema version", true, func(s *DatabaseSnapshot) { s.Version = "1747" }, DatabaseIncompatible, false},
		{"unknown metadata", true, func(s *DatabaseSnapshot) { s.SchemaVerified = false }, DatabaseNotVerified, false},
		{"unsupported package or SELinux baseline", true, func(s *DatabaseSnapshot) { s.EnvironmentOK = false }, DatabaseNotVerified, false},
		{"external listener", true, func(s *DatabaseSnapshot) { s.RuntimeOK = false }, DatabaseComplete, false},
		{"conflicting credentials", true, func(s *DatabaseSnapshot) { s.Secrets = "conflict" }, DatabaseComplete, false},
		{"wrong administrative credential", true, func(s *DatabaseSnapshot) { s.AdminCredentialOK = false }, DatabasePartial, false},
		{"wrong application credential", true, func(s *DatabaseSnapshot) { s.ApplicationAuthOK = false }, DatabaseComplete, false},
		{"global privilege escalation", true, func(s *DatabaseSnapshot) { s.GrantsOK = false }, DatabaseComplete, false},
		{"conflicting config", false, func(s *DatabaseSnapshot) { s.Config = "conflict" }, DatabaseEmpty, false},
		{"initialized stopped unknown", false, func(s *DatabaseSnapshot) { s.Initialized = true }, DatabaseNotVerified, false},
		{"empty database with routine", true, func(s *DatabaseSnapshot) { s.Tables = 0; s.OtherObjects = 1; s.Version = "" }, DatabasePartial, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			i, _, _ := databaseInputs(t, tc.complete)
			tc.edit(&i.Snapshot)
			if i.Snapshot.State() != tc.want || databaseReady(i.Snapshot) != tc.ready {
				t.Fatal(i.Snapshot.State(), databaseReady(i.Snapshot))
			}
		})
	}
}
func TestDatabasePlanAndNoOp(t *testing.T) {
	for _, standard := range []bool{false, true} {
		t.Run(map[bool]string{false: "legacy", true: "standard"}[standard], func(t *testing.T) {
			dbAuth := dbAuth
			if standard {
				original := dbAuth
				dbAuth = func(s DatabaseStage) DatabaseLabAuthorization {
					a := original(s)
					a.Scope = StandardAuthorization
					a.SnapshotReference = ""
					a.OperatorEvidenceSHA256 = ""
					a.CandidateAcknowledged = false
					return a
				}
			}

			i, m, src := databaseInputs(t, false)
			s, e := i.Prepare(m, src)
			if e != nil {
				t.Fatal(e)
			}
			again, e := i.Prepare(m, src)
			if e != nil || s.Digest() != again.Digest() || len(s.Graph().Actions()) != 6 {
				t.Fatal("determinism", e)
			}
			for j, a := range s.Graph().Actions() {
				if a.StageID != "S03" || a.Command.Timeout <= 0 || j > 0 && (len(a.Dependencies) != 1 || a.Dependencies[0] != s.Graph().Actions()[j-1].ID) {
					t.Fatal(a)
				}
			}
			f := &dbFake{s: i.Snapshot}
			dir := filepath.Join(t.TempDir(), "state")
			o, e := ExecuteDatabaseLab(context.Background(), s, m, dbAuth(s), dir, f, f, false, nil)
			if e != nil || len(o.Traces) != 6 || o.NoOp || !databaseSatisfied(f.s) {
				t.Fatal(e, o)
			}
			for _, r := range o.Journal.Actions {
				if r.State != journal.Succeeded || r.Retry != journal.Never {
					t.Fatal(r)
				}
			}
			o, e = ExecuteDatabaseLab(context.Background(), s, m, dbAuth(s), dir, f, f, true, nil)
			if e != nil || !o.NoOp || f.calls != 6 {
				t.Fatal("replay", e)
			}
			i.Snapshot = f.s
			s, e = i.Prepare(m, src)
			if e != nil || len(s.Graph().Actions()) != 0 {
				t.Fatal(e)
			}
			o, e = ExecuteDatabaseLab(context.Background(), s, m, dbAuth(s), filepath.Join(t.TempDir(), "noop"), f, f, false, nil)
			if e != nil || !o.NoOp || f.calls != 6 {
				t.Fatal(e)
			}
			if VerifyDatabaseForApplication(f.s, src) != nil || VerifyDatabaseForApplication(f.s, DatabaseSource{}) == nil {
				t.Fatal("S05 source gate")
			}
		})
	}
}
func TestDatabaseInputRejection(t *testing.T) {
	for _, name := range []string{"wrong host", "stale foundation", "stale RPM inventory", "wrong secret reference", "unknown source", "transaction", "repository prep", "unavailable package", "partial schema"} {
		t.Run(name, func(t *testing.T) {
			i, m, src := databaseInputs(t, false)
			switch name {
			case "wrong host":
				i.Snapshot.Host = hash("foreign")
			case "stale foundation":
				i.Snapshot.Foundation = hash("stale")
			case "stale RPM inventory":
				i.Parent.Parent.Inventory.Host = hash("foreign")
			case "wrong secret reference":
				i.Parent.Parent.Config.Credentials.Database.Type = config.Environment
			case "unknown source":
				src = DatabaseSource{}
			case "transaction":
				i.Parent.Parent.Transaction = &Transaction{}
			case "repository prep":
				i.Parent.Parent.PrepareRepositoryTrust = true
			case "unavailable package":
				i.Parent.Snapshot.RPM.Installed = nil
			case "partial schema":
				i.Snapshot = completeDatabase(i.Snapshot)
				i.Snapshot.Tables--
			}
			if _, e := i.Prepare(m, src); e == nil {
				t.Fatal("accepted invalid input")
			}
		})
	}
}
func TestDatabaseAuthorizationAndLock(t *testing.T) {
	for _, name := range []string{"scope", "host", "stage", "parent", "manifest", "source", "snapshot", "operator", "candidate", "stale state", "lock"} {
		t.Run(name, func(t *testing.T) {
			i, m, src := databaseInputs(t, false)
			s, _ := i.Prepare(m, src)
			a := dbAuth(s)
			f := &dbFake{s: i.Snapshot}
			dir := filepath.Join(t.TempDir(), "state")
			switch name {
			case "scope":
				a.Scope = "production"
			case "host":
				a.Host = hash("other")
			case "stage":
				a.StageSHA256 = hash("other")
			case "parent":
				a.ParentSHA256 = hash("other")
			case "manifest":
				a.ManifestSHA256 = hash("other")
			case "source":
				a.SourceSHA256 = hash("other")
			case "snapshot":
				a.SnapshotReference = ""
			case "operator":
				a.OperatorEvidenceSHA256 = ""
			case "candidate":
				a.CandidateAcknowledged = false
			case "stale state":
				f.s.Config = "valid"
			case "lock":
				j, e := journal.Acquire(dir)
				if e != nil {
					t.Fatal(e)
				}
				defer j.Close()
			}
			if _, e := ExecuteDatabaseLab(context.Background(), s, m, a, dir, f, f, false, nil); e == nil || f.calls != 0 {
				t.Fatal("unauthorized execution", e)
			}
		})
	}
}
func TestDatabaseFailureAndRecovery(t *testing.T) {
	for _, kind := range []execution.Category{execution.Timeout, execution.Canceled, execution.Exit, execution.OutputLimit, execution.Start} {
		t.Run(string(kind), func(t *testing.T) {
			i, m, src := databaseInputs(t, false)
			s, _ := i.Prepare(m, src)
			f := &dbFake{s: i.Snapshot, failAt: 4, failure: kind, partial: true}
			dir := filepath.Join(t.TempDir(), "state")
			o, e := ExecuteDatabaseLab(context.Background(), s, m, dbAuth(s), dir, f, f, false, nil)
			if !errors.Is(e, &execution.Error{Category: kind}) || len(o.Traces) != 4 || o.Journal.Actions[3].State != journal.Failed {
				t.Fatal(e)
			}
			f.failure = ""
			_, e = ExecuteDatabaseLab(context.Background(), s, m, dbAuth(s), dir, f, f, true, nil)
			if e == nil || f.calls != 4 {
				t.Fatal("partial DDL replayed", e)
			}
		})
	}
	t.Run("zero exit false postcondition", func(t *testing.T) {
		i, m, src := databaseInputs(t, false)
		s, _ := i.Prepare(m, src)
		f := &dbFake{s: i.Snapshot, noChange: true}
		o, e := ExecuteDatabaseLab(context.Background(), s, m, dbAuth(s), filepath.Join(t.TempDir(), "state"), f, f, false, nil)
		if !errors.Is(e, &execution.Error{Category: execution.Postcondition}) || len(o.Traces) != 1 {
			t.Fatal(e)
		}
	})
	t.Run("complete import incomplete journal", func(t *testing.T) {
		i, m, src := databaseInputs(t, true)
		i.Snapshot.AccountExists = false
		i.Snapshot.GrantsOK = false
		i.Snapshot.ApplicationAuthOK = false
		i.Snapshot.Exists = false
		i.Snapshot.Version = ""
		i.Snapshot.CatalogSHA = ""
		i.Snapshot.Tables = 0
		i.Snapshot.Indexes = 0
		i.Snapshot.SeedsOK = false
		i.Snapshot.AdminCredentialOK = false
		s, e := i.Prepare(m, src)
		if e != nil {
			t.Fatal(e)
		}
		f := &dbFake{s: i.Snapshot, crashAt: 1}
		dir := filepath.Join(t.TempDir(), "state")
		func() {
			defer func() {
				if recover() == nil {
					t.Error("synthetic interruption did not occur")
				}
			}()
			_, _ = ExecuteDatabaseLab(context.Background(), s, m, dbAuth(s), dir, f, f, false, nil)
		}()
		file, e := os.Open(filepath.Join(dir, "journal.json"))
		if e != nil {
			t.Fatal(e)
		}
		pending, e := journal.Decode(file)
		file.Close()
		if e != nil || pending.Actions[0].State != journal.Running || f.s.State() != DatabaseComplete {
			t.Fatal("missing durable running intent for completed DDL", e)
		}
		f.crashAt = 0
		o, e := ExecuteDatabaseLab(context.Background(), s, m, dbAuth(s), dir, f, f, true, nil)
		if e != nil || len(o.Traces) != 2 || f.calls != 3 {
			t.Fatal("completed schema replay", e)
		}
	})
	t.Run("partial credential persistence blocks resume", func(t *testing.T) {
		i, m, src := databaseInputs(t, false)
		s, _ := i.Prepare(m, src)
		f := &dbFake{s: i.Snapshot, failAt: 2, failure: execution.Exit}
		dir := filepath.Join(t.TempDir(), "state")
		_, e := ExecuteDatabaseLab(context.Background(), s, m, dbAuth(s), dir, f, f, false, nil)
		if e == nil {
			t.Fatal("missing persistence failure")
		}
		f.s.Secrets = "conflict"
		f.failAt = 0
		o, e := ExecuteDatabaseLab(context.Background(), s, m, dbAuth(s), dir, f, f, true, nil)
		if e == nil || f.calls != 2 || o.Journal.Actions[1].State != journal.ReconciliationRequired {
			t.Fatal("partial credential files replayed", e)
		}
	})
	t.Run("credential drift", func(t *testing.T) {
		i, m, src := databaseInputs(t, false)
		s, _ := i.Prepare(m, src)
		f := &dbFake{s: i.Snapshot}
		dir := filepath.Join(t.TempDir(), "state")
		if _, e := ExecuteDatabaseLab(context.Background(), s, m, dbAuth(s), dir, f, f, false, nil); e != nil {
			t.Fatal(e)
		}
		f.s.CredentialID = hash("replacement files")
		if _, e := ExecuteDatabaseLab(context.Background(), s, m, dbAuth(s), dir, f, f, true, nil); e == nil || f.calls != 6 {
			t.Fatal("credential drift accepted")
		}
	})
}
func TestDatabaseSourceAndSecretBoundaries(t *testing.T) {
	_, m, _ := databaseInputs(t, false)
	p := filepath.Join(t.TempDir(), "corrupt.zip")
	if e := os.WriteFile(p, []byte("corrupted archive"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := LoadDatabaseSource(context.Background(), m, p); e == nil {
		t.Fatal("unverified source")
	}
	i, m, src := databaseInputs(t, false)
	s, e := i.Prepare(m, src)
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(struct {
		Stage         json.RawMessage
		Graph         json.RawMessage
		Authorization DatabaseLabAuthorization
	}{s.JSON(), s.Graph().JSON(), dbAuth(s)})
	if strings.Contains(string(raw), src.create) || strings.Contains(string(raw), src.seed) {
		t.Fatal("SQL source leaked into graph")
	}
	n, _ := NewNative(nil)
	r := DatabaseNative{Native: n, Source: DatabaseSource{}}.Runner(s)
	if r.Validate(s.Graph().Actions()[0].Command) == nil {
		t.Fatal("substituted source accepted")
	}
}
