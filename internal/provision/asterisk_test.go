//go:build linux

package provision

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grimange/almadial/internal/artifacts"
	"github.com/grimange/almadial/internal/execution"
	"github.com/grimange/almadial/internal/journal"
)

func asteriskInputs(t *testing.T) (AsteriskInputs, artifacts.Manifest, AsteriskSource) {
	p, m, _ := databaseInputs(t, true)
	src := AsteriskSource{[]byte("synthetic source; never executable"), []byte("synthetic PJ")}
	s := AsteriskSnapshot{Host: p.Snapshot.Host, Foundation: p.Parent.Snapshot.Digest(), Database: p.Snapshot.Digest(), Source: src.Digest(), Toolchain: hash("synthetic GCC/make"), EnvironmentOK: true, Build: "absent", Install: "absent", Configuration: "absent", Unit: "absent", Modules: []AsteriskModule{}}
	return AsteriskInputs{p, s, src.Digest()}, m, src
}
func astAuth(s AsteriskStage) AsteriskLabAuthorization {
	d := s.doc()
	return AsteriskLabAuthorization{"ALM-19/disposable-lab", d.Before.Host, s.Digest(), d.Parent, d.Manifest, d.Source, hash("synthetic fresh snapshot"), hash("synthetic explicit scope"), true}
}

type astFake struct {
	s                      AsteriskSnapshot
	calls, failAt, crashAt int
	drift, omit            bool
}

func (f *astFake) InspectAsterisk(context.Context) (AsteriskSnapshot, error) {
	s := f.s
	s.Modules = append([]AsteriskModule{}, s.Modules...)
	return s, nil
}
func (f *astFake) Validate(execution.Command) error { return nil }
func (f *astFake) Run(_ context.Context, c execution.Command) (execution.Result, error) {
	f.calls++
	if f.failAt == f.calls {
		return execution.Result{Started: true, ExitCode: 1}, errors.New("synthetic bounded failure")
	}
	switch c.Args[1] {
	case "asterisk.00.build":
		f.s.Build = hash("build")
		for _, m := range AsteriskModules() {
			f.s.Modules = append(f.s.Modules, AsteriskModule{Name: m, Source: true, Built: true})
		}
	case "asterisk.01.install", "asterisk.01.libraries", "asterisk.01.resources":
		f.s.Install = hash("install")
		for j := range f.s.Modules {
			f.s.Modules[j].Installed = true
		}
	case "asterisk.02.config":
		f.s.Configuration = hash("config")
	case "asterisk.03.unit":
		f.s.Unit = hash("unit")
	case "asterisk.04.service":
		f.s.Active = true
		f.s.Enabled = true
		f.s.RuntimeOK = true
		f.s.ServiceID = "service1"
		for j := range f.s.Modules {
			f.s.Modules[j].Loaded = true
			f.s.Modules[j].Pass = true
		}
		if f.omit {
			f.s.Modules[7].Loaded = false
			f.s.Modules[7].Pass = false
			f.s.RuntimeOK = false
		}
	case "asterisk.05.restart":
		f.s.ServiceID = "service2"
		f.s.RestartVerified = true
	}
	if f.drift {
		f.s.Database = hash("unrelated schema mutation")
	}
	if f.crashAt == f.calls {
		panic("synthetic loss after mutation before outcome")
	}
	return execution.Result{Started: true, ExitCode: 0}, nil
}
func TestAsteriskPlanRuntimeAndNoOp(t *testing.T) {
	i, m, src := asteriskInputs(t)
	s, e := i.Prepare(m, src)
	if e != nil {
		t.Fatal(e)
	}
	again, _ := i.Prepare(m, src)
	if s.Digest() != again.Digest() || len(s.Graph().Actions()) != 6 {
		t.Fatal("nondeterministic graph")
	}
	f := &astFake{s: i.Snapshot}
	dir := filepath.Join(t.TempDir(), "journal")
	o, e := ExecuteAsteriskLab(context.Background(), s, m, astAuth(s), dir, f, f, false, nil)
	if e != nil || o.NoOp || !asteriskSatisfied(f.s) {
		t.Fatal(e, o)
	}
	for _, v := range o.Journal.Actions {
		if v.State != journal.Succeeded || v.Retry != journal.Never || v.Evidence == "" {
			t.Fatal(v)
		}
	}
	o, e = ExecuteAsteriskLab(context.Background(), s, m, astAuth(s), dir, f, f, true, nil)
	if e != nil || !o.NoOp || f.calls != 6 {
		t.Fatal("resume", e)
	}
	i.Snapshot = f.s
	s, e = i.Prepare(m, src)
	if e != nil || len(s.Graph().Actions()) != 0 {
		t.Fatal(e)
	}
	o, e = ExecuteAsteriskLab(context.Background(), s, m, astAuth(s), filepath.Join(t.TempDir(), "noop"), f, f, false, nil)
	if e != nil || !o.NoOp || f.calls != 6 {
		t.Fatal("fresh no-op", e)
	}
}
func TestAsteriskRejectInputs(t *testing.T) {
	for _, name := range []string{"wrong host", "stale database", "stale foundation", "changed source", "missing source", "missing dependency", "wrong toolchain", "operator config", "partial build", "partial install", "conflicting unit", "missing authorization", "wrong signer scope", "stale plan", "missing snapshot", "candidate authorization"} {
		t.Run(name, func(t *testing.T) {
			i, m, src := asteriskInputs(t)
			switch name {
			case "wrong host":
				i.Snapshot.Host = hash("other")
			case "stale database":
				i.Snapshot.Database = hash("other")
			case "stale foundation":
				i.Snapshot.Foundation = hash("other")
			case "changed source":
				i.SourceSHA256 = hash("other")
			case "missing source":
				src = AsteriskSource{}
			case "missing dependency":
				i.Snapshot.EnvironmentOK = false
			case "wrong toolchain":
				i.Snapshot.Toolchain = ""
			case "operator config":
				i.Snapshot.Configuration = "conflict"
			case "partial build":
				i.Snapshot.Build = "invalid"
			case "partial install":
				i.Snapshot.Install = "invalid"
			case "conflicting unit":
				i.Snapshot.Unit = "conflict"
			}
			s, e := i.Prepare(m, src)
			if strings.HasPrefix(name, "missing authorization") || name == "wrong signer scope" || name == "stale plan" || name == "missing snapshot" || name == "candidate authorization" {
				if e != nil {
					t.Fatal(e)
				}
				a := astAuth(s)
				switch name {
				case "missing authorization":
					a = AsteriskLabAuthorization{}
				case "wrong signer scope":
					a.Scope = "production"
				case "stale plan":
					a.StageSHA256 = hash("stale")
				case "missing snapshot":
					a.SnapshotReference = ""
				case "candidate authorization":
					a.CandidateAcknowledged = false
				}
				f := &astFake{s: i.Snapshot}
				_, e = ExecuteAsteriskLab(context.Background(), s, m, a, filepath.Join(t.TempDir(), "state"), f, f, false, nil)
				if f.calls != 0 {
					t.Fatal("mutation before authorization")
				}
			}
			if e == nil {
				t.Fatal("accepted invalid input")
			}
		})
	}
}
func TestAsteriskModuleContract(t *testing.T) {
	i, _, _ := asteriskInputs(t)
	f := astFake{s: i.Snapshot}
	for _, id := range []string{"asterisk.00.build", "asterisk.01.install", "asterisk.02.config", "asterisk.03.unit", "asterisk.04.service", "asterisk.05.restart"} {
		_, _ = f.Run(context.Background(), asteriskActionCommand(id))
	}
	if !asteriskSatisfied(f.s) {
		t.Fatal("fixture")
	}
	for j, m := range f.s.Modules {
		for _, field := range []string{"source", "build", "installed", "loaded", "pass"} {
			t.Run(m.Name+"/"+field, func(t *testing.T) {
				s := f.s
				s.Modules = append([]AsteriskModule{}, f.s.Modules...)
				switch field {
				case "source":
					s.Modules[j].Source = false
				case "build":
					s.Modules[j].Built = false
				case "installed":
					s.Modules[j].Installed = false
				case "loaded":
					s.Modules[j].Loaded = false
				case "pass":
					s.Modules[j].Pass = false
				}
				if asteriskSatisfied(s) {
					t.Fatal("accepted missing native requirement")
				}
			})
		}
	}
	for _, field := range []string{"identity", "startup", "restart", "selinux", "capability"} {
		t.Run(field, func(t *testing.T) {
			s := f.s
			switch field {
			case "identity":
				s.ServiceID = ""
			case "startup":
				s.Active = false
			case "restart":
				s.RestartVerified = false
			case "selinux":
				s.EnvironmentOK = false
			case "capability":
				s.RuntimeOK = false
			}
			if asteriskSatisfied(s) {
				t.Fatal("false runtime success")
			}
		})
	}
}
func TestAsteriskActionFailureAndBaselinePreservation(t *testing.T) {
	for j := 1; j <= 6; j++ {
		t.Run(fmt.Sprint(j), func(t *testing.T) {
			i, m, src := asteriskInputs(t)
			s, _ := i.Prepare(m, src)
			f := &astFake{s: i.Snapshot, failAt: j}
			o, e := ExecuteAsteriskLab(context.Background(), s, m, astAuth(s), filepath.Join(t.TempDir(), "state"), f, f, false, nil)
			if e == nil || f.calls != j || len(o.Traces) != j {
				t.Fatal("false success or unsafe continuation", e)
			}
			if f.s.Database != i.Snapshot.Database {
				t.Fatal("baseline changed")
			}
		})
	}
	for _, kind := range []string{"unrelated database mutation", "missing runtime module"} {
		t.Run(kind, func(t *testing.T) {
			i, m, src := asteriskInputs(t)
			s, _ := i.Prepare(m, src)
			f := &astFake{s: i.Snapshot, drift: kind == "unrelated database mutation", omit: kind == "missing runtime module"}
			_, e := ExecuteAsteriskLab(context.Background(), s, m, astAuth(s), filepath.Join(t.TempDir(), "state"), f, f, false, nil)
			if e == nil {
				t.Fatal("failed postcondition accepted")
			}
		})
	}
}
func TestAsteriskInterruptedRecovery(t *testing.T) {
	for j := 1; j <= 6; j++ {
		t.Run(fmt.Sprint(j), func(t *testing.T) {
			i, m, src := asteriskInputs(t)
			s, _ := i.Prepare(m, src)
			f := &astFake{s: i.Snapshot, crashAt: j}
			dir := filepath.Join(t.TempDir(), "state")
			func() {
				defer func() {
					if recover() == nil {
						t.Fatal("expected loss")
					}
				}()
				_, _ = ExecuteAsteriskLab(context.Background(), s, m, astAuth(s), dir, f, f, false, nil)
			}()
			f.crashAt = 0
			_, e := ExecuteAsteriskLab(context.Background(), s, m, astAuth(s), dir, f, f, true, nil)
			if j == 6 {
				if e == nil {
					t.Fatal("restart without durable outcome must require review")
				}
			} else if e != nil {
				t.Fatal("independently satisfied action not reconciled", e)
			}
		})
	}
	i, m, src := asteriskInputs(t)
	s, _ := i.Prepare(m, src)
	f := &astFake{s: i.Snapshot}
	dir := filepath.Join(t.TempDir(), "state")
	lock, e := journal.Acquire(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer lock.Close()
	if _, e := ExecuteAsteriskLab(context.Background(), s, m, astAuth(s), dir, f, f, false, nil); e == nil || f.calls != 0 {
		t.Fatal("lock bypass")
	}
}
func TestAsteriskListenerPolicy(t *testing.T) {
	good := "tcp LISTEN 0 10 127.0.0.1:5038 0.0.0.0:* users:((\"asterisk\",pid=123,fd=1))\ntcp LISTEN 0 10 127.0.0.1:8088 0.0.0.0:* users:((\"asterisk\",pid=123,fd=2))\n"
	for _, tc := range []struct {
		name, raw string
		ok        bool
	}{{"local", good, true}, {"PJ resolver", good + "udp UNCONN 0 0 0.0.0.0:45000 0.0.0.0:* users:((\"asterisk\",pid=123,fd=9))\n" + "udp UNCONN 0 0 [::]:45001 [::]:* users:((\"asterisk\",pid=123,fd=10))", true}, {"public AMI", strings.Replace(good, "127.0.0.1:5038", "0.0.0.0:5038", 1), false}, {"public HTTP", strings.Replace(good, "127.0.0.1:8088", "[::]:8088", 1), false}, {"wrong local", strings.Replace(good, "127.0.0.1:8088", "127.0.0.2:8088", 1), false}, {"wrong owner", strings.ReplaceAll(good, "pid=123,", "pid=124,"), false}, {"absent HTTP", strings.Split(good, "\n")[0], false}, {"SIP exposed", good + "udp UNCONN 0 0 0.0.0.0:5060 0.0.0.0:* users:((\"asterisk\",pid=123,fd=3))", false}} {
		t.Run(tc.name, func(t *testing.T) {
			if (verifyAsteriskListeners(tc.raw, 123) == nil) != tc.ok {
				t.Fatal(tc.raw)
			}
		})
	}
}
func TestAsteriskSourceRejectsCorruption(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "source")
	if e := os.WriteFile(p, []byte("corrupted source"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := sourceBytes(p, asteriskArchiveSHA); e == nil {
		t.Fatal("digest bypass")
	}
	link := filepath.Join(d, "symlink")
	_ = os.Symlink(p, link)
	if _, e := sourceBytes(link, rawSHA([]byte("corrupted source"))); e == nil {
		t.Fatal("symlink accepted")
	}
	if verifyAsteriskLayout([]byte("wrong version archive")) == nil {
		t.Fatal("invalid source accepted")
	}
}

func TestAsteriskReceiptBackedRuntimeRepair(t *testing.T) {
	for _, state := range []string{"libraries-missing", "resources-missing"} {
		t.Run(state, func(t *testing.T) {
			i, m, src := asteriskInputs(t)
			i.Snapshot.Build = hash("verified compiled receipt")
			i.Snapshot.Install = state
			i.Snapshot.Configuration = hash("preserved configuration")
			i.Snapshot.Unit = hash("preserved unit")
			for _, name := range AsteriskModules() {
				i.Snapshot.Modules = append(i.Snapshot.Modules, AsteriskModule{Name: name, Source: true, Built: true, Installed: true})
			}
			s, e := i.Prepare(m, src)
			if e != nil {
				t.Fatal(e)
			}
			if len(s.Graph().Actions()) != 3 {
				t.Fatal("repair must not rebuild or replace config", s.Graph().Actions())
			}
			f := &astFake{s: i.Snapshot}
			o, e := ExecuteAsteriskLab(context.Background(), s, m, astAuth(s), filepath.Join(t.TempDir(), "journal"), f, f, false, nil)
			if e != nil || !asteriskSatisfied(f.s) || o.NoOp {
				t.Fatal(e, o)
			}
		})
	}
}

func TestAsteriskPinnedLayout(t *testing.T) {
	for _, name := range []string{"valid", "wrong version", "system PJPROJECT", "missing configure", "missing AGI"} {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			z := gzip.NewWriter(&buf)
			w := tar.NewWriter(z)
			for _, path := range []string{"configure", "Makefile", ".version", "third-party/pjproject/Makefile", "third-party/pjproject/pjproject-2.17.tar.bz2.md5", "res/res_agi.c", "apps/app_confbridge.c", "channels/chan_pjsip.c"} {
				if (name == "system PJPROJECT" && strings.Contains(path, "pjproject")) || (name == "missing configure" && path == "configure") || (name == "missing AGI" && path == "res/res_agi.c") {
					continue
				}
				data := []byte("fixture")
				if path == ".version" {
					data = []byte(AsteriskVersion)
					if name == "wrong version" {
						data = []byte("22.11.0")
					}
				}
				if e := w.WriteHeader(&tar.Header{Name: "asterisk-22.12.0/" + path, Mode: 0644, Size: int64(len(data))}); e != nil {
					t.Fatal(e)
				}
				if _, e := w.Write(data); e != nil {
					t.Fatal(e)
				}
			}
			if e := w.Close(); e != nil {
				t.Fatal(e)
			}
			if e := z.Close(); e != nil {
				t.Fatal(e)
			}
			if (verifyAsteriskLayout(buf.Bytes()) == nil) != (name == "valid") {
				t.Fatal("layout verification mismatch")
			}
		})
	}
}
