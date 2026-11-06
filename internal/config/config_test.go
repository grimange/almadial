package config

import (
	"encoding/json"
	"errors"
	"io"
	"net/netip"
	"os"
	"strings"
	"testing"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name + ".json")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
func basic(t *testing.T) Config {
	t.Helper()
	c, err := Load(strings.NewReader(fixture(t, "basic")))
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func ptr(n int) *int { return &n }
func TestValidInputs(t *testing.T) {
	for _, name := range []string{"basic", "webrtc"} {
		t.Run(name, func(t *testing.T) {
			c, err := Load(strings.NewReader(fixture(t, name)))
			if err != nil {
				t.Fatal(err)
			}
			if err := Validate(c); err != nil {
				t.Fatal(err)
			}
			data, err := json.Marshal(c)
			if err != nil {
				t.Fatal(err)
			}
			again, err := Load(strings.NewReader(string(data)))
			if err != nil {
				t.Fatal(err)
			}
			d1, _ := Digest(c)
			d2, _ := Digest(again)
			if d1 != d2 {
				t.Fatal("effective input changed on serialization")
			}
		})
	}
	cases := []struct {
		name   string
		change func(*Config)
	}{
		{"IPv6 IP certificate without DNS", func(c *Config) {
			c.Network.BindAddress = netip.MustParseAddr("2001:db8::20")
			c.Network.AdvertiseAddress = c.Network.BindAddress
			c.WebRTC = WebRTC{true, ptr(8089), &TLS{"/tls/cert", "/tls/key"}}
		}},
		{"wildcard explicit advertised", func(c *Config) { c.Network.BindAddress = netip.IPv4Unspecified() }},
		{"separate advertised", func(c *Config) { c.Network.AdvertiseAddress = netip.MustParseAddr("198.51.100.20") }},
		{"hostname", func(c *Config) { c.Network.Hostname = "dial.example.invalid" }},
		{"single label hostname", func(c *Config) { c.Network.Hostname = "dial" }},
		{"maximum RTP pair", func(c *Config) { c.RTP = RTP{ptr(65534), ptr(65535)} }},
		{"network suffix not inferable without prefix", func(c *Config) {
			c.Network.BindAddress = netip.MustParseAddr("192.0.2.0")
			c.Network.AdvertiseAddress = c.Network.BindAddress
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := basic(t)
			tc.change(&c)
			if err := Validate(c); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestDefaults(t *testing.T) {
	raw, err := Decode(strings.NewReader(fixture(t, "basic")))
	if err != nil {
		t.Fatal(err)
	}
	if raw.SIP.Port != nil || raw.Network.AdvertiseAddress.IsValid() {
		t.Fatal("Decode applied defaults")
	}
	c := Defaults(raw)
	if *c.SIP.Port != 5060 || *c.RTP.Start != 10000 || *c.RTP.End != 20000 || c.WebRTC.Enabled || c.WebRTC.WSSPort != nil || c.Network.Hostname != "" || c.Network.AdvertiseAddress != c.Network.BindAddress {
		t.Fatal("unexpected defaults")
	}
	raw.SIP.Port = ptr(0)
	raw.Network.Hostname = "DIAL.Example.Invalid."
	c = Defaults(raw)
	if *c.SIP.Port != 0 || c.Network.Hostname != "dial.example.invalid" {
		t.Fatal("explicit input overwritten or normalization missing")
	}
	*c.SIP.Port = 5061
	if *raw.SIP.Port != 0 {
		t.Fatal("defaults aliases input port")
	}
	d1, _ := json.Marshal(Defaults(c))
	d2, _ := json.Marshal(Defaults(Defaults(c)))
	if string(d1) != string(d2) {
		t.Fatal("defaulting not idempotent")
	}
}
func TestStrictDecode(t *testing.T) {
	base := fixture(t, "basic")
	cases := []struct{ name, input, field string }{
		{"malformed", "{", "configuration"},
		{"invalid UTF-8", "{\"network\":{\"hostname\":\"" + string([]byte{255}) + "\"}}", "configuration"},
		{"dotted key", `{"network.bind_address":"192.0.2.10"}`, "configuration"},
		{"recursive root key", `{"configuration":{}}`, "configuration"},
		{"root array", "[]", "configuration"},
		{"root null", "null", "configuration"},
		{"unknown", `{"inline_password":"FAKE_SECRET"}`, "configuration"},
		{"case mismatch", strings.Replace(base, "schema_version", "Schema_Version", 1), "configuration"},
		{"nested unknown", strings.Replace(base, `"type": "env"`, `"token": "FAKE_SECRET", "type": "env"`, 1), "credentials.database"},
		{"duplicate", strings.Replace(base, `"schema_version": 1`, `"schema_version": 1,"schema_version": 1`, 1), "schema_version"},
		{"nested duplicate", strings.Replace(base, `"type": "env"`, `"type": "env", "type": "env"`, 1), "credentials.database.type"},
		{"null", strings.Replace(base, `"schema_version": 1`, `"schema_version": null`, 1), "schema_version"},
		{"null object", `{"network":null}`, "network"},
		{"fraction", strings.Replace(base, `"schema_version": 1`, `"schema_version": 1.5`, 1), "schema_version"},
		{"huge integer", `{"schema_version":999999999999999999999999}`, "schema_version"},
		{"string integer", `{"sip":{"port":"5060"}}`, "sip.port"},
		{"wrong boolean", `{"webrtc":{"enabled":1}}`, "webrtc.enabled"},
		{"wrong string", `{"network":{"hostname":false}}`, "network.hostname"},
		{"unknown secret key", `{"FAKE_SECRET":"value"}`, "configuration"},
		{"wrong nested object", `{"credentials":[]}`, "credentials"},
		{"invalid address", `{"network":{"bind_address":"FAKE_SECRET"}}`, "network.bind_address"},
		{"CIDR network", `{"network":{"bind_address":"192.0.2.0/24"}}`, "network.bind_address"},
		{"empty address", `{"network":{"bind_address":""}}`, "network.bind_address"},
		{"trailing document", base + ` {}`, "configuration"},
		{"trailing data", base + ` FAKE_SECRET`, "configuration"},
		{"oversized", strings.Repeat(" ", maxDocumentBytes+1), "configuration"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Decode(strings.NewReader(tc.input))
			assertField(t, err, tc.field)
			if strings.Contains(err.Error(), "FAKE_SECRET") {
				t.Fatal("diagnostic leaked input")
			}
		})
	}
	for _, name := range []string{"invalid-inline", "invalid-unattended"} {
		t.Run(name, func(t *testing.T) {
			if _, err := Load(strings.NewReader(fixture(t, name))); err == nil {
				t.Fatal("invalid fixture accepted")
			}
		})
	}
}
func assertField(t *testing.T, err error, field string) {
	t.Helper()
	var fe *FieldError
	if !errors.As(err, &fe) || fe.Field != field {
		t.Fatalf("wanted error field %s, got %v", field, err)
	}
}
func TestValidationFailures(t *testing.T) {
	cases := []struct {
		name, field string
		change      func(*Config)
	}{
		{"unsupported version", "schema_version", func(c *Config) { c.SchemaVersion = 2 }},
		{"missing version", "schema_version", func(c *Config) { c.SchemaVersion = 0 }},
		{"missing bind", "network.bind_address", func(c *Config) { c.Network.BindAddress = netip.Addr{} }},
		{"ambiguous wildcard", "network.advertise_address", func(c *Config) {
			c.Network.BindAddress = netip.IPv4Unspecified()
			c.Network.AdvertiseAddress = netip.Addr{}
		}},
		{"unspecified advertised", "network.advertise_address", func(c *Config) { c.Network.AdvertiseAddress = netip.IPv4Unspecified() }},
		{"mixed families", "network.advertise_address", func(c *Config) { c.Network.AdvertiseAddress = netip.MustParseAddr("2001:db8::10") }},
		{"multicast", "network.bind_address", func(c *Config) { c.Network.BindAddress = netip.MustParseAddr("224.0.0.1") }},
		{"loopback", "network.bind_address", func(c *Config) { c.Network.BindAddress = netip.MustParseAddr("127.0.0.1") }},
		{"broadcast", "network.bind_address", func(c *Config) { c.Network.BindAddress = netip.MustParseAddr("255.255.255.255") }},
		{"link local", "network.bind_address", func(c *Config) { c.Network.BindAddress = netip.MustParseAddr("fe80::1") }},
		{"scoped", "network.bind_address", func(c *Config) { c.Network.BindAddress = netip.MustParseAddr("fe80::1%eth0") }},
		{"mapped", "network.bind_address", func(c *Config) { c.Network.BindAddress = netip.MustParseAddr("::ffff:192.0.2.1") }},
		{"zero port", "sip.port", func(c *Config) { c.SIP.Port = ptr(0) }},
		{"negative port", "sip.port", func(c *Config) { c.SIP.Port = ptr(-1) }},
		{"large port", "sip.port", func(c *Config) { c.SIP.Port = ptr(65536) }},
		{"missing effective port", "sip.port", func(c *Config) { c.SIP.Port = nil }},
		{"RTP zero", "rtp.start", func(c *Config) { c.RTP.Start = ptr(0) }},
		{"RTP end overflow", "rtp.end", func(c *Config) { c.RTP.End = ptr(65536) }},
		{"RTP reversed", "rtp.end", func(c *Config) { c.RTP.Start = ptr(20000); c.RTP.End = ptr(10000) }},
		{"RTP single port", "rtp.end", func(c *Config) { c.RTP.End = ptr(10000) }},
		{"RTP odd start", "rtp.start", func(c *Config) { c.RTP.Start = ptr(10001) }},
		{"UDP conflict", "sip.port", func(c *Config) { c.SIP.Port = ptr(10000) }},
		{"disabled WSS port", "webrtc", func(c *Config) { c.WebRTC.WSSPort = ptr(8089) }},
		{"disabled TLS", "webrtc", func(c *Config) { c.WebRTC.TLS = &TLS{} }},
		{"missing TLS", "webrtc.tls", func(c *Config) { c.WebRTC = WebRTC{Enabled: true, WSSPort: ptr(8089)} }},
		{"WSS conflict", "webrtc.wss_port", func(c *Config) { c.WebRTC = WebRTC{Enabled: true, WSSPort: ptr(5060)} }},
		{"WSS invalid", "webrtc.wss_port", func(c *Config) { c.WebRTC = WebRTC{Enabled: true, WSSPort: ptr(0)} }},
		{"missing certificate", "webrtc.tls.certificate_file", func(c *Config) { c.WebRTC = WebRTC{true, ptr(8089), &TLS{PrivateKeyFile: "/key"}} }},
		{"missing key", "webrtc.tls.private_key_file", func(c *Config) { c.WebRTC = WebRTC{true, ptr(8089), &TLS{CertificateFile: "/cert"}} }},
		{"same TLS files", "webrtc.tls.private_key_file", func(c *Config) { c.WebRTC = WebRTC{true, ptr(8089), &TLS{"/cert", "/cert"}} }},
		{"missing reference", "credentials.database.type", func(c *Config) { c.Credentials.Database = SecretRef{} }},
		{"unsupported ref", "credentials.database.type", func(c *Config) { c.Credentials.Database.Type = "FAKE_SECRET" }},
		{"empty env", "credentials.database.source", func(c *Config) { c.Credentials.Database.Source = "" }},
		{"env assignment", "credentials.database.source", func(c *Config) { c.Credentials.Database.Source = "NAME=FAKE_SECRET" }},
		{"env dollar expansion", "credentials.database.source", func(c *Config) { c.Credentials.Database.Source = "${FAKE_SECRET}" }},
		{"env leading digit", "credentials.database.source", func(c *Config) { c.Credentials.Database.Source = "1FAKE_SECRET" }},
		{"file relative", "credentials.database.source", func(c *Config) { c.Credentials.Database = SecretRef{File, "FAKE_SECRET"} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := basic(t)
			tc.change(&c)
			err := Validate(c)
			assertField(t, err, tc.field)
			if strings.Contains(err.Error(), "FAKE_SECRET") {
				t.Fatal("diagnostic leaked input")
			}
		})
	}
}
func TestHostnames(t *testing.T) {
	for _, s := range []string{".", "https://dial.example", "bad name", "-dial", "dial-", "a..b", "a_b", strings.Repeat("a", 64), strings.Repeat("a.", 127) + "aa", "192.0.2.1", "díal"} {
		t.Run(s, func(t *testing.T) {
			c := basic(t)
			c.Network.Hostname = s
			assertField(t, Validate(c), "network.hostname")
		})
	}
}
func TestFileReferences(t *testing.T) {
	for _, p := range []string{"", "/", "relative/key", "/a/../key", "/a//key", "/a/", "/a\x00key", "/a\nkey", "-----BEGIN PRIVATE KEY-----FAKE_SECRET", "/-----BEGIN PRIVATE KEY-----", "/a\\key", " /a"} {
		t.Run(p, func(t *testing.T) {
			c := basic(t)
			c.Credentials.Database = SecretRef{File, p}
			assertField(t, Validate(c), "credentials.database.source")
			c = basic(t)
			c.WebRTC = WebRTC{true, ptr(8089), &TLS{p, "/key"}}
			assertField(t, Validate(c), "webrtc.tls.certificate_file")
		})
	}
}
func TestUnattendedAndEffectiveInputs(t *testing.T) {
	fromFile := basic(t)
	direct := Config{SchemaVersion: 1, Network: Network{BindAddress: netip.MustParseAddr("192.0.2.10")}, Credentials: Credentials{
		Database: SecretRef{Environment, "ALMADIAL_DB_PASSWORD"}, AMI: SecretRef{Environment, "ALMADIAL_AMI_PASSWORD"}, SIP: SecretRef{Environment, "ALMADIAL_SIP_PASSWORD"}, Admin: SecretRef{Environment, "ALMADIAL_ADMIN_PASSWORD"},
	}}
	a, err := Digest(fromFile)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Digest(direct)
	if err != nil {
		t.Fatal(err)
	}
	if a != b || len(a) != 64 {
		t.Fatal("file and direct effective input identities disagree")
	}
	assertField(t, ValidateUnattended(direct, Approval{}), "approval")
	assertField(t, ValidateUnattended(direct, Approval{true, "bad"}), "approval.config_sha256")
	approval := Approval{true, a}
	if err := ValidateUnattended(direct, approval); err != nil {
		t.Fatal(err)
	}
	direct.Network.Hostname = "changed.example.invalid"
	assertField(t, ValidateUnattended(direct, approval), "approval.config_sha256")
	direct.Credentials.Admin = SecretRef{}
	assertField(t, ValidateUnattended(direct, approval), "credentials.admin.type")
}

type brokenReader struct{}

func (brokenReader) Read([]byte) (int, error) { return 0, errors.New("FAKE_SECRET") }
func TestReadErrorsAndNoSecretResolution(t *testing.T) {
	_, err := Decode(brokenReader{})
	assertField(t, err, "configuration")
	if strings.Contains(err.Error(), "FAKE_SECRET") {
		t.Fatal("reader error leaked")
	}
	c := basic(t)
	c.Credentials.Database = SecretRef{File, "/does/not/exist/almadial-secret"}
	if err := Validate(c); err != nil {
		t.Fatal("validation performed file I/O", err)
	}
	t.Setenv("ALMADIAL_AMI_PASSWORD", "FAKE_RESOLVED_VALUE")
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "FAKE_RESOLVED_VALUE") {
		t.Fatal("secret resolved")
	}
	if _, err := Decode(io.LimitReader(strings.NewReader(string(data)), int64(len(data)))); err != nil {
		t.Fatal(err)
	}
}

func TestRejectedLoadReturnsZero(t *testing.T) {
	input := strings.Replace(fixture(t, "basic"), `"schema_version": 1`, `"schema_version": 2`, 1)
	c, err := Load(strings.NewReader(input))
	assertField(t, err, "schema_version")
	if c.Network.BindAddress.IsValid() || c.Credentials.Database.Source != "" {
		t.Fatal("partial invalid result escaped")
	}
	c = basic(t)
	for _, name := range []string{".", "K.example"} {
		c.Network.Hostname = name
		if err := Validate(Defaults(c)); err == nil {
			t.Fatal("normalization concealed invalid hostname")
		}
	}
}
func TestApprovalCoversEveryInputGroup(t *testing.T) {
	original := basic(t)
	digest, err := Digest(original)
	if err != nil {
		t.Fatal(err)
	}
	changes := []func(*Config){
		func(c *Config) { c.SIP.Port = ptr(5062) }, func(c *Config) { c.RTP.End = ptr(20002) },
		func(c *Config) { c.Network.AdvertiseAddress = netip.MustParseAddr("198.51.100.1") },
		func(c *Config) { c.Credentials.SIP.Source = "DIFFERENT_SECRET_SOURCE" },
		func(c *Config) { c.WebRTC = WebRTC{true, ptr(8089), &TLS{"/cert", "/key"}} },
	}
	for i, change := range changes {
		t.Run(string(rune('A'+i)), func(t *testing.T) {
			c := basic(t)
			change(&c)
			assertField(t, ValidateUnattended(c, Approval{true, digest}), "approval.config_sha256")
		})
	}
}
