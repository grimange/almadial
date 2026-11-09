package operatorui

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/netip"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/grimange/almadial/internal/artifacts"
	"github.com/grimange/almadial/internal/config"
	"github.com/grimange/almadial/internal/plan"
	"github.com/grimange/almadial/internal/preflight"
)

func fixture(t *testing.T) (config.Config, preflight.Snapshot, preflight.Requirements, artifacts.Manifest) {
	t.Helper()
	f, e := os.Open("../config/testdata/basic.json")
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	c, e := config.Load(f)
	if e != nil {
		t.Fatal(e)
	}
	mf, e := os.Open("../../manifests/stack.lock.yaml")
	if e != nil {
		t.Fatal(e)
	}
	defer mf.Close()
	m, e := artifacts.Load(mf)
	if e != nil {
		t.Fatal(e)
	}
	s := preflight.Snapshot{HostID: "synthetic-wizard", OS: "almalinux", Release: "10.2", Architecture: "amd64", CPUs: 4, CPUFlags: []string{"sse2"}, TotalMemory: 8192, AvailableMemory: 4096, Disks: []preflight.Disk{{Path: "/", Available: 99999}}, Addresses: []netip.Addr{c.Network.BindAddress}, SELinux: "Enforcing", TimeSync: "yes", Known: map[string]bool{}}
	for _, k := range preflight.RequiredProbes() {
		s.Known[k] = true
	}
	r := preflight.Requirements{Profile: "SYNTHETIC-NOT-PRODUCTION", MinCPUs: 1, CPUFlags: []string{"sse2"}, MinTotalMemory: 1024, MinAvailableMemory: 512, MinDiskAvailable: 4096}
	return c, s, r, m
}
func TestConfigurationJourneys(t *testing.T) {
	for _, tc := range []struct {
		name   string
		input  string
		loaded bool
		web    bool
	}{
		{"IP-only fresh", "192.0.2.10\n\n\n\n\n\nno\nenv:DB\nenv:AMI\nenv:SIP\nenv:ADMIN\n", false, false},
		{"loaded empty keeps defaults", strings.Repeat("\n", 11), true, false},
		{"invalid IP syntax", "bad\n192.0.2.10\n" + strings.Repeat("\n", 10), true, false},
		{"invalid port syntax", strings.Repeat("\n", 3) + "bad\n5060\n" + strings.Repeat("\n", 7), true, false},
		{"invalid port correction", strings.Repeat("\n", 3) + "0\n" + strings.Repeat("\n", 7) + "5060\n", true, false},
		{"invalid credential correction", strings.Repeat("\n", 7) + "env:bad-name\n" + strings.Repeat("\n", 3) + "file:/fixture/db\n", true, false},
		{"WebRTC enabled", strings.Repeat("\n", 6) + "yes\n" + strings.Repeat("\n", 4) + "\n/fixture/cert\n/fixture/key\n", true, true},
		{"WebRTC TLS correction", strings.Repeat("\n", 6) + "yes\n" + strings.Repeat("\n", 4) + "\nrelative\n/fixture/key\n/fixture/cert\n", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _, _, _ := fixture(t)
			if !tc.loaded {
				c = config.Config{}
			}
			var out bytes.Buffer
			p := NewPrompts(context.Background(), strings.NewReader(tc.input), &out)
			got, e := Configure(p, c)
			if e != nil {
				t.Fatalf("%v\n%s", e, &out)
			}
			if e = config.Validate(got); e != nil {
				t.Fatal(e)
			}
			if got.Network.Hostname != "" || got.WebRTC.Enabled != tc.web {
				t.Fatal("unwanted hostname/WebRTC")
			}
			if strings.Contains(out.String(), "\x1b") {
				t.Fatal("ANSI output")
			}
			t.Logf("valid config digest produced; WebRTC=%t; hostname optional", got.WebRTC.Enabled)
		})
	}
}
func TestPromptFailures(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		want        error
	}{
		{"EOF", "", ErrInput}, {"partial final line", "yes", ErrInput}, {"cancel", "cancel\n", ErrCanceled}, {"control C", "\x03\n", ErrCanceled}, {"oversized", strings.Repeat("x", 4097) + "\n", ErrInput},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := NewPrompts(context.Background(), strings.NewReader(tc.input), io.Discard)
			if _, e := p.Ask("question", ""); !errors.Is(e, tc.want) {
				t.Fatal(e)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	r, w := io.Pipe()
	defer r.Close()
	defer w.Close()
	p := NewPrompts(ctx, r, io.Discard)
	done := make(chan error, 1)
	go func() { _, e := p.Ask("question", ""); done <- e }()
	cancel()
	select {
	case e := <-done:
		if e != ErrCanceled {
			t.Fatal(e)
		}
	case <-time.After(time.Second):
		t.Fatal("cancel blocked")
	}
}
func TestApprovalDecisions(t *testing.T) {
	c, s, r, m := fixture(t)
	prep, e := Prepare(c, s, r, m)
	if e != nil || !prep.Plan.Eligible() {
		t.Fatal("fixture not eligible", e)
	}
	cd, _ := config.Digest(c)
	good := Decision{true, cd, prep.Plan.Digest()}
	for _, tc := range []struct {
		name           string
		change         func(*Preparation, *Decision)
		authorized, ok bool
	}{
		{"explicit approved fixture", func(*Preparation, *Decision) {}, true, true},
		{"missing intent", func(_ *Preparation, d *Decision) { d.Approved = false }, true, false},
		{"missing config digest", func(_ *Preparation, d *Decision) { d.ConfigSHA256 = "" }, true, false},
		{"wrong config digest", func(_ *Preparation, d *Decision) { d.ConfigSHA256 = strings.Repeat("0", 64) }, true, false},
		{"wrong plan", func(_ *Preparation, d *Decision) { d.PlanSHA256 = strings.Repeat("0", 64) }, true, false},
		{"missing plan", func(_ *Preparation, d *Decision) { d.PlanSHA256 = "" }, true, false},
		{"changed config even fresh digest", func(p *Preparation, d *Decision) {
			p.Config.Network.Hostname = "changed.example.invalid"
			d.ConfigSHA256, _ = config.Digest(p.Config)
		}, true, false},
		{"changed plan", func(p *Preparation, _ *Decision) {
			changed := s
			changed.HostID = "changed-host"
			p.Plan, _ = plan.New(c, changed, r, m.Digest())
		}, true, false},
		{"failed preflight", func(p *Preparation, d *Decision) {
			changed := s
			changed.SELinux = "Permissive"
			p.Plan, _ = plan.New(c, changed, r, m.Digest())
			d.PlanSHA256 = p.Plan.Digest()
		}, true, false},
		{"candidate", func(*Preparation, *Decision) {}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, d := prep, good
			tc.change(&p, &d)
			e := validateDecision(p, d, tc.authorized)
			if (e == nil) != tc.ok {
				t.Fatal(e)
			}
		})
	}
	if good.Validate(prep) == nil {
		t.Fatal("public candidate bypass")
	}
	if _, e = prep.Approve(context.Background(), NewPrompts(context.Background(), strings.NewReader("yes\n"), io.Discard)); e != ErrBlocked {
		t.Fatal("candidate prompt bypass")
	}
}
func TestInteractiveApproval(t *testing.T) {
	c, s, r, m := fixture(t)
	p, _ := Prepare(c, s, r, m)
	for _, tc := range []struct {
		name, input string
		ok          bool
	}{{"yes", "yes\n", true}, {"case", "YeS\n", true}, {"no", "no\n", false}, {"empty", "\n", false}, {"EOF", "", false}, {"partial yes", "yes", false}, {"cancel", "cancel\n", false}, {"invalid then yes", "maybe\nyes\n", true}, {"invalid then no", "maybe\nno\n", false}} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			ctx := context.Background()
			d, e := p.approve(ctx, NewPrompts(ctx, strings.NewReader(tc.input), &out), true)
			if (e == nil) != tc.ok || d.Approved != tc.ok {
				t.Fatalf("approved=%v err=%v", d.Approved, e)
			}
			if !strings.Contains(out.String(), p.Plan.Digest()) {
				t.Fatal("missing plan binding")
			}
		})
	}
}
func TestPresentationAndEquivalentPlans(t *testing.T) {
	c, s, r, m := fixture(t)
	prep, _ := Prepare(c, s, r, m)
	var output bytes.Buffer
	if e := prep.Present(&output); e != nil {
		t.Fatal(e)
	}
	for _, word := range []string{"VALID", "CANDIDATE", "NOT_RUN", "NOT STARTED", "S03", "irreversible", "prerequisites:", "postconditions:", "WARNING"} {
		if !strings.Contains(output.String(), word) {
			t.Fatal("missing presentation", word)
		}
	}
	p := NewPrompts(context.Background(), strings.NewReader(strings.Repeat("\n", 11)), io.Discard)
	interactive, e := Configure(p, c)
	if e != nil {
		t.Fatal(e)
	}
	again, _ := Prepare(interactive, s, r, m)
	if again.Plan.Digest() != prep.Plan.Digest() {
		t.Fatal("interactive/unattended plan differs")
	}
	s.Known["firewall"] = false
	blocked, _ := Prepare(c, s, r, m)
	output.Reset()
	blocked.Present(&output)
	if !strings.Contains(output.String(), "NOT_VERIFIED") || !strings.Contains(output.String(), "installation cannot continue") {
		t.Fatal("missing blocker guidance")
	}
}
