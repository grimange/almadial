package preflight

import (
	"context"
	"errors"
	"net/netip"
	"reflect"
	"strings"
	"testing"

	"github.com/grimange/almadial/internal/config"
)

func fixtureConfig() config.Config {
	ref := config.SecretRef{Type: config.Environment, Source: "TEST_SOURCE"}
	return config.Defaults(config.Config{SchemaVersion: 1, Network: config.Network{BindAddress: netip.MustParseAddr("192.0.2.10")}, Credentials: config.Credentials{Database: ref, AMI: ref, SIP: ref, Admin: ref}})
}
func fixtureBudget() Requirements {
	return Requirements{"test-only-budget", 1, []string{"sse2"}, 1024, 512, 4096}
}
func fixtureHost() Snapshot {
	s := Snapshot{HostID: strings.Repeat("a", 32), OS: "almalinux", Release: "10.2", Kernel: "fixture", Architecture: "amd64", CPUs: 4, CPUFlags: []string{"sse2", "avx"}, TotalMemory: 8192, AvailableMemory: 4096, Disks: []Disk{{"/", 100000, 90000}, {"/usr", 100000, 90000}, {"/var", 100000, 90000}}, Addresses: []netip.Addr{netip.MustParseAddr("192.0.2.10")}, SELinux: "Enforcing", Firewall: "empty nftables ruleset", TimeSync: "yes", Known: map[string]bool{}}
	for _, k := range RequiredProbes() {
		s.Known[k] = true
	}
	return s
}
func TestAssessment(t *testing.T) {
	cases := []struct {
		name    string
		change  func(*Snapshot, *config.Config, *Requirements)
		blocked bool
		want    string
	}{
		{"supported", func(*Snapshot, *config.Config, *Requirements) {}, false, ""},
		{"IPv6", func(s *Snapshot, c *config.Config, r *Requirements) {
			c.Network.BindAddress = netip.MustParseAddr("2001:db8::1")
			c.Network.AdvertiseAddress = c.Network.BindAddress
			s.Addresses = []netip.Addr{c.Network.BindAddress}
		}, false, ""},
		{"distribution", func(s *Snapshot, c *config.Config, r *Requirements) { s.OS = "other" }, true, "os"},
		{"unqualified-minor", func(s *Snapshot, c *config.Config, r *Requirements) { s.Release = "10.3" }, true, "os"},
		{"architecture", func(s *Snapshot, c *config.Config, r *Requirements) { s.Architecture = "arm64" }, true, "architecture"},
		{"CPU", func(s *Snapshot, c *config.Config, r *Requirements) { s.CPUFlags = nil }, true, "cpu-budget"},
		{"CPU-count", func(s *Snapshot, c *config.Config, r *Requirements) { s.CPUs = 0 }, true, "cpu-budget"},
		{"memory-total", func(s *Snapshot, c *config.Config, r *Requirements) { s.TotalMemory = 100 }, true, "memory-budget"},
		{"memory-available", func(s *Snapshot, c *config.Config, r *Requirements) { s.AvailableMemory = 0 }, true, "memory-budget"},
		{"disk", func(s *Snapshot, c *config.Config, r *Requirements) { s.Disks[0].Available = 1 }, true, "disk-budget"},
		{"missing-budget", func(s *Snapshot, c *config.Config, r *Requirements) { *r = Requirements{} }, true, "budget"},
		{"SIP", func(s *Snapshot, c *config.Config, r *Requirements) {
			s.Listeners = []Listener{{"udp", c.Network.BindAddress, 5060}}
		}, true, "listener-udp"},
		{"RTP", func(s *Snapshot, c *config.Config, r *Requirements) {
			s.Listeners = []Listener{{"udp", c.Network.BindAddress, 15000}}
		}, true, "listener-udp"},
		{"TCP-independent-RTP", func(s *Snapshot, c *config.Config, r *Requirements) {
			s.Listeners = []Listener{{"tcp", c.Network.BindAddress, 15000}}
		}, false, ""},
		{"different-bind", func(s *Snapshot, c *config.Config, r *Requirements) {
			s.Listeners = []Listener{{"udp", netip.MustParseAddr("192.0.2.11"), 5060}}
		}, false, ""},
		{"v6-wildcard", func(s *Snapshot, c *config.Config, r *Requirements) {
			s.Listeners = []Listener{{"udp", netip.IPv6Unspecified(), 5060}}
		}, true, "listener-udp"},
		{"WSS", func(s *Snapshot, c *config.Config, r *Requirements) {
			c.WebRTC.Enabled = true
			c.WebRTC.TLS = &config.TLS{CertificateFile: "/test/cert", PrivateKeyFile: "/test/key"}
			*c = config.Defaults(*c)
			s.Listeners = []Listener{{"tcp", c.Network.BindAddress, 8089}}
		}, true, "listener-tcp"},
		{"WSS-disabled", func(s *Snapshot, c *config.Config, r *Requirements) {
			s.Listeners = []Listener{{"tcp", c.Network.BindAddress, 8089}}
		}, false, ""},
		{"management", func(s *Snapshot, c *config.Config, r *Requirements) {
			s.Listeners = []Listener{{"tcp", netip.MustParseAddr("127.0.0.1"), 3306}}
		}, true, "listener-tcp"},
		{"services", func(s *Snapshot, c *config.Config, r *Requirements) {
			s.Services = []string{"asterisk.service loaded active running"}
		}, true, "services"},
		{"packages", func(s *Snapshot, c *config.Config, r *Requirements) { s.Packages = []string{"mariadb-server"} }, true, "packages"},
		{"ownership", func(s *Snapshot, c *config.Config, r *Requirements) { s.OwnedPaths = []string{"/etc/asterisk"} }, true, "ownership"},
		{"SELinux", func(s *Snapshot, c *config.Config, r *Requirements) { s.SELinux = "Permissive" }, true, "selinux"},
		{"time", func(s *Snapshot, c *config.Config, r *Requirements) { s.TimeSync = "no" }, true, "time"},
		{"unassigned-bind", func(s *Snapshot, c *config.Config, r *Requirements) { s.Addresses = nil }, true, "network"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, c, r := fixtureHost(), fixtureConfig(), fixtureBudget()
			tc.change(&s, &c, &r)
			f, e := Assess(c, s, r)
			if e != nil {
				t.Fatal(e)
			}
			if Eligible(f) == tc.blocked {
				t.Fatalf("eligibility: %+v", f)
			}
			if tc.want != "" {
				found := false
				for _, x := range f {
					if strings.HasPrefix(x.ID, tc.want) && x.Blocking {
						found = true
					}
				}
				if !found {
					t.Fatalf("missing %s", tc.want)
				}
			}
		})
	}
	for _, probe := range RequiredProbes() {
		t.Run("unknown-"+probe, func(t *testing.T) {
			s := fixtureHost()
			s.Known[probe] = false
			f, _ := Assess(fixtureConfig(), s, fixtureBudget())
			if Eligible(f) {
				t.Fatal("unverified passed")
			}
			for _, x := range f {
				if x.ID == probe && x.Status == NotVerified {
					return
				}
			}
			t.Fatal("missing unverified finding")
		})
	}
}

type fakeSource struct {
	files    map[string]string
	commands map[Operation]string
	fail     string
	trace    []string
}

func fixtureSource() *fakeSource {
	return &fakeSource{files: map[string]string{
		"/etc/machine-id": strings.Repeat("a", 32), "/etc/os-release": "ID=almalinux\nVERSION_ID=\"10.2\"", "/proc/sys/kernel/osrelease": "fixture", "/proc/cpuinfo": "processor : 0\nflags : sse2 avx\n", "/proc/meminfo": "MemTotal: 8192 kB\nMemAvailable: 4096 kB\n", "/sys/fs/selinux/enforce": "1", "/etc/resolv.conf": "nameserver 192.0.2.53"}, commands: map[Operation]string{Network: `[{"addr_info":[{"local":"192.0.2.10"}]}]`, Sockets: "", Packages: "base-package\nmariadb-connector-c\n", Services: "", Firewall: `{"nftables":[]}`, Clock: "yes"}}
}
func (f *fakeSource) Architecture() (string, error) {
	f.trace = append(f.trace, "uname")
	return "amd64", nil
}
func (f *fakeSource) Processes() ([]string, error) {
	f.trace = append(f.trace, "read process comm")
	if f.fail == "processes" {
		return nil, errors.New("SENSITIVE")
	}
	return strings.Fields(f.files["process-names"]), nil
}
func (f *fakeSource) Read(p string) ([]byte, error) {
	f.trace = append(f.trace, "read "+p)
	v, ok := f.files[p]
	if !ok || f.fail == p {
		return nil, errors.New("SENSITIVE fixture failure")
	}
	return []byte(v), nil
}
func (f *fakeSource) Disk(p string) (Disk, error) {
	f.trace = append(f.trace, "statfs "+p)
	if f.fail == p {
		return Disk{}, errors.New("sensitive")
	}
	return Disk{p, 100000, 90000}, nil
}
func (f *fakeSource) Exists(p string) (bool, error) {
	f.trace = append(f.trace, "lstat "+p)
	return p == "/sys/fs/selinux", nil
}
func (f *fakeSource) Run(ctx context.Context, op Operation) ([]byte, error) {
	f.trace = append(f.trace, "run "+string(op))
	if _, _, e := command(op); e != nil {
		return nil, e
	}
	if f.fail == string(op) {
		return nil, errors.New("SENSITIVE output")
	}
	return []byte(f.commands[op]), nil
}
func TestCollectorReadOnlyTrace(t *testing.T) {
	f := fixtureSource()
	s := Collect(context.Background(), f)
	findings, e := Assess(fixtureConfig(), s, fixtureBudget())
	if e != nil || !Eligible(findings) {
		t.Fatalf("%v %s", e, Human(findings))
	}
	t.Log(strings.Join(f.trace, "\n"))
	allowedReads := map[string]bool{"uname": true, "read process comm": true}
	for p := range f.files {
		allowedReads["read "+p] = true
	}
	for _, p := range []string{"/", "/usr", "/var"} {
		allowedReads["statfs "+p] = true
	}
	for _, p := range []string{"/etc/asterisk", "/etc/astguiclient.conf", "/usr/share/astguiclient", "/srv/www/htdocs/vicidial", "/var/lib/mysql", "/var/lib/almadial"} {
		allowedReads["lstat "+p] = true
	}
	for _, op := range []Operation{Network, Sockets, Packages, Services, Firewall, Clock} {
		allowedReads["run "+string(op)] = true
	}
	for _, entry := range f.trace {
		if !allowedReads[entry] {
			t.Fatalf("unexpected operation: %s", entry)
		}
	}
	for _, op := range []Operation{"install", "start", "nft add", "sh", "download"} {
		if _, _, e := command(op); e == nil {
			t.Fatal("mutation permitted")
		}
		if _, e := (Linux{}).Run(context.Background(), op); e == nil {
			t.Fatal("production allowlist bypassed")
		}
	}
}
func TestCollectorFailures(t *testing.T) {
	cases := []struct{ name, target, probe, value string }{
		{"os-read", "/etc/os-release", "os", ""}, {"CPU-read", "/proc/cpuinfo", "cpu", ""}, {"memory-read", "/proc/meminfo", "memory", ""}, {"disk-stat", "/var", "disk", ""}, {"command", "sockets", "listeners", ""}, {"firewall-permission", "firewall", "firewall", ""}, {"packages", "packages", "packages", ""}, {"services", "services", "services", ""}, {"processes", "processes", "processes", ""}, {"time", "clock", "time", ""}, {"network", "network", "network", ""},
		{"socket-format", "", "listeners", "garbage"}, {"clock-format", "", "time", "garbage"}, {"service-format", "", "services", "garbage"}, {"network-format", "", "network", "garbage"}, {"firewall-format", "", "firewall", "garbage"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := fixtureSource()
			f.fail = tc.target
			if tc.value != "" {
				ops := map[string]Operation{"listeners": Sockets, "time": Clock, "services": Services, "network": Network, "firewall": Firewall}
				f.commands[ops[tc.probe]] = tc.value
			}
			s := Collect(context.Background(), f)
			if s.Known[tc.probe] {
				t.Fatal("failed/malformed probe treated verified")
			}
			fs, _ := Assess(fixtureConfig(), s, fixtureBudget())
			if Eligible(fs) {
				t.Fatal("failure passed")
			}
			if strings.Contains(Human(fs), "SENSITIVE") {
				t.Fatal("raw command error leaked")
			}
		})
	}
}
func TestSocketParsing(t *testing.T) {
	for _, tc := range []struct {
		raw string
		ok  bool
	}{
		{"udp UNCONN 0 0 0.0.0.0:5060 0.0.0.0:*", true}, {"tcp LISTEN 0 128 [::]:8089 [::]:*", true}, {"tcp LISTEN 0 128 *:443 *:*", true}, {"", true}, {"junk", false}, {"tcp LISTEN 0 128 1.2.3.4:bad *:*", false}, {"tcp LISTEN 0 128 [fe80::1%eth0]:80 *:*", false},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			_, e := parseSockets(tc.raw)
			if (e == nil) != tc.ok {
				t.Fatalf("%v", e)
			}
		})
	}
}
func TestFingerprint(t *testing.T) {
	s, c, r := fixtureHost(), fixtureConfig(), fixtureBudget()
	h, _ := Fingerprint(c, s, r)
	s.AvailableMemory--
	s.Disks[0].Available--
	s.Kernel = "patch variation"
	s.Listeners = []Listener{{"tcp", netip.IPv6Unspecified(), 9999}}
	s.CPUFlags = append(s.CPUFlags, "extra")
	h2, _ := Fingerprint(c, s, r)
	if h != h2 {
		t.Fatal("irrelevant variation invalidated")
	}
	s.HostID = strings.Repeat("b", 32)
	h3, _ := Fingerprint(c, s, r)
	if h == h3 {
		t.Fatal("host identity not bound")
	}
	s = fixtureHost()
	s.Services = []string{"asterisk"}
	h4, _ := Fingerprint(c, s, r)
	if reflect.DeepEqual(h, h4) {
		t.Fatal("collision not bound")
	}
}

func TestExactCommandAllowlist(t *testing.T) {
	expected := map[Operation][]string{
		Network: {"/usr/sbin/ip", "-j", "address", "show"}, Sockets: {"/usr/sbin/ss", "-H", "-lntu", "-n"}, Packages: {"/usr/bin/rpm", "-qa", "--qf", "%{NAME}\n"}, Services: {"/usr/bin/systemctl", "list-units", "--all", "--type=service", "--no-legend", "--no-pager", "--plain"}, Firewall: {"/usr/sbin/nft", "--json", "--stateless", "list", "ruleset"}, Clock: {"/usr/bin/timedatectl", "show", "--property=NTPSynchronized", "--value"},
	}
	for op, want := range expected {
		path, args, err := command(op)
		if err != nil || !reflect.DeepEqual(append([]string{path}, args...), want) {
			t.Fatalf("unexpected argv for %s", op)
		}
	}
}
func TestMalformedFileProbes(t *testing.T) {
	cases := []struct{ path, value, probe string }{
		{"/etc/machine-id", strings.Repeat("z", 32), "identity"},
		{"/proc/meminfo", "MemTotal: 8192 kB\nMemAvailable: invalid kB", "memory"},
		{"/proc/cpuinfo", "processor : 0\nflags : sse2\nprocessor : 1", "cpu"},
		{"/sys/fs/selinux/enforce", "unexpected", "selinux"},
	}
	for _, tc := range cases {
		t.Run(tc.probe, func(t *testing.T) {
			f := fixtureSource()
			f.files[tc.path] = tc.value
			s := Collect(context.Background(), f)
			if s.Known[tc.probe] {
				t.Fatal("malformed file marked known")
			}
		})
	}
}
func TestInformationalFindingsAndFailures(t *testing.T) {
	s := fixtureHost()
	c := fixtureConfig()
	c.Network.Hostname = "example.invalid"
	f, e := Assess(c, s, fixtureBudget())
	if e != nil || !Eligible(f) {
		t.Fatal("optional DNS/firewall warning should not block")
	}
	var warnings int
	for _, x := range f {
		if x.Status == Warning {
			warnings++
		}
	}
	if warnings != 2 {
		t.Fatal("missing warnings")
	}
	fake := fixtureSource()
	fake.fail = "firewall"
	s = Collect(context.Background(), fake)
	if len(s.Failures) != 1 || s.Failures[0].Probe != "firewall" || s.Failures[0].ExitCode != -1 {
		t.Fatal("missing sanitized probe failure")
	}
	if Eligible(nil) {
		t.Fatal("empty report must not advertise readiness")
	}
	if packageConflict("mariadb-connector-c") {
		t.Fatal("client library is not a server collision")
	}
}

func TestListenerProtocolAndMalformedFacts(t *testing.T) {
	s := fixtureHost()
	c := fixtureConfig()
	s.Listeners = []Listener{{"tcp", c.Network.BindAddress, 5060}}
	fs, e := Assess(c, s, fixtureBudget())
	if e != nil || !Eligible(fs) {
		t.Fatal("TCP socket must not collide with configured UDP SIP")
	}
	s.Listeners = []Listener{{"unknown", c.Network.BindAddress, 5060}}
	fs, _ = Assess(c, s, fixtureBudget())
	if Eligible(fs) {
		t.Fatal("malformed socket should block")
	}
	for _, f := range fs {
		if f.ID == "listeners" && f.Status == Pass {
			t.Fatal("contradictory healthy listener finding")
		}
	}
}
func TestCollectorDeterminism(t *testing.T) {
	a := Collect(context.Background(), fixtureSource())
	b := Collect(context.Background(), fixtureSource())
	if !reflect.DeepEqual(a, b) {
		t.Fatal("collector ordering unstable")
	}
}

func TestProcessAndFirewallObservations(t *testing.T) {
	f := fixtureSource()
	f.files["process-names"] = "AST_update.pl perl httpd mariadbd"
	f.commands[Services] = "firewalld.service loaded active running firewall"
	s := Collect(context.Background(), f)
	if len(s.Processes) != 3 || len(s.FirewallServices) != 1 {
		t.Fatal("missing process/firewall observations")
	}
	fs, _ := Assess(fixtureConfig(), s, fixtureBudget())
	if Eligible(fs) {
		t.Fatal("existing processes passed")
	}
	f = fixtureSource()
	s = Collect(context.Background(), f)
	if s.Firewall != "empty nftables ruleset" {
		t.Fatal("empty JSON ruleset misclassified")
	}
}
