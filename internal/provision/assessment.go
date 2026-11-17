package provision

import (
	"reflect"
	"sort"
	"strings"
)

type RepoObservation struct {
	ID           string   `json:"id"`
	Enabled      bool     `json:"enabled"`
	Mirrorlist   string   `json:"mirrorlist"`
	Metalink     string   `json:"metalink"`
	BaseURLs     []string `json:"baseurls"`
	Keys         []string `json:"keys"`
	GPGCheck     bool     `json:"gpgcheck"`
	TLSVerify    bool     `json:"tls_verify"`
	RepoGPGCheck bool     `json:"repo_gpgcheck"`
}
type Installed struct {
	Name   string `json:"name"`
	EVR    string `json:"evr"`
	Arch   string `json:"arch"`
	Signer string `json:"signer"`
}

func (p Installed) NEVRA() string { return p.Name + "-" + p.EVR + "." + p.Arch }

type Snapshot struct {
	Host              string            `json:"host"` // SHA-256 of machine-id, never a hostname or address
	OS                string            `json:"os"`
	Release           string            `json:"release"`
	Arch              string            `json:"arch"`
	Repositories      []RepoObservation `json:"repositories"`
	Installed         []Installed       `json:"installed"`
	KeyHashes         []string          `json:"key_hashes"` // public keys actually imported in RPM
	AlmaKeyFileSHA256 string            `json:"alma_key_file_sha256"`
	HistorySHA256     string            `json:"history_sha256"` // canonical read-only DNF transaction records
	HistoryComplete   bool              `json:"history_complete"`
}

func canonical(s Snapshot) Snapshot {
	b := s
	b.Repositories = append([]RepoObservation{}, s.Repositories...)
	for i := range b.Repositories {
		b.Repositories[i].BaseURLs = append([]string{}, b.Repositories[i].BaseURLs...)
		b.Repositories[i].Keys = append([]string{}, b.Repositories[i].Keys...)
		sort.Strings(b.Repositories[i].BaseURLs)
		sort.Strings(b.Repositories[i].Keys)
	}
	sort.Slice(b.Repositories, func(i, j int) bool { return b.Repositories[i].ID < b.Repositories[j].ID })
	b.Installed = append([]Installed{}, s.Installed...)
	for i := range b.Installed {
		if b.Installed[i].Name == "gpg-pubkey" && b.Installed[i].Arch == "" {
			b.Installed[i].Arch = "(none)"
		}
	}
	sort.Slice(b.Installed, func(i, j int) bool { return b.Installed[i].NEVRA() < b.Installed[j].NEVRA() })
	b.KeyHashes = append([]string{}, s.KeyHashes...)
	sort.Strings(b.KeyHashes)
	return b
}
func (s Snapshot) Digest() string { return hash(canonical(s)) }
func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

// ValidateTrust permits disabled unrelated definitions, but rejects any active
// source outside the four M0 Alma repos. EPEL is transaction-local and restricted
// to the six qualified Perl packages. No repository files are ever rewritten.
func (s Snapshot) ValidateTrust() error {
	return s.validateTrust(true)
}

// The key-preparation action alone may start without the pinned EPEL key.
// All other host/repository requirements remain identical.
func (s Snapshot) validateTrust(requireEPEL bool) error {
	if !shaPattern.MatchString(s.Host) || s.OS != "almalinux" || (s.Release != "10.2" && s.Release != "10.3") || s.Arch != "x86_64" || !shaPattern.MatchString(s.HistorySHA256) || !s.HistoryComplete || s.AlmaKeyFileSHA256 != almaKeySHA || !contains(s.KeyHashes, almaKeySHA) || (requireEPEL && !contains(s.KeyHashes, epelKeySHA)) {
		return errPolicy
	}
	seen := map[string]bool{}
	for _, r := range s.Repositories {
		if seen[r.ID] {
			return errPolicy
		}
		seen[r.ID] = true
		if !r.Enabled {
			continue
		}
		p, ok := repository(r.ID)
		if !ok || !p.Enabled || !r.GPGCheck || !r.TLSVerify || r.Metalink != "" || r.Mirrorlist != p.Source || len(r.BaseURLs) != 0 || len(r.Keys) != 1 || r.Keys[0] != almaKeyPath {
			return errPolicy
		}
	}
	for _, r := range Repositories() {
		if r.Enabled {
			found := false
			for _, o := range s.Repositories {
				if o.ID == r.ID && o.Enabled {
					found = true
				}
			}
			if !found {
				return errPolicy
			}
		}
	}
	ids := map[string]bool{}
	for _, p := range canonical(s).Installed {
		if p.NEVRA() == epelKeyRPM.NEVRA() && !contains(s.KeyHashes, epelKeySHA) {
			return errPolicy
		}
		if !namePattern.MatchString(p.Name) || !evrPattern.MatchString(p.EVR) || p.Arch == "" || ids[p.NEVRA()] {
			return errPolicy
		}
		ids[p.NEVRA()] = true
	}
	return nil
}

var epelKeyRPM = Installed{"gpg-pubkey", "0:e37ed158-65785fa9", "(none)", ""}

func verifyKeyImport(before, after Snapshot) error {
	if before.validateTrust(false) != nil || after.ValidateTrust() != nil {
		return errPolicy
	}
	expected := canonical(before)
	if !contains(expected.KeyHashes, epelKeySHA) {
		for _, p := range expected.Installed {
			if p.NEVRA() == epelKeyRPM.NEVRA() {
				return errPolicy
			}
		}
		expected.KeyHashes = append(expected.KeyHashes, epelKeySHA)
		expected.Installed = append(expected.Installed, epelKeyRPM)
	}
	if expected.Digest() != after.Digest() {
		return errPolicy
	}
	return nil
}

type Status string

const (
	Satisfied    Status = "SATISFIED"
	Required     Status = "REQUIRED"
	Incompatible Status = "INCOMPATIBLE"
	Unavailable  Status = "UNAVAILABLE"
	NotVerified  Status = "NOT_VERIFIED"
)

type Assessment struct {
	Package Package `json:"package"`
	Status  Status  `json:"status"`
}

// Assess uses exact M0 EVR equality, including epoch. RPM comparison and closure
// selection belong to DNF. No implicit upgrades/downgrades are approved here.
func Assess(s Snapshot, candidates []Package, resolved bool) []Assessment {
	out := []Assessment{}
	trusted := s.ValidateTrust() == nil
	packages := packagesFor(s.Release)
	if packages == nil {
		// Retain the historical diagnostic shape for incomplete inventories;
		// failed trust keeps every entry NOT_VERIFIED and cannot authorize RPMs.
		packages = Packages()
	}
	for _, p := range packages {
		a := Assessment{p, NotVerified}
		if trusted {
			installed := false
			for _, i := range s.Installed {
				if i.Name == p.Name {
					installed = true
					a.Status = Incompatible
					r, _ := repository(p.Repository)
					if i.EVR == p.EVR && i.Arch == p.Arch && i.Signer == r.Signer {
						a.Status = Satisfied
					}
					if a.Status == Incompatible {
						break
					}
				}
			}
			if !installed && resolved {
				a.Status = Unavailable
				for _, c := range candidates {
					if c == p {
						a.Status = Required
						break
					}
				}
			}
		}
		out = append(out, a)
	}
	return out
}

type Transaction struct {
	BeforeSHA256 string      `json:"before_sha256"`
	Install      []Candidate `json:"install"`
	Remove       []Package   `json:"remove"`
}
type Candidate struct {
	Package
	SHA256 string `json:"sha256"`
}

func normalize(t Transaction) Transaction {
	t.Install = append([]Candidate{}, t.Install...)
	sort.Slice(t.Install, func(i, j int) bool {
		return t.Install[i].NEVRA()+t.Install[i].Repository < t.Install[j].NEVRA()+t.Install[j].Repository
	})
	t.Remove = sortedPackages(t.Remove)
	return t
}
func (t Transaction) Digest() string { return hash(normalize(t)) }
func (t Transaction) Validate(s Snapshot) error {
	if s.ValidateTrust() != nil || t.BeforeSHA256 != s.Digest() || len(t.Remove) > len(upgradesFor(s.Release)) || len(t.Install) > 2048 {
		return errPolicy
	}
	removed := map[string]bool{}
	for _, old := range t.Remove {
		if removed[old.Name] {
			return errPolicy
		}
		approved := false
		for _, u := range upgradesFor(s.Release) {
			if old != u.Before {
				continue
			}
			for _, c := range t.Install {
				if c == u.After {
					approved = true
				}
			}
		}
		if !approved {
			return errPolicy
		}
		found := false
		for _, i := range s.Installed {
			if i.Name == old.Name && i.EVR == old.EVR && i.Arch == old.Arch && i.Signer == "dee5c11cc2a1e572" {
				found = true
			}
		}
		if !found {
			return errPolicy
		}
		removed[old.Name] = true
	}
	cs := []Package{}
	seen := map[string]bool{}
	for _, c := range t.Install {
		if !validPackage(c.Package) || !shaPattern.MatchString(c.SHA256) || seen[c.Name] || !strings.Contains(c.EVR, ".el10") {
			return errPolicy
		}
		seen[c.Name] = true
		if !approvedPayload(s.Release, c) {
			return errPolicy
		}
		if pinned := PinnedRPMHash(c.Name); s.Release == "10.2" && pinned != "" && c.SHA256 != pinned {
			return errPolicy
		}
		for _, i := range s.Installed {
			if i.Name == c.Name {
				if !removed[c.Name] {
					return errPolicy
				}
				matched := false
				for _, old := range t.Remove {
					if old.Name == i.Name && old.EVR == i.EVR && old.Arch == i.Arch {
						matched = true
					}
				}
				if !matched {
					return errPolicy
				}
			}
		} // No unpaired erasure, reinstall, downgrade or unrelated upgrade.
		if c.Repository == "epel-almadial" {
			qualified := false
			for _, p := range packagesFor(s.Release) {
				if p == c.Package {
					qualified = true
				}
			}
			if !qualified {
				return errPolicy
			}
		}
		cs = append(cs, c.Package)
	}
	for _, a := range Assess(s, cs, true) {
		if a.Status != Satisfied && a.Status != Required {
			return errPolicy
		}
	}
	return nil
}

// Verify independently checks the complete inventory delta, required direct RPMs,
// signatures and unchanged repository/key policy. A successful process is not proof.
func Verify(before, after Snapshot, t Transaction) error {
	if t.Validate(before) != nil || after.ValidateTrust() != nil || before.Host != after.Host || !reflect.DeepEqual(canonical(before).Repositories, canonical(after).Repositories) || !reflect.DeepEqual(canonical(before).KeyHashes, canonical(after).KeyHashes) {
		return errPolicy
	}
	// Headers alone cannot prove that DNF committed its transaction and final
	// scriptlets. A crash after RPM writes but before history commit remains
	// uncertain even when every requested header is present.
	if len(t.Install) > 0 && before.HistorySHA256 == after.HistorySHA256 {
		return errPolicy
	}
	expected := map[string]Installed{}
	for _, i := range before.Installed {
		expected[i.NEVRA()] = i
	}
	for _, old := range t.Remove {
		delete(expected, old.NEVRA())
	}
	for _, c := range t.Install {
		r, _ := repository(c.Repository)
		i := Installed{c.Name, c.EVR, c.Arch, r.Signer}
		expected[i.NEVRA()] = i
	}
	if len(expected) != len(after.Installed) {
		return errPolicy
	}
	for _, i := range after.Installed {
		if want, ok := expected[i.NEVRA()]; !ok || want != i {
			return errPolicy
		}
	}
	for _, a := range Assess(after, nil, false) {
		if a.Status != Satisfied {
			return errPolicy
		}
	}
	return nil
}
