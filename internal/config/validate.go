package config

import (
	"net/netip"
	"path"
	"strings"
	"unicode"
)

// Validate checks effective input (call Defaults first). Errors follow a stable
// field order. It never reads TLS/secret files, environment variables or DNS.
func Validate(c Config) error {
	if c.SchemaVersion != SchemaVersion {
		return invalid("schema_version", "must be supported version 1")
	}
	if err := address("network.bind_address", c.Network.BindAddress, true); err != nil {
		return err
	}
	if err := address("network.advertise_address", c.Network.AdvertiseAddress, false); err != nil {
		return err
	}
	if c.Network.BindAddress.Is4() != c.Network.AdvertiseAddress.Is4() {
		return invalid("network.advertise_address", "must use the bind address family")
	}
	if c.Network.Hostname != "" && !hostname(c.Network.Hostname) {
		return invalid("network.hostname", "must be a DNS hostname of at most 253 characters")
	}
	for _, p := range []struct {
		field string
		value *int
	}{
		{"sip.port", c.SIP.Port}, {"rtp.start", c.RTP.Start}, {"rtp.end", c.RTP.End},
	} {
		if err := port(p.field, p.value); err != nil {
			return err
		}
	}
	if *c.RTP.Start >= *c.RTP.End {
		return invalid("rtp.end", "must exceed rtp.start to contain an RTP/RTCP pair")
	}
	if *c.RTP.Start%2 != 0 {
		return invalid("rtp.start", "must be even for RTP/RTCP pairing")
	}
	if *c.SIP.Port >= *c.RTP.Start && *c.SIP.Port <= *c.RTP.End {
		return invalid("sip.port", "must not overlap the UDP RTP range")
	}
	if c.WebRTC.Enabled {
		if err := port("webrtc.wss_port", c.WebRTC.WSSPort); err != nil {
			return err
		}
		// Keep distinct service ports as a schema policy, even though SIP UDP and
		// WSS TCP could technically share a number on some hosts.
		if *c.WebRTC.WSSPort == *c.SIP.Port {
			return invalid("webrtc.wss_port", "must differ from sip.port")
		}
		if c.WebRTC.TLS == nil {
			return invalid("webrtc.tls", "certificate and private-key file references are required")
		}
		if !filePath(c.WebRTC.TLS.CertificateFile) {
			return invalid("webrtc.tls.certificate_file", "must be a clean absolute Linux file path")
		}
		if !filePath(c.WebRTC.TLS.PrivateKeyFile) {
			return invalid("webrtc.tls.private_key_file", "must be a clean absolute Linux file path")
		}
		if c.WebRTC.TLS.CertificateFile == c.WebRTC.TLS.PrivateKeyFile {
			return invalid("webrtc.tls.private_key_file", "must differ from certificate_file")
		}
	} else if c.WebRTC.WSSPort != nil || c.WebRTC.TLS != nil {
		return invalid("webrtc", "WSS/TLS settings require enabled=true")
	}
	for _, ref := range []struct {
		field string
		value SecretRef
	}{
		{"credentials.database", c.Credentials.Database}, {"credentials.ami", c.Credentials.AMI},
		{"credentials.sip", c.Credentials.SIP}, {"credentials.admin", c.Credentials.Admin},
	} {
		switch ref.value.Type {
		case Environment:
			if !envName(ref.value.Source) {
				return invalid(ref.field+".source", "must be an environment-variable name")
			}
		case File:
			if !filePath(ref.value.Source) {
				return invalid(ref.field+".source", "must be a clean absolute Linux file path")
			}
		default:
			return invalid(ref.field+".type", "must be env or file")
		}
	}
	return nil
}

func address(field string, ip netip.Addr, bind bool) error {
	if !ip.IsValid() {
		return invalid(field, "an explicit valid IP address is required")
	}
	if ip.Is4In6() || ip.Zone() != "" {
		return invalid(field, "mapped IPv4 and scoped IPv6 addresses are unsupported")
	}
	if bind && ip.IsUnspecified() {
		return nil
	}
	if !ip.IsGlobalUnicast() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip == netip.MustParseAddr("255.255.255.255") {
		return invalid(field, "must be a unicast host address (private addresses are supported)")
	}
	return nil
}
func port(field string, p *int) error {
	if p == nil || *p < 1 || *p > 65535 {
		return invalid(field, "must be an integer from 1 through 65535")
	}
	return nil
}
func hostname(value string) bool {
	if len(value) > 253 {
		return false
	}
	if _, err := netip.ParseAddr(value); err == nil {
		return false
	}
	for _, label := range strings.Split(value, ".") {
		if len(label) < 1 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, c := range label {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-') {
				return false
			}
		}
	}
	return true
}
func filePath(value string) bool {
	if !path.IsAbs(value) || value == "/" || path.Clean(value) != value || strings.Contains(value, "-----BEGIN") || strings.Contains(value, "\\") {
		return false
	}
	for _, c := range value {
		if unicode.IsControl(c) {
			return false
		}
	}
	return strings.TrimSpace(value) == value
}
func envName(value string) bool {
	for i, c := range value {
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c == '_' || i > 0 && c >= '0' && c <= '9') {
			return false
		}
	}
	return len(value) > 0
}
