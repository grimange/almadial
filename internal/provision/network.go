package provision

import (
	"errors"
	"fmt"
	"net/netip"
	"regexp"
)

var errNetwork = errors.New("S09 isolated network contract failed; preserve management access and inspect the named firewall postcondition")

// NetworkRequest deliberately supports only the approved isolated topology.
// It does not publish an external address, open telephony ports or change binds.
type NetworkRequest struct {
	Mode, Interface, ManagementAddress string
	SSHPort                            uint16
	RTPStart, RTPEnd                   uint16
}

var networkInterface = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.:-]{0,14}$`)

func networkRequestValid(r NetworkRequest) bool {
	ip, e := netip.ParseAddr(r.ManagementAddress)
	base := e == nil && ip.Is4() && ip.IsGlobalUnicast() && !ip.IsLoopback() && networkInterface.MatchString(r.Interface) && r.Interface != "lo" && r.SSHPort > 0
	if r.Mode == "isolated" {
		return base && ip.IsPrivate() && r.RTPStart == 0 && r.RTPEnd == 0
	}
	return base && r.Mode == "remote" && r.RTPStart >= 1024 && r.RTPStart%2 == 0 && r.RTPEnd > r.RTPStart && r.SSHPort != 443

}

// NetworkPolicy uses a native firewalld policy, ahead of zones, without
// reassigning interfaces or changing operator zones. ANY excludes HOST.
func NetworkPolicy(r NetworkRequest) (string, error) {
	if !networkRequestValid(r) {
		return "", errNetwork
	}
	extra := ""
	if r.Mode == "remote" {
		extra = fmt.Sprintf("  <port port=\"443\" protocol=\"tcp\"/>\n  <port port=\"%d-%d\" protocol=\"udp\"/>\n", r.RTPStart, r.RTPEnd)
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<!-- AlmaDial S09: isolated host; SSH and essential IP control traffic only. -->
<policy target="DROP" priority="-20000">
  <short>AlmaDial isolated host</short>
  <ingress-zone name="ANY"/>
  <egress-zone name="HOST"/>
  <port port="%d" protocol="tcp"/>
%s  <protocol value="icmp"/>
  <protocol value="ipv6-icmp"/>
</policy>
`, r.SSHPort, extra), nil
}
