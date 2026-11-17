// Package artifacts verifies source identities without installing or executing them.
package artifacts

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sort"

	"go.yaml.in/yaml/v3"
)

const manifestLimit = 64 << 10

type leaf struct{ value, tag string }

type Source struct {
	Version string `yaml:"version"`
	Source  string `yaml:"source"`
	SHA256  string `yaml:"sha256"`
}
type RPM struct {
	NEVRA      string `yaml:"nevra"`
	Repository string `yaml:"repository"`
}
type manifestDocument struct {
	SchemaVersion int    `yaml:"schema_version"`
	Profile       string `yaml:"profile"`
	Status        string `yaml:"status"`
	OS            struct {
		ID          string `yaml:"id"`
		Major       int    `yaml:"major"`
		Release     string `yaml:"release"`
		Arch        string `yaml:"arch"`
		ImageSHA256 string `yaml:"image_sha256"`
	} `yaml:"os"`
	Asterisk struct {
		Major       int    `yaml:"major_candidate"`
		Version     string `yaml:"version"`
		Source      string `yaml:"source"`
		SHA256      string `yaml:"sha256"`
		Fingerprint string `yaml:"signing_fingerprint"`
	} `yaml:"asterisk"`
	VICIdial struct {
		Source        string `yaml:"source"`
		Revision      int    `yaml:"revision"`
		TreeSHA256    string `yaml:"exported_tree_sha256"`
		SchemaVersion int    `yaml:"schema_version"`
		Provenance    string `yaml:"provenance_state"`
		Archive       string `yaml:"authenticated_archive"`
		SHA256        string `yaml:"archive_sha256"`
		Evidence      string `yaml:"provenance_evidence"`
	} `yaml:"vicidial"`
	WebRTC struct {
		Name         string `yaml:"name"`
		Revision     string `yaml:"revision"`
		SHA256       string `yaml:"sha256"`
		SIPJSVersion string `yaml:"sipjs_version"`
		SIPJSSHA256  string `yaml:"sipjs_sha256"`
	} `yaml:"webrtc"`
	PJProject    Source `yaml:"pjproject"`
	NetTelnet    Source `yaml:"perl_net_telnet"`
	Verification struct {
		Runtime string `yaml:"runtime_state"`
		Fixture string `yaml:"fixture_id"`
		Digest  string `yaml:"evidence_digest"`
	} `yaml:"verification"`
	MariaDB RPM `yaml:"mariadb"`
	PHP     RPM `yaml:"php"`
	Perl    RPM `yaml:"perl"`
	Apache  RPM `yaml:"apache"`
	SRTP    RPM `yaml:"srtp"`
	Jansson RPM `yaml:"jansson"`
}

// Manifest is constructed only by strict decoding against the reviewed M0 policy.
// Its identity is SHA-256 of the original document bytes (including comments).
type Manifest struct {
	doc     manifestDocument
	digest  string
	origins map[string][]string
}

func Load(r io.Reader) (Manifest, error) {
	b, err := io.ReadAll(io.LimitReader(r, manifestLimit+1))
	if err != nil || len(b) > manifestLimit {
		return Manifest{}, errors.New("manifest: unreadable or exceeds 64 KiB")
	}
	dec := yaml.NewDecoder(bytes.NewReader(b))
	var node yaml.Node
	if err = dec.Decode(&node); err != nil {
		return Manifest{}, errors.New("manifest: malformed YAML")
	}
	var extra yaml.Node
	if dec.Decode(&extra) != io.EOF {
		return Manifest{}, errors.New("manifest: expected exactly one document")
	}
	if len(node.Content) != 1 {
		return Manifest{}, errors.New("manifest: empty document")
	}
	var identity manifestDocument
	if node.Decode(&identity) != nil {
		return Manifest{}, errors.New("manifest: invalid OS selection")
	}
	policy := manifestLeaves(identity.OS.Release)
	if policy == nil {
		return Manifest{}, errors.New("manifest: unsupported AlmaLinux release; select explicit 10.3 current or historical 10.2 policy")
	}
	seen := map[string]bool{}
	if err = validatePolicyNode(node.Content[0], "", seen, policy); err != nil {
		return Manifest{}, err
	}
	keys := make([]string, 0, len(policy))
	for k := range policy {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !seen[k] {
			return Manifest{}, fmt.Errorf("manifest: missing required field %s", k)
		}
	}
	var d manifestDocument
	if err = node.Decode(&d); err != nil {
		return Manifest{}, errors.New("manifest: invalid field type")
	}
	return Manifest{doc: d, digest: digestBytes(b)}, nil
}
func validateNode(n *yaml.Node, path string, seen map[string]bool) error {
	return validatePolicyNode(n, path, seen, approvedLeaves)
}
func validatePolicyNode(n *yaml.Node, path string, seen map[string]bool, policy map[string]leaf) error {
	if n.Anchor != "" || n.Kind == yaml.AliasNode {
		return errors.New("manifest: aliases and anchors are not supported")
	}
	if n.Kind == yaml.MappingNode {
		if n.ShortTag() != "!!map" {
			return errors.New("manifest: unsupported mapping tag")
		}
		keys := map[string]bool{}
		for i := 0; i < len(n.Content); i += 2 {
			k := n.Content[i]
			if k.Kind != yaml.ScalarNode || k.ShortTag() != "!!str" || k.Anchor != "" || keys[k.Value] {
				return errors.New("manifest: duplicate or ambiguous key")
			}
			keys[k.Value] = true
			next := k.Value
			if path != "" {
				next = path + "." + next
			}
			// Unknown names may contain sensitive input; never echo them.
			known := false
			for p := range policy {
				if p == next || len(p) > len(next) && p[:len(next)+1] == next+"." {
					known = true
					break
				}
			}
			if !known {
				return errors.New("manifest: unknown field")
			}
			if err := validatePolicyNode(n.Content[i+1], next, seen, policy); err != nil {
				return err
			}
		}
		return nil
	}
	expected, ok := policy[path]
	if !ok || n.Kind != yaml.ScalarNode {
		return errors.New("manifest: unsupported structure")
	}
	// Preserve existing unquoted numeric version scalars; quoted versions are equivalent.
	tag := n.ShortTag()
	if tag != "!!"+expected.tag && !(expected.tag == "float" && tag == "!!str") {
		return fmt.Errorf("manifest: incorrect type for %s", path)
	}
	if n.Value != expected.value {
		return fmt.Errorf("manifest: %s differs from reviewed M0 identity", path)
	}
	seen[path] = true
	return nil
}
func digestBytes(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func (m Manifest) Digest() string { return m.digest }
func (m Manifest) Status() string { return m.doc.Status }

// InstallationAuthorized deliberately never grants authority. The checked-in
// policy is candidate/TBD; future promotion requires independent maintainer review.
func (m Manifest) InstallationAuthorized() bool { return false }

// Packages preserves exact NEVRA and repository origin, not RPM signature proof.
// A later native RPM transaction must independently authenticate official keys.
func (m Manifest) Packages() map[string]RPM {
	return map[string]RPM{"mariadb": m.doc.MariaDB, "php": m.doc.PHP, "perl": m.doc.Perl, "apache": m.doc.Apache, "srtp": m.doc.SRTP, "jansson": m.doc.Jansson}
}

func (m Manifest) Release() string { return m.doc.OS.Release }
