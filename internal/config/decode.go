package config

import (
	"bytes"
	"encoding/json"
	"io"
	"net/netip"
	"strconv"
	"strings"
	"unicode/utf8"
)

const maxDocumentBytes = 1 << 20

// Decode checks the JSON shape and returns typed input without applying defaults
// or semantic validation. Only one object, at most 1 MiB, is accepted. Read errors
// are redacted because a reader's error may contain private input or paths.
func Decode(r io.Reader) (Config, error) {
	var zero Config
	data, err := io.ReadAll(io.LimitReader(r, maxDocumentBytes+1))
	if err != nil {
		return zero, invalid("configuration", "cannot read JSON document")
	}
	if len(data) > maxDocumentBytes {
		return zero, invalid("configuration", "JSON document exceeds 1 MiB")
	}
	if !utf8.Valid(data) {
		return zero, invalid("configuration", "JSON must be valid UTF-8")
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := guard(dec, "configuration"); err != nil {
		return zero, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return zero, invalid("configuration", "expected exactly one JSON document")
	}
	dec = json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var c Config
	if err := dec.Decode(&c); err != nil {
		return zero, invalid("configuration", "invalid typed JSON value")
	}
	return c, nil
}

func Load(r io.Reader) (Config, error) {
	c, err := Decode(r)
	if err != nil {
		return Config{}, err
	}
	c = Defaults(c)
	if err = Validate(c); err != nil {
		return Config{}, err
	}
	return c, nil
}

// The small explicit shape table enforces exact spelling, rejects duplicate
// keys/nulls and gives value-free errors before encoding/json's typed decoder.
func shape(path string) string {
	switch path {
	case "configuration", "network", "sip", "rtp", "webrtc", "webrtc.tls", "credentials", "credentials.database", "credentials.ami", "credentials.sip", "credentials.admin":
		return "object"
	case "schema_version", "sip.port", "rtp.start", "rtp.end", "webrtc.wss_port":
		return "integer"
	case "network.bind_address", "network.advertise_address":
		return "address"
	case "network.hostname", "webrtc.tls.certificate_file", "webrtc.tls.private_key_file", "credentials.database.type", "credentials.database.source", "credentials.ami.type", "credentials.ami.source", "credentials.sip.type", "credentials.sip.source", "credentials.admin.type", "credentials.admin.source":
		return "string"
	case "webrtc.enabled":
		return "boolean"
	default:
		return ""
	}
}

func guard(dec *json.Decoder, path string) error {
	token, err := dec.Token()
	if err != nil {
		return invalid(path, "malformed JSON")
	}
	kind := shape(path)
	switch kind {
	case "object":
		if token != json.Delim('{') {
			return invalid(path, "must be a JSON object")
		}
		seen := map[string]bool{}
		for dec.More() {
			key, err := dec.Token()
			if err != nil {
				return invalid(path, "malformed JSON object")
			}
			name, ok := key.(string)
			if !ok {
				return invalid(path, "malformed JSON field")
			}
			child := name
			if path != "configuration" {
				child = path + "." + name
			}
			if name == "configuration" || strings.Contains(name, ".") || shape(child) == "" {
				return invalid(path, "unknown field (field name omitted)")
			}
			if seen[name] {
				return invalid(child, "duplicate field")
			}
			seen[name] = true
			if err := guard(dec, child); err != nil {
				return err
			}
		}
		end, err := dec.Token()
		if err != nil || end != json.Delim('}') {
			return invalid(path, "malformed JSON object")
		}
	case "integer":
		n, ok := token.(json.Number)
		if !ok {
			return invalid(path, "must be an integer")
		}
		if _, err := strconv.ParseInt(string(n), 10, strconv.IntSize); err != nil {
			return invalid(path, "must be a representable integer")
		}
	case "boolean":
		if _, ok := token.(bool); !ok {
			return invalid(path, "must be a boolean")
		}
	case "string", "address":
		value, ok := token.(string)
		if !ok {
			return invalid(path, "must be a string")
		}
		if kind == "address" {
			if _, err := netip.ParseAddr(value); err != nil {
				return invalid(path, "must be an IPv4 or IPv6 address without a prefix")
			}
		}
	}
	return nil
}
