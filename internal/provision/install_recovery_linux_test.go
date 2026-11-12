//go:build linux

package provision

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grimange/almadial/internal/execution"
	"github.com/grimange/almadial/internal/journal"
)

// Injection exists only in this test binary. These fixtures exercise the unified
// adapter and the real journal, not DNF, DDL or native service implementations.
// Native completed-effect reconciliation is qualified separately on copied lab
// journals; component tests retain ownership of partial-stage retry decisions.
type installObservation func(context.Context, journal.Record) (journal.Observation, error)

func (f installObservation) Observe(c context.Context, r journal.Record) (journal.Observation, error) {
	return f(c, r)
}

func TestInstallationBoundaryInjection(t *testing.T) {
	for _, label := range installStages[:10] {
		stage := strings.Fields(label)[0]
		for _, boundary := range []string{"before-effect", "after-effect"} {
			t.Run(stage+"/"+boundary, func(t *testing.T) {
				dir := filepath.Join(t.TempDir(), "install")
				owner, err := journal.Acquire(dir)
				if err != nil {
					t.Fatal(err)
				}
				defer owner.Close()
				r := installRun{ctx: context.Background(), state: dir, lock: owner}
				id := stage + ".fixture"
				graph, err := execution.NewGraph([]execution.Action{{ID: id, StageID: stage, Description: "ALM-30 adapter fixture", Dependencies: []string{}, Preconditions: []string{"eligible"}, Postconditions: []string{"complete"}, Command: execution.Command{Executable: "/fixture/effect", Timeout: time.Second, StdoutLimit: 128, StderrLimit: 128}}})
				if err != nil {
					t.Fatal(err)
				}
				retry := journal.Idempotent
				// Unsafe absent transaction/schema state remains explicitly blocked; the
				// accepted component observers decide whether real state is safely absent.
				if stage == "S01" || stage == "S03" {
					retry = journal.Never
				}
				doc, err := journal.New(journal.Identity{Host: hash("host"), HostState: hash("hoststate"), Config: hash("config"), Manifest: hash("manifest"), Plan: hash("plan"), Policy: hash("policy"), Graph: graph.Digest()}, graph, map[string]journal.Retry{id: retry})
				if err != nil {
					t.Fatal(err)
				}
				effects := 0
				observer := installObservation(func(context.Context, journal.Record) (journal.Observation, error) {
					state := journal.Absent
					if effects == 1 {
						state = journal.Satisfied
					}
					return journal.Observation{State: state, Evidence: hash(fmt.Sprint(effects))}, nil
				})
				injected := errors.New("ALM-30 deterministic interruption")
				first := true
				execute := func(before int, resume bool) error {
					if before != 0 {
						t.Fatal("original approved input replaced")
					}
					s, e := journal.Acquire(filepath.Join(dir, stage))
					if e != nil {
						return e
					}
					defer s.Close()
					if resume {
						e = s.Resume(context.Background(), doc, observer)
					} else {
						e = s.Initialize(doc)
					}
					if e != nil {
						return e
					}
					observed, _ := s.Snapshot()
					if observed.Actions[0].State == journal.Succeeded {
						return nil
					}
					if e = s.Begin(id); e != nil {
						return e
					}
					if first && boundary == "before-effect" {
						first = false
						return injected
					}
					effects++
					if first {
						first = false
						return injected
					}
					return s.Finish(context.Background(), execution.Trace{ActionID: id, StageID: stage, Outcome: "succeeded", Result: execution.Result{Started: true}}, observer)
				}
				inspect := func() (int, error) { return effects, nil }
				if _, err = driveInstallStage(&r, stage, inspect, execute); !errors.Is(err, injected) {
					t.Fatal(err)
				}
				var receipt int
				if err = r.read(stage+"-accepted", &receipt); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("false completion", err)
				}
				interrupted, err := journal.Inspect(filepath.Join(dir, stage))
				if err != nil || interrupted.Actions[0].State != journal.Running {
					t.Fatal(interrupted, err)
				}
				_, err = driveInstallStage(&r, stage, inspect, execute)
				if boundary == "before-effect" && retry == journal.Never {
					if !errors.Is(err, journal.ErrReconcile) || effects != 0 {
						t.Fatal("unsafe replay", err, effects)
					}
					return
				}
				if err != nil || effects != 1 {
					t.Fatal(err, effects)
				}
				final, err := journal.Inspect(filepath.Join(dir, stage))
				if err != nil || final.Identity != doc.Identity || final.Actions[0].State != journal.Succeeded {
					t.Fatal(final, err)
				}
				_, err = driveInstallStage(&r, stage, inspect, func(int, bool) error { t.Fatal("receipt replay"); return nil })
				if err != nil || effects != 1 {
					t.Fatal(err, effects)
				}
			})
		}
	}
}

// The child acknowledges lock ownership on stdout, then waits for stdin EOF.
// No sleeps, polling or alternate lock implementation establish concurrency.
func TestInstallationLockProcess(t *testing.T) {
	if os.Getenv("ALM30_LOCK_CHILD") == "1" {
		owner, err := journal.Acquire(os.Getenv("ALM30_LOCK_DIR"))
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println("owned")
		_, _ = io.Copy(io.Discard, os.Stdin)
		if err = owner.Close(); err != nil {
			t.Fatal(err)
		}
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	dir := filepath.Join(t.TempDir(), "install")
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestInstallationLockProcess$")
	cmd.Env = append(os.Environ(), "ALM30_LOCK_CHILD=1", "ALM30_LOCK_DIR="+dir)
	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { in.Close(); _ = cmd.Wait() }()
	ack := make([]byte, 6)
	if _, err = io.ReadFull(out, ack); err != nil || string(ack) != "owned\n" {
		t.Fatal("ownership handshake", err, string(ack))
	}
	for _, attempt := range []string{"install", "resume"} {
		t.Run(attempt, func(t *testing.T) {
			if s, e := journal.Acquire(dir); !errors.Is(e, journal.ErrBusy) {
				if s != nil {
					s.Close()
				}
				t.Fatal("concurrent owner admitted", e)
			}
		})
	}
	in.Close()
	if err = cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	owner, err := journal.Acquire(dir)
	if err != nil {
		t.Fatal("released lock not reusable", err)
	}
	owner.Close()
}
