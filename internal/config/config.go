// Package config defines non-secret installation inputs. It performs no host
// inspection, DNS lookup, secret resolution, or installation operations.
package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/netip"
	"strings"
)

const SchemaVersion = 1

type Config struct {
	SchemaVersion int         `json:"schema_version"`
	Network       Network     `json:"network"`
	SIP           SIP         `json:"sip"`
	RTP           RTP         `json:"rtp"`
	WebRTC        WebRTC      `json:"webrtc"`
	Credentials   Credentials `json:"credentials"`
}

type Network struct {
	BindAddress      netip.Addr `json:"bind_address"`
	AdvertiseAddress netip.Addr `json:"advertise_address,omitempty,omitzero"`
	Hostname         string     `json:"hostname,omitempty"`
}

// Pointer ports distinguish omission (use the documented default) from zero
// (invalid). Defaults never overwrite an explicit value.
type SIP struct {
	Port *int `json:"port,omitempty"`
}
type RTP struct {
	Start *int `json:"start,omitempty"`
	End   *int `json:"end,omitempty"`
}
type WebRTC struct {
	Enabled bool `json:"enabled"`
	WSSPort *int `json:"wss_port,omitempty"`
	TLS     *TLS `json:"tls,omitempty"`
}
type TLS struct {
	CertificateFile string `json:"certificate_file"`
	PrivateKeyFile  string `json:"private_key_file"`
}

type ReferenceType string

const (
	Environment ReferenceType = "env"
	File        ReferenceType = "file"
)

// SecretRef contains a source identifier only, never a resolved secret value.
// A file source must identify an absolute path; an env source a variable name.
type SecretRef struct {
	Type   ReferenceType `json:"type"`
	Source string        `json:"source"`
}
type Credentials struct {
	Database SecretRef `json:"database"`
	AMI      SecretRef `json:"ami"`
	SIP      SecretRef `json:"sip"`
	Admin    SecretRef `json:"admin"`
}

// FieldError deliberately excludes user-supplied values, including unknown keys.
type FieldError struct{ Field, Reason string }

func (e *FieldError) Error() string      { return e.Field + ": " + e.Reason }
func invalid(field, reason string) error { return &FieldError{field, reason} }

// Defaults returns a normalized copy. No environment-derived defaults exist.
// Schema version, bind address and credential sources always remain explicit.
func Defaults(c Config) Config {
	port := func(p *int, fallback int) *int {
		v := fallback
		if p != nil {
			v = *p
		}
		return &v
	}
	c.SIP.Port = port(c.SIP.Port, 5060)
	c.RTP.Start = port(c.RTP.Start, 10000)
	c.RTP.End = port(c.RTP.End, 20000)
	c.Network.Hostname = strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			return r + ('a' - 'A')
		}
		return r
	}, c.Network.Hostname)
	if len(c.Network.Hostname) > 1 {
		c.Network.Hostname = strings.TrimSuffix(c.Network.Hostname, ".")
	}
	if c.Network.BindAddress.IsValid() && !c.Network.BindAddress.IsUnspecified() && !c.Network.AdvertiseAddress.IsValid() {
		c.Network.AdvertiseAddress = c.Network.BindAddress
	}
	if c.WebRTC.Enabled {
		c.WebRTC.WSSPort = port(c.WebRTC.WSSPort, 8089)
	} else if c.WebRTC.WSSPort != nil {
		c.WebRTC.WSSPort = port(c.WebRTC.WSSPort, 0)
	}
	if c.WebRTC.TLS != nil {
		tls := *c.WebRTC.TLS
		c.WebRTC.TLS = &tls
	}
	return c
}

// Digest binds approval to the complete normalized, validated non-secret input.
// It is an input identity, not proof that the host or an installation is safe.
func Digest(c Config) (string, error) {
	c = Defaults(c)
	if err := Validate(c); err != nil {
		return "", err
	}
	data, err := json.Marshal(c)
	if err != nil {
		return "", invalid("configuration", "cannot encode validated inputs")
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// Approval is supplied separately by a future operator-facing workflow. It is
// not read from installation configuration, and does not authorize a command.
type Approval struct {
	Approved     bool
	ConfigSHA256 string
}

func ValidateUnattended(c Config, approval Approval) error {
	digest, err := Digest(c)
	if err != nil {
		return err
	}
	if !approval.Approved {
		return invalid("approval", "explicit unattended approval is required")
	}
	if approval.ConfigSHA256 != digest {
		return invalid("approval.config_sha256", "must match the validated effective configuration")
	}
	return nil
}
