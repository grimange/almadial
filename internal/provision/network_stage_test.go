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

type networkFake struct {
	s     NetworkSnapshot
	calls int
	crash bool
}

func (f *networkFake) InspectNetwork(context.Context) (NetworkSnapshot, error) { return f.s, nil }
func (f *networkFake) Validate(execution.Command) error                        { return nil }
func (f *networkFake) Run(context.Context, execution.Command) (execution.Result, error) {
	f.calls++
	f.s.Files = hash("files")
	f.s.Services = hash("services")
	f.s.Health = hash("policy health")
	f.s.Current = hash("new effective policy")
	if f.crash {
		panic("after completed effect")
	}
	return execution.Result{Started: true}, nil
}
func networkAuth(s NetworkStage) NetworkLabAuthorization {
	d := s.doc()
	return NetworkLabAuthorization{Scope: "ALM-25/disposable-lab", Host: d.Before.Host, StageSHA256: s.Digest(), ParentSHA256: d.Parent, ManifestSHA256: d.Manifest, PolicySHA256: NetworkPolicyDigest(), SnapshotReference: hash("snapshot"), OperatorEvidenceSHA256: hash("authority"), CandidateAcknowledged: true, DialingInactive: true, ManagementRecoveryVerified: true}
}
func TestNetworkExecutionRecoveryAndNoOp(t *testing.T) {
	for _, standard := range []bool{false, true} {
		t.Run(map[bool]string{false: "legacy", true: "standard"}[standard], func(t *testing.T) {
			networkAuth := networkAuth
			if standard {
				original := networkAuth
				networkAuth = func(s NetworkStage) NetworkLabAuthorization {
					a := original(s)
					a.Scope = StandardAuthorization
					a.SnapshotReference = ""
					a.OperatorEvidenceSHA256 = ""
					a.CandidateAcknowledged = false
					a.ManagementRecoveryVerified = false
					return a
				}
			}

			for _, crash := range []bool{false, true} {
				t.Run(map[bool]string{false: "normal", true: "completed interrupted journal"}[crash], func(t *testing.T) {
					p, m, src, perl, ast, db := runtimeFixture(t)
					p.Snapshot.Files = hash("rf")
					p.Snapshot.Services = hash("rs")
					p.Snapshot.Health = hash("rh")
					w := WebInputs{p, WebSnapshot{Host: p.Snapshot.Host, Foundation: p.Snapshot.Digest(), Foreign: hash("foreign"), Files: hash("wf"), Services: hash("ws"), Health: hash("wh")}}
					parent := TLSInputs{Parent: w, Request: TLSRequest{"provided", "localhost", "/cert.pem", "/key.pem", "/trust.pem"}, Certificate: TLSCertificate{LeafSHA256: hash("leaf"), ChainSHA256: hash("chain"), TrustSHA256: hash("trust")}, Snapshot: TLSSnapshot{Host: w.Snapshot.Host, Foundation: w.Snapshot.Digest(), Foreign: hash("tlsforeign"), Current: hash("tls current"), Files: hash("tlsfiles"), Services: hash("tlsservices"), Health: hash("tlshealth")}}
					i := NetworkInputs{Parent: parent, Request: NetworkRequest{"isolated", "eth0", "10.1.2.3", 22, 0, 0}, Snapshot: NetworkSnapshot{Host: parent.Snapshot.Host, Foundation: parent.Snapshot.Digest(), Foreign: hash("foreign"), Current: hash("old"), Files: "absent", Services: "absent", Health: "absent"}}
					s, e := i.Prepare(m, src, perl, ast, db)
					if e != nil || len(s.Graph().Actions()) != 1 {
						t.Fatal(e)
					}
					again, _ := i.Prepare(m, src, perl, ast, db)
					if s.Digest() != again.Digest() {
						t.Fatal("nondeterministic")
					}
					if !crash {
						t.Run("native failure trace and journal", func(t *testing.T) {
							r := &networkDiagnosticRunner{vicidialGenerationProbeRunner: vicidialGenerationProbeRunner{
								output: "distinct stdout",
								result: execution.Result{Started: true, ExitCode: 7, Stderr: "distinct stderr"},
								err:    &execution.Error{Category: execution.Exit, ExitCode: 7, Detail: "native fixture failed"},
							}}
							n := NetworkNative{Request: i.Request}
							n.TLS.Web.Runtime.Application.Source = src
							n.TLS.Web.Runtime.Application.Asterisk.Database.Native.Runner = r
							out, err := ExecuteNetworkLab(context.Background(), s, m, networkAuth(s), filepath.Join(t.TempDir(), "failure"), &networkFake{s: i.Snapshot}, n.Runner(s), false, nil)
							if err == nil || len(out.Traces) != 1 || out.Journal == nil || r.calls != 1 {
								t.Fatal("missing failed action evidence", err, out)
							}
							trace := out.Traces[0]
							if trace.ActionID != "network.00.activate" || trace.StageID != "S09" || trace.Result.ExitCode != 7 || trace.Result.Stdout != "distinct stdout" || !strings.Contains(trace.Result.Stderr, "distinct stderr") || out.Journal.Actions[0].ExitCode != 7 || out.Journal.Actions[0].State != journal.Failed {
								t.Fatal("native diagnostic lost through stage", out)
							}
						})
					}
					f := &networkFake{s: i.Snapshot, crash: crash}
					dir := filepath.Join(t.TempDir(), "state")
					if crash {
						func() {
							defer func() {
								if recover() == nil {
									t.Fatal("missing interruption")
								}
							}()
							_, _ = ExecuteNetworkLab(context.Background(), s, m, networkAuth(s), dir, f, f, false, nil)
						}()
						f.crash = false
					}
					out, e := ExecuteNetworkLab(context.Background(), s, m, networkAuth(s), dir, f, f, crash, nil)
					if e != nil || f.calls != 1 || !networkSatisfied(f.s) {
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
					out, e = ExecuteNetworkLab(context.Background(), s, m, networkAuth(s), filepath.Join(t.TempDir(), "noop"), f, f, false, nil)
					if e != nil || !out.NoOp || f.calls != 1 {
						t.Fatal(e, out)
					}
					for _, defect := range []string{"state", "authority", "lock"} {
						t.Run(defect, func(t *testing.T) {
							a := networkAuth(s)
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
							_, e := ExecuteNetworkLab(context.Background(), s, m, a, d, f, f, false, nil)
							f.s = before
							if e == nil || f.calls != 1 {
								t.Fatal("invalid execution admitted")
							}
						})
					}
				})
			}
		})
	}
}
