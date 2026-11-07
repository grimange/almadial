// Package preflight separates read-only host observations from installation policy.
package preflight

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/netip"
	"sort"
	"strings"

	"github.com/grimange/almadial/internal/config"
)

type Status string

const (
	Pass        Status = "PASS"
	Warning     Status = "WARNING"
	Fail        Status = "FAIL"
	NotVerified Status = "NOT_VERIFIED"
)

type Finding struct {
	ID       string `json:"id"`
	Status   Status `json:"status"`
	Blocking bool   `json:"blocking"`
	Message  string `json:"message"`
}
type Listener struct {
	Protocol string
	Address  netip.Addr
	Port     int
}
type Disk struct {
	Path             string
	Total, Available uint64
}

// Missing/failed probes are represented by Known, never by zero-valued success.
// Raw subprocess diagnostics are deliberately not retained (they may be sensitive).
type ProbeFailure struct {
	Probe    string
	ExitCode int
}
type Snapshot struct {
	Processes                                 []string
	Failures                                  []ProbeFailure
	FirewallServices                          []string
	HostID, OS, Release, Kernel, Architecture string
	CPUs                                      int
	CPUFlags                                  []string
	TotalMemory, AvailableMemory              uint64
	Disks                                     []Disk
	Listeners                                 []Listener
	Addresses                                 []netip.Addr
	DNSConfigured                             bool
	Packages, Services, OwnedPaths            []string
	SELinux, Firewall, TimeSync               string
	FirewallSHA256                            string
	Known                                     map[string]bool
}

// Requirements is an explicit, reviewed installation budget, not a capacity
// guarantee. M0 lab sizing is not a universal minimum. No implicit defaults exist.
type Requirements struct {
	Profile                                              string
	MinCPUs                                              int
	CPUFlags                                             []string
	MinTotalMemory, MinAvailableMemory, MinDiskAvailable uint64
}

func RequiredProbes() []string {
	return []string{"identity", "os", "network", "architecture", "cpu", "memory", "disk", "listeners", "packages", "services", "processes", "ownership", "selinux", "firewall", "time"}
}

func Assess(c config.Config, s Snapshot, r Requirements) ([]Finding, error) {
	c = config.Defaults(c)
	if err := config.Validate(c); err != nil {
		return nil, err
	}
	var out []Finding
	add := func(id string, status Status, block bool, message string) {
		out = append(out, Finding{id, status, block, message})
	}
	for _, id := range RequiredProbes() {
		if !s.Known[id] {
			add(id, NotVerified, true, "read-only probe incomplete; establish state before installation")
		}
	}
	check := func(id string, ok bool, message string) {
		if ok {
			add(id, Pass, false, "verified")
		} else {
			add(id, Fail, true, message)
		}
	}
	if s.Known["identity"] {
		check("identity", s.HostID != "", "host identity unavailable")
	}
	if s.Known["os"] {
		check("os", s.OS == "almalinux" && s.Release == "10.2", "only AlmaLinux 10.2 is qualified; other releases require compatibility review")
	}
	if s.Known["network"] {
		ok := false
		for _, a := range s.Addresses {
			if a == c.Network.BindAddress || (c.Network.BindAddress.IsUnspecified() && a.IsGlobalUnicast() && a.Is4() == c.Network.BindAddress.Is4()) {
				ok = true
			}
		}
		check("network", ok, "configured bind address is not assigned locally")
		if c.Network.Hostname != "" && !s.DNSConfigured {
			add("dns", Warning, false, "hostname supplied but resolver configuration not observed; name resolution requires future host verification")
		}
	}
	if s.Known["architecture"] {
		check("architecture", s.Architecture == "amd64", "selected stack requires x86_64")
	}
	if r.Profile == "" || r.MinCPUs < 1 || len(r.CPUFlags) == 0 || r.MinTotalMemory == 0 || r.MinAvailableMemory == 0 || r.MinDiskAvailable == 0 {
		add("budget", NotVerified, true, "reviewed CPU ISA and installation resource budget required")
	} else {
		if s.Known["cpu"] {
			ok := s.CPUs >= r.MinCPUs
			for _, f := range r.CPUFlags {
				if !contains(s.CPUFlags, f) {
					ok = false
				}
			}
			check("cpu-budget", ok, "CPU count or required instruction capability missing")
		}
		if s.Known["memory"] {
			check("memory-budget", s.TotalMemory >= r.MinTotalMemory && s.AvailableMemory >= r.MinAvailableMemory, "insufficient total or currently available memory")
		}
		if s.Known["disk"] {
			ok := len(s.Disks) > 0
			for _, d := range s.Disks {
				if d.Available < r.MinDiskAvailable {
					ok = false
				}
			}
			check("disk-budget", ok, "insufficient available space on installation filesystems")
		}
	}
	if s.Known["packages"] {
		check("packages", len(s.Packages) == 0, "existing telephony/database/web packages require ownership review; fresh installation only")
	}
	if s.Known["services"] {
		check("services", len(s.Services) == 0, "existing relevant service units require ownership review")
	}
	if s.Known["processes"] {
		check("processes", len(s.Processes) == 0, "existing database, telephony or web processes require ownership review")
	}
	if s.Known["ownership"] {
		check("ownership", len(s.OwnedPaths) == 0, "existing application paths require ownership review; do not overwrite")
	}
	if s.Known["selinux"] {
		check("selinux", s.SELinux == "Enforcing", "SELinux must be Enforcing; preflight will not change it")
	}
	if s.Known["firewall"] {
		add("firewall", Warning, false, "ruleset observed; future bounded network policy requires review, no remote reachability claim")
	}
	if s.Known["time"] {
		check("time", s.TimeSync == "yes", "clock synchronization not established")
	}
	if s.Known["listeners"] {
		conflict := false
		valid := true
		for _, l := range s.Listeners {
			if !l.Address.IsValid() || l.Port < 1 || l.Port > 65535 || (l.Protocol != "tcp" && l.Protocol != "udp") {
				valid = false
				add("listeners-format", NotVerified, true, "unrecognized socket observation")
				continue
			}
			hit := false
			// HTTP/HTTPS use server-wide listeners; management ports remain loopback-only.
			if l.Protocol == "tcp" {
				hit = l.Port == 80 || l.Port == 443 || l.Port == 3306 || l.Port == 5038 || l.Port == 4577
				if c.WebRTC.Enabled && l.Port == *c.WebRTC.WSSPort && overlaps(l.Address, c.Network.BindAddress) {
					hit = true
				}
			}
			if l.Protocol == "udp" && overlaps(l.Address, c.Network.BindAddress) {
				hit = l.Port == *c.SIP.Port || (l.Port >= *c.RTP.Start && l.Port <= *c.RTP.End)
			}
			if hit {
				conflict = true
				add(fmt.Sprintf("listener-%s-%05d-%s", l.Protocol, l.Port, l.Address), Fail, true, "required listener or RTP allocation overlaps existing socket; identify owner or change configuration")
			}
		}
		if !conflict && valid {
			add("listeners", Pass, false, "no current required socket collisions; recheck at apply time, RTP reservation is not proven")
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ID == out[j].ID {
			return out[i].Message < out[j].Message
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}
func contains(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}
func overlaps(a, b netip.Addr) bool {
	return a.IsUnspecified() || b.IsUnspecified() || a.Unmap() == b.Unmap()
}
func Eligible(fs []Finding) bool {
	if len(fs) == 0 {
		return false
	}
	for _, f := range fs {
		if f.Blocking {
			return false
		}
	}
	return true
}
func hash(v any) string {
	b, _ := json.Marshal(v)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// Fingerprint excludes volatile counters, available bytes above the budget,
// process IDs and irrelevant sockets. It includes eligibility and relevant facts.
func Fingerprint(c config.Config, s Snapshot, r Requirements) (string, error) {
	fs, err := Assess(c, s, r)
	if err != nil {
		return "", err
	}
	flags := append([]string(nil), r.CPUFlags...)
	sort.Strings(flags)
	r.CPUFlags = flags
	addresses := []string{}
	for _, a := range s.Addresses {
		if a.IsGlobalUnicast() {
			addresses = append(addresses, a.String())
		}
	}
	sort.Strings(addresses)
	disks := make([]string, 0, len(s.Disks))
	for _, d := range s.Disks {
		disks = append(disks, d.Path)
	}
	sort.Strings(disks)
	canonical := func(xs []string) []string { v := append([]string(nil), xs...); sort.Strings(v); return v }
	return hash(struct {
		Host, OS, Release, Arch, Security, Clock, Firewall                              string
		Flags, Packages, Services, Paths, Disks, FirewallServices, Addresses, Processes []string
		Budget                                                                          Requirements
		Findings                                                                        []Finding
	}{s.HostID, s.OS, s.Release, s.Architecture, s.SELinux, s.TimeSync, s.FirewallSHA256, flags, canonical(s.Packages), canonical(s.Services), canonical(s.OwnedPaths), disks, canonical(s.FirewallServices), addresses, canonical(s.Processes), r, fs}), nil
}
func Human(findings []Finding) string {
	var b strings.Builder
	for _, f := range findings {
		fmt.Fprintf(&b, "%s: %s — %s\n", f.ID, f.Status, f.Message)
	}
	if Eligible(findings) {
		b.WriteString("Installation eligibility: ELIGIBLE\n")
	} else {
		b.WriteString("Installation eligibility: BLOCKED\n")
	}
	b.WriteString("Host changes requested by preflight: NONE\n")
	return b.String()
}
