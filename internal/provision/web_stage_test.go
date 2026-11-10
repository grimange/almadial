//go:build linux

package provision

import (
	"context"
	"github.com/grimange/almadial/internal/execution"
	"github.com/grimange/almadial/internal/journal"
	"path/filepath"
	"strings"
	"testing"
)

type webFake struct {
	s       WebSnapshot
	calls   int
	crashAt int
}

func (f *webFake) InspectWeb(context.Context) (WebSnapshot, error) { return f.s, nil }
func (f *webFake) Validate(execution.Command) error                { return nil }
func (f *webFake) Run(_ context.Context, c execution.Command) (execution.Result, error) {
	f.calls++
	switch c.Args[1] {
	case "web.00.files":
		f.s.Files = hash("files")
	case "web.01.services":
		f.s.Services = hash("services")
	case "web.02.verify":
		f.s.Health = hash("authenticated")
	}
	if f.crashAt == f.calls {
		panic("interrupted after effect")
	}
	return execution.Result{Started: true}, nil
}
func webAuth(s WebStage) WebLabAuthorization {
	d := s.doc()
	return WebLabAuthorization{Scope: "ALM-23/disposable-lab", Host: d.Before.Host, StageSHA256: s.Digest(), ParentSHA256: d.Parent, ManifestSHA256: d.Manifest, PolicySHA256: WebPolicyDigest(), SnapshotReference: hash("snapshot"), OperatorEvidenceSHA256: hash("authority"), CandidateAcknowledged: true, DialingInactive: true}
}
func TestWebExecutionAndRecovery(t *testing.T) {
	for _, crashAt := range []int{0, 1, 2, 3} {
		crash := crashAt > 0
		t.Run(map[int]string{0: "normal", 1: "interrupted-files", 2: "interrupted-services", 3: "interrupted-setup"}[crashAt], func(t *testing.T) {
			p, m, src, perl, ast, db := runtimeFixture(t)
			p.Snapshot.Files = hash("runtime-files")
			p.Snapshot.Services = hash("runtime-services")
			p.Snapshot.Health = hash("runtime-health")
			i := WebInputs{p, WebSnapshot{Host: p.Snapshot.Host, Foundation: p.Snapshot.Digest(), Foreign: hash("foreign"), Files: "absent", Services: "absent", Health: "absent"}}
			s, e := i.Prepare(m, src, perl, ast, db)
			if e != nil || len(s.Graph().Actions()) != 3 {
				t.Fatal(e)
			}
			s2, _ := i.Prepare(m, src, perl, ast, db)
			if s.Digest() != s2.Digest() {
				t.Fatal("nondeterministic")
			}
			f := &webFake{s: i.Snapshot, crashAt: crashAt}
			dir := filepath.Join(t.TempDir(), "state")
			if crash {
				func() {
					defer func() {
						if recover() == nil {
							t.Fatal("missing interruption")
						}
					}()
					_, _ = ExecuteWebLab(context.Background(), s, m, webAuth(s), dir, f, f, false, nil)
				}()
				f.crashAt = 0
			}
			out, e := ExecuteWebLab(context.Background(), s, m, webAuth(s), dir, f, f, crash, nil)
			if e != nil || f.calls != 3 || !webSatisfied(f.s) {
				t.Fatal(out, e, f.calls)
			}
			if crash {
				for _, a := range out.Journal.Actions {
					if a.State != journal.Succeeded {
						t.Fatal(a)
					}
				}
			}
			i.Snapshot = f.s
			s, e = i.Prepare(m, src, perl, ast, db)
			if e != nil {
				t.Fatal(e)
			}
			out, e = ExecuteWebLab(context.Background(), s, m, webAuth(s), filepath.Join(t.TempDir(), "noop"), f, f, false, nil)
			if e != nil || !out.NoOp || f.calls != 3 {
				t.Fatal(out, e)
			}
			f.s.Health = "absent"
			_, e = ExecuteWebLab(context.Background(), s, m, webAuth(s), filepath.Join(t.TempDir(), "stale"), f, f, false, nil)
			if e == nil {
				t.Fatal("stale NO_OP")
			}
		})
	}
}
func TestWebDependencyAuthorizationAndLock(t *testing.T) {
	p, m, src, perl, ast, db := runtimeFixture(t)
	p.Snapshot.Files = hash("f")
	p.Snapshot.Services = hash("s")
	p.Snapshot.Health = hash("h")
	i := WebInputs{p, WebSnapshot{Host: p.Snapshot.Host, Foundation: p.Snapshot.Digest(), Foreign: hash("foreign"), Files: "absent", Services: "absent", Health: "absent"}}
	s, e := i.Prepare(m, src, perl, ast, db)
	if e != nil {
		t.Fatal(e)
	}
	for _, kind := range []string{"scope", "identity", "foreign", "lock"} {
		t.Run(kind, func(t *testing.T) {
			f := &webFake{s: i.Snapshot}
			a := webAuth(s)
			dir := filepath.Join(t.TempDir(), "state")
			switch kind {
			case "scope":
				a.Scope = "production"
			case "identity":
				a.ManifestSHA256 = hash("stale")
			case "foreign":
				f.s.Foreign = hash("changed")
			case "lock":
				lock, e := journal.Acquire(dir)
				if e != nil {
					t.Fatal(e)
				}
				defer lock.Close()
			}
			if _, e := ExecuteWebLab(context.Background(), s, m, a, dir, f, f, false, nil); e == nil || f.calls != 0 {
				t.Fatal("unsafe execution", e)
			}
		})
	}
	i.Parent.Snapshot.Health = "absent"
	if _, e = i.Prepare(m, src, perl, ast, db); e == nil {
		t.Fatal("incomplete upstream")
	}
}
func TestWebSetupHandoffAndConfiguration(t *testing.T) {
	cols := []string{"first_login_trigger", "other"}
	for _, s := range []string{`["59","4142"]`, `["4E","4142"]`} {
		got, e := webNormalizeSetup(cols, s)
		if e != nil || got != `["59","4142"]` {
			t.Fatal(got, e)
		}
	}
	for _, s := range []string{`[null,"4142"]`, `["58","4142"]`, `["4E"]`, `[]\n[]`} {
		if _, e := webNormalizeSetup(cols, s); e == nil {
			t.Fatal("invalid handoff", s)
		}
	}
	for _, bad := range []string{"Listen ", "0.0.0.0", "SSLEngine", "password="} {
		if strings.Contains(WebVhost(), bad) {
			t.Fatal(bad)
		}
	}
	for _, good := range []string{"<VirtualHost 127.0.0.1:80>", "Require local", "/run/php-fpm/www.sock", "options\\.php"} {
		if !strings.Contains(WebVhost(), good) {
			t.Fatal(good)
		}
	}
}
