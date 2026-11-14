//go:build linux

package provision

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/grimange/almadial/internal/execution"
	"github.com/grimange/almadial/internal/journal"
)

type phoneFake struct {
	s     PhoneSnapshot
	calls int
	crash bool
}

func (f *phoneFake) InspectPhone(context.Context) (PhoneSnapshot, error) { return f.s, nil }
func (f *phoneFake) Validate(execution.Command) error                    { return nil }
func (f *phoneFake) Run(context.Context, execution.Command) (execution.Result, error) {
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
func phoneAuth(s PhoneStage) PhoneLabAuthorization {
	d := s.doc()
	return PhoneLabAuthorization{Scope: "ALM-26/disposable-lab", Host: d.Before.Host, StageSHA256: s.Digest(), ParentSHA256: d.Parent, ManifestSHA256: d.Manifest, PolicySHA256: PhonePolicyDigest(), SnapshotReference: hash("snapshot"), OperatorEvidenceSHA256: hash("authority"), CandidateAcknowledged: true, DialingInactive: true, ManagementRecoveryVerified: true}
}
func TestPhoneExecutionRecoveryAndNoOp(t *testing.T) {
	for _, crash := range []bool{false, true} {
		t.Run(map[bool]string{false: "normal", true: "completed interrupted journal"}[crash], func(t *testing.T) {
			p, m, src, perl, ast, db := runtimeFixture(t)
			p.Snapshot.Files = hash("rf")
			p.Snapshot.Services = hash("rs")
			p.Snapshot.Health = hash("rh")
			w := WebInputs{p, WebSnapshot{Host: p.Snapshot.Host, Foundation: p.Snapshot.Digest(), Foreign: hash("foreign"), Files: hash("wf"), Services: hash("ws"), Health: hash("wh")}}
			parent := TLSInputs{Parent: w, Request: TLSRequest{"provided", "localhost", "/cert.pem", "/key.pem", "/trust.pem"}, Certificate: TLSCertificate{LeafSHA256: hash("leaf"), ChainSHA256: hash("chain"), TrustSHA256: hash("trust")}, Snapshot: TLSSnapshot{Host: w.Snapshot.Host, Foundation: w.Snapshot.Digest(), Foreign: hash("tlsforeign"), Current: hash("tls current"), Files: hash("tlsfiles"), Services: hash("tlsservices"), Health: hash("tlshealth")}}
			network := NetworkInputs{Parent: parent, Request: NetworkRequest{"isolated", "eth0", "10.1.2.3", 22, 0, 0}, Snapshot: NetworkSnapshot{Host: parent.Snapshot.Host, Foundation: parent.Snapshot.Digest(), Foreign: hash("nf"), Current: hash("nc"), Files: hash("nfiles"), Services: hash("ns"), Health: hash("nh")}}
			phone := PhoneSource{files: map[string][]byte{"fixture": []byte("verified fixture")}}
			i := PhoneInputs{Parent: network, Request: PhoneRequest{phoneIdentity, "localhost", "wss://localhost:8089/ws"}, PhoneSourceSHA256: phone.Digest(), Snapshot: PhoneSnapshot{Host: network.Snapshot.Host, Foundation: network.Snapshot.Digest(), Foreign: hash("foreign"), Current: hash("old"), Files: "absent", Services: "absent", Health: "absent"}}

			s, e := i.Prepare(m, src, perl, ast, db, phone)
			if e != nil || len(s.Graph().Actions()) != 1 {
				t.Fatal(e)
			}
			again, _ := i.Prepare(m, src, perl, ast, db, phone)
			if s.Digest() != again.Digest() {
				t.Fatal("nondeterministic")
			}
			f := &phoneFake{s: i.Snapshot, crash: crash}
			dir := filepath.Join(t.TempDir(), "state")
			if crash {
				func() {
					defer func() {
						if recover() == nil {
							t.Fatal("missing interruption")
						}
					}()
					_, _ = ExecutePhoneLab(context.Background(), s, m, phoneAuth(s), dir, f, f, false, nil)
				}()
				f.crash = false
			}
			out, e := ExecutePhoneLab(context.Background(), s, m, phoneAuth(s), dir, f, f, crash, nil)
			if e != nil || f.calls != 1 || !phoneSatisfied(f.s) {
				t.Fatal(e, out)
			}
			if out.Journal.Actions[0].State != journal.Succeeded {
				t.Fatal(out)
			}
			i.Snapshot = f.s
			s, e = i.Prepare(m, src, perl, ast, db, phone)
			if e != nil || len(s.Graph().Actions()) != 0 {
				t.Fatal(e)
			}
			out, e = ExecutePhoneLab(context.Background(), s, m, phoneAuth(s), filepath.Join(t.TempDir(), "noop"), f, f, false, nil)
			if e != nil || !out.NoOp || f.calls != 1 {
				t.Fatal(e, out)
			}
			for _, defect := range []string{"state", "authority", "lock"} {
				t.Run(defect, func(t *testing.T) {
					a := phoneAuth(s)
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
					_, e := ExecutePhoneLab(context.Background(), s, m, a, d, f, f, false, nil)
					f.s = before
					if e == nil || f.calls != 1 {
						t.Fatal("invalid execution admitted")
					}
				})
			}
		})
	}
}
