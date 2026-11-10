//go:build linux

package provision

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/grimange/almadial/internal/artifacts"
	"github.com/grimange/almadial/internal/journal"
)

// These tests compose accepted stage fixtures and APIs. They do not model a
// top-level installer, replace native inspectors, or run application workloads.
func TestBackendCompleteChain(t *testing.T) {
	i, m, src, perl, ast, db := runtimeFixture(t)
	i.Snapshot.Files, i.Snapshot.Services, i.Snapshot.Health = hash("files"), hash("services"), hash("health")
	system := i.Parent.Parent.Parent.Parent
	if !systemSatisfied(system.Snapshot, system.Budget) {
		t.Fatal("S01/S02 postconditions incomplete")
	}
	for _, a := range Assess(system.Snapshot.RPM, nil, false) {
		if a.Status != Satisfied {
			t.Fatal("S01", a)
		}
	}
	s2, e2 := system.Prepare(m)
	s3, e3 := i.Parent.Parent.Parent.Prepare(m, db)
	s4, e4 := i.Parent.Parent.Prepare(m, ast)
	s5, e5 := i.Parent.Prepare(m, src, perl, ast, db)
	s6, e6 := i.Prepare(m, src, perl, ast, db)
	for _, err := range []error{e2, e3, e4, e5, e6} {
		if err != nil {
			t.Fatal(err)
		}
	}
	for stage, count := range map[string]int{"S02": len(s2.Graph().Actions()), "S03": len(s3.Graph().Actions()), "S04": len(s4.Graph().Actions()), "S05": len(s5.Graph().Actions()), "S06": len(s6.Graph().Actions())} {
		if count != 0 {
			t.Fatalf("%s unnecessarily reprovisions complete state: %d", stage, count)
		}
	}
	again, err := i.Prepare(m, src, perl, ast, db)
	if err != nil || again.Digest() != s6.Digest() || again.Graph().Digest() != s6.Graph().Digest() {
		t.Fatal("assessment is not deterministic", err)
	}
	f := &runtimeFake{s: i.Snapshot}
	out, err := ExecuteRuntimeLab(context.Background(), s6, m, runtimeAuth(s6), filepath.Join(t.TempDir(), "state"), f, f, false, nil)
	if err != nil || !out.NoOp || f.calls != 0 || len(out.Traces) != 0 {
		t.Fatal("complete chain replayed effects", out, err)
	}
}

func TestBackendUpstreamBlocksDependents(t *testing.T) {
	for _, stage := range []string{"S01", "S02", "S03", "S04", "S05"} {
		t.Run(stage, func(t *testing.T) {
			i, m, src, perl, ast, db := runtimeFixture(t)
			// Rebind observations after injecting the defect so rejection cannot
			// be explained solely by an obsolete parent digest.
			system := &i.Parent.Parent.Parent.Parent
			switch stage {
			case "S01":
				system.Snapshot.RPM.Installed = fixture("gcc").Installed
				system.Parent.Inventory = system.Snapshot.RPM
			case "S02":
				system.Snapshot.Directories[0].Exists = false
			case "S03":
				i.Parent.Parent.Parent.Snapshot.SchemaVerified = false
				i.Parent.Parent.Parent.Snapshot.Tables--
			case "S04":
				i.Parent.Parent.Snapshot.Modules[0].Loaded = false
				i.Parent.Parent.Snapshot.Modules[0].Pass = false
			case "S05":
				i.Parent.Snapshot.Configuration = "absent"
			}
			i.Parent.Parent.Parent.Snapshot.Foundation = system.Snapshot.Digest()
			i.Parent.Parent.Snapshot.Foundation = system.Snapshot.Digest()
			i.Parent.Parent.Snapshot.Database = i.Parent.Parent.Parent.Snapshot.Digest()
			i.Parent.Snapshot.Foundation = system.Snapshot.Digest()
			i.Parent.Snapshot.Database = i.Parent.Parent.Snapshot.Database
			i.Parent.Snapshot.Asterisk = i.Parent.Parent.Snapshot.Digest()
			i.Snapshot.Foundation = i.Parent.Snapshot.Digest()
			if stage == "S01" || stage == "S02" {
				if _, err := i.Parent.Parent.Parent.Prepare(m, db); err == nil {
					t.Fatal(stage, "did not block S03")
				}
			}
			if stage != "S05" {
				if _, err := i.Parent.Prepare(m, src, perl, ast, db); err == nil {
					t.Fatal(stage, "did not block S05")
				}
			}
			if s, err := i.Prepare(m, src, perl, ast, db); err == nil || s.Digest() != "" {
				t.Fatal(stage, "produced an executable S06 plan")
			}
		})
	}
}

func TestBackendMaterialIdentityPropagates(t *testing.T) {
	for _, input := range []string{"config", "manifest"} {
		t.Run(input, func(t *testing.T) {
			i, m, src, perl, ast, db := runtimeFixture(t)
			original, err := i.Prepare(m, src, perl, ast, db)
			if err != nil {
				t.Fatal(err)
			}
			if input == "config" {
				i.Parent.Parent.Parent.Parent.Parent.Config.Network.Hostname = "changed.example.invalid"
			} else {
				data, err := os.ReadFile("../../manifests/stack.lock.yaml")
				if err != nil {
					t.Fatal(err)
				}
				// Comment changes retain the pinned candidate policy but change its
				// immutable document identity. No alternate stack is authorized.
				m, err = artifacts.Load(strings.NewReader("# integration identity fixture\n" + string(data)))
				if err != nil {
					t.Fatal(err)
				}
			}
			changed, err := i.Prepare(m, src, perl, ast, db)
			if err != nil || changed.doc().Parent == original.doc().Parent || changed.Digest() == original.Digest() {
				t.Fatal("material input did not invalidate dependent identity", err)
			}
			if input == "config" && changed.doc().Config == original.doc().Config || input == "manifest" && changed.doc().Manifest == original.doc().Manifest {
				t.Fatal("changed input identity was lost")
			}
			f := &runtimeFake{s: i.Snapshot}
			if _, err := ExecuteRuntimeLab(context.Background(), changed, m, runtimeAuth(original), filepath.Join(t.TempDir(), "state"), f, f, false, nil); err == nil || f.calls != 0 {
				t.Fatal("stale authorization executed changed chain", err)
			}
		})
	}
}

func TestBackendCopiedJournalRecovery(t *testing.T) {
	i, m, src, perl, ast, db := runtimeFixture(t)
	s, err := i.Prepare(m, src, perl, ast, db)
	if err != nil {
		t.Fatal(err)
	}
	f := &runtimeFake{s: i.Snapshot}
	completed, err := ExecuteRuntimeLab(context.Background(), s, m, runtimeAuth(s), filepath.Join(t.TempDir(), "original"), f, f, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	before := f.s
	for _, field := range []string{"interrupted", "config", "manifest", "plan", "host", "graph", "action_metadata"} {
		t.Run(field, func(t *testing.T) {
			// Copy a validated completed journal, never alter live state. The
			// fixture represents lost final outcome after all effects completed.
			encoded, err := completed.Journal.JSON()
			if err != nil {
				t.Fatal(err)
			}
			doc, err := journal.Decode(strings.NewReader(string(encoded)))
			if err != nil {
				t.Fatal(err)
			}
			last := &doc.Actions[len(doc.Actions)-1]
			last.State, last.Evidence, last.ExitCode = journal.Running, "", -1
			switch field {
			case "config":
				doc.Identity.Config = hash("changed config")
			case "manifest":
				doc.Identity.Manifest = hash("changed manifest")
			case "plan":
				doc.Identity.Plan = hash("changed plan")
			case "host":
				doc.Identity.Host = hash("different host")
			case "graph":
				doc.Identity.Graph = hash("changed graph")
			case "action_metadata":
				doc.Actions[0].Retry = journal.Never
			}
			encoded, err = doc.JSON()
			if err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(t.TempDir(), "copy")
			if err := os.Mkdir(dir, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "journal.json"), encoded, 0600); err != nil {
				t.Fatal(err)
			}
			f.calls = 0
			out, err := ExecuteRuntimeLab(context.Background(), s, m, runtimeAuth(s), dir, f, f, true, nil)
			if field == "interrupted" {
				if err != nil || !out.NoOp || out.Journal == nil || out.Journal.Actions[len(doc.Actions)-1].State != journal.Succeeded {
					t.Fatal("complete-state reconciliation failed", out, err)
				}
			} else {
				if !errors.Is(err, journal.ErrIdentity) || !strings.Contains(err.Error(), field) {
					t.Fatal("identity mismatch not identified", err)
				}
				actual, err := os.ReadFile(filepath.Join(dir, "journal.json"))
				if err != nil || string(actual) != string(encoded) {
					t.Fatal("rejected journal overwritten", err)
				}
			}
			if f.calls != 0 || !reflect.DeepEqual(f.s, before) || len(out.Traces) != 0 {
				t.Fatal("recovery replayed accepted effects")
			}
		})
	}
}

func TestBackendCrossStageExecutionLock(t *testing.T) {
	i, m, src, perl, ast, db := runtimeFixture(t)
	s, err := i.Prepare(m, src, perl, ast, db)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "installation")
	owner, err := journal.Acquire(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	// The accepted lock is installation-directory scoped. Both stage APIs
	// must reject the same owner before invoking their mutation runners.
	f := &runtimeFake{s: i.Snapshot}
	if _, err := ExecuteRuntimeLab(context.Background(), s, m, runtimeAuth(s), dir, f, f, false, nil); !errors.Is(err, journal.ErrBusy) || f.calls != 0 {
		t.Fatal("S06 bypassed installation owner", err)
	}
	database := i.Parent.Parent.Parent
	dbStage, err := database.Prepare(m, db)
	if err != nil {
		t.Fatal(err)
	}
	df := &dbFake{s: database.Snapshot}
	if _, err := ExecuteDatabaseLab(context.Background(), dbStage, m, dbAuth(dbStage), dir, df, df, false, nil); !errors.Is(err, journal.ErrBusy) || df.calls != 0 {
		t.Fatal("S03 bypassed S06 installation owner", err)
	}
}
