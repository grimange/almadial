//go:build linux

package provision

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/grimange/almadial/internal/artifacts"
	"github.com/grimange/almadial/internal/config"
	"github.com/grimange/almadial/internal/execution"
	"github.com/grimange/almadial/internal/journal"
)

func systemFixture(complete bool) SystemSnapshot {
	s := SystemSnapshot{RPM: fixture(), SELinux: "Enforcing", Policy: "targeted", LocalContextsSHA256: hash("vendor contexts"), SystemUIDMax: 999, SystemGIDMax: 999,
		Accounts: []AccountObservation{{Name: "root", UID: 0, GID: 0}, {Name: "apache", UID: 48, GID: 48, Home: "/usr/share/httpd", Shell: "/sbin/nologin", Groups: []string{"apache"}}, {Name: "mysql", UID: 27, GID: 27, Home: "/var/lib/mysql", Shell: "/sbin/nologin", Groups: []string{"mysql"}}},
		Groups:   []GroupObservation{{"root", 0}, {"apache", 48}, {"mysql", 27}}, Recording: StorageObservation{Mount: "/", Source: "/dev/test", Filesystem: "xfs", Options: "rw,seclabel", AvailableBytes: 8 << 30, AvailableInodes: 100000}}
	if complete {
		s.Groups = append(s.Groups, GroupObservation{"asterisk", 900})
		s.Accounts = append(s.Accounts, AccountObservation{Name: "asterisk", UID: 900, GID: 900, Home: "/var/lib/asterisk", Shell: "/sbin/nologin", Groups: []string{"asterisk"}})
	}
	for j, p := range ManagedDirectories() {
		d := DirectoryObservation{Path: p.Path, DefaultContext: expectedContext(p)}
		if complete {
			a, _ := account(s, p.Owner)
			g, _ := group(s, p.Group)
			d.Exists = true
			d.UID = a.UID
			d.GID = g.GID
			d.Mode = p.Mode
			d.Label = d.DefaultContext
			d.Device = 1
			d.Inode = uint64(j + 100)
		}
		s.Directories = append(s.Directories, d)
	}
	return s
}
func systemBudget() RecordingBudget {
	return RecordingBudget{Profile: "synthetic-test-only", Mount: "/", Source: "/dev/test", Filesystem: "xfs", MinAvailableBytes: 1 << 30, MinFreeInodes: 10000}
}
func systemInputs(t *testing.T, complete bool) (SystemInputs, artifacts.Manifest) {
	t.Helper()
	_, m := inputs(t)
	f, e := os.Open("../config/testdata/basic.json")
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	c, e := config.Load(f)
	if e != nil {
		t.Fatal(e)
	}
	i := SystemInputs{Parent: Inputs{Config: c}, Snapshot: systemFixture(complete), Budget: systemBudget()}
	i.Parent.Preflight.HostID = strings.Repeat("1", 32)
	h := sha256.Sum256([]byte(i.Parent.Preflight.HostID))
	i.Snapshot.RPM.Host = hex.EncodeToString(h[:])
	i.Parent.Inventory = i.Snapshot.RPM
	return i, m
}
func systemAuth(s SystemStage) SystemLabAuthorization {
	d := s.doc()
	return SystemLabAuthorization{"ALM-17/disposable-lab", d.Before.RPM.Host, s.Digest(), d.Parent, d.Manifest, hash("synthetic snapshot"), hash("synthetic unit-test permission"), true}
}
func finding(fs []SystemFinding, id string) Status {
	for _, f := range fs {
		if f.ID == id {
			return f.Status
		}
	}
	return ""
}

func TestSystemAssessment(t *testing.T) {
	for _, tc := range []struct {
		name, id string
		want     Status
		edit     func(*SystemSnapshot, *RecordingBudget)
	}{
		{"existing identity", "identity.asterisk", Satisfied, func(*SystemSnapshot, *RecordingBudget) {}},
		{"missing account", "identity.asterisk", CreateRequired, func(s *SystemSnapshot, _ *RecordingBudget) { s.Accounts = s.Accounts[:3] }},
		{"missing group", "identity.asterisk", Conflict, func(s *SystemSnapshot, _ *RecordingBudget) { s.Groups = s.Groups[:3] }},
		{"wrong home", "identity.asterisk", Conflict, func(s *SystemSnapshot, _ *RecordingBudget) { s.Accounts[3].Home = "/foreign" }},
		{"interactive shell", "identity.asterisk", Conflict, func(s *SystemSnapshot, _ *RecordingBudget) { s.Accounts[3].Shell = "/bin/bash" }},
		{"uid collision", "identity.asterisk", Conflict, func(s *SystemSnapshot, _ *RecordingBudget) {
			s.Accounts = append(s.Accounts, AccountObservation{Name: "foreign", UID: 900})
		}},
		{"gid collision", "identity.asterisk", Conflict, func(s *SystemSnapshot, _ *RecordingBudget) {
			s.Groups = append(s.Groups, GroupObservation{"foreign", 900})
		}},
		{"wrong primary group", "identity.asterisk", Conflict, func(s *SystemSnapshot, _ *RecordingBudget) { s.Accounts[3].GID = 48 }},
		{"interactive uid range", "identity.asterisk", Conflict, func(s *SystemSnapshot, _ *RecordingBudget) { s.Accounts[3].UID = 1001 }},
		{"extra privileges", "identity.asterisk", Conflict, func(s *SystemSnapshot, _ *RecordingBudget) {
			s.Accounts[3].Groups = append(s.Accounts[3].Groups, "wheel")
		}},
		{"vendor absent", "identity.apache", NotVerified, func(s *SystemSnapshot, _ *RecordingBudget) { s.Accounts = append(s.Accounts[:1], s.Accounts[2:]...) }},
		{"vendor conflict", "identity.mysql", Conflict, func(s *SystemSnapshot, _ *RecordingBudget) { s.Accounts[2].Shell = "/bin/bash" }},
		{"missing directory", "directory.00", Required, func(s *SystemSnapshot, _ *RecordingBudget) {
			s.Directories[0] = DirectoryObservation{Path: s.Directories[0].Path, DefaultContext: s.Directories[0].DefaultContext}
		}},
		{"correct directory", "directory.00", Satisfied, func(*SystemSnapshot, *RecordingBudget) {}},
		{"incorrect owner", "directory.00", Conflict, func(s *SystemSnapshot, _ *RecordingBudget) { s.Directories[0].UID = 48 }},
		{"incorrect mode", "directory.00", Conflict, func(s *SystemSnapshot, _ *RecordingBudget) { s.Directories[0].Mode = 0755 }},
		{"world writable", "directory.00", Conflict, func(s *SystemSnapshot, _ *RecordingBudget) { s.Directories[0].Mode = 0777 }},
		{"special bits", "directory.00", Conflict, func(s *SystemSnapshot, _ *RecordingBudget) { s.Directories[0].Mode = 04700 }},
		{"file at directory", "directory.00", Conflict, func(s *SystemSnapshot, _ *RecordingBudget) { s.Directories[0].Unsafe = true }},
		{"symlink component", "directory.00", Conflict, func(s *SystemSnapshot, _ *RecordingBudget) {
			s.Directories[0].Unsafe = true
			s.Directories[0].Exists = false
		}},
		{"wrong recording owner", "directory.14", Conflict, func(s *SystemSnapshot, _ *RecordingBudget) { s.Directories[14].UID = 48 }},
		{"missing path probe", "directory.00", NotVerified, func(s *SystemSnapshot, _ *RecordingBudget) { s.Directories = s.Directories[1:] }},
		{"enforcing", "selinux", Satisfied, func(*SystemSnapshot, *RecordingBudget) {}},
		{"permissive", "selinux", NotVerified, func(s *SystemSnapshot, _ *RecordingBudget) { s.SELinux = "Permissive" }},
		{"disabled", "selinux", NotVerified, func(s *SystemSnapshot, _ *RecordingBudget) { s.SELinux = "Disabled" }},
		{"wrong policy", "selinux", NotVerified, func(s *SystemSnapshot, _ *RecordingBudget) { s.Policy = "mls" }},
		{"unknown persistent mapping", "selinux", NotVerified, func(s *SystemSnapshot, _ *RecordingBudget) { s.LocalContextsSHA256 = "" }},
		{"unsupported type", "directory.14", NotVerified, func(s *SystemSnapshot, _ *RecordingBudget) {
			s.Directories[14].DefaultContext = "system_u:object_r:invented_t:s0"
		}},
		{"missing persistent mapping", "directory.14", NotVerified, func(s *SystemSnapshot, _ *RecordingBudget) { s.Directories[14].DefaultContext = "" }},
		{"actual label mismatch", "directory.14", ReconciliationRequired, func(s *SystemSnapshot, _ *RecordingBudget) {
			s.Directories[14].Label = "unconfined_u:object_r:default_t:s0"
		}},
		{"capacity satisfied", "recording.budget", Satisfied, func(*SystemSnapshot, *RecordingBudget) {}},
		{"insufficient bytes", "recording.budget", Incompatible, func(s *SystemSnapshot, _ *RecordingBudget) { s.Recording.AvailableBytes = 1 }},
		{"insufficient inodes", "recording.budget", Incompatible, func(s *SystemSnapshot, _ *RecordingBudget) { s.Recording.AvailableInodes = 1 }},
		{"missing budget", "recording.budget", NotVerified, func(_ *SystemSnapshot, b *RecordingBudget) { *b = RecordingBudget{} }},
		{"unexpected mount", "recording.budget", Conflict, func(s *SystemSnapshot, _ *RecordingBudget) { s.Recording.Mount = "/other" }},
		{"unexpected source", "recording.budget", Conflict, func(s *SystemSnapshot, _ *RecordingBudget) { s.Recording.Source = "/dev/other" }},
		{"read only mount", "recording.budget", Conflict, func(s *SystemSnapshot, _ *RecordingBudget) { s.Recording.Options = "ro,seclabel" }},
		{"unsupported filesystem", "recording.budget", Conflict, func(s *SystemSnapshot, b *RecordingBudget) { s.Recording.Filesystem = "nfs"; b.Filesystem = "nfs" }},
		{"rpm drift", "rpm", NotVerified, func(s *SystemSnapshot, _ *RecordingBudget) { s.RPM.Installed = s.RPM.Installed[1:] }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, b := systemFixture(true), systemBudget()
			tc.edit(&s, &b)
			if got := finding(AssessSystem(s, b), tc.id); got != tc.want {
				t.Fatalf("%s got %s want %s", tc.id, got, tc.want)
			}
		})
	}
}

type systemFake struct {
	s          SystemSnapshot
	calls      int
	failure    execution.Category
	failAt     int
	lie, extra bool
}

func (f *systemFake) InspectSystem(ctx context.Context) (SystemSnapshot, error) {
	if ctx.Err() != nil {
		return SystemSnapshot{}, ctx.Err()
	}
	var s SystemSnapshot
	_ = json.Unmarshal([]byte(jsonString(f.s)), &s)
	return s, nil
}
func (f *systemFake) Validate(c execution.Command) error {
	r, _ := execution.NewRunner(nil)
	return r.Validate(c)
}
func (f *systemFake) Run(_ context.Context, c execution.Command) (execution.Result, error) {
	f.calls++
	r := execution.Result{Started: true, ExitCode: 0}
	if f.failure != "" && f.calls == f.failAt {
		return r, &execution.Error{Category: f.failure, ExitCode: -1}
	}
	if f.lie {
		return r, nil
	}
	switch c.Executable {
	case "/usr/sbin/groupadd":
		f.s.Groups = append(f.s.Groups, GroupObservation{"asterisk", 900})
	case "/usr/sbin/useradd":
		f.s.Accounts = append(f.s.Accounts, AccountObservation{Name: "asterisk", UID: 900, GID: 900, Home: "/var/lib/asterisk", Shell: "/sbin/nologin", Groups: []string{"asterisk"}})
	case "/proc/self/exe":
		var req SystemDirectoryRequest
		_ = json.Unmarshal([]byte(c.Args[2]), &req)
		for j, p := range ManagedDirectories() {
			if p.Path == req.Before.Path {
				a, _ := account(f.s, p.Owner)
				g, _ := group(f.s, p.Group)
				for k, existing := range f.s.Directories {
					if existing.Path == p.Path {
						f.s.Directories[k] = DirectoryObservation{Path: p.Path, Exists: true, UID: a.UID, GID: g.GID, Mode: p.Mode, Device: 1, Inode: uint64(j + 100), Label: expectedContext(p), DefaultContext: expectedContext(p)}
					}
				}
			}
		}
	default:
		return r, errSystem
	}
	if f.extra {
		f.s.Accounts[1].Home = "/foreign"
	}
	return r, nil
}

func TestSystemPlanAndExecution(t *testing.T) {
	i, m := systemInputs(t, false)
	s, e := i.Prepare(m)
	if e != nil {
		t.Fatal(e)
	}
	if len(s.Graph().Actions()) != 18 {
		t.Fatal("expected group, user and sixteen directories")
	}
	before := jsonString(i)
	again, e := i.Prepare(m)
	if e != nil || s.Digest() != again.Digest() || before != jsonString(i) {
		t.Fatal("non-deterministic or mutable plan")
	}
	for j, a := range s.Graph().Actions() {
		if j > 0 && !reflect.DeepEqual(a.Dependencies, []string{s.Graph().Actions()[j-1].ID}) {
			t.Fatal("ordering")
		}
		if a.Command.Timeout == 0 || a.Command.Executable == "/bin/sh" {
			t.Fatal("unbounded or shell action")
		}
	}
	i.Snapshot.Accounts[0].Name = "mutated"
	if a, ok := account(s.doc().Before, "root"); !ok || a.UID != 0 {
		t.Fatal("mutable stage")
	}
	f := &systemFake{s: s.doc().Before}
	dir := filepath.Join(t.TempDir(), "state")
	o, e := ExecuteSystemLab(context.Background(), s, m, systemAuth(s), dir, f, f, false, nil)
	if e != nil || len(o.Traces) != 18 || o.NoOp {
		t.Fatal(e, o)
	}
	for _, r := range o.Journal.Actions {
		if r.State != journal.Succeeded {
			t.Fatal(r)
		}
	}
	if !systemSatisfied(f.s, systemBudget()) {
		t.Fatal("missing postconditions")
	}
	if a, _ := account(f.s, "apache"); a.UID != 48 {
		t.Fatal("vendor changed")
	}
	o, e = ExecuteSystemLab(context.Background(), s, m, systemAuth(s), dir, f, f, true, nil)
	if e != nil || !o.NoOp || f.calls != 18 {
		t.Fatal("resume replay", e)
	}
	i, m = systemInputs(t, true)
	noop, e := i.Prepare(m)
	if e != nil || len(noop.Graph().Actions()) != 0 {
		t.Fatal("second plan", e)
	}
	f = &systemFake{s: i.Snapshot}
	o, e = ExecuteSystemLab(context.Background(), noop, m, systemAuth(noop), filepath.Join(t.TempDir(), "noop"), f, f, false, nil)
	if e != nil || !o.NoOp || f.calls != 0 {
		t.Fatal("no-op", e)
	}
}
func TestSystemFailuresAndRecovery(t *testing.T) {
	for _, standard := range []bool{false, true} {
		t.Run(map[bool]string{false: "legacy", true: "standard"}[standard], func(t *testing.T) {
			systemAuth := systemAuth
			if standard {
				original := systemAuth
				systemAuth = func(s SystemStage) SystemLabAuthorization {
					a := original(s)
					a.Scope = StandardAuthorization
					a.SnapshotReference = ""
					a.OperatorEvidenceSHA256 = ""
					a.CandidateAcknowledged = false
					return a
				}
			}

			for _, kind := range []execution.Category{execution.Timeout, execution.Canceled, execution.Exit, execution.Start, execution.OutputLimit} {
				t.Run(string(kind), func(t *testing.T) {
					i, m := systemInputs(t, false)
					s, _ := i.Prepare(m)
					f := &systemFake{s: i.Snapshot, failure: kind, failAt: 2}
					dir := filepath.Join(t.TempDir(), "state")
					o, e := ExecuteSystemLab(context.Background(), s, m, systemAuth(s), dir, f, f, false, nil)
					if !errors.Is(e, &execution.Error{Category: kind}) || len(o.Traces) != 2 {
						t.Fatal(e)
					}
					if f.calls != 2 {
						t.Fatal("continued after failure")
					}
					f.failure = ""
					o, e = ExecuteSystemLab(context.Background(), s, m, systemAuth(s), dir, f, f, true, nil)
					if e != nil || len(o.Traces) != 17 || f.calls != 19 {
						t.Fatal("safe absent action did not resume", e)
					}
				})
			}
			for _, tc := range []struct {
				name       string
				lie, extra bool
			}{{"zero exit missing postcondition", true, false}, {"unexpected vendor change", false, true}} {
				t.Run(tc.name, func(t *testing.T) {
					i, m := systemInputs(t, false)
					s, _ := i.Prepare(m)
					f := &systemFake{s: i.Snapshot, lie: tc.lie, extra: tc.extra}
					o, e := ExecuteSystemLab(context.Background(), s, m, systemAuth(s), filepath.Join(t.TempDir(), "state"), f, f, false, nil)
					if !errors.Is(e, &execution.Error{Category: execution.Postcondition}) || o.Journal.Actions[0].State != journal.Failed {
						t.Fatal(e)
					}
				})
			}
		})
	}
}
func TestSystemAuthorizationAndFreshness(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*SystemLabAuthorization, *systemFake)
	}{
		{"unauthorized", func(a *SystemLabAuthorization, _ *systemFake) { a.Scope = "production" }},
		{"ALM16 permission", func(a *SystemLabAuthorization, _ *systemFake) { a.Scope = "ALM-16/disposable-lab" }},
		{"candidate not acknowledged", func(a *SystemLabAuthorization, _ *systemFake) { a.CandidateAcknowledged = false }},
		{"changed plan", func(a *SystemLabAuthorization, _ *systemFake) { a.StageSHA256 = hash("changed") }},
		{"changed manifest", func(a *SystemLabAuthorization, _ *systemFake) { a.ManifestSHA256 = hash("changed") }},
		{"missing snapshot", func(a *SystemLabAuthorization, _ *systemFake) { a.SnapshotReference = "" }},
		{"changed host", func(_ *SystemLabAuthorization, f *systemFake) { f.s.RPM.Host = hash("changed") }},
		{"changed directory", func(_ *SystemLabAuthorization, f *systemFake) { f.s.Directories[0].Exists = true }},
		{"changed mapping", func(_ *SystemLabAuthorization, f *systemFake) { f.s.LocalContextsSHA256 = hash("changed") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			i, m := systemInputs(t, false)
			s, _ := i.Prepare(m)
			a := systemAuth(s)
			f := &systemFake{s: i.Snapshot}
			tc.edit(&a, f)
			_, e := ExecuteSystemLab(context.Background(), s, m, a, filepath.Join(t.TempDir(), "state"), f, f, false, nil)
			if e == nil || f.calls != 0 {
				t.Fatal("mutation allowed", e)
			}
		})
	}
	i, m := systemInputs(t, false)
	s, _ := i.Prepare(m)
	f := &systemFake{s: i.Snapshot}
	dir := filepath.Join(t.TempDir(), "state")
	lock, e := journal.Acquire(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer lock.Close()
	if _, e = ExecuteSystemLab(context.Background(), s, m, systemAuth(s), dir, f, f, false, nil); !errors.Is(e, journal.ErrBusy) || f.calls != 0 {
		t.Fatal("lock bypass", e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e = ExecuteSystemLab(ctx, s, m, systemAuth(s), filepath.Join(t.TempDir(), "cancel"), f, f, false, nil); e == nil || f.calls != 0 {
		t.Fatal("cancellation bypass")
	}
}
func TestSystemInterruptedDirectory(t *testing.T) {
	for _, state := range []string{"absent", "complete", "partial", "drift"} {
		t.Run(state, func(t *testing.T) {
			i, m := systemInputs(t, true)
			i.Snapshot.Directories[0] = DirectoryObservation{Path: ManagedDirectories()[0].Path, DefaultContext: expectedContext(ManagedDirectories()[0])}
			s, e := i.Prepare(m)
			if e != nil {
				t.Fatal(e)
			}
			a := systemAuth(s)
			dir := filepath.Join(t.TempDir(), "state")
			session, _ := journal.Acquire(dir)
			doc, _ := s.journalDocument(a)
			if e = session.Initialize(doc); e != nil {
				t.Fatal(e)
			}
			id := s.Graph().Actions()[0].ID
			if e = session.Begin(id); e != nil {
				t.Fatal(e)
			}
			session.Close()
			f := &systemFake{s: i.Snapshot}
			if state != "absent" {
				f.s.Directories[0] = systemFixture(true).Directories[0]
			}
			if state == "partial" {
				f.s.Directories[0].Mode = 0755
			}
			if state == "drift" {
				f.s.Directories[0].Inode = 777
				f.s.Directories[0].UID = 48
			}
			o, e := ExecuteSystemLab(context.Background(), s, m, a, dir, f, f, true, nil)
			if state == "partial" || state == "drift" {
				if e == nil || f.calls != 0 {
					t.Fatal("blind replay", e)
				}
				return
			}
			if e != nil {
				t.Fatal(e)
			}
			if state == "complete" && (!o.NoOp || f.calls != 0) {
				t.Fatal("completed action replayed")
			}
			if state == "absent" && f.calls != 1 {
				t.Fatal("safe missing action not resumed")
			}
		})
	}
}
func TestSystemSafePathInspection(t *testing.T) {
	root := t.TempDir()
	foreign := filepath.Join(root, "foreign")
	if e := os.WriteFile(foreign, []byte("operator bytes"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.Symlink(root, filepath.Join(root, "link")); e != nil {
		t.Fatal(e)
	}
	if e := os.Link(foreign, filepath.Join(root, "hardlink")); e != nil {
		t.Fatal(e)
	}
	for _, path := range []string{foreign, filepath.Join(root, "hardlink"), filepath.Join(root, "link"), filepath.Join(root, "link", "missing"), root + "/../escape"} {
		if f, e := safeDirectory(path); e == nil {
			f.Close()
			t.Fatal("unsafe path accepted")
		}
	}
	if _, e := safeDirectory(filepath.Join(root, "missing")); !errors.Is(e, os.ErrNotExist) {
		t.Fatal("missing path")
	}
	f, e := safeDirectory(root)
	if e != nil {
		t.Fatal(e)
	}
	f.Close()
	if os.Geteuid() != 0 {
		denied := filepath.Join(root, "denied")
		if e := os.Mkdir(denied, 0000); e != nil {
			t.Fatal(e)
		}
		defer os.Chmod(denied, 0700)
		if f, e := safeDirectory(filepath.Join(denied, "child")); e == nil {
			f.Close()
			t.Fatal("inaccessible parent accepted")
		} else if !errors.Is(e, errSystem) || !observeDirectory(filepath.Join(denied, "child")).Unsafe {
			t.Fatal("inaccessible parent was not classified as unsafe", e)
		}
	}
	b, _ := os.ReadFile(foreign)
	if string(b) != "operator bytes" {
		t.Fatal("foreign contents changed")
	}
	n, _ := NewNative(nil)
	if e := ApplySystemDirectory(context.Background(), 0, SystemDirectoryRequest{Before: DirectoryObservation{Path: "/etc/almadial"}, HostState: hash("no running intent")}, n); e == nil {
		t.Fatal("helper bypass")
	}
}
func TestSystemCanonicalAssessment(t *testing.T) {
	s := systemFixture(true)
	digest := s.Digest()
	fs := AssessSystem(s, systemBudget())
	s.Accounts[0], s.Accounts[2] = s.Accounts[2], s.Accounts[0]
	s.Directories[0], s.Directories[2] = s.Directories[2], s.Directories[0]
	s.Recording.AvailableBytes--
	if s.Digest() != digest || !reflect.DeepEqual(fs, AssessSystem(s, systemBudget())) {
		t.Fatal("ordering or incidental space changed policy")
	}
	for _, edit := range []func(*SystemSnapshot){func(s *SystemSnapshot) { s.Accounts = append(s.Accounts, s.Accounts[0]) }, func(s *SystemSnapshot) { s.Groups = append(s.Groups, s.Groups[0]) }, func(s *SystemSnapshot) { s.Directories = append(s.Directories, s.Directories[0]) }} {
		s := systemFixture(true)
		edit(&s)
		if systemReady(s, systemBudget()) {
			t.Fatal("duplicate observations")
		}
	}
}

func TestSystemEquivalentPlanIdentity(t *testing.T) {
	i, m := systemInputs(t, false)
	first, e := i.Prepare(m)
	if e != nil {
		t.Fatal(e)
	}
	reverse := func(n int, swap func(int, int)) {
		for j := 0; j < n/2; j++ {
			swap(j, n-j-1)
		}
	}
	reverse(len(i.Snapshot.Accounts), func(a, b int) {
		i.Snapshot.Accounts[a], i.Snapshot.Accounts[b] = i.Snapshot.Accounts[b], i.Snapshot.Accounts[a]
	})
	reverse(len(i.Snapshot.Groups), func(a, b int) {
		i.Snapshot.Groups[a], i.Snapshot.Groups[b] = i.Snapshot.Groups[b], i.Snapshot.Groups[a]
	})
	reverse(len(i.Snapshot.Directories), func(a, b int) {
		i.Snapshot.Directories[a], i.Snapshot.Directories[b] = i.Snapshot.Directories[b], i.Snapshot.Directories[a]
	})
	second, e := i.Prepare(m)
	if e != nil || first.Digest() != second.Digest() || string(first.Graph().JSON()) != string(second.Graph().JSON()) {
		t.Fatal("equivalent inventory changed immutable stage identity", e)
	}
}

func TestSystemDefaultLabelRepair(t *testing.T) {
	i, m := systemInputs(t, true)
	i.Snapshot.Directories[0].Label = "unconfined_u:object_r:default_t:s0"
	before := i.Snapshot.Directories[0]
	s, e := i.Prepare(m)
	if e != nil || len(s.Graph().Actions()) != 1 {
		t.Fatal("expected only exact default-label repair", e)
	}
	f := &systemFake{s: s.doc().Before}
	o, e := ExecuteSystemLab(context.Background(), s, m, systemAuth(s), filepath.Join(t.TempDir(), "state"), f, f, false, nil)
	after, _ := directory(f.s, before.Path)
	if e != nil || len(o.Traces) != 1 || after.Inode != before.Inode || after.UID != before.UID || after.GID != before.GID || after.Mode != before.Mode || after.Label != before.DefaultContext {
		t.Fatal("label repair changed metadata or failed", e, after)
	}
}

type systemReadFixture struct {
	result execution.Result
	err    error
}

func (f systemReadFixture) Validate(c execution.Command) error {
	r, _ := execution.NewRunner(nil)
	return r.Validate(c)
}
func (f systemReadFixture) Run(_ context.Context, c execution.Command) (execution.Result, error) {
	if c.Executable != "/usr/bin/getent" || len(c.Args) != 1 || c.Args[0] != "passwd" || c.Timeout <= 0 || c.StdoutLimit > 4<<20 || c.StderrLimit > 16<<10 {
		return execution.Result{}, errSystem
	}
	return f.result, f.err
}
func TestSystemBoundedObservationFailures(t *testing.T) {
	for _, tc := range []struct {
		name   string
		result execution.Result
		err    error
		want   bool
	}{
		{"success", execution.Result{Started: true, Stdout: " root:x:0:0:root:/root:/bin/bash\n"}, nil, true},
		{"not started", execution.Result{}, nil, false},
		{"nonzero", execution.Result{Started: true, ExitCode: 1}, nil, false},
		{"stdout limit", execution.Result{Started: true, StdoutTruncated: true}, nil, false},
		{"stderr limit", execution.Result{Started: true, StderrTruncated: true}, nil, false},
		{"timeout", execution.Result{}, &execution.Error{Category: execution.Timeout, ExitCode: -1}, false},
		{"cancellation", execution.Result{}, context.Canceled, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n := Native{Runner: systemReadFixture{tc.result, tc.err}}
			s, e := n.systemRead(context.Background(), "/usr/bin/getent", "passwd")
			if (e == nil) != tc.want || (tc.want && s != "root:x:0:0:root:/root:/bin/bash") {
				t.Fatal(s, e)
			}
		})
	}
}
