// Package provision implements only S01's qualified RPM dependency contract.
// Native DNF resolves dependencies. No code here compares or solves RPM versions.
package provision

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
)

// M0-08 replay's direct requests, normalized using M0-02 metadata and observed
// RPM epochs (notably MariaDB=3, Perl/Time-HiRes=4, mod_ssl=1). Source packages
// and optional M0-02 investigations are deliberately absent.
//
//go:embed packages.json
var packageJSON []byte

// Seven coupled util-linux RPM updates needed by the pinned M0 libuuid-devel
// on the recorded clean 10.2 image. This exception is an exact reviewed set,
// not permission for arbitrary upgrades. Native RPM confirms upgrade direction.
//
//go:embed upgrades.json
var upgradeJSON []byte

type Upgrade struct {
	Before Package   `json:"before"`
	After  Candidate `json:"after"`
}

func Upgrades() []Upgrade { var p []Upgrade; _ = json.Unmarshal(upgradeJSON, &p); return p }

// Official EPEL 10 public key retained and SHA-256 qualified by M0-08.
// Source: https://dl.fedoraproject.org/pub/epel/RPM-GPG-KEY-EPEL-10
//
//go:embed epel.asc
var epelPublicKey string

type Package struct {
	Name       string `json:"name"`
	EVR        string `json:"evr"`
	Arch       string `json:"arch"`
	Repository string `json:"repository"`
}

func (p Package) NEVRA() string { return p.Name + "-" + p.EVR + "." + p.Arch }

type Repository struct {
	ID          string `json:"id"`
	Source      string `json:"source"`
	KeySHA256   string `json:"key_sha256"`
	Fingerprint string `json:"fingerprint"`
	Signer      string `json:"signer"`
	Enabled     bool   `json:"enabled"`
}

const almaKeySHA = "de57c3c0a2dcde88ff5b894e74335f55a0ec074815f36d472270d937c16f077b"
const epelKeySHA = "de390fc168eae5ab2852e9e93d34a0b9ddf05cf9ce90ee28d97de26a4b1f6b93"
const epelSource = "https://dl.fedoraproject.org/pub/epel/10.2/Everything/x86_64/"
const almaKeyPath = "file:///etc/pki/rpm-gpg/RPM-GPG-KEY-AlmaLinux-10"

func Repositories() []Repository {
	out := []Repository{}
	for _, id := range []string{"appstream", "baseos", "crb", "extras"} {
		out = append(out, Repository{id, "https://mirrors.almalinux.org/mirrorlist/10/" + id, almaKeySHA, "EE6DB7B98F5BF5EDD9DA0DE5DEE5C11CC2A1E572", "dee5c11cc2a1e572", true})
	}
	return append(out, Repository{"epel-almadial", epelSource, epelKeySHA, "7D8D15CBFC4E62688591FB2633D98517E37ED158", "33d98517e37ed158", false})
}
func Packages() []Package { var p []Package; _ = json.Unmarshal(packageJSON, &p); return p }

// These two exact RPM archives were retained and hash-qualified in M0-08.
func PinnedRPMHash(name string) string {
	switch name {
	case "perl-Switch":
		return "fd470d089ae27612fdf331345dafa46f893f25da95a257360857dc49025bdf48"
	case "perl-Crypt-Eksblowfish":
		return "5cdeb146add4bf332e9ff3b0eba6f11149d0efddb83b0049b3329f295218f3f8"
	}
	return ""
}
func PolicyDigest() string {
	return hash(struct {
		Version      string
		Packages     []Package
		Repositories []Repository
		Upgrades     []Upgrade
	}{"alm16-v3-reviewed-util-linux-updates", Packages(), Repositories(), Upgrades()})
}
func hash(v any) string {
	b, _ := json.Marshal(v)
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

var shaPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
var namePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9+_.-]{0,127}$`)
var evrPattern = regexp.MustCompile(`^[0-9]+:[A-Za-z0-9+_.~^]+-[A-Za-z0-9+_.~^]+$`)
var errPolicy = errors.New("RPM policy not verified; inspect repository trust, package identity and approved transaction")

func repository(id string) (Repository, bool) {
	for _, r := range Repositories() {
		if r.ID == id {
			return r, true
		}
	}
	return Repository{}, false
}
func validPackage(p Package) bool {
	_, ok := repository(p.Repository)
	return ok && namePattern.MatchString(p.Name) && evrPattern.MatchString(p.EVR) && (p.Arch == "x86_64" || p.Arch == "noarch")
}
func sortedPackages(p []Package) []Package {
	out := append([]Package{}, p...)
	sort.Slice(out, func(i, j int) bool { return out[i].NEVRA()+out[i].Repository < out[j].NEVRA()+out[j].Repository })
	return out
}
