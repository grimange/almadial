package provision

import (
	"encoding/xml"
	"strings"
	"testing"
)

func TestNetworkIsolatedPolicy(t *testing.T) {
	r := NetworkRequest{"isolated", "eth0", "10.160.90.233", 2222, 0, 0}
	a, e := NetworkPolicy(r)
	if e != nil {
		t.Fatal(e)
	}
	b, _ := NetworkPolicy(r)
	if a != b {
		t.Fatal("nondeterministic policy")
	}
	var p struct {
		Target   string `xml:"target,attr"`
		Priority int    `xml:"priority,attr"`
		Ports    []struct {
			Port     string `xml:"port,attr"`
			Protocol string `xml:"protocol,attr"`
		} `xml:"port"`
		In struct {
			Name string `xml:"name,attr"`
		} `xml:"ingress-zone"`
		Out struct {
			Name string `xml:"name,attr"`
		} `xml:"egress-zone"`
	}
	if xml.Unmarshal([]byte(a), &p) != nil || p.Target != "DROP" || p.Priority != -20000 || p.In.Name != "ANY" || p.Out.Name != "HOST" || len(p.Ports) != 1 || p.Ports[0].Port != "2222" || p.Ports[0].Protocol != "tcp" {
		t.Fatal(a)
	}
	for _, port := range []string{"443", "8089", "3306", "5038", "4577", "5060", "10000"} {
		if strings.Contains(a, `port="`+port+`"`) {
			t.Fatal("unapproved grant")
		}
	}
}
func TestNetworkRejectUnsupportedTopology(t *testing.T) {
	for _, r := range []NetworkRequest{{"public", "eth0", "10.1.2.3", 22, 0, 0}, {"isolated", "eth0", "8.8.8.8", 22, 0, 0}, {"isolated", "lo", "127.0.0.1", 22, 0, 0}, {"isolated", "eth0\n", "10.1.2.3", 22, 0, 0}, {"isolated", "eth0", "10.1.2.3", 0, 0, 0}, {"isolated", "eth0", "fe80::1", 22, 0, 0}} {
		if _, e := NetworkPolicy(r); e == nil {
			t.Fatal(r)
		}
	}
}
