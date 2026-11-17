//go:build linux

package provision

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/grimange/almadial/internal/artifacts"
	"github.com/grimange/almadial/internal/execution"
)

func vicidialFixture() VICIdialSource {
	return VICIdialSource{files: map[string][]byte{
		"www/vicidial/admin.php":             []byte("<?php /* synthetic, never executed */ ?>"),
		"www/agc/vicidial.php":               []byte("<?php /* synthetic agent */ ?>"),
		"bin/AST_update.pl":                  []byte("#!/usr/bin/perl\n# synthetic fixture\n"),
		"agi/test.agi":                       []byte("#!/usr/bin/perl\n# synthetic AGI\n"),
		"sounds/test.wav":                    []byte("synthetic non-audio fixture"),
		"docs/LICENSE.txt":                   []byte("synthetic fixture notice"),
		"extras/MySQL_AST_CREATE_tables.sql": []byte("must never be deployed by S05"),
		"extras/first_server_install.sql":    []byte("must never be imported by S05"),
	}}
}

func TestVICIdialDestinationManifest(t *testing.T) {
	s := vicidialFixture()
	f, content, e := s.deployment()
	if e != nil || len(f) != 6 {
		t.Fatal(e, f)
	}
	previous := ""
	for _, v := range f {
		if v.Target <= previous || !vicidialTarget(v.Target) || v.SHA256 != rawSHA(content[v.Target]) || strings.HasSuffix(v.Target, ".sql") {
			t.Fatal(v)
		}
		previous = v.Target
	}
	before := s.Digest()
	inv := s.Inventory()
	inv[0].SHA256 = hash("forged observation")
	content[f[0].Target][0] ^= 1
	f[0].Mode = 0777
	if before != s.Digest() {
		t.Fatal("public observations altered retained source")
	}
	if (VICIdialSource{}).Digest() != "" {
		t.Fatal("empty source became verified")
	}
}

func TestVICIdialRetainedRepresentationBinding(t *testing.T) {
	s := vicidialFixture()
	before := s.Digest()
	s.archive = []byte("synthetic alternative representation")
	s.archiveSHA = rawSHA(s.archive)
	if !s.retainedArchiveVerified() || s.Digest() != before {
		t.Fatal("verified representation changed canonical source identity")
	}
	c := s.archiveInstallCommand()
	if c.Args[2] == vicidialFilesAdapter || !strings.Contains(c.Args[2], s.archiveSHA) {
		t.Fatal("installation did not bind verified representation")
	}
	s.archive[0] ^= 1
	if s.retainedArchiveVerified() || s.archiveInstallCommand().Args[2] != vicidialFilesAdapter {
		t.Fatal("changed retained archive accepted")
	}
	s.archiveSHA = databaseArchiveSHA
	if s.archiveInstallCommand().Args[2] != vicidialFilesAdapter {
		t.Fatal("historical adapter changed")
	}
}

func validVICIdialObservations(f []VICIdialFile) []VICIdialFileObservation {
	o := []VICIdialFileObservation{}
	for _, v := range f {
		o = append(o, VICIdialFileObservation{v.Target, "regular", v.SHA256, v.Owner, v.Group, v.Label, v.Mode, 1})
	}
	return o
}

func TestVICIdialFileAssessment(t *testing.T) {
	for _, name := range []string{"fresh", "complete", "partial", "foreign preserved", "missing observation", "wrong content", "wrong owner", "wrong group", "wrong mode", "wrong label", "symlink", "directory", "hardlink", "duplicate observation", "duplicate target", "unsafe destination", "unexpected destination", "unsafe manifest mode"} {
		t.Run(name, func(t *testing.T) {
			f, _ := vicidialFixture().Deployment()
			o := validVICIdialObservations(f)
			want := "COMPLETE"
			switch name {
			case "fresh":
				for j := range o {
					o[j] = VICIdialFileObservation{Target: f[j].Target, Kind: "absent"}
				}
				want = "ABSENT"
			case "partial":
				o[0] = VICIdialFileObservation{Target: f[0].Target, Kind: "absent"}
				want = "PARTIAL"
			case "foreign preserved":
				o = append(o, VICIdialFileObservation{Target: "/var/www/html/operator.txt", Kind: "regular", SHA256: hash("operator data")})
			case "missing observation":
				o = o[1:]
			case "wrong content":
				o[0].SHA256 = hash("foreign edits")
			case "wrong owner":
				o[0].Owner = "apache"
			case "wrong group":
				o[0].Group = "operator"
			case "wrong mode":
				o[0].Mode = 0777
			case "wrong label":
				o[0].Label = "unconfined_t"
			case "symlink":
				o[0].Kind = "symlink"
			case "directory":
				o[0].Kind = "directory"
			case "hardlink":
				o[0].Links = 2
			case "duplicate observation":
				o = append(o, o[0])
			case "duplicate target":
				f = append(f, f[0])
			case "unsafe destination":
				f[0].Target = "/var/www/html/../secrets"
			case "unexpected destination":
				f[0].Target = "/etc/passwd"
			case "unsafe manifest mode":
				f[0].Mode = 04755
			}
			a, e := AssessVICIdialFiles(f, o)
			if name == "fresh" || name == "complete" || name == "partial" || name == "foreign preserved" {
				if e != nil || a.State != want || !shaPattern.MatchString(a.Identity) {
					t.Fatal(a, e)
				}
			} else if e == nil {
				t.Fatal("unsafe state accepted", a)
			}
		})
	}
}

func zipFixture(t *testing.T, names []string, modes []os.FileMode) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for i, name := range names {
		h := &zip.FileHeader{Name: name, Method: zip.Store}
		if modes != nil {
			h.SetMode(modes[i])
		}
		w, e := z.CreateHeader(h)
		if e != nil {
			t.Fatal(e)
		}
		if !strings.HasSuffix(name, "/") {
			if _, e = w.Write([]byte("synthetic data")); e != nil {
				t.Fatal(e)
			}
		}
	}
	if e := z.Close(); e != nil {
		t.Fatal(e)
	}
	return b.Bytes()
}

func TestVICIdialZIPPaths(t *testing.T) {
	for _, name := range []string{"valid", "traversal", "absolute", "wrong root", "duplicate", "symlink", "file parent", "backslash", "control"} {
		t.Run(name, func(t *testing.T) {
			names := []string{vicidialRoot + "www/agc/test.php"}
			var modes []os.FileMode
			switch name {
			case "traversal":
				names[0] = vicidialRoot + "../escape"
			case "absolute":
				names[0] = "/etc/passwd"
			case "wrong root":
				names[0] = "r4038/www/test.php"
			case "duplicate":
				names = append(names, names[0])
			case "symlink":
				modes = []os.FileMode{os.ModeSymlink | 0777}
			case "file parent":
				names = append(names, names[0]+"/child")
			case "backslash":
				names[0] = vicidialRoot + "www\\escape"
			case "control":
				names[0] = vicidialRoot + "file\nname"
			}
			f, e := readVICIdialZIP(context.Background(), zipFixture(t, names, modes))
			if name == "valid" {
				if e != nil || len(f) != 1 {
					t.Fatal(e)
				}
			} else if e == nil {
				t.Fatal("unsafe ZIP accepted")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := readVICIdialZIP(ctx, zipFixture(t, []string{vicidialRoot + "version"}, nil)); e == nil {
		t.Fatal("cancellation ignored")
	}
	if _, e := readVICIdialZIP(context.Background(), []byte("corrupt")); e == nil {
		t.Fatal("corruption accepted")
	}
}

func TestVICIdialRejectUnverifiedArchiveAndDerivation(t *testing.T) {
	f, e := os.Open("../../manifests/stack.lock.yaml")
	if e != nil {
		t.Fatal(e)
	}
	m, e := artifacts.Load(f)
	f.Close()
	if e != nil {
		t.Fatal(e)
	}
	p := filepath.Join(t.TempDir(), "unapproved.zip")
	if e := os.WriteFile(p, zipFixture(t, []string{vicidialRoot + "version"}, nil), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := LoadVICIdialSource(context.Background(), m, p); e == nil {
		t.Fatal("unapproved source accepted")
	}
	if _, e := vicidialProcps("bin/ADMIN_keepalive_ALL.pl", []byte("wrong source")); e == nil {
		t.Fatal("unapproved patch input accepted")
	}
}

func TestVICIdialRuntimeRequiredPostconditions(t *testing.T) {
	r := VICIdialRuntime{Database: hash("db"), Asterisk: hash("ast"), Perl: hash("perl"), PHP: hash("php"), Apache: hash("httpd"), Configuration: hash("config"), DatabaseReadOnlyVerified: true, RegistrationVerified: true, AMIAuthenticated: true, AMILocalOnly: true, HTTPLocalOnly: true, NoSIPTransports: true, SELinuxEnforcing: true, GeneratorVerified: true, ReloadVerified: true, ForeignEditsAbsent: true, WebEntryPoints: true, ProtectedCredentials: true, SchedulingDeferred: true}
	if !r.Satisfied() {
		t.Fatal("complete observation rejected")
	}
	typeOf := reflect.TypeOf(r)
	for j := 0; j < typeOf.NumField(); j++ {
		t.Run(typeOf.Field(j).Name, func(t *testing.T) {
			copy := r
			v := reflect.ValueOf(&copy).Elem().Field(j)
			v.Set(reflect.Zero(v.Type()))
			if copy.Satisfied() {
				t.Fatal("missing required postcondition accepted")
			}
		})
	}
}

func TestVICIdialDatabaseRecognitionIsReadOnly(t *testing.T) {
	i, _, src := databaseInputs(t, true)
	before := i.Snapshot.Digest()
	if e := VerifyDatabaseForApplication(i.Snapshot, src); e != nil || i.Snapshot.Digest() != before {
		t.Fatal(e)
	}
	for _, name := range []string{"missing table", "missing seed", "wrong catalog", "missing grant", "partial schema"} {
		t.Run(name, func(t *testing.T) {
			s := i.Snapshot
			switch name {
			case "missing table":
				s.Tables--
			case "missing seed":
				s.SeedsOK = false
			case "wrong catalog":
				s.CatalogSHA = hash("wrong")
			case "missing grant":
				s.GrantsOK = false
			case "partial schema":
				s.Version = "0"
			}
			if e := VerifyDatabaseForApplication(s, src); e == nil {
				t.Fatal("incomplete S03 accepted")
			}
		})
	}
}

type vicidialFilesFake struct {
	result execution.Result
	err    error
	calls  int
}

func (f *vicidialFilesFake) RunInput(_ context.Context, c execution.Command, b []byte) (execution.Result, error) {
	f.calls++
	if c.Executable != "/usr/bin/python3" || c.Args[len(c.Args)-1] != "inspect" || !json.Valid(b) {
		panic("unexpected effectful inspection")
	}
	return f.result, f.err
}
func TestVICIdialBoundedFileInspection(t *testing.T) {
	s := vicidialFixture()
	files, _ := s.Deployment()
	b, _ := json.Marshal(validVICIdialObservations(files))
	for _, name := range []string{"valid", "exit failure", "truncated", "invalid JSON", "missing observations"} {
		t.Run(name, func(t *testing.T) {
			f := &vicidialFilesFake{result: execution.Result{Started: true, ExitCode: 0, Stdout: string(b)}}
			switch name {
			case "exit failure":
				f.result.ExitCode = 1
			case "truncated":
				f.result.StdoutTruncated = true
			case "invalid JSON":
				f.result.Stdout = "not JSON"
			case "missing observations":
				f.result.Stdout = "[]"
			}
			a, e := readVICIdialFiles(context.Background(), s, f)
			if name == "valid" {
				if e != nil || a.State != "COMPLETE" {
					t.Fatal(a, e)
				}
			} else if e == nil {
				t.Fatal("incomplete inspection accepted")
			}
		})
	}
}

func TestVICIdialAGIModuleRuntimeLayout(t *testing.T) {
	s := VICIdialSource{files: map[string][]byte{"extras/Asterisk.pm": []byte("fixture base"), "extras/Asterisk/AGI.pm": []byte("fixture AGI")}}
	files, _, e := s.deployment()
	if e != nil || len(files) != 2 {
		t.Fatal(e, files)
	}
	for _, f := range files {
		if !vicidialTarget(f.Target) || !strings.HasPrefix(f.Target, "/usr/local/share/perl5/5.40/Asterisk") || f.Mode != 0644 || f.Owner != "root" || f.Group != "root" {
			t.Fatal(f)
		}
	}
	if vicidialTarget("/usr/local/share/perl5/5.40/Asterisk/Foreign.pm") {
		t.Fatal("unreviewed Perl destination allowed")
	}
}

func TestVICIdialOtherPerlLibrariesKeepApplicationLayout(t *testing.T) {
	s := VICIdialSource{files: map[string][]byte{"extras/Asterisk/Manager.pm": []byte("fixture")}}
	files, _, e := s.deployment()
	if e != nil || len(files) != 1 || files[0].Target != vicidialHome+"/libs/Asterisk/Manager.pm" || !vicidialTarget(files[0].Target) {
		t.Fatal(e, files)
	}
}
