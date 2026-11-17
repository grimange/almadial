//go:build linux

package provision

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grimange/almadial/internal/artifacts"
	"github.com/grimange/almadial/internal/execution"
	"github.com/grimange/almadial/internal/journal"
)

func runtimeFixture(t *testing.T) (RuntimeInputs, artifacts.Manifest, VICIdialSource, VICIdialPerlSource, AsteriskSource, DatabaseSource) {
	p, m, src, perl, ast, db := vicidialStageFixture(t)
	p.Snapshot.Files = hash("files")
	p.Snapshot.Perl = hash("perl")
	p.Snapshot.Configuration = hash("config")
	p.Snapshot.Registration = hash("registration")
	p.Snapshot.Generated = hash("generation")
	p.Snapshot.Web = hash("web")
	p.Snapshot.Runtime = hash("runtime")
	return RuntimeInputs{p, RuntimeSnapshot{Host: p.Snapshot.Host, Foundation: p.Snapshot.Digest(), Foreign: hash("foreign schedules"), Files: "absent", Services: "absent", Health: "absent"}}, m, src, perl, ast, db
}
func runtimeAuth(s RuntimeStage) RuntimeLabAuthorization {
	d := s.doc()
	return RuntimeLabAuthorization{Scope: "ALM-21/disposable-lab", Host: d.Before.Host, StageSHA256: s.Digest(), ParentSHA256: d.Parent, ManifestSHA256: d.Manifest, PolicySHA256: RuntimePolicyDigest(), SnapshotReference: hash("synthetic snapshot"), OperatorEvidenceSHA256: hash("synthetic authority"), CandidateAcknowledged: true, DialingInactive: true}
}

type runtimeFake struct {
	s                       RuntimeSnapshot
	calls, failAt, crashAt  int
	noChange, foreignChange bool
}

func (f *runtimeFake) InspectRuntime(context.Context) (RuntimeSnapshot, error) { return f.s, nil }
func (f *runtimeFake) Validate(execution.Command) error                        { return nil }
func (f *runtimeFake) Run(_ context.Context, c execution.Command) (execution.Result, error) {
	f.calls++
	if f.calls == f.failAt {
		return execution.Result{Started: true, ExitCode: 1}, errors.New("synthetic dependency failure")
	}
	if !f.noChange {
		switch c.Args[1] {
		case "runtime.00.files":
			f.s.Files = hash("files")
		case "runtime.01.services":
			f.s.Services = hash("services")
			f.s.Health = hash("health")
		case "runtime.02.verify":
			f.s.Health = hash("health")
		}
	}
	if f.foreignChange {
		f.s.Foreign = hash("operator schedule changed")
	}
	if f.calls == f.crashAt {
		panic("synthetic interrupted journal outcome")
	}
	return execution.Result{Started: true}, nil
}
func TestRuntimeExecutionFreshNoOpAndDrift(t *testing.T) {
	for _, standard := range []bool{false, true} {
		t.Run(map[bool]string{false: "legacy", true: "standard"}[standard], func(t *testing.T) {
			runtimeAuth := runtimeAuth
			if standard {
				original := runtimeAuth
				runtimeAuth = func(s RuntimeStage) RuntimeLabAuthorization {
					a := original(s)
					a.Scope = StandardAuthorization
					a.SnapshotReference = ""
					a.OperatorEvidenceSHA256 = ""
					a.CandidateAcknowledged = false
					return a
				}
			}

			i, m, src, perl, ast, db := runtimeFixture(t)
			s, e := i.Prepare(m, src, perl, ast, db)
			if e != nil {
				t.Fatal(e)
			}
			s2, e := i.Prepare(m, src, perl, ast, db)
			if e != nil || s2.Digest() != s.Digest() || len(s.Graph().Actions()) != 3 {
				t.Fatal("non-deterministic plan", e)
			}
			f := &runtimeFake{s: i.Snapshot}
			out, e := ExecuteRuntimeLab(context.Background(), s, m, runtimeAuth(s), filepath.Join(t.TempDir(), "state"), f, f, false, nil)
			if e != nil || out.NoOp || f.calls != 3 || !runtimeSatisfied(f.s) {
				t.Fatal(out, e)
			}
			for _, r := range out.Journal.Actions {
				if r.State != journal.Succeeded || r.Evidence == "" {
					t.Fatal(r)
				}
			}
			i.Snapshot = f.s
			s, e = i.Prepare(m, src, perl, ast, db)
			if e != nil {
				t.Fatal(e)
			}
			out, e = ExecuteRuntimeLab(context.Background(), s, m, runtimeAuth(s), filepath.Join(t.TempDir(), "noop"), f, f, false, nil)
			if e != nil || !out.NoOp || len(out.Traces) != 0 || f.calls != 3 {
				t.Fatal(out, e)
			}
			f.s.Health = "absent"
			if _, e = ExecuteRuntimeLab(context.Background(), s, m, runtimeAuth(s), filepath.Join(t.TempDir(), "stale"), f, f, false, nil); e == nil {
				t.Fatal("stale health accepted")
			}
			i.Snapshot = f.s
			s, e = i.Prepare(m, src, perl, ast, db)
			if e != nil || len(s.Graph().Actions()) != 1 {
				t.Fatal("fresh health drift missed", e)
			}
		})
	}
}
func TestRuntimeAuthorizationAndFoundationBoundaries(t *testing.T) {
	for _, name := range []string{"scope", "host", "stage", "parent", "manifest", "policy", "snapshot", "operator", "candidate", "dialing", "stale", "foreign", "lock", "no-postcondition", "failed-start"} {
		t.Run(name, func(t *testing.T) {
			i, m, src, perl, ast, db := runtimeFixture(t)
			s, e := i.Prepare(m, src, perl, ast, db)
			if e != nil {
				t.Fatal(e)
			}
			a := runtimeAuth(s)
			f := &runtimeFake{s: i.Snapshot}
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
			case "policy":
				a.PolicySHA256 = hash("other")
			case "snapshot":
				a.SnapshotReference = ""
			case "operator":
				a.OperatorEvidenceSHA256 = ""
			case "candidate":
				a.CandidateAcknowledged = false
			case "dialing":
				a.DialingInactive = false
			case "stale":
				f.s.Files = "partial"
			case "foreign":
				f.foreignChange = true
			case "lock":
				lock, e := journal.Acquire(dir)
				if e != nil {
					t.Fatal(e)
				}
				defer lock.Close()
			case "no-postcondition":
				f.noChange = true
			case "failed-start":
				f.failAt = 2
			}
			if _, e = ExecuteRuntimeLab(context.Background(), s, m, a, dir, f, f, false, nil); e == nil {
				t.Fatal("unsafe request passed")
			}
			if name != "foreign" && name != "no-postcondition" && name != "failed-start" && f.calls != 0 {
				t.Fatal("mutation before safety gate")
			}
		})
	}
}
func TestRuntimeInterruptedReconciliation(t *testing.T) {
	for _, at := range []int{1, 2, 3} {
		t.Run(string(rune('0'+at)), func(t *testing.T) {
			i, m, src, perl, ast, db := runtimeFixture(t)
			s, e := i.Prepare(m, src, perl, ast, db)
			if e != nil {
				t.Fatal(e)
			}
			f := &runtimeFake{s: i.Snapshot, crashAt: at}
			dir := filepath.Join(t.TempDir(), "state")
			func() {
				defer func() {
					if recover() == nil {
						t.Error("interruption missing")
					}
				}()
				_, _ = ExecuteRuntimeLab(context.Background(), s, m, runtimeAuth(s), dir, f, f, false, nil)
			}()
			f.crashAt = 0
			out, e := ExecuteRuntimeLab(context.Background(), s, m, runtimeAuth(s), dir, f, f, true, nil)
			if e != nil || !runtimeSatisfied(f.s) || f.calls != 3 {
				t.Fatal("valid completed state replayed", out, e, f.calls)
			}
		})
	}
}
func TestRuntimeCronAndActivationContract(t *testing.T) {
	cron := RuntimeCron()
	for _, line := range strings.Split(cron, "\n") {
		if strings.Contains(line, ".pl") && !strings.HasPrefix(line, "#") {
			t.Fatal("dialing-inactive contract executes cron")
		}
	}
	for _, bad := range []string{"password=", "Secret:", "VARDB_pass", "$("} {
		if strings.Contains(cron, bad) {
			t.Fatal("unsafe cron content", bad)
		}
	}
	if !strings.Contains(cron, "CRON_TZ=UTC") || !strings.Contains(cron, "# * * * * * root /usr/share/astguiclient/ADMIN_keepalive_ALL.pl") {
		t.Fatal("schedule provenance lost")
	}
	if RuntimePolicyDigest() == "" || RuntimeCron() != cron {
		t.Fatal("unstable generation")
	}
	for _, id := range []string{"fastagi", "updater", "listener"} {
		u := runtimeUnit(id, "test.pl", id == "fastagi")
		for _, must := range []string{"User=asterisk", "Group=asterisk", "KillMode=control-group", "Restart=on-failure", "ExecStartPre="} {
			if !strings.Contains(u, must) {
				t.Fatal(must)
			}
		}
	}
	if _, e := (VICIdialSource{}).runtimeScript("FastAGI_log.pl"); e == nil {
		t.Fatal("unverified script accepted")
	}
	if _, e := runtimeReplace("foreign", "expected", "replacement", 1); e == nil {
		t.Fatal("unknown source preimage accepted")
	}
}
