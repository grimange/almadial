package provision

import (
	"path"
	"strings"
)

// These observations carry no file contents, SQL rows, credentials or CLI output.
type VICIdialFileObservation struct {
	Target, Kind, SHA256, Owner, Group, Label string
	Mode                                      uint32
	Links                                     uint64
}

type VICIdialFilesAssessment struct {
	State             string
	Missing, Verified []string
	Identity          string
}

func vicidialTarget(p string) bool {
	if path.Clean(p) != p || strings.ContainsAny(p, "\x00\r\n\\:") {
		return false
	}
	for _, root := range []string{vicidialWeb, vicidialHome, "/var/lib/asterisk/agi-bin", "/var/lib/asterisk/sounds", "/usr/share/doc/almadial-vicidial"} {
		if strings.HasPrefix(p, root+"/") {
			return true
		}
	}
	return p == "/usr/local/share/perl5/5.40/Asterisk.pm" || p == "/usr/local/share/perl5/5.40/Asterisk/AGI.pm"
}

// AssessVICIdialFiles recognizes exact approved bytes and metadata. A missing
// file can be completed, but a foreign inode/content/mode is never overwritten.
// Observations outside this explicit manifest are preserved and do not authorize
// deletion. Native collection must additionally verify every parent directory.
func AssessVICIdialFiles(files []VICIdialFile, observed []VICIdialFileObservation) (VICIdialFilesAssessment, error) {
	a := VICIdialFilesAssessment{State: "ABSENT", Missing: []string{}, Verified: []string{}}
	if len(files) == 0 {
		return a, errVICIdial
	}
	byPath := map[string]VICIdialFileObservation{}
	for _, f := range observed {
		if _, ok := byPath[f.Target]; ok {
			return a, errVICIdial
		}
		byPath[f.Target] = f
	}
	previous := ""
	for _, f := range files {
		if !vicidialTarget(f.Target) || f.Target <= previous || !shaPattern.MatchString(f.SourceSHA256) || !shaPattern.MatchString(f.SHA256) || f.Owner != "root" || f.Mode&0022 != 0 || f.Mode&0400 == 0 || f.Mode&^0777 != 0 || f.Group == "" || f.Label == "" || f.Source == "" || (f.Writer != "upstream-r4037" && f.Writer != "almadial-m0-procps") {
			return a, errVICIdial
		}
		previous = f.Target
		o, ok := byPath[f.Target]
		if !ok {
			// Missing observations are incomplete inspection, never absent files.
			return a, errVICIdial
		}
		if o.Kind == "absent" {
			a.Missing = append(a.Missing, f.Target)
			continue
		}
		if o.Kind != "regular" || o.SHA256 != f.SHA256 || o.Owner != f.Owner || o.Group != f.Group || o.Mode != f.Mode || o.Label != f.Label || o.Links != 1 {
			a.State = "CONFLICT"
			return a, errVICIdial
		}
		a.Verified = append(a.Verified, f.Target)
	}
	if len(a.Missing) == 0 {
		a.State = "COMPLETE"
	} else if len(a.Verified) != 0 {
		a.State = "PARTIAL"
	}
	a.Identity = hash(struct {
		Files   []VICIdialFile
		Missing []string
	}{files, a.Missing})
	return a, nil
}

// Runtime observations are separate from file installation: source extraction,
// file presence or a PHP response alone cannot satisfy S05.
type VICIdialRuntime struct {
	Database, Asterisk, Perl, PHP, Apache, Configuration  string
	DatabaseReadOnlyVerified, RegistrationVerified        bool
	AMIAuthenticated, AMILocalOnly, HTTPLocalOnly         bool
	NoSIPTransports, SELinuxEnforcing                     bool
	GeneratorVerified, ReloadVerified, ForeignEditsAbsent bool
	WebEntryPoints, ProtectedCredentials                  bool
	SchedulingDeferred                                    bool
}

func (r VICIdialRuntime) Satisfied() bool {
	for _, v := range []string{r.Database, r.Asterisk, r.Perl, r.PHP, r.Apache, r.Configuration} {
		if !shaPattern.MatchString(v) {
			return false
		}
	}
	return r.DatabaseReadOnlyVerified && r.RegistrationVerified && r.AMIAuthenticated && r.AMILocalOnly && r.HTTPLocalOnly && r.NoSIPTransports && r.SELinuxEnforcing && r.GeneratorVerified && r.ReloadVerified && r.ForeignEditsAbsent && r.WebEntryPoints && r.ProtectedCredentials && r.SchedulingDeferred
}
