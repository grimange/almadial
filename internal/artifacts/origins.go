package artifacts

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"path"
	"strings"
)

// WithAcquisitionOrigins attaches a release-reviewed location policy without
// changing the accepted source manifest, versions, hashes or plan identity.
// This is configuration, never an assertion of third-party distribution rights.
// Only exact-byte archives are supported; SVN repackaging is not permitted.
func WithAcquisitionOrigins(m Manifest, input io.Reader) (Manifest, error) {
	var p struct {
		Schema         int      `json:"schema"`
		ManifestSHA256 string   `json:"manifest_sha256"`
		Stores         []string `json:"private_content_addressed_stores"`
	}
	b, err := io.ReadAll(io.LimitReader(input, manifestLimit+1))
	if err != nil || len(b) > manifestLimit {
		return Manifest{}, errors.New("origins: unreadable or oversized policy")
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if dec.Decode(&p) != nil || dec.Decode(new(any)) != io.EOF || p.Schema != 1 || m.digest == "" || p.ManifestSHA256 != m.Digest() || len(p.Stores) == 0 || len(p.Stores) > 4 {
		return Manifest{}, errors.New("origins: expected bounded policy bound to the exact source manifest")
	}
	seen := map[string]bool{}
	for _, base := range p.Stores {
		u, e := url.Parse(base)
		if !approvedHTTPS(base) || e != nil || strings.HasSuffix(base, "/") || path.Clean(u.Path) != u.Path || seen[base] {
			return Manifest{}, errors.New("origins: unique canonical HTTPS store required")
		}
		seen[base] = true
	}
	m.origins = map[string][]string{}
	for _, name := range installComponents {
		s, e := m.artifact(name)
		if e != nil {
			return Manifest{}, e
		}
		for _, base := range p.Stores {
			m.origins[name] = append(m.origins[name], base+"/sha256/"+s.sha256+"/"+name+".artifact")
		}
	}
	return m, nil
}

// AcquisitionPolicy binds an explicitly selected location to the existing exact
// manifest. Location selection cannot change approved archive identities.
func AcquisitionPolicy(m Manifest, store string) ([]byte, error) {
	b, err := json.Marshal(struct {
		Schema         int      `json:"schema"`
		ManifestSHA256 string   `json:"manifest_sha256"`
		Stores         []string `json:"private_content_addressed_stores"`
	}{1, m.Digest(), []string{store}})
	if err != nil {
		return nil, err
	}
	if _, err = WithAcquisitionOrigins(m, bytes.NewReader(b)); err != nil {
		return nil, err
	}
	return b, nil
}
