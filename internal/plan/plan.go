// Package plan describes future changes; it never executes them or grants approval.
package plan

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/grimange/almadial/internal/config"
	"github.com/grimange/almadial/internal/preflight"
)

const SchemaVersion = 1

type Stage struct {
	ID             string   `json:"id"`
	Description    string   `json:"description"`
	Risk           string   `json:"risk"`
	Prerequisites  []string `json:"prerequisites"`
	Inputs         []string `json:"inputs"`
	Changes        []string `json:"changes"`
	Postconditions []string `json:"postconditions"`
}
type document struct {
	SchemaVersion  int                 `json:"schema_version"`
	ConfigSHA256   string              `json:"config_sha256"`
	ManifestSHA256 string              `json:"manifest_sha256"`
	HostSHA256     string              `json:"host_sha256"`
	Stages         []Stage             `json:"stages"`
	Findings       []preflight.Finding `json:"findings"`
	Eligible       bool                `json:"eligible"`
}

// Plan owns only canonical serialized bytes. Accessors decode fresh copies;
// caller-owned input slices/maps and returned collections cannot mutate it.
type Plan struct{ data string }

func New(c config.Config, s preflight.Snapshot, r preflight.Requirements, manifestSHA256 string) (Plan, error) {
	raw, err := hex.DecodeString(manifestSHA256)
	if err != nil || len(raw) != 32 || strings.ToLower(manifestSHA256) != manifestSHA256 {
		return Plan{}, errors.New("manifest: expected lowercase SHA-256 identity from the future verified manifest contract")
	}
	cd, err := config.Digest(c)
	if err != nil {
		return Plan{}, err
	}
	hd, err := preflight.Fingerprint(c, s, r)
	if err != nil {
		return Plan{}, err
	}
	fs, err := preflight.Assess(c, s, r)
	if err != nil {
		return Plan{}, err
	}
	stages := stages(config.Defaults(c))
	d := document{SchemaVersion, cd, manifestSHA256, hd, stages, fs, preflight.Eligible(fs)}
	b, err := json.Marshal(d)
	if err != nil {
		return Plan{}, err
	}
	return Plan{string(b)}, nil
}
func (p Plan) doc() (document, error) {
	var d document
	if p.data == "" {
		return d, errors.New("plan is uninitialized")
	}
	err := json.Unmarshal([]byte(p.data), &d)
	return d, err
}
func (p Plan) JSON() ([]byte, error) {
	if _, err := p.doc(); err != nil {
		return nil, err
	}
	return []byte(p.data), nil
}
func (p Plan) Digest() string {
	if p.data == "" {
		return ""
	}
	h := sha256.Sum256([]byte(p.data))
	return hex.EncodeToString(h[:])
}
func (p Plan) Stages() []Stage               { d, _ := p.doc(); return d.Stages }
func (p Plan) Findings() []preflight.Finding { d, _ := p.doc(); return d.Findings }
func (p Plan) Eligible() bool                { d, err := p.doc(); return err == nil && d.Eligible }
func (p Plan) Human() (string, error) {
	d, err := p.doc()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "AlmaDial installation plan %s\nConfig: %s\nManifest: %s\nHost: %s\n", p.Digest(), d.ConfigSHA256, d.ManifestSHA256, d.HostSHA256)
	b.WriteString(preflight.Human(d.Findings))
	for _, s := range d.Stages {
		fmt.Fprintf(&b, "%s [%s] %s\n  prerequisites: %s\n  inputs: %s\n  intended changes: %s\n  expected postconditions: %s\n", s.ID, s.Risk, s.Description, strings.Join(s.Prerequisites, ", "), strings.Join(s.Inputs, ", "), strings.Join(s.Changes, "; "), strings.Join(s.Postconditions, "; "))
	}
	b.WriteString("No stages executed. Separate operator approval and fresh revalidation required.\n")
	return b.String(), nil
}

// Revalidate is pure. The future executor must collect a fresh snapshot, verify
// manifest provenance and obtain actual approval independently immediately before
// any mutation; this helper neither collects observations nor executes stages.
func (p Plan) Revalidate(c config.Config, s preflight.Snapshot, r preflight.Requirements, manifest string) error {
	if !p.Eligible() {
		return errors.New("plan blocked or uninitialized; regenerate after resolving findings")
	}
	fresh, err := New(c, s, r, manifest)
	if err != nil {
		return err
	}
	if !fresh.Eligible() {
		return errors.New("fresh preflight blocked; regenerate and obtain renewed approval")
	}
	if fresh.data != p.data {
		return errors.New("plan stale: material inputs changed; regenerate and obtain renewed approval")
	}
	return nil
}
func stages(c config.Config) []Stage {
	web := c.WebRTC.Enabled
	s := []Stage{
		{"S00", "Revalidate supported clean host", "read-only", nil, []string{"effective config digest", "verified manifest identity", "fresh host snapshot", "reviewed resource/ISA budget"}, []string{"none"}, []string{"host eligible; material inputs match approved plan"}},
		{"S01", "Install pinned dependencies", "system", []string{"S00"}, []string{"verified manifest RPM and source identities"}, []string{"install approved RPM and PHP/Perl build/runtime dependencies"}, []string{"signatures and selected versions verified; no foreign repositories"}},
		{"S02", "Create filesystem and service identities", "system", []string{"S01"}, []string{"reviewed file ownership contract"}, []string{"create service accounts and owned directories only"}, []string{"ownership, modes, SELinux labels and storage verified"}},
		{"S03", "Initialize MariaDB", "irreversible", []string{"S02"}, []string{"schema 1748 derivation", "database secret reference"}, []string{"configure socket-only MariaDB and import fresh derived schema"}, []string{"345 tables; strict SQL mode; no TCP listener; stop on partial schema"}},
		{"S04", "Build and configure native Asterisk", "system", []string{"S01", "S02"}, []string{"Asterisk 22.12.0 and bundled PJPROJECT 2.17 verified sources"}, []string{"compile required modules; deploy owned base configuration and local AMI"}, []string{"version/modules; PJSIP and ConfBridge loaded; no generated file collisions"}},
		{"S05", "Install VICIdial and compatibility derivations", "system", []string{"S03", "S04"}, []string{"r4037 pinned source and derivation hashes", "credential references"}, []string{"deploy preserved upstream and derived PHP/Perl assets; register server; run actual configuration generator"}, []string{"schema and AMI verified; generated PJSIP/dialplan/ConfBridge recognized; regeneration stable"}},
		{"S06", "Configure runtime scheduling", "system", []string{"S05"}, []string{"bounded upstream heartbeat and FastAGI contracts"}, []string{"configure owned scheduling and process supervision"}, []string{"AST_update heartbeat current; no duplicate jobs; FastAGI loopback-only"}},
		{"S07", "Configure Apache/PHP", "system", []string{"S05"}, []string{"approved webroot ownership and PHP derivations"}, []string{"deploy owned vhost and PHP runtime configuration"}, []string{"application authentication works; document root not broadly writable"}},
		{"S08", "Configure bounded network policy", "security", []string{"S06", "S07"}, []string{"bind and advertised IP; optional hostname; reviewed firewall policy"}, []string{"configure approved local listeners, SELinux contexts and bounded firewall rules"}, []string{"management endpoints loopback/socket-only; SELinux Enforcing; external profiles not claimed"}},
	}
	s[8].Changes = append(s[8].Changes, fmt.Sprintf("bind %s; advertise %s; SIP UDP %d; RTP UDP %d-%d", c.Network.BindAddress, c.Network.AdvertiseAddress, *c.SIP.Port, *c.RTP.Start, *c.RTP.End))
	if web {
		s[8].Changes = append(s[8].Changes, fmt.Sprintf("WSS TCP %d; certificate identity %s", *c.WebRTC.WSSPort, certificateIdentity(c)))
		s[8].Description = "Configure TLS/WSS and bounded network policy"
		s[8].Inputs = append(s[8].Inputs, "certificate and private-key file references; IP or hostname identity")
		s[8].Changes = append(s[8].Changes, "deploy referenced TLS material without disclosing private key")
		s[8].Postconditions = append(s[8].Postconditions, "certificate trust and identity validated; WSS available")
		s = append(s, Stage{"S09", "Deploy self-hosted VICIphone", "system", []string{"S04", "S05", "S06", "S07", "S08"}, []string{"VICIphone 3.0 pinned assets and credential POST overlays"}, []string{"deploy verified local browser-phone assets and integration"}, []string{"secure registration; ICE/DTLS-SRTP; bidirectional browser media; manual recovery and logout cleanup"}})
	}
	deps := []string{"S06", "S07", "S08"}
	if web {
		deps = append(deps, "S09")
	}
	s = append(s, Stage{"S10", "Verify integrated installation", "test-effects", deps, []string{"explicit local synthetic fixtures; separately approved reboot test"}, []string{"bounded local call/recording and service persistence tests"}, []string{"agent login; conference; recording; controls; teardown; reboot recovery; no public calls"}}, Stage{"S11", "Handover", "report", []string{"S10"}, []string{"verified postcondition evidence"}, []string{"produce redacted operator results and recovery guidance"}, []string{"no temporary secrets or orphaned test calls; limitations disclosed"}})
	return s
}

func certificateIdentity(c config.Config) string {
	if c.Network.Hostname != "" {
		return c.Network.Hostname
	}
	return c.Network.AdvertiseAddress.String()
}

// ConfigSHA256 and ManifestSHA256 expose immutable approval bindings.
func (p Plan) ConfigSHA256() string   { d, _ := p.doc(); return d.ConfigSHA256 }
func (p Plan) ManifestSHA256() string { d, _ := p.doc(); return d.ManifestSHA256 }
