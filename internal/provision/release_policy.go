package provision

import (
	_ "embed"
	"encoding/json"
)

// ReleasePackagePolicy is one reviewed distribution set, not a version solver.
// DNF resolves only these repositories; the complete approved payload set bounds
// dependencies and replacements. Historical 10.2 bytes and digests stay intact.
type ReleasePackagePolicy struct {
	Version      string            `json:"version"`
	Release      string            `json:"release"`
	ImageSHA256  string            `json:"image_sha256"`
	Repositories map[string]string `json:"repositories"`
	Packages     []Package         `json:"packages"`
	Closure      []Candidate       `json:"closure"`
	Upgrades     []Upgrade         `json:"upgrades"`
}

//go:embed packages-alma103.json
var releasePolicyJSON []byte

func currentPackagePolicy() *ReleasePackagePolicy {
	var p ReleasePackagePolicy
	if json.Unmarshal(releasePolicyJSON, &p) != nil {
		return nil
	}
	return &p
}
func PackagePolicyDigest(release string) string {
	if release == "10.2" {
		return PolicyDigest()
	}
	if release == "10.3" {
		return hash(currentPackagePolicy())
	}
	return ""
}
func packagesFor(release string) []Package {
	if release == "10.3" {
		return currentPackagePolicy().Packages
	}
	if release == "10.2" {
		return Packages()
	}
	return nil
}

// RequiredPackages returns the exact direct RPM requirements for diagnostics.
// The returned copy cannot change the immutable provisioning policy.
func RequiredPackages(release string) []Package {
	return append([]Package(nil), packagesFor(release)...)
}
func upgradesFor(release string) []Upgrade {
	if release == "10.3" {
		return currentPackagePolicy().Upgrades
	}
	return Upgrades()
}
func approvedPayload(release string, c Candidate) bool {
	if release == "10.2" {
		return true
	}
	if release != "10.3" {
		return false
	}
	for _, p := range currentPackagePolicy().Closure {
		if p == c {
			return true
		}
	}
	return false
}

// PackageSelection is part of the immutable installation request. BeforeSHA256
// binds the observed host before any RPM key import or package transaction.
type PackageSelection struct {
	PolicySHA256 string
	Transaction  Transaction
}

func resolutionSnapshot(s Snapshot) Snapshot {
	s = canonical(s)
	if !contains(s.KeyHashes, epelKeySHA) {
		s.KeyHashes = append(s.KeyHashes, epelKeySHA)
		s.Installed = append(s.Installed, epelKeyRPM)
	}
	return canonical(s)
}
func (p PackageSelection) validate(s Snapshot) error {
	if s.Release != "10.3" || s.validateTrust(false) != nil || p.PolicySHA256 != PackagePolicyDigest(s.Release) || p.Transaction.BeforeSHA256 != s.Digest() {
		return errPolicy
	}
	t := p.Transaction
	trusted := resolutionSnapshot(s)
	t.BeforeSHA256 = trusted.Digest()
	return t.Validate(trusted)
}
