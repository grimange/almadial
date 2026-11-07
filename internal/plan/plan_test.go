package plan

import (
	"bytes"
	"encoding/json"
	"net/netip"
	"strings"
	"testing"

	"github.com/grimange/almadial/internal/config"
	"github.com/grimange/almadial/internal/preflight"
)

func inputs() (config.Config, preflight.Snapshot, preflight.Requirements) {
	ref := config.SecretRef{Type: config.Environment, Source: "FIXTURE_SOURCE"}
	c := config.Defaults(config.Config{SchemaVersion: 1, Network: config.Network{BindAddress: netip.MustParseAddr("192.0.2.10")}, Credentials: config.Credentials{Database: ref, AMI: ref, SIP: ref, Admin: ref}})
	s := preflight.Snapshot{HostID: strings.Repeat("a", 32), OS: "almalinux", Release: "10.2", Kernel: "fixture", Architecture: "amd64", CPUs: 4, CPUFlags: []string{"sse2", "avx"}, TotalMemory: 8192, AvailableMemory: 4096, Disks: []preflight.Disk{{Path: "/", Total: 100000, Available: 90000}}, Addresses: []netip.Addr{c.Network.BindAddress}, SELinux: "Enforcing", Firewall: "empty nftables ruleset", TimeSync: "yes", Known: map[string]bool{}}
	for _, k := range preflight.RequiredProbes() {
		s.Known[k] = true
	}
	r := preflight.Requirements{Profile: "synthetic-test-budget", MinCPUs: 1, CPUFlags: []string{"sse2"}, MinTotalMemory: 1024, MinAvailableMemory: 512, MinDiskAvailable: 4096}
	return c, s, r
}

const manifest = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestDeterministicOutputAndStages(t *testing.T) {
	for _, web := range []bool{false, true} {
		t.Run(map[bool]string{false: "basic", true: "webrtc"}[web], func(t *testing.T) {
			c, s, r := inputs()
			if web {
				c.WebRTC.Enabled = true
				c.WebRTC.TLS = &config.TLS{CertificateFile: "/fixture/certificate", PrivateKeyFile: "/fixture/private-key"}
				c = config.Defaults(c)
			}
			p, e := New(c, s, r, manifest)
			if e != nil {
				t.Fatal(e)
			}
			if !p.Eligible() {
				t.Fatal("fixture blocked")
			}
			b, _ := p.JSON()
			if !json.Valid(b) {
				t.Fatal("invalid JSON")
			}
			again, e := New(c, s, r, manifest)
			if e != nil || p.Digest() != again.Digest() {
				t.Fatal("unstable identity")
			}
			text, e := p.Human()
			if e != nil || !strings.Contains(text, "No stages executed") {
				t.Fatal("human output")
			}
			if strings.Contains(string(b), "FIXTURE_SOURCE") {
				t.Fatal("reference source unexpectedly included; digest suffices")
			}
			seen := map[string]bool{}
			last := ""
			for _, stage := range p.Stages() {
				if stage.ID <= last {
					t.Fatal("unstable stage order")
				}
				for _, dep := range stage.Prerequisites {
					if !seen[dep] {
						t.Fatalf("%s before %s", stage.ID, dep)
					}
				}
				if len(stage.Inputs) == 0 || len(stage.Changes) == 0 || len(stage.Postconditions) == 0 || stage.Risk == "" {
					t.Fatal("incomplete stage contract")
				}
				seen[stage.ID] = true
				last = stage.ID
			}
			if seen["S09"] != web {
				t.Fatal("optional feature mismatch")
			}
			if !web && (strings.Contains(text, "certificate and private-key") || strings.Contains(text, "FQDN required")) {
				t.Fatal("basic requires TLS/DNS")
			}
			if e = p.Revalidate(c, s, r, manifest); e != nil {
				t.Fatal(e)
			}
			t.Logf("fixture web=%v plan_sha256=%s config_digest_bound=true", web, p.Digest())
			t.Logf("PLAN_JSON=%s", b)
			t.Logf("PLAN_HUMAN=%s", text)
		})
	}
}
func TestImmutability(t *testing.T) {
	c, s, r := inputs()
	p, e := New(c, s, r, manifest)
	if e != nil {
		t.Fatal(e)
	}
	before, _ := p.JSON()
	digest := p.Digest()
	*c.SIP.Port = 5070
	c.Credentials.Admin.Source = "CHANGED"
	s.Known["os"] = false
	s.CPUFlags[0] = "changed"
	s.Disks[0].Available = 0
	r.CPUFlags[0] = "changed"
	stages := p.Stages()
	stages[0].Description = "changed"
	stages[1].Prerequisites[0] = "changed"
	stages[0].Inputs[0] = "changed"
	stages[0].Changes[0] = "changed"
	stages[0].Postconditions[0] = "changed"
	p.Findings()[0].Message = "changed"
	beforeCopy, _ := p.JSON()
	beforeCopy[0] = '!'
	after, _ := p.JSON()
	if !bytes.Equal(before, after) || digest != p.Digest() {
		t.Fatal("caller mutated plan")
	}
	c, s, r = inputs()
	again, _ := New(c, s, r, manifest)
	if p.Digest() != again.Digest() {
		t.Fatal("not reproducible after caller mutation")
	}
}
func TestFreshness(t *testing.T) {
	cases := []struct {
		name  string
		edit  func(*config.Config, *preflight.Snapshot, *preflight.Requirements, *string)
		stale bool
	}{
		{"config", func(c *config.Config, s *preflight.Snapshot, r *preflight.Requirements, m *string) {
			*c.SIP.Port = 5070
		}, true},
		{"host", func(c *config.Config, s *preflight.Snapshot, r *preflight.Requirements, m *string) {
			s.HostID = strings.Repeat("b", 32)
		}, true},
		{"OS", func(c *config.Config, s *preflight.Snapshot, r *preflight.Requirements, m *string) {
			s.Release = "10.3"
		}, true},
		{"CPU", func(c *config.Config, s *preflight.Snapshot, r *preflight.Requirements, m *string) { s.CPUFlags = nil }, true},
		{"resource-threshold", func(c *config.Config, s *preflight.Snapshot, r *preflight.Requirements, m *string) {
			s.AvailableMemory = 0
		}, true},
		{"listener", func(c *config.Config, s *preflight.Snapshot, r *preflight.Requirements, m *string) {
			s.Listeners = []preflight.Listener{{Protocol: "udp", Address: c.Network.BindAddress, Port: 5060}}
		}, true},
		{"service", func(c *config.Config, s *preflight.Snapshot, r *preflight.Requirements, m *string) {
			s.Services = []string{"asterisk.service"}
		}, true},
		{"probe-incomplete", func(c *config.Config, s *preflight.Snapshot, r *preflight.Requirements, m *string) {
			s.Known["firewall"] = false
		}, true},
		{"firewall-rules", func(c *config.Config, s *preflight.Snapshot, r *preflight.Requirements, m *string) {
			s.FirewallSHA256 = strings.Repeat("b", 64)
		}, true},
		{"manifest", func(c *config.Config, s *preflight.Snapshot, r *preflight.Requirements, m *string) {
			*m = strings.Repeat("b", 64)
		}, true},
		{"budget", func(c *config.Config, s *preflight.Snapshot, r *preflight.Requirements, m *string) { r.MinCPUs = 2 }, true},
		{"irrelevant", func(c *config.Config, s *preflight.Snapshot, r *preflight.Requirements, m *string) {
			s.Kernel = "patch"
			s.AvailableMemory--
			s.Disks[0].Available--
			s.Listeners = []preflight.Listener{{Protocol: "tcp", Address: netip.IPv6Unspecified(), Port: 9999}}
			s.CPUFlags = append(s.CPUFlags, "more")
		}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, s, r := inputs()
			p, _ := New(c, s, r, manifest)
			m := manifest
			tc.edit(&c, &s, &r, &m)
			if (p.Revalidate(c, s, r, m) != nil) != tc.stale {
				t.Fatal("freshness mismatch")
			}
		})
	}
}
func TestRejectedInputs(t *testing.T) {
	c, s, r := inputs()
	for _, m := range []string{"", "bad", strings.Repeat("A", 64)} {
		if _, e := New(c, s, r, m); e == nil {
			t.Fatal("bad manifest accepted")
		}
	}
	c.SchemaVersion = 2
	if _, e := New(c, s, r, manifest); e == nil {
		t.Fatal("bad config accepted")
	}
	c, s, r = inputs()
	s.OS = "other"
	p, e := New(c, s, r, manifest)
	if e != nil || p.Eligible() {
		t.Fatal("blocked plan should be inspectable")
	}
	if p.Revalidate(c, s, r, manifest) == nil {
		t.Fatal("blocked plan revalidated")
	}
	var zero Plan
	if zero.Eligible() || zero.Digest() != "" {
		t.Fatal("zero plan valid")
	}
	if _, e := zero.JSON(); e == nil {
		t.Fatal("zero serialized")
	}
	if _, e := zero.Human(); e == nil {
		t.Fatal("zero rendered")
	}
}

func TestEquivalentObservationOrdering(t *testing.T) {
	c, s, r := inputs()
	s.CPUFlags = append(s.CPUFlags, "flag1")
	p, _ := New(c, s, r, manifest)
	s.CPUFlags = []string{"flag1", "avx", "sse2"}
	again, _ := New(c, s, r, manifest)
	if p.Digest() != again.Digest() {
		t.Fatal("unordered observations affected identity")
	}
}
