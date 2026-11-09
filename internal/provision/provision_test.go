package provision

import (
	"context"
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
	"github.com/grimange/almadial/internal/plan"
	"github.com/grimange/almadial/internal/preflight"
)

func fixture(missing ...string) Snapshot {
	s := Snapshot{Host: strings.Repeat("a", 64), OS: "almalinux", Release: "10.2", Arch: "x86_64", HistorySHA256: hash("history"), HistoryComplete: true, AlmaKeyFileSHA256: almaKeySHA, KeyHashes: []string{almaKeySHA, epelKeySHA}, Repositories: []RepoObservation{}, Installed: []Installed{}}
	for _, r := range Repositories() {
		if r.Enabled {
			s.Repositories = append(s.Repositories, RepoObservation{ID: r.ID, Enabled: true, Mirrorlist: r.Source, BaseURLs: []string{}, Keys: []string{almaKeyPath}, GPGCheck: true, TLSVerify: true})
		}
	}
	for _, p := range Packages() {
		if contains(missing, p.Name) {
			continue
		}
		r, _ := repository(p.Repository)
		s.Installed = append(s.Installed, Installed{p.Name, p.EVR, p.Arch, r.Signer})
	}
	return canonical(s)
}
func transaction(s Snapshot) Transaction {
	t := Transaction{BeforeSHA256: s.Digest(), Install: []Candidate{}, Remove: []Package{}}
	for _, p := range Packages() {
		found := false
		for _, i := range s.Installed {
			if i.Name == p.Name {
				found = true
			}
		}
		if !found {
			h := PinnedRPMHash(p.Name)
			if h == "" {
				h = hash(p)
			}
			t.Install = append(t.Install, Candidate{p, h})
		}
	}
	return normalize(t)
}
func applied(s Snapshot, t Transaction) Snapshot {
	b, _ := json.Marshal(s)
	var after Snapshot
	_ = json.Unmarshal(b, &after)
	kept := after.Installed[:0]
	for _, i := range after.Installed {
		removed := false
		for _, old := range t.Remove {
			if old.NEVRA() == i.NEVRA() {
				removed = true
			}
		}
		if !removed {
			kept = append(kept, i)
		}
	}
	after.Installed = kept
	for _, p := range t.Install {
		r, _ := repository(p.Repository)
		after.Installed = append(after.Installed, Installed{p.Name, p.EVR, p.Arch, r.Signer})
	}
	after.HistorySHA256 = hash("new history")
	return canonical(after)
}
func inputs(t *testing.T) (plan.Plan, artifacts.Manifest) {
	t.Helper()
	f, e := os.Open("../../manifests/stack.lock.yaml")
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	m, e := artifacts.Load(f)
	if e != nil {
		t.Fatal(e)
	}
	f2, e := os.Open("../config/testdata/basic.json")
	if e != nil {
		t.Fatal(e)
	}
	defer f2.Close()
	c, e := config.Load(f2)
	if e != nil {
		t.Fatal(e)
	}
	p, e := plan.New(c, preflight.Snapshot{}, preflight.Requirements{}, m.Digest())
	if e != nil {
		t.Fatal(e)
	}
	return p, m
}
func auth(s Stage) LabAuthorization {
	d := s.doc()
	return LabAuthorization{"ALM-16/disposable-lab", d.Before.Host, s.Digest(), d.Parent, d.Manifest, hash("synthetic snapshot reference"), hash("synthetic permission; unit test only"), true}
}

func TestM0Policy(t *testing.T) {
	ps := Packages()
	if len(ps) != 44 {
		t.Fatal(len(ps))
	}
	for _, p := range ps {
		if !validPackage(p) {
			t.Fatal(p)
		}
	}
	if ps[0].Name != "autoconf" {
		t.Fatal("unordered policy")
	}
	ps[0].Name = "corrupted"
	if Packages()[0].Name == "corrupted" {
		t.Fatal("mutable policy")
	}
	if !shaPattern.MatchString(PolicyDigest()) {
		t.Fatal("missing policy digest")
	}
	for _, p := range Packages() {
		if contains([]string{"asterisk", "vicidial", "pjproject-devel", "perl-Net-Telnet", "perl-core", "sox"}, p.Name) {
			t.Fatal("unqualified/source-managed package", p.Name)
		}
	}
}
func TestRepositoryTrust(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*Snapshot)
	}{
		{"wrong OS", func(s *Snapshot) { s.OS = "centos" }},
		{"wrong release", func(s *Snapshot) { s.Release = "10.3" }},
		{"wrong architecture", func(s *Snapshot) { s.Arch = "aarch64" }},
		{"unknown host", func(s *Snapshot) { s.Host = "" }},
		{"foreign enabled repository", func(s *Snapshot) {
			s.Repositories = append(s.Repositories, RepoObservation{ID: "foreign", Enabled: true})
		}},
		{"mirror substitution", func(s *Snapshot) { s.Repositories[0].Mirrorlist = "https://example.invalid/" }},
		{"baseurl substitution", func(s *Snapshot) { s.Repositories[0].BaseURLs = []string{"https://example.invalid/"} }},
		{"metalink substitution", func(s *Snapshot) { s.Repositories[0].Metalink = "https://example.invalid/" }},
		{"disabled CRB", func(s *Snapshot) {
			for i := range s.Repositories {
				if s.Repositories[i].ID == "crb" {
					s.Repositories[i].Enabled = false
				}
			}
		}},
		{"signature disabled", func(s *Snapshot) { s.Repositories[0].GPGCheck = false }},
		{"TLS disabled", func(s *Snapshot) { s.Repositories[0].TLSVerify = false }},
		{"untrusted key file", func(s *Snapshot) { s.AlmaKeyFileSHA256 = hash("wrong key") }},
		{"untrusted imported key", func(s *Snapshot) { s.KeyHashes = []string{almaKeySHA} }},
		{"key URL substitution", func(s *Snapshot) { s.Repositories[0].Keys = []string{"file:///wrong"} }},
		{"duplicate definition", func(s *Snapshot) { s.Repositories = append(s.Repositories, s.Repositories[0]) }},
		{"incomplete DNF history", func(s *Snapshot) { s.HistoryComplete = false }},
		{"unknown DNF history", func(s *Snapshot) { s.HistorySHA256 = "" }},
		{"duplicate RPM", func(s *Snapshot) { s.Installed = append(s.Installed, s.Installed[0]) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := fixture()
			tc.edit(&s)
			if s.ValidateTrust() == nil {
				t.Fatal("accepted unsafe state")
			}
			for _, a := range Assess(s, nil, false) {
				if a.Status != NotVerified {
					t.Fatal(a)
				}
			}
		})
	}
	s := fixture()
	s.Repositories = append(s.Repositories, RepoObservation{ID: "admin-disabled", Enabled: false})
	if e := s.ValidateTrust(); e != nil {
		t.Fatal("disabled administrator definition rejected", e)
	}
}
func TestAssessment(t *testing.T) {
	for _, tc := range []struct {
		name     string
		missing  []string
		edit     func(*Snapshot)
		resolved bool
		want     Status
	}{
		{"already satisfied", nil, nil, false, Satisfied},
		{"required", []string{"autoconf"}, nil, true, Required},
		{"not yet resolved", []string{"autoconf"}, nil, false, NotVerified},
		{"wrong version", nil, func(s *Snapshot) { s.Installed[0].EVR = "0:0-1.el10" }, true, Incompatible},
		{"wrong epoch", nil, func(s *Snapshot) { s.Installed[0].EVR = "1:2.71-13.el10" }, true, Incompatible},
		{"wrong architecture", nil, func(s *Snapshot) { s.Installed[0].Arch = "aarch64" }, true, Incompatible},
		{"unsigned installed RPM", nil, func(s *Snapshot) { s.Installed[0].Signer = "" }, true, Incompatible},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := fixture(tc.missing...)
			if tc.edit != nil {
				tc.edit(&s)
			}
			cs := []Package{}
			for _, c := range transaction(s).Install {
				cs = append(cs, c.Package)
			}
			a := Assess(s, cs, tc.resolved)
			if a[0].Status != tc.want {
				t.Fatalf("got %s", a[0].Status)
			}
		})
	}
	s := fixture("autoconf")
	if Assess(s, nil, true)[0].Status != Unavailable {
		t.Fatal("missing package falsely available")
	}
}
func TestTransactionBoundary(t *testing.T) {
	s := fixture("autoconf", "perl-Switch")
	valid := transaction(s)
	if e := valid.Validate(s); e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct {
		name string
		edit func(*Transaction)
	}{
		{"stale inventory", func(v *Transaction) { v.BeforeSHA256 = hash("old inventory") }},
		{"erase", func(v *Transaction) { v.Remove = append(v.Remove, Packages()[0]) }},
		{"missing dependency", func(v *Transaction) { v.Install = v.Install[1:] }},
		{"foreign repository", func(v *Transaction) { v.Install[0].Repository = "foreign" }},
		{"foreign architecture", func(v *Transaction) { v.Install[0].Arch = "aarch64" }},
		{"cross release", func(v *Transaction) { v.Install[0].EVR = "0:1-1.el9" }},
		{"unknown checksum", func(v *Transaction) { v.Install[0].SHA256 = "" }},
		{"rebuild pinned EPEL archive", func(v *Transaction) { v.Install[1].SHA256 = hash("rebuilt archive") }},
		{"unreviewed EPEL", func(v *Transaction) {
			v.Install = append(v.Install, Candidate{Package{"perl-extra", "0:1-1.el10", "noarch", "epel-almadial"}, hash("extra")})
		}},
		{"upgrade existing package", func(v *Transaction) {
			p := Packages()[1]
			p.EVR = "0:9-1.el10"
			v.Install = append(v.Install, Candidate{p, hash("upgrade")})
		}},
		{"duplicate candidate", func(v *Transaction) { v.Install = append(v.Install, v.Install[0]) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, _ := json.Marshal(valid)
			var tx Transaction
			_ = json.Unmarshal(b, &tx)
			tc.edit(&tx)
			if tx.Validate(s) == nil {
				t.Fatal("accepted outside transaction policy")
			}
		})
	}
	transitive := Candidate{Package{"required-library", "0:1-1.el10", "x86_64", "baseos"}, hash("native solver closure")}
	valid.Install = append(valid.Install, transitive)
	if e := valid.Validate(s); e != nil {
		t.Fatal("qualified native closure rejected", e)
	}
}
func TestPostconditions(t *testing.T) {
	s := fixture("autoconf")
	tx := transaction(s)
	after := applied(s, tx)
	if e := Verify(s, after, tx); e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct {
		name string
		edit func(*Snapshot)
	}{
		{"absent package", func(a *Snapshot) { a.Installed = a.Installed[1:] }},
		{"wrong installed signature", func(a *Snapshot) { a.Installed[0].Signer = "untrusted" }},
		{"unexpected package", func(a *Snapshot) {
			a.Installed = append(a.Installed, Installed{"unexpected", "0:1-1.el10", "x86_64", ""})
		}},
		{"repository drift", func(a *Snapshot) { a.Repositories[0].Enabled = false }},
		{"partial history", func(a *Snapshot) { a.HistoryComplete = false }},
		{"host drift", func(a *Snapshot) { a.Host = hash("other host") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, _ := json.Marshal(after)
			var a Snapshot
			_ = json.Unmarshal(b, &a)
			tc.edit(&a)
			if Verify(s, a, tx) == nil {
				t.Fatal("false installed success")
			}
		})
	}
}
func TestDeterministicStage(t *testing.T) {
	p, m := inputs(t)
	s := fixture("autoconf", "gcc")
	tx := transaction(s)
	a, e := PreparePackages(p, m, s, tx)
	if e != nil {
		t.Fatal(e)
	}
	for i, j := 0, len(s.Installed)-1; i < j; i, j = i+1, j-1 {
		s.Installed[i], s.Installed[j] = s.Installed[j], s.Installed[i]
	}
	s.Repositories[0], s.Repositories[3] = s.Repositories[3], s.Repositories[0]
	tx.Install[0], tx.Install[1] = tx.Install[1], tx.Install[0]
	b, e := PreparePackages(p, m, s, tx)
	if e != nil || a.Digest() != b.Digest() || a.Graph().Digest() != b.Graph().Digest() || !reflect.DeepEqual(a.JSON(), b.JSON()) {
		t.Fatal("nondeterministic stage", e)
	}
	x := a.Graph().Actions()
	x[0].Command.Args[0] = "corrupt"
	if a.Graph().Actions()[0].Command.Args[0] != "-B" {
		t.Fatal("mutable graph")
	}
	if m.InstallationAuthorized() {
		t.Fatal("candidate manifest authorized")
	}
	if _, e = PreparePackages(plan.Plan{}, m, s, tx); e == nil {
		t.Fatal("missing M1 plan accepted")
	}
	if _, e = PreparePackages(p, artifacts.Manifest{}, s, tx); e == nil {
		t.Fatal("unauthorized/unknown manifest accepted")
	}
}

// Controlled native command boundary. It does not emulate dependency resolution:
// transaction fixtures are explicit native-solver observations supplied by tests.
type fakeNative struct {
	s            Snapshot
	tx           Transaction
	after        Snapshot
	failure      *execution.Error
	badOutput    bool
	calls        int
	mutations    int
	onRun        func()
	inspectCalls int
	driftAt      int
}

func (f *fakeNative) Validate(execution.Command) error { return nil }
func (f *fakeNative) Run(ctx context.Context, c execution.Command) (execution.Result, error) {
	f.calls++
	r := execution.Result{Started: true, ExitCode: 0}
	if ctx.Err() != nil {
		return r, &execution.Error{Category: execution.Canceled, ExitCode: -1}
	}
	mode := c.Args[3]
	if mode == "inspect" {
		f.inspectCalls++
		s := f.s
		if f.driftAt > 0 && f.inspectCalls >= f.driftAt {
			s.HistorySHA256 = hash("concurrent DNF change")
		}
		b, _ := json.Marshal(s)
		r.Stdout = string(b)
		return r, nil
	}
	f.mutations++
	if f.onRun != nil {
		f.onRun()
	}
	if f.failure != nil {
		r.ExitCode = f.failure.ExitCode
		return r, f.failure
	}
	if mode == "apply" || mode == "trust" {
		f.s = f.after
	}
	b, _ := json.Marshal(f.tx)
	r.Stdout = string(b)
	if f.badOutput {
		r.Stdout = "untrusted metadata output"
	}
	return r, nil
}
func TestJournaledProvisioning(t *testing.T) {
	p, m := inputs(t)
	for _, mode := range []string{"resolve", "apply"} {
		t.Run(mode, func(t *testing.T) {
			s := fixture("autoconf")
			tx := transaction(s)
			var st Stage
			var e error
			if mode == "resolve" {
				st, e = PrepareResolution(p, m, s)
			} else {
				st, e = PreparePackages(p, m, s, tx)
			}
			if e != nil {
				t.Fatal(e)
			}
			dir := filepath.Join(t.TempDir(), "state")
			f := &fakeNative{s: s, tx: tx, after: applied(s, tx)}
			f.onRun = func() {
				d, e := journal.Inspect(dir)
				if e != nil || d.Actions[0].State != journal.Running {
					t.Fatal("process started before durable intent", d, e)
				}
				other, e := journal.Acquire(dir)
				if other != nil {
					other.Close()
				}
				if !errors.Is(e, journal.ErrBusy) {
					t.Fatal("exclusive ownership missing", e)
				}
			}
			out, e := ExecuteLab(context.Background(), st, m, auth(st), dir, Native{f}, false, nil)
			if e != nil || out.Journal.Actions[0].State != journal.Succeeded || f.mutations != 1 || out.Transaction.Digest() != tx.Digest() {
				t.Fatal(out, e)
			}
			if mode == "apply" {
				second := transaction(f.s)
				noop, e := PreparePackages(p, m, f.s, second)
				if e != nil {
					t.Fatal(e)
				}
				out, e = ExecuteLab(context.Background(), noop, m, auth(noop), "/must-not-be-created", Native{f}, false, nil)
				if e != nil || !out.NoOp || f.mutations != 1 {
					t.Fatal("second run changed packages", out, e)
				}
			}
		})
	}
}
func TestRejectedExecution(t *testing.T) {
	p, m := inputs(t)
	s := fixture("autoconf")
	tx := transaction(s)
	st, _ := PreparePackages(p, m, s, tx)
	for _, tc := range []struct {
		name   string
		change func(*LabAuthorization, *fakeNative)
	}{
		{"no authorization", func(a *LabAuthorization, _ *fakeNative) { *a = LabAuthorization{} }},
		{"wrong host", func(a *LabAuthorization, _ *fakeNative) { a.Host = hash("other host") }},
		{"wrong parent plan", func(a *LabAuthorization, _ *fakeNative) { a.ParentSHA256 = hash("other plan") }},
		{"wrong transaction plan", func(a *LabAuthorization, _ *fakeNative) { a.StageSHA256 = hash("old graph") }},
		{"wrong manifest", func(a *LabAuthorization, _ *fakeNative) { a.ManifestSHA256 = hash("other manifest") }},
		{"missing snapshot record", func(a *LabAuthorization, _ *fakeNative) { a.SnapshotReference = "" }},
		{"missing operator evidence", func(a *LabAuthorization, _ *fakeNative) { a.OperatorEvidenceSHA256 = "" }},
		{"normal installation scope", func(a *LabAuthorization, _ *fakeNative) { a.Scope = "production" }},
		{"candidate not acknowledged", func(a *LabAuthorization, _ *fakeNative) { a.CandidateAcknowledged = false }},
		{"stale package state", func(_ *LabAuthorization, f *fakeNative) { f.s.HistorySHA256 = hash("changed") }},
		{"changed under lock", func(_ *LabAuthorization, f *fakeNative) { f.driftAt = 2 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := auth(st)
			f := &fakeNative{s: s, tx: tx, after: applied(s, tx)}
			tc.change(&a, f)
			_, e := ExecuteLab(context.Background(), st, m, a, filepath.Join(t.TempDir(), "state"), Native{f}, false, nil)
			if e == nil || f.mutations != 0 {
				t.Fatal("unauthorized command", e)
			}
		})
	}
	dir := filepath.Join(t.TempDir(), "state")
	lock, e := journal.Acquire(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer lock.Close()
	f := &fakeNative{s: s, tx: tx}
	_, e = ExecuteLab(context.Background(), st, m, auth(st), dir, Native{f}, false, nil)
	if !errors.Is(e, journal.ErrBusy) || f.mutations != 0 {
		t.Fatal("lock contention bypassed", e)
	}
}
func TestProvisioningFailures(t *testing.T) {
	p, m := inputs(t)
	s := fixture("autoconf")
	tx := transaction(s)
	st, _ := PreparePackages(p, m, s, tx)
	for _, category := range []execution.Category{execution.Exit, execution.Timeout, execution.Canceled, execution.Start, execution.OutputLimit} {
		t.Run(string(category), func(t *testing.T) {
			f := &fakeNative{s: s, tx: tx, failure: &execution.Error{Category: category, ExitCode: -1}}
			out, e := ExecuteLab(context.Background(), st, m, auth(st), filepath.Join(t.TempDir(), "state"), Native{f}, false, nil)
			var ex *execution.Error
			if !errors.As(e, &ex) || ex.Category != category || out.Journal.Actions[0].State != journal.Failed {
				t.Fatal(out, e)
			}
		})
	}
	f := &fakeNative{s: s, tx: tx, after: s}
	out, e := ExecuteLab(context.Background(), st, m, auth(st), filepath.Join(t.TempDir(), "state"), Native{f}, false, nil)
	var ex *execution.Error
	if !errors.As(e, &ex) || ex.Category != execution.Postcondition || out.Journal.Actions[0].Failure != execution.Postcondition {
		t.Fatal("exit zero became false success", out, e)
	}
	res, _ := PrepareResolution(p, m, s)
	f = &fakeNative{s: s, tx: tx, badOutput: true}
	out, e = ExecuteLab(context.Background(), res, m, auth(res), filepath.Join(t.TempDir(), "state"), Native{f}, false, nil)
	if e == nil || out.Journal.Actions[0].State != journal.Failed {
		t.Fatal("invalid metadata accepted", out, e)
	}
}
