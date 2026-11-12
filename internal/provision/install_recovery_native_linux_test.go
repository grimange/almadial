//go:build linux

package provision

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grimange/almadial/internal/artifacts"
	"github.com/grimange/almadial/internal/journal"
)

// This explicitly opted-in qualification runs only on the disposable ALM-29 D
// host. It changes copied journals, never the canonical installation journal.
// It is absent from the shipped CLI and is skipped by normal repository tests.
func TestInstallationNativeCopiedRecovery(t *testing.T) {
	root := os.Getenv("ALM30_NATIVE_OUTPUT")
	if root == "" {
		t.Skip("explicit disposable-host qualification only")
	}
	machine, err := os.ReadFile("/etc/machine-id")
	if err != nil || strings.TrimSpace(string(machine)) != "4c7b492940923838a0700adba65ed3eb" || os.Geteuid() != 0 || !strings.HasPrefix(root, "/var/lib/almadial-alm30/") {
		t.Fatal("qualification identity/output boundary")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	raw, err := os.ReadFile("/var/lib/almadial-inputs/plan.json")
	if err != nil {
		t.Fatal(err)
	}
	raw = bytes.TrimSpace(raw)
	request, err := InstallPlanRequest(raw)
	if err != nil {
		t.Fatal(err)
	}
	mb, err := os.ReadFile("/var/lib/almadial-inputs/stack.lock.yaml")
	if err != nil {
		t.Fatal(err)
	}
	m, err := artifacts.Load(bytes.NewReader(mb))
	if err != nil {
		t.Fatal(err)
	}
	sources, err := LoadInstallSources(ctx, m, request.Sources)
	if err != nil {
		t.Fatal(err)
	}
	p, err := DecodeInstallPlan(raw, m, sources)
	if err != nil {
		t.Fatal(err)
	}
	native, err := NewNative(nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = AssessInstall(ctx, p, m, sources, native, "/var/lib/almadial-install"); err != nil {
		t.Fatal("initial fresh postconditions", err)
	}
	for n := 1; n <= 10; n++ {
		name := fmt.Sprintf("S%02d", n)
		t.Run(name, func(t *testing.T) {
			dest := filepath.Join(root, name)
			if err := os.MkdirAll(dest, 0700); err != nil {
				t.Fatal(err)
			}
			err := filepath.WalkDir("/var/lib/almadial-install", func(path string, d fs.DirEntry, e error) error {
				if e != nil {
					return e
				}
				rel, e := filepath.Rel("/var/lib/almadial-install", path)
				if e != nil {
					return e
				}
				target := filepath.Join(dest, rel)
				if d.IsDir() {
					return os.MkdirAll(target, 0700)
				}
				if d.Type()&os.ModeSymlink != 0 {
					return fmt.Errorf("unexpected state symlink")
				}
				if d.Name() == "apply.lock" {
					return nil
				}
				b, e := os.ReadFile(path)
				if e != nil {
					return e
				}
				f, e := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
				if e != nil {
					return e
				}
				_, e = f.Write(b)
				ce := f.Close()
				if e != nil {
					return e
				}
				return ce
			})
			if err != nil {
				t.Fatal(err)
			}
			journalStage := name
			if n == 1 {
				journalStage = "S01-apply"
			}
			original, err := journal.Inspect(filepath.Join(dest, journalStage))
			if err != nil {
				t.Fatal(err)
			}
			if len(original.Actions) == 0 {
				t.Fatal("no representative native action")
			}
			interrupted := original
			interrupted.Actions = append([]journal.Record(nil), original.Actions...)
			last := len(interrupted.Actions) - 1
			interrupted.Actions[last].State = journal.Running
			interrupted.Actions[last].Evidence = ""
			interrupted.Actions[last].Failure = ""
			interrupted.Actions[last].ExitCode = -1
			if err = interrupted.Validate(); err != nil {
				t.Fatal(err)
			}
			b, _ := json.Marshal(interrupted)
			if err = os.WriteFile(filepath.Join(dest, journalStage, "journal.json"), b, 0600); err != nil {
				t.Fatal(err)
			}
			// Reconcile through the accepted observer only. No stage executor,
			// action runner, receipt deletion, or provisioning entry point is called.
			r := installRun{ctx: ctx, m: m, sources: sources, native: native, state: dest}
			observer, err := nativeCopiedObserver(&r, name)
			if err != nil {
				t.Fatal(err)
			}
			session, err := journal.Acquire(filepath.Join(dest, journalStage))
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close()
			expected := original
			expected.Actions = append([]journal.Record(nil), original.Actions...)
			for k := range expected.Actions {
				expected.Actions[k].State = journal.Pending
				expected.Actions[k].Evidence = ""
				expected.Actions[k].Failure = ""
				expected.Actions[k].ExitCode = -1
			}
			err = session.Resume(ctx, expected, observer)
			final, e := session.Snapshot()
			if e != nil {
				t.Fatal(e)
			}
			if err != nil {
				if !errors.Is(err, journal.ErrReconcile) {
					t.Fatal(err)
				}
				if final.Actions[last].State != journal.ReconciliationRequired {
					t.Fatal("missing explicit block")
				}
				t.Logf("stage=%s action=%s outcome=explicit-review-required zero_executor_calls=true", name, original.Actions[last].ID)
				return
			}
			if final.Identity != original.Identity || final.Actions[last].State != journal.Succeeded {
				t.Fatal("reconciliation identity/outcome")
			}
			session.Close()
			session, err = journal.Acquire(filepath.Join(dest, journalStage))
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close()
			if err = session.Resume(ctx, expected, observer); err != nil {
				t.Fatal("repeat reconciliation", err)
			}
			t.Logf("stage=%s action=%s outcome=reconciled identity_preserved=true zero_executor_calls=true", name, original.Actions[last].ID)
		})
	}
}

// Construct only the existing read-only observers, using original immutable
// stage inputs. Deliberately no execution runner is exposed to this function.
func nativeCopiedObserver(r *installRun, name string) (journal.Observer, error) {
	var ph PhoneInputs
	if e := r.read("S10-accepted", &ph); e != nil {
		return nil, e
	}
	phone := r.phoneNative(ph)
	net := phone.Network
	tls := net.TLS
	web := tls.Web
	rt := web.Runtime
	app := rt.Application
	ast := app.Asterisk
	db := ast.Database
	switch name {
	case "S01":
		var i Inputs
		if e := r.read("S01-apply", &i); e != nil {
			return nil, e
		}
		s, e := i.Prepare(r.m)
		if e != nil {
			return nil, e
		}
		return &checks{native: r.native, d: s.doc()}, nil
	case "S02":
		var i SystemInputs
		if e := r.read(name+"-before", &i); e != nil {
			return nil, e
		}
		s, e := i.Prepare(r.m)
		if e != nil {
			return nil, e
		}
		return &systemChecks{host: r.native, d: s.doc()}, nil
	case "S03":
		var i DatabaseInputs
		if e := r.read(name+"-before", &i); e != nil {
			return nil, e
		}
		s, e := i.Prepare(r.m, r.sources.Database)
		if e != nil {
			return nil, e
		}
		return &databaseChecks{host: db, d: s.doc()}, nil
	case "S04":
		var i AsteriskInputs
		if e := r.read(name+"-before", &i); e != nil {
			return nil, e
		}
		s, e := i.Prepare(r.m, r.sources.Asterisk)
		if e != nil {
			return nil, e
		}
		return &asteriskChecks{host: ast, d: s.doc()}, nil
	case "S05":
		var i VICIdialInputs
		if e := r.read(name+"-before", &i); e != nil {
			return nil, e
		}
		s, e := i.Prepare(r.m, r.sources.Application, r.sources.Perl, r.sources.Asterisk, r.sources.Database)
		if e != nil {
			return nil, e
		}
		return &vicidialChecks{host: app, d: s.doc()}, nil
	case "S06":
		var i RuntimeInputs
		if e := r.read(name+"-before", &i); e != nil {
			return nil, e
		}
		return &runtimeChecks{host: rt, before: i.Snapshot}, nil
	case "S07":
		var i WebInputs
		if e := r.read(name+"-before", &i); e != nil {
			return nil, e
		}
		return &webChecks{host: web, before: i.Snapshot}, nil
	case "S08":
		var i TLSInputs
		if e := r.read(name+"-before", &i); e != nil {
			return nil, e
		}
		return &tlsChecks{host: tls, before: i.Snapshot}, nil
	case "S09":
		var i NetworkInputs
		if e := r.read(name+"-before", &i); e != nil {
			return nil, e
		}
		return &networkChecks{host: net, before: i.Snapshot}, nil
	case "S10":
		var i PhoneInputs
		if e := r.read(name+"-before", &i); e != nil {
			return nil, e
		}
		return &phoneChecks{host: phone, before: i.Snapshot}, nil
	}
	return nil, fmt.Errorf("unknown stage")
}
