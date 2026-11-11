//go:build linux

package provision

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grimange/almadial/internal/execution"
	"github.com/grimange/almadial/internal/journal"
)

type tlsFake struct {
	s     TLSSnapshot
	calls int
	crash bool
}

func (f *tlsFake) InspectTLS(context.Context) (TLSSnapshot, error) { return f.s, nil }
func (f *tlsFake) Validate(execution.Command) error                { return nil }
func (f *tlsFake) Run(context.Context, execution.Command) (execution.Result, error) {
	f.calls++
	f.s.Files = hash("files")
	f.s.Services = hash("services")
	f.s.Health = hash("certificate")
	f.s.Current = hash("new generation")
	if f.crash {
		panic("after completed effect")
	}
	return execution.Result{Started: true}, nil
}
func tlsAuth(s TLSStage) TLSLabAuthorization {
	d := s.doc()
	return TLSLabAuthorization{Scope: "ALM-24/disposable-lab", Host: d.Before.Host, StageSHA256: s.Digest(), ParentSHA256: d.Parent, ManifestSHA256: d.Manifest, PolicySHA256: TLSPolicyDigest(), SnapshotReference: hash("snapshot"), OperatorEvidenceSHA256: hash("authority"), CandidateAcknowledged: true, DialingInactive: true}
}
func TestTLSExecutionRecoveryAndNoOp(t *testing.T) {
	for _, crash := range []bool{false, true} {
		t.Run(map[bool]string{false: "normal", true: "completed interrupted journal"}[crash], func(t *testing.T) {
			p, m, src, perl, ast, db := runtimeFixture(t)
			p.Snapshot.Files = hash("rf")
			p.Snapshot.Services = hash("rs")
			p.Snapshot.Health = hash("rh")
			w := WebInputs{p, WebSnapshot{Host: p.Snapshot.Host, Foundation: p.Snapshot.Digest(), Foreign: hash("foreign"), Files: hash("wf"), Services: hash("ws"), Health: hash("wh")}}
			i := TLSInputs{Parent: w, Request: TLSRequest{"provided", "localhost", "/cert.pem", "/key.pem", "/trust.pem"}, Certificate: TLSCertificate{LeafSHA256: hash("leaf"), ChainSHA256: hash("chain"), TrustSHA256: hash("trust")}, Snapshot: TLSSnapshot{Host: w.Snapshot.Host, Foundation: w.Snapshot.Digest(), Foreign: hash("tlsforeign"), Current: hash("old"), Files: "absent", Services: "absent", Health: "absent"}}
			s, e := i.Prepare(m, src, perl, ast, db)
			if e != nil || len(s.Graph().Actions()) != 1 {
				t.Fatal(e)
			}
			again, _ := i.Prepare(m, src, perl, ast, db)
			if s.Digest() != again.Digest() {
				t.Fatal("nondeterministic")
			}
			f := &tlsFake{s: i.Snapshot, crash: crash}
			dir := filepath.Join(t.TempDir(), "state")
			if crash {
				func() {
					defer func() {
						if recover() == nil {
							t.Fatal("missing interruption")
						}
					}()
					_, _ = ExecuteTLSLab(context.Background(), s, m, tlsAuth(s), dir, f, f, false, nil)
				}()
				f.crash = false
			}
			out, e := ExecuteTLSLab(context.Background(), s, m, tlsAuth(s), dir, f, f, crash, nil)
			if e != nil || f.calls != 1 || !tlsSatisfied(f.s) {
				t.Fatal(e, out)
			}
			if out.Journal.Actions[0].State != journal.Succeeded {
				t.Fatal(out)
			}
			i.Snapshot = f.s
			s, e = i.Prepare(m, src, perl, ast, db)
			if e != nil || len(s.Graph().Actions()) != 0 {
				t.Fatal(e)
			}
			out, e = ExecuteTLSLab(context.Background(), s, m, tlsAuth(s), filepath.Join(t.TempDir(), "noop"), f, f, false, nil)
			if e != nil || !out.NoOp || f.calls != 1 {
				t.Fatal(e, out)
			}
			for _, defect := range []string{"state", "authority", "lock"} {
				t.Run(defect, func(t *testing.T) {
					a := tlsAuth(s)
					d := filepath.Join(t.TempDir(), "reject")
					before := f.s
					switch defect {
					case "state":
						f.s.Health = "absent"
					case "authority":
						a.Host = hash("wrong")
					case "lock":
						lock, e := journal.Acquire(d)
						if e != nil {
							t.Fatal(e)
						}
						defer lock.Close()
					}
					_, e := ExecuteTLSLab(context.Background(), s, m, a, d, f, f, false, nil)
					f.s = before
					if e == nil || f.calls != 1 {
						t.Fatal("invalid execution admitted")
					}
				})
			}
		})
	}
}
func TestTLSClosedHandoff(t *testing.T) {
	good := `tcp LISTEN 0 511 127.0.0.1:80 0.0.0.0:* users:(("httpd",pid=2400,fd=3))`
	secure := `tcp LISTEN 0 511 127.0.0.1:443 0.0.0.0:* users:(("httpd",pid=2400,fd=4))`
	if verifyVICIdialWebListeners(good+"\n"+secure, 2400) == nil || verifyVICIdialWebListenersTLS(good+"\n"+secure, 2400, true) != nil {
		t.Fatal("handoff widened ordinary S05 policy")
	}
	for _, r := range []TLSRequest{{"acme", "localhost", "/c", "/k", "/t"}, {"provided", "localhost\nListen 443", "/c", "/k", "/t"}, {"provided", "localhost", "relative", "/k", "/t"}} {
		if tlsRequestValid(r) {
			t.Fatal("unsafe request")
		}
	}
}

func TestTLSAsteriskListenerHandoff(t *testing.T) {
	base := "tcp LISTEN 0 10 127.0.0.1:5038 0.0.0.0:* users:((\"asterisk\",pid=123,fd=1))\ntcp LISTEN 0 10 127.0.0.1:8088 0.0.0.0:* users:((\"asterisk\",pid=123,fd=2))\n"
	tls := base + "tcp LISTEN 0 10 127.0.0.1:8089 0.0.0.0:* users:((\"asterisk\",pid=123,fd=3))\n"
	if verifyAsteriskListeners(tls, 123) == nil || verifyAsteriskListenersTLS(tls, 123, true) != nil {
		t.Fatal("closed S08 listener handoff")
	}
	for _, changed := range []string{strings.Replace(tls, "127.0.0.1:8089", "0.0.0.0:8089", 1), strings.Replace(tls, "pid=123,fd=3", "pid=124,fd=3", 1), strings.Replace(tls, "127.0.0.1:8089", "127.0.0.1:5060", 1)} {
		if verifyAsteriskListenersTLS(changed, 123, true) == nil {
			t.Fatal("unsafe listener accepted")
		}
	}
}
