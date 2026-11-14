package provision

import (
	"strings"
	"testing"
)

func TestRemoteNetworkAndTLSContracts(t *testing.T) {
	r := NetworkRequest{Mode: "remote", Interface: "eth0", ManagementAddress: "192.0.2.10", SSHPort: 22, RTPStart: 12000, RTPEnd: 12019}
	p, e := NetworkPolicy(r)
	if e != nil {
		t.Fatal(e)
	}
	for _, required := range []string{`port="443" protocol="tcp"`, `port="12000-12019" protocol="udp"`, `port="22" protocol="tcp"`, `target="DROP"`} {
		if !strings.Contains(p, required) {
			t.Fatal(required)
		}
	}
	for _, forbidden := range []string{`port="3306"`, `port="8088"`, `port="8089"`, `port="5038"`, `port="4577"`} {
		if strings.Contains(p, forbidden) {
			t.Fatal(forbidden)
		}
	}
	r.RTPStart = 12001
	if _, e = NetworkPolicy(r); e == nil {
		t.Fatal("odd RTP start accepted")
	}
	r.Mode = "isolated"
	r.ManagementAddress = "10.0.0.2"
	if _, e = NetworkPolicy(r); e == nil {
		t.Fatal("lab media permission accepted")
	}
	v := RemoteTLSVhost("dialer.example.test")
	for _, required := range []string{"Listen 0.0.0.0:443 https", "ServerName dialer.example.test", "Require all granted", `ProxyPass "/ws" "ws://127.0.0.1:8088/ws"`, "SSLProtocol -all +TLSv1.2 +TLSv1.3"} {
		if !strings.Contains(v, required) {
			t.Fatal(required)
		}
	}
	lab := TLSVhost("localhost")
	if strings.Contains(lab, "ProxyPass") || strings.Contains(lab, "Require all granted") || !strings.Contains(lab, "Listen 127.0.0.1:443") {
		t.Fatal("lab TLS contract changed")
	}
}

func TestRemoteWebListenerHandoffIsNarrow(t *testing.T) {
	const http = `tcp LISTEN 0 511 127.0.0.1:80 0.0.0.0:* users:(("httpd",pid=123,fd=4))`
	const https = `tcp LISTEN 0 511 0.0.0.0:443 0.0.0.0:* users:(("httpd",pid=123,fd=5))`
	for _, tc := range []struct {
		name, sockets   string
		tls, remote, ok bool
	}{
		{"remote verified", http + "\n" + https, true, true, true},
		{"ordinary foundation rejects public", http + "\n" + https, false, false, false},
		{"lab handoff rejects public", http + "\n" + https, true, false, false},
		{"remote flag needs TLS", http + "\n" + https, false, true, false},
		{"public HTTP rejected", strings.Replace(http, "127.0.0.1:80", "0.0.0.0:80", 1) + "\n" + https, true, true, false},
		{"wrong owner rejected", http + "\n" + strings.Replace(https, "pid=123", "pid=999", 1), true, true, false},
		{"internal database rejected", http + "\n" + https + "\ntcp LISTEN 0 511 0.0.0.0:3306 0.0.0.0:* users:((\"httpd\",pid=123,fd=6))", true, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := verifyVICIdialWebListenersTLS(tc.sockets, 123, tc.tls, tc.remote) == nil; got != tc.ok {
				t.Fatalf("listener admitted=%v, want %v", got, tc.ok)
			}
		})
	}
}
