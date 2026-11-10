//go:build linux

package provision

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/grimange/almadial/internal/artifacts"
	"github.com/grimange/almadial/internal/execution"
	"github.com/grimange/almadial/internal/journal"
)

func vicidialStageFixture(t *testing.T) (VICIdialInputs, artifacts.Manifest, VICIdialSource, VICIdialPerlSource, AsteriskSource, DatabaseSource) {
	t.Helper()
	p, m, ast := asteriskInputs(t)
	f := &astFake{s: p.Snapshot}
	for _, id := range []string{"asterisk.00.build", "asterisk.01.install", "asterisk.02.config", "asterisk.03.unit", "asterisk.04.service", "asterisk.05.restart"} {
		_, _ = f.Run(context.Background(), asteriskActionCommand(id))
	}
	p.Snapshot = f.s
	source := VICIdialSource{files: map[string][]byte{"version": []byte("synthetic source; never executable")}, archive: []byte("synthetic archive")}
	perl := VICIdialPerlSource{[]byte("synthetic archive"), []byte("synthetic module"), []byte("synthetic attribution")}
	s := VICIdialSnapshot{Host: f.s.Host, Foundation: f.s.Foundation, Database: f.s.Database, Asterisk: f.s.Digest(), Source: source.Digest(), Dependency: perl.Digest(), GeneratorInputs: hash("synthetic source rows"), Files: "absent", Perl: "absent", Configuration: "absent", Registration: "absent", Generated: "absent", Web: "absent", Runtime: "absent"}
	return VICIdialInputs{p, s, source.Digest(), perl.Digest()}, m, source, perl, ast, DatabaseSource{"synthetic unit-test CREATE", "synthetic unit-test seed"}
}
func vicidialAuth(s VICIdialStage) VICIdialLabAuthorization {
	d := s.doc()
	return VICIdialLabAuthorization{Scope: "ALM-20/disposable-lab", Host: d.Before.Host, StageSHA256: s.Digest(), ParentSHA256: d.Parent, ManifestSHA256: d.Manifest, SourceSHA256: d.Source, DependencySHA256: d.Dependency, SnapshotReference: hash("synthetic verified snapshot"), OperatorEvidenceSHA256: hash("synthetic explicit ALM-20 scope"), CandidateAcknowledged: true}
}

type vicidialStageFake struct {
	s                             VICIdialSnapshot
	calls, failAt, crashAt        int
	partialFiles, noChange, drift bool
}

func (f *vicidialStageFake) InspectVICIdial(context.Context) (VICIdialSnapshot, error) {
	return f.s, nil
}
func (f *vicidialStageFake) Validate(execution.Command) error { return nil }
func (f *vicidialStageFake) Run(_ context.Context, c execution.Command) (execution.Result, error) {
	f.calls++
	if f.calls == f.failAt {
		return execution.Result{Started: true, ExitCode: 1}, errors.New("synthetic failure")
	}
	if !f.noChange {
		v := hash(c.Args[1])
		switch c.Args[1] {
		case "vicidial.00.files":
			f.s.Files = v
			if f.partialFiles {
				f.s.Files = "partial"
			}
		case "vicidial.01.perl":
			f.s.Perl = v
		case "vicidial.02.configuration":
			f.s.Configuration = v
		case "vicidial.03.registration":
			f.s.Registration = v
		case "vicidial.04.generate":
			f.s.Generated = v
		case "vicidial.05.web":
			f.s.Web = v
		case "vicidial.06.verify":
			f.s.Runtime = v
		default:
			panic("unapproved S05 action")
		}
	}
	if f.drift {
		f.s.Database = hash("unrelated changed schema")
	}
	if f.calls == f.crashAt {
		panic("synthetic interruption before durable outcome")
	}
	return execution.Result{Started: true}, nil
}
func TestVICIdialStageExecutionAndFreshNoOp(t *testing.T) {
	i, m, src, dep, ast, db := vicidialStageFixture(t)
	s, e := i.Prepare(m, src, dep, ast, db)
	if e != nil {
		t.Fatal(e)
	}
	again, e := i.Prepare(m, src, dep, ast, db)
	if e != nil || s.Digest() != again.Digest() || len(s.Graph().Actions()) != 7 {
		t.Fatal("non-deterministic plan", e)
	}
	f := &vicidialStageFake{s: i.Snapshot}
	dir := filepath.Join(t.TempDir(), "state")
	o, e := ExecuteVICIdialLab(context.Background(), s, m, vicidialAuth(s), dir, f, f, false, nil)
	if e != nil || o.NoOp || !vicidialSatisfied(f.s) || f.calls != 7 {
		t.Fatal(e, o)
	}
	for _, a := range o.Journal.Actions {
		if a.State != journal.Succeeded || a.Evidence == "" {
			t.Fatal(a)
		}
	}
	o, e = ExecuteVICIdialLab(context.Background(), s, m, vicidialAuth(s), dir, f, f, true, nil)
	if e != nil || !o.NoOp || f.calls != 7 {
		t.Fatal("resume", e)
	}
	i.Snapshot = f.s
	s, e = i.Prepare(m, src, dep, ast, db)
	if e != nil || len(s.Graph().Actions()) != 0 {
		t.Fatal(e)
	}
	o, e = ExecuteVICIdialLab(context.Background(), s, m, vicidialAuth(s), filepath.Join(t.TempDir(), "noop"), f, f, false, nil)
	if e != nil || !o.NoOp || f.calls != 7 {
		t.Fatal("fresh NO_OP", e)
	}
}
func TestVICIdialAuthorizationAndStalePlan(t *testing.T) {
	i, m, src, dep, ast, db := vicidialStageFixture(t)
	s, e := i.Prepare(m, src, dep, ast, db)
	if e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct {
		name   string
		change func(*VICIdialLabAuthorization)
	}{
		{"scope", func(a *VICIdialLabAuthorization) { a.Scope = "ALM-19/disposable-lab" }},
		{"host", func(a *VICIdialLabAuthorization) { a.Host = hash("other host") }},
		{"stage", func(a *VICIdialLabAuthorization) { a.StageSHA256 = hash("other stage") }},
		{"parent", func(a *VICIdialLabAuthorization) { a.ParentSHA256 = hash("other plan") }},
		{"manifest", func(a *VICIdialLabAuthorization) { a.ManifestSHA256 = hash("other manifest") }},
		{"source", func(a *VICIdialLabAuthorization) { a.SourceSHA256 = hash("other source") }},
		{"dependency", func(a *VICIdialLabAuthorization) { a.DependencySHA256 = hash("other dependency") }},
		{"snapshot", func(a *VICIdialLabAuthorization) { a.SnapshotReference = "" }},
		{"operator scope", func(a *VICIdialLabAuthorization) { a.OperatorEvidenceSHA256 = "" }},
		{"candidate acknowledgement", func(a *VICIdialLabAuthorization) { a.CandidateAcknowledged = false }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := vicidialAuth(s)
			tc.change(&a)
			f := &vicidialStageFake{s: i.Snapshot}
			_, e := ExecuteVICIdialLab(context.Background(), s, m, a, filepath.Join(t.TempDir(), "state"), f, f, false, nil)
			if e == nil || f.calls != 0 {
				t.Fatal("authorization bypass")
			}
		})
	}
	f := &vicidialStageFake{s: i.Snapshot}
	f.s.Files = "partial"
	if _, e := ExecuteVICIdialLab(context.Background(), s, m, vicidialAuth(s), filepath.Join(t.TempDir(), "stale"), f, f, false, nil); e == nil || f.calls != 0 {
		t.Fatal("stale plan executed")
	}
}
func TestVICIdialStageLockAndPostconditions(t *testing.T) {
	i, m, src, dep, ast, db := vicidialStageFixture(t)
	s, e := i.Prepare(m, src, dep, ast, db)
	if e != nil {
		t.Fatal(e)
	}
	dir := filepath.Join(t.TempDir(), "lock")
	lock, e := journal.Acquire(dir)
	if e != nil {
		t.Fatal(e)
	}
	f := &vicidialStageFake{s: i.Snapshot}
	_, e = ExecuteVICIdialLab(context.Background(), s, m, vicidialAuth(s), dir, f, f, false, nil)
	lock.Close()
	if e == nil || f.calls != 0 {
		t.Fatal("exclusive lock bypass")
	}
	for _, name := range []string{"exit failure", "false success", "unrelated mutation", "partial files"} {
		t.Run(name, func(t *testing.T) {
			f := &vicidialStageFake{s: i.Snapshot}
			switch name {
			case "exit failure":
				f.failAt = 1
			case "false success":
				f.noChange = true
			case "unrelated mutation":
				f.drift = true
			case "partial files":
				f.partialFiles = true
			}
			o, e := ExecuteVICIdialLab(context.Background(), s, m, vicidialAuth(s), filepath.Join(t.TempDir(), "state"), f, f, false, nil)
			if e == nil || f.calls != 1 || o.Journal.Actions[0].State == journal.Succeeded {
				t.Fatal("incorrect completion", e, o)
			}
		})
	}
}
func TestVICIdialInterruptedReconciliation(t *testing.T) {
	for _, tc := range []struct {
		name    string
		action  int
		partial bool
		blocked bool
	}{
		{"complete files incomplete journal", 1, false, false}, {"matching partial files safe completion", 1, true, false},
		{"registration complete incomplete journal", 4, false, false}, {"generator complete incomplete journal", 5, false, false},
		{"uncertain registration", 4, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			i, m, src, dep, ast, db := vicidialStageFixture(t)
			s, e := i.Prepare(m, src, dep, ast, db)
			if e != nil {
				t.Fatal(e)
			}
			f := &vicidialStageFake{s: i.Snapshot, crashAt: tc.action, partialFiles: tc.partial}
			dir := filepath.Join(t.TempDir(), "state")
			func() {
				defer func() {
					if recover() == nil {
						t.Fatal("interruption did not fire")
					}
				}()
				_, _ = ExecuteVICIdialLab(context.Background(), s, m, vicidialAuth(s), dir, f, f, false, nil)
			}()
			f.crashAt = 0
			f.partialFiles = false
			if tc.blocked {
				f.s.Registration = "partial"
			}
			_, e = ExecuteVICIdialLab(context.Background(), s, m, vicidialAuth(s), dir, f, f, true, nil)
			if (e != nil) != tc.blocked {
				t.Fatal(e)
			}
			if !tc.blocked && !vicidialSatisfied(f.s) {
				t.Fatal("incomplete reconciliation")
			}
		})
	}
}
