package preflight

import (
	"errors"
	"net"
	"net/netip"
	"strconv"
	"strings"
)

func parseSockets(output string) ([]Listener, error) {
	var result []Listener
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		f := strings.Fields(line)
		if len(f) < 6 {
			return nil, errors.New("malformed socket observation")
		}
		protocol := f[0]
		if protocol != "tcp" && protocol != "udp" {
			return nil, errors.New("unsupported socket protocol")
		}
		host, port, err := net.SplitHostPort(f[4])
		if err != nil {
			return nil, errors.New("malformed socket endpoint")
		}
		// ss '*' does not establish family; IPv6 wildcard conservatively overlaps both.
		if host == "*" {
			host = "::"
		}
		addr, err := netip.ParseAddr(host)
		if err != nil || addr.Zone() != "" {
			return nil, errors.New("unrecognized socket address")
		}
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return nil, errors.New("invalid socket port")
		}
		result = append(result, Listener{protocol, addr, n})
	}
	return result, nil
}
