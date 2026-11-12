//go:build linux

package provision

import (
	"context"
	"errors"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grimange/almadial/internal/journal"
	"github.com/grimange/almadial/internal/preflight"
)

func TestInstallationImmutableIdentity(t *testing.T) {
	rt, m, app, perl, ast, db := runtimeFixture(t)
	initial := rt.Parent.Parent.Parent.Parent.Parent
	initial.Requirements = preflight.Requirements{Profile: "fixture-only", MinCPUs: 1, CPUFlags: []string{"sse2"}, MinTotalMemory: 1024, MinAvailableMemory: 512, MinDiskAvailable: 4096}
	initial.Preflight = preflight.Snapshot{HostID: initial.Preflight.HostID, OS: "almalinux", Release: "10.2", Architecture: "amd64", CPUs: 4, CPUFlags: []string{"sse2"}, TotalMemory: 8192, AvailableMemory: 4096, Disks: []preflight.Disk{{Path: "/", Total: 100000, Available: 90000}, {Path: "/usr", Total: 100000, Available: 90000}, {Path: "/var", Total: 100000, Available: 90000}}, Addresses: []netip.Addr{initial.Config.Network.BindAddress}, SELinux: "Enforcing", Firewall: "empty nftables ruleset", TimeSync: "yes", Known: map[string]bool{}}
	for _, key := range preflight.RequiredProbes() {
		initial.Preflight.Known[key] = true
	}
	r := InstallRequest{Config: initial.Config, Requirements: initial.Requirements, TLS: TLSRequest{"provided", "localhost", "/cert.pem", "/key.pem", "/trust.pem"}, Network: NetworkRequest{"isolated", "eth0", "10.1.2.3", 22}, Phone: PhoneRequest{phoneIdentity, "localhost", "wss://localhost:8089/ws"}, Authorization: InstallAuthorization{"xu7wxvch94ii2iw38hpb8d58", initial.Inventory.Host, hash("snapshot"), hash("authority"), true, true}}
	src := InstallSources{db, ast, app, perl, PhoneSource{files: map[string][]byte{"fixture": []byte("verified fixture")}}}
	cert := TLSCertificate{LeafSHA256: hash("leaf")}
	p, e := PrepareInstall(r, initial, m, src, cert)
	if e != nil {
		t.Fatal(e)
	}
	again, e := DecodeInstallPlan(p.JSON(), m, src)
	if e != nil || p.Digest() != again.Digest() {
		t.Fatal("plan roundtrip", e)
	}
	b := p.JSON()
	b[0] = '!'
	if p.JSON()[0] != '{' {
		t.Fatal("mutable plan")
	}
	for _, tc := range []string{"host", "authorization", "recovery", "source", "config"} {
		t.Run(tc, func(t *testing.T) {
			bad := r
			s := src
			switch tc {
			case "host":
				bad.Authorization.Host = hash("other")
			case "authorization":
				bad.Authorization.CandidateAcknowledged = false
			case "recovery":
				bad.Authorization.ManagementRecoveryVerified = false
			case "source":
				s.Application = VICIdialSource{}
			case "config":
				bad.Config.SchemaVersion = 999
			}
			q, err := PrepareInstall(bad, initial, m, s, cert)
			if tc == "source" {
				if err == nil && q.Digest() == p.Digest() {
					t.Fatal("source identity not bound")
				}
			} else if err == nil {
				t.Fatal("unsafe request accepted")
			}
		})
	}
	if _, e = DecodeInstallPlan(append(p.JSON(), []byte(` {}`)...), m, src); e == nil {
		t.Fatal("trailing plan accepted")
	}
}

func TestInstallationStageBoundaryRecovery(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "install")
	lock, e := journal.Acquire(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer lock.Close()
	r := installRun{ctx: context.Background(), state: dir, lock: lock}
	observations, effects := 0, 0
	inspect := func() (int, error) { observations++; return effects, nil }
	fail := errors.New("confirmed upstream failure")
	if _, e = driveInstallStage(&r, "S03", inspect, func(before int, resume bool) error { effects++; return fail }); !errors.Is(e, fail) {
		t.Fatal(e)
	}
	var accepted int
	if e = r.read("S03-accepted", &accepted); !errors.Is(e, os.ErrNotExist) {
		t.Fatal("failed stage accepted", e)
	}
	// The stage executor owns recovery. The adapter passes the same approved
	// before-input rather than replacing it with post-interruption observations.
	_, e = driveInstallStage(&r, "S03", inspect, func(before int, resume bool) error {
		if before != 0 {
			t.Fatal("replaced original input")
		}
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
	n := observations
	_, e = driveInstallStage(&r, "S03", inspect, func(int, bool) error { t.Fatal("replayed completed effect"); return nil })
	if e != nil || observations != n || effects != 1 {
		t.Fatal(e, observations, effects)
	}
	// Receipt reuse is deliberately not the final-host health test: the workflow
	// performs recursive fresh S01-S10 postconditions before returning NO_OP.
	if _, e = journal.Acquire(dir); !errors.Is(e, journal.ErrBusy) {
		t.Fatal("concurrent install bypassed lock", e)
	}
}

// Exercise the actual Linux exec boundary with a request larger than MAX_ARG_STRLEN.
// Invalid mode cannot apply a transaction, even on hosts with the DNF modules.
func TestInstallationLargeRPMTransport(t *testing.T) {
	if _, e := os.Stat("/usr/bin/python3"); e != nil {
		t.Skip("Python unavailable")
	}
	n, e := NewNative(nil)
	if e != nil {
		t.Fatal(e)
	}
	c := command("invalid", map[string]string{"padding": strings.Repeat("x", 160<<10)})
	before := jsonString(c)
	r, e := n.Runner.Run(context.Background(), c)
	if !r.Started || r.ExitCode == 0 || e == nil {
		t.Fatalf("large request did not reach bounded subprocess: %+v %v", r, e)
	}
	if before != jsonString(c) {
		t.Fatal("approved command was mutated")
	}
}

func TestInstallationVerifiedPauseAndResume(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "install")
	lock, err := journal.Acquire(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	r := installRun{ctx: context.Background(), state: dir, lock: lock, pauseAfter: "S03"}
	effects := 0
	inspect := func() (int, error) { return effects, nil }
	execute := func(int, bool) error { effects++; return nil }
	if _, err = driveInstallStage(&r, "S03", inspect, execute); !errors.Is(err, ErrInstallPaused) {
		t.Fatalf("verified checkpoint did not pause: %v", err)
	}
	r.pauseAfter = ""
	if _, err = driveInstallStage(&r, "S03", inspect, execute); err != nil || effects != 1 {
		t.Fatalf("resume replayed completed stage: effects=%d err=%v", effects, err)
	}
}
