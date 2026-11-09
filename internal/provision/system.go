package provision

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

// S02 prepares the M0 runtime layout and plan §5.4 footprint. Application
// binaries, configuration files, credentials and historical recordings are not
// created or replaced by this stage.
type ServiceIdentity struct {
	Name, Group, Home, Shell string
	Vendor                   bool
}

func ServiceIdentities() []ServiceIdentity {
	return []ServiceIdentity{
		{"apache", "apache", "/usr/share/httpd", "/sbin/nologin", true},
		{"mysql", "mysql", "/var/lib/mysql", "/sbin/nologin", true},
		{"asterisk", "asterisk", "/var/lib/asterisk", "/sbin/nologin", false},
	}
}

type ManagedDirectory struct {
	Path, Owner, Group, SELinuxType string
	Mode                            uint32
}

// Every operation addresses the directory inode only. Existing contents remain
// untouched. Modes and owners follow M0-08 base() and plan §5.4; monitorDONE is
// the plan's processing/archive location, separate from M0's live monitor path.
func ManagedDirectories() []ManagedDirectory {
	return []ManagedDirectory{
		{"/etc/almadial", "root", "root", "etc_t", 0700},
		{"/etc/almadial/secrets", "root", "root", "etc_t", 0700},
		{"/var/lib/almadial", "root", "root", "var_lib_t", 0700},
		{"/var/lib/almadial/artifacts", "root", "root", "var_lib_t", 0700},
		{"/var/lib/almadial/backups", "root", "root", "var_lib_t", 0700},
		{"/var/log/almadial", "root", "root", "var_log_t", 0700},
		{"/var/log/almadial/evidence", "root", "root", "var_log_t", 0700},
		{"/etc/asterisk", "root", "asterisk", "etc_t", 0750},
		{"/etc/asterisk/keys", "root", "asterisk", "etc_t", 0750},
		{"/var/lib/asterisk", "asterisk", "asterisk", "var_lib_t", 0750},
		{"/var/lib/asterisk/agi-bin", "asterisk", "asterisk", "bin_t", 0750},
		{"/var/lib/asterisk/sounds", "asterisk", "asterisk", "var_lib_t", 0750},
		{"/var/log/asterisk", "asterisk", "asterisk", "var_log_t", 0750},
		{"/var/spool/asterisk", "asterisk", "asterisk", "var_spool_t", 0750},
		{"/var/spool/asterisk/monitor", "asterisk", "asterisk", "var_spool_t", 0750},
		{"/var/spool/asterisk/monitorDONE", "asterisk", "asterisk", "var_spool_t", 0750},
	}
}

type AccountObservation struct {
	Name, Home, Shell string
	UID, GID          uint32
	Groups            []string
}
type GroupObservation struct {
	Name string
	GID  uint32
}
type DirectoryObservation struct {
	Path                  string
	Exists, Unsafe        bool
	UID, GID, Mode        uint32
	Device, Inode         uint64
	Label, DefaultContext string
}
type StorageObservation struct {
	Mount, Source, Filesystem, Options string
	AvailableBytes, AvailableInodes    uint64
}

// RecordingBudget has no defaults. Its identity is included in the immutable
// stage; a lab reserve never becomes a production capacity recommendation.
type RecordingBudget struct {
	Profile, Mount, Source, Filesystem string
	MinAvailableBytes, MinFreeInodes   uint64
}
type SystemSnapshot struct {
	RPM                 Snapshot
	SELinux, Policy     string
	LocalContextsSHA256 string
	SystemUIDMax        uint32
	SystemGIDMax        uint32
	Accounts            []AccountObservation
	Groups              []GroupObservation
	Directories         []DirectoryObservation
	Recording           StorageObservation
}

func canonicalSystem(s SystemSnapshot) SystemSnapshot {
	s.Accounts = append([]AccountObservation{}, s.Accounts...)
	for i := range s.Accounts {
		s.Accounts[i].Groups = append([]string{}, s.Accounts[i].Groups...)
		sort.Strings(s.Accounts[i].Groups)
	}
	s.Groups = append([]GroupObservation{}, s.Groups...)
	s.Directories = append([]DirectoryObservation{}, s.Directories...)
	sort.Slice(s.Accounts, func(i, j int) bool { return s.Accounts[i].Name < s.Accounts[j].Name })
	sort.Slice(s.Groups, func(i, j int) bool { return s.Groups[i].Name < s.Groups[j].Name })
	sort.Slice(s.Directories, func(i, j int) bool { return s.Directories[i].Path < s.Directories[j].Path })
	s.RPM = canonical(s.RPM)
	return s
}
func (s SystemSnapshot) Digest() string {
	s = canonicalSystem(s)
	// Free space changes independently of provisioning. Check the approved budget
	// afresh, while binding the mount identity rather than incidental free counts.
	s.Recording.AvailableBytes, s.Recording.AvailableInodes = 0, 0
	return hash(s)
}
func SystemPolicyDigest() string {
	return hash(struct {
		Version     string
		Identities  []ServiceIdentity
		Directories []ManagedDirectory
	}{"alm17-v1-default-policy-no-replacement", ServiceIdentities(), ManagedDirectories()})
}

const (
	CreateRequired         Status = "CREATE_REQUIRED"
	ReconciliationRequired Status = "RECONCILIATION_REQUIRED"
	Conflict               Status = "CONFLICT"
)

type SystemFinding struct {
	ID     string
	Status Status
	Detail string
}

var errSystem = errors.New("service/storage policy not verified; review bounded findings and approved identity")

func account(s SystemSnapshot, name string) (AccountObservation, bool) {
	for _, a := range s.Accounts {
		if a.Name == name {
			return a, true
		}
	}
	return AccountObservation{}, false
}
func group(s SystemSnapshot, name string) (GroupObservation, bool) {
	for _, g := range s.Groups {
		if g.Name == name {
			return g, true
		}
	}
	return GroupObservation{}, false
}
func directory(s SystemSnapshot, path string) (DirectoryObservation, bool) {
	for _, d := range s.Directories {
		if d.Path == path {
			return d, true
		}
	}
	return DirectoryObservation{}, false
}
func expectedContext(d ManagedDirectory) string { return "system_u:object_r:" + d.SELinuxType + ":s0" }
func identityValid(s SystemSnapshot, p ServiceIdentity, a AccountObservation, g GroupObservation) bool {
	if a.UID == 0 || a.UID > s.SystemUIDMax || g.GID == 0 || g.GID > s.SystemGIDMax || a.GID != g.GID || a.Home != p.Home || a.Shell != p.Shell || len(a.Groups) != 1 || a.Groups[0] != p.Group {
		return false
	}
	for _, other := range s.Accounts {
		if other.Name != a.Name && other.UID == a.UID {
			return false
		}
	}
	for _, other := range s.Groups {
		if other.Name != g.Name && other.GID == g.GID {
			return false
		}
	}
	return true
}
func directorySatisfied(s SystemSnapshot, p ManagedDirectory, d DirectoryObservation) bool {
	a, ok := account(s, p.Owner)
	g, gok := group(s, p.Group)
	return ok && gok && d.Exists && !d.Unsafe && d.UID == a.UID && d.GID == g.GID && d.Mode == p.Mode && d.DefaultContext == expectedContext(p) && d.Label == d.DefaultContext
}

// AssessSystem is pure and deterministic. Conflicting existing metadata is
// never automatically adopted: no ownership receipt or replacement authority is
// inferred from a familiar path name. Only a default-label repair is allowed on
// an otherwise compatible directory inode.
func AssessSystem(s SystemSnapshot, b RecordingBudget) []SystemFinding {
	out := []SystemFinding{}
	add := func(id string, st Status, detail string) { out = append(out, SystemFinding{id, st, detail}) }
	trusted := s.RPM.ValidateTrust() == nil
	for _, a := range Assess(s.RPM, nil, false) {
		if a.Status != Satisfied {
			trusted = false
		}
	}
	if !trusted {
		add("rpm", NotVerified, "accepted ALM-16 baseline required")
	} else {
		add("rpm", Satisfied, "qualified direct requirements and repository trust")
	}
	if s.SELinux != "Enforcing" || s.Policy != "targeted" || !shaPattern.MatchString(s.LocalContextsSHA256) {
		add("selinux", NotVerified, "Enforcing targeted policy and persistent mappings required")
	} else {
		add("selinux", Satisfied, "Enforcing targeted policy")
	}
	if s.SystemUIDMax == 0 || s.SystemGIDMax == 0 {
		add("nss", NotVerified, "system identity range unavailable")
	}
	for _, kind := range []string{"account", "group", "path"} {
		seen := map[string]bool{}
		var names []string
		switch kind {
		case "account":
			for _, a := range s.Accounts {
				names = append(names, a.Name)
			}
		case "group":
			for _, g := range s.Groups {
				names = append(names, g.Name)
			}
		case "path":
			for _, d := range s.Directories {
				names = append(names, d.Path)
			}
		}
		for _, name := range names {
			if seen[name] {
				add("duplicate."+kind, NotVerified, "ambiguous observation")
				break
			}
			seen[name] = true
		}
	}
	root, rok := account(s, "root")
	rg, rgok := group(s, "root")
	if !rok || !rgok || root.UID != 0 || root.GID != 0 || rg.GID != 0 {
		add("root", NotVerified, "root identity unavailable")
	}
	for _, p := range ServiceIdentities() {
		a, ok := account(s, p.Name)
		g, gok := group(s, p.Group)
		st := Satisfied
		if !ok || !gok {
			st = CreateRequired
			if p.Vendor {
				st = NotVerified
			}
			if ok {
				st = Conflict
			}
			if gok {
				for _, other := range s.Groups {
					if other.Name != g.Name && other.GID == g.GID {
						st = Conflict
					}
				}
				if g.GID == 0 || g.GID > s.SystemGIDMax {
					st = Conflict
				}
			}
		} else if !identityValid(s, p, a, g) {
			st = Conflict
		}
		add("identity."+p.Name, st, "vendor identities are preserved; numeric reassignment and extra privileges are rejected")
	}
	for i, p := range ManagedDirectories() {
		d, ok := directory(s, p.Path)
		st := Satisfied
		switch {
		case !ok:
			st = NotVerified
		case d.Unsafe:
			st = Conflict
		case d.DefaultContext != expectedContext(p):
			st = NotVerified
		case !d.Exists:
			st = Required
		case !directorySatisfied(s, p, d):
			a, aok := account(s, p.Owner)
			g, gok := group(s, p.Group)
			st = Conflict
			if aok && gok && d.UID == a.UID && d.GID == g.GID && d.Mode == p.Mode {
				st = ReconciliationRequired
			}
		}
		add(fmt.Sprintf("directory.%02d", i), st, "exact directory inode only; existing contents and foreign metadata preserved")
	}
	st := Satisfied
	if b.Profile == "" || b.Mount == "" || b.Source == "" || b.Filesystem == "" || b.MinAvailableBytes == 0 || b.MinFreeInodes == 0 {
		st = NotVerified
	} else if s.Recording.Mount != b.Mount || s.Recording.Source != b.Source || s.Recording.Filesystem != b.Filesystem || (b.Filesystem != "xfs" && b.Filesystem != "ext4") || !contains(splitOptions(s.Recording.Options), "rw") {
		st = Conflict
	} else if s.Recording.AvailableBytes < b.MinAvailableBytes || s.Recording.AvailableInodes < b.MinFreeInodes {
		st = Incompatible
	}
	add("recording.budget", st, "explicit free-byte/inode reserve on the reviewed writable local mount")
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
func systemReady(s SystemSnapshot, b RecordingBudget) bool {
	for _, f := range AssessSystem(s, b) {
		if f.Status != Satisfied && f.Status != Required && f.Status != CreateRequired && f.Status != ReconciliationRequired {
			return false
		}
	}
	return true
}
func systemSatisfied(s SystemSnapshot, b RecordingBudget) bool {
	for _, f := range AssessSystem(s, b) {
		if f.Status != Satisfied {
			return false
		}
	}
	return true
}
func jsonString(v any) string { b, _ := json.Marshal(v); return string(b) }
