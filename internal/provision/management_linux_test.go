//go:build linux

package provision

import (
	"fmt"
	"net/netip"
	"strings"
	"testing"
)

func TestManagementSSHDiscovery(t *testing.T) {
	for _, tc := range []struct {
		name, effective, observed string
		port                      uint16
	}{
		{"22", "port 22\nlistenaddress 0.0.0.0:22\nlistenaddress [::]:22", `LISTEN 0 128 0.0.0.0:22 0.0.0.0:* users:(("sshd",pid=1,fd=3))`, 22},
		{"2222", "port 2222\nlistenaddress 0.0.0.0:2222", `LISTEN 0 128 0.0.0.0:2222 0.0.0.0:* users:(("sshd",pid=1,fd=3))`, 2222},
		{"multiple ports", "port 22\nport 2222\nlistenaddress 0.0.0.0:22", `LISTEN 0 128 0.0.0.0:22 0.0.0.0:* users:(("sshd",pid=1,fd=3))`, 0},
		{"listener differs", "port 22\nlistenaddress 0.0.0.0:22", `LISTEN 0 128 0.0.0.0:2222 0.0.0.0:* users:(("sshd",pid=1,fd=3))`, 0},
		{"socket owned", "port 22\nlistenaddress 0.0.0.0:22", `LISTEN 0 128 0.0.0.0:22 0.0.0.0:* users:(("systemd",pid=1,fd=3))`, 0},
		{"wrong address", "port 22\nlistenaddress 10.2.3.9:22", `LISTEN 0 128 10.2.3.9:22 0.0.0.0:* users:(("sshd",pid=1,fd=3))`, 0},
		{"ipv6 only", "port 22\nlistenaddress [::]:22", `LISTEN 0 128 [::]:22 [::]:* users:(("sshd",pid=1,fd=3))`, 0},
		{"ambiguous listener", "port 22\nlistenaddress 0.0.0.0:22", `LISTEN 0 128 0.0.0.0:22 0.0.0.0:* users:(("sshd",pid=1,fd=3))` + "\n" + `LISTEN 0 128 10.2.3.4:22 0.0.0.0:* users:(("sshd",pid=1,fd=4))`, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			port, err := parseManagementSSH(tc.effective, tc.observed, "10.2.3.4")
			if port != tc.port || (err == nil) != (tc.port != 0) {
				t.Fatal(port, err)
			}
			if err != nil && !strings.Contains(err.Error(), "will not reconfigure SSH") {
				t.Fatal(err)
			}
		})
	}
}
func TestManagementIPv4InterfacePairs(t *testing.T) {
	raw := `[{"ifname":"lo","flags":["UP"],"addr_info":[{"family":"inet","local":"127.0.0.1","scope":"host"}]},{"ifname":"ens3","flags":["UP"],"addr_info":[{"family":"inet","local":"10.2.3.4","scope":"global"},{"family":"inet6","local":"2001:db8::1","scope":"global"}]},{"ifname":"enp1s0","flags":["UP"],"addr_info":[{"family":"inet","local":"192.0.2.2","scope":"global"}]},{"ifname":"eth0","flags":[],"addr_info":[{"family":"inet","local":"10.9.9.9","scope":"global"}]}]`
	pairs, err := parseIPv4Interfaces([]byte(raw))
	if err != nil || fmt.Sprint(pairs) != "[{192.0.2.2 enp1s0} {10.2.3.4 ens3}]" {
		t.Fatal(pairs, err)
	}
	for _, bad := range []string{"invalid", "[]", `[{"ifname":"lo","flags":["UP"],"addr_info":[{"family":"inet","local":"169.254.1.2","scope":"global"}]}]`} {
		if _, err := parseIPv4Interfaces([]byte(bad)); err == nil {
			t.Fatal(bad)
		}
	}
}

func TestManagementPairAndDirectDNSContract(t *testing.T) {
	pairs := []IPv4Interface{{"10.2.3.4", "ens3"}, {"10.2.3.5", "enp1s0"}}
	r := NetworkRequest{ManagementAddress: "10.2.3.4", Interface: "ens3"}
	if err := validateManagementPair(pairs, r); err != nil {
		t.Fatal(err)
	}
	r.Interface = "eth0"
	if err := validateManagementPair(pairs, r); err == nil || !strings.Contains(err.Error(), "not assigned") {
		t.Fatal(err)
	}
	r.Interface = "ens3"
	r.ManagementAddress = "203.0.113.4"
	if err := validateManagementPair(pairs, r); err == nil || !strings.Contains(err.Error(), "NAT") {
		t.Fatal(err)
	}
	if err := validateManagementDNS([]netip.Addr{netip.MustParseAddr("10.2.3.4")}, "10.2.3.4"); err != nil {
		t.Fatal(err)
	}
	for _, ips := range [][]netip.Addr{nil, {netip.MustParseAddr("203.0.113.4")}, {netip.MustParseAddr("10.2.3.4"), netip.MustParseAddr("10.2.3.5")}} {
		if err := validateManagementDNS(ips, "10.2.3.4"); err == nil {
			t.Fatal("unsupported DNS/NAT accepted")
		}
	}
}
