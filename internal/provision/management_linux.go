//go:build linux

package provision

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

type IPv4Interface struct{ Address, Interface string }

// A subprocess cannot make discovery retain unbounded output.
type managementBuffer struct{ bytes.Buffer }

func (b *managementBuffer) Write(p []byte) (int, error) {
	if len(p) > 1<<20-b.Len() {
		return 0, errors.New("management observation exceeds 1 MiB")
	}
	return b.Buffer.Write(p)
}

func managementOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	var out managementBuffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("read-only management discovery failed for %s; inspect the host configuration before installing", name)
	}
	return out.Bytes(), nil
}

func parseIPv4Interfaces(raw []byte) ([]IPv4Interface, error) {
	var links []struct {
		Ifname, Operstate string
		Flags             []string
		AddrInfo          []struct{ Family, Local, Scope string } `json:"addr_info"`
	}
	if err := json.Unmarshal(raw, &links); err != nil {
		return nil, fmt.Errorf("cannot decode host IPv4/interface observations: %w", err)
	}
	var pairs []IPv4Interface
	seen := map[IPv4Interface]bool{}
	for _, l := range links {
		if !strings.Contains(" "+strings.Join(l.Flags, " ")+" ", " UP ") {
			continue
		}
		for _, a := range l.AddrInfo {
			ip, err := netip.ParseAddr(a.Local)
			if err != nil || a.Family != "inet" || a.Scope != "global" || !ip.Is4() || !ip.IsGlobalUnicast() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}
			p := IPv4Interface{ip.String(), l.Ifname}
			if !seen[p] {
				pairs = append(pairs, p)
				seen[p] = true
			}
		}
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].Interface == pairs[j].Interface {
			return pairs[i].Address < pairs[j].Address
		}
		return pairs[i].Interface < pairs[j].Interface
	})
	if len(pairs) == 0 {
		return nil, fmt.Errorf("no active global IPv4/interface pair found; configure a directly attached server IPv4 before installing")
	}
	return pairs, nil
}

func DiscoverIPv4Interfaces(ctx context.Context) ([]IPv4Interface, error) {
	b, err := managementOutput(ctx, "/usr/sbin/ip", "-j", "address", "show")
	if err != nil {
		return nil, err
	}
	return parseIPv4Interfaces(b)
}

func parseManagementSSH(effective, listeners, address string) (uint16, error) {
	fail := func() (uint16, error) {
		return 0, fmt.Errorf("unsupported or ambiguous SSH management configuration: require one effective sshd port and matching sshd-owned IPv4 listener on the selected address or wildcard; inspect sshd -T and ss -lntp (including socket activation and Match/ListenAddress settings); AlmaDial will not reconfigure SSH")
	}
	ports := map[uint16]bool{}
	var configured []netip.AddrPort
	for _, line := range strings.Split(effective, "\n") {
		f := strings.Fields(line)
		if len(f) != 2 {
			continue
		}
		switch f[0] {
		case "port":
			n, e := strconv.ParseUint(f[1], 10, 16)
			if e != nil || n == 0 {
				return fail()
			}
			ports[uint16(n)] = true
		case "listenaddress":
			ap, e := netip.ParseAddrPort(f[1])
			if e != nil {
				return fail()
			}
			configured = append(configured, ap)
		}
	}
	if len(ports) != 1 || len(configured) == 0 {
		return fail()
	}
	var port uint16
	for p := range ports {
		port = p
	}
	ip, e := netip.ParseAddr(address)
	if e != nil || !ip.Is4() {
		return fail()
	}
	for _, ap := range configured {
		if ap.Port() != port || (!ap.Addr().IsUnspecified() && ap.Addr().Unmap() != ip) {
			return fail()
		}
	}
	matched := false
	for _, line := range strings.Split(listeners, "\n") {
		if !strings.Contains(line, `"sshd"`) {
			continue
		}
		f := strings.Fields(line)
		if len(f) < 6 {
			return fail()
		}
		ap, e := netip.ParseAddrPort(f[3])
		if e != nil || ap.Port() != port {
			return fail()
		}
		if !ap.Addr().IsUnspecified() && ap.Addr().Unmap() != ip {
			return fail()
		}
		known := false
		for _, c := range configured {
			if c == ap {
				known = true
			}
		}
		if !known {
			return fail()
		}
		if ap.Addr().Is4() && (ap.Addr().IsUnspecified() || ap.Addr() == ip) {
			matched = true
		}
	}
	if !matched {
		return fail()
	}
	return port, nil
}

func DiscoverManagementSSH(ctx context.Context, address string) (uint16, error) {
	e, e1 := managementOutput(ctx, "/usr/sbin/sshd", "-T")
	if e1 != nil {
		return 0, e1
	}
	l, e1 := managementOutput(ctx, "/usr/sbin/ss", "-H", "-lntp")
	if e1 != nil {
		return 0, e1
	}
	return parseManagementSSH(string(e), string(l), address)
}

// VerifyManagement is read-only and runs before standard provisioning, including
// resume. DNS equality enforces the supported direct-address contract; it does
// not establish internet reachability or media traversal.
func VerifyManagement(ctx context.Context, r NetworkRequest, hostname string) error {
	pairs, err := DiscoverIPv4Interfaces(ctx)
	if err != nil {
		return err
	}
	if err = validateManagementPair(pairs, r); err != nil {
		return err
	}
	port, err := DiscoverManagementSSH(ctx, r.ManagementAddress)
	if err != nil {
		return err
	}
	if port != r.SSHPort {
		return fmt.Errorf("effective SSH port changed: plan requires %d, observed %d; preserve the saved plan and reconcile management access before resuming", r.SSHPort, port)
	}
	if r.Mode == "remote" {
		ips, e := net.DefaultResolver.LookupNetIP(ctx, "ip4", hostname)
		if e != nil || len(ips) == 0 {
			return fmt.Errorf("FQDN must resolve to the directly assigned IPv4 before provisioning")
		}
		return validateManagementDNS(ips, r.ManagementAddress)
	}
	return nil
}

func validateManagementPair(pairs []IPv4Interface, r NetworkRequest) error {
	found := false
	for _, p := range pairs {
		if p.Address == r.ManagementAddress && p.Interface == r.Interface {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("selected IPv4 %s is not assigned to active interface %s; public-to-private NAT mappings are unsupported", r.ManagementAddress, r.Interface)
	}
	return nil
}

func validateManagementDNS(ips []netip.Addr, address string) error {
	if len(ips) == 0 {
		return fmt.Errorf("FQDN has no IPv4 address; configure direct-address DNS before provisioning")
	}
	for _, ip := range ips {
		if ip.Unmap().String() != address {
			return fmt.Errorf("FQDN IPv4 differs from selected local address; multiple-address DNS and public-to-private NAT mappings are unsupported")
		}
	}
	return nil
}
