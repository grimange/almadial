// Command almadial-media-lab guards the single ALM-27 disposable-lab experiment.
// It is an acceptance tool, not an installation stage or production interface.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/grimange/almadial/internal/execution"
	"github.com/grimange/almadial/internal/journal"
)

func hash(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

type checks struct {
	runner         *execution.Runner
	repo, evidence string
	scenario       string
	profile        string
}

func (c checks) command(mode string) execution.Command {
	script := "m3_05_run.py"
	if c.scenario == "reboot" {
		script = "m3_05_reboot.py"
	}
	if c.scenario == "agent" {
		script = "m3_05_agent.py"
	}
	args := []string{filepath.Join(c.repo, "scripts/lab", script), mode, c.evidence}
	if c.profile != "" {
		args = append(args, c.profile)
	}
	return execution.Command{Executable: "/usr/bin/python3", Args: args, Timeout: 8 * time.Minute, StdoutLimit: 65536, StderrLimit: 65536}
}
func (c checks) Supports(id string) bool { return id == "media.preflight" || id == "media.cleaned" }
func (c checks) Check(ctx context.Context, id string) error {
	mode := "inspect"
	if id == "media.cleaned" {
		mode = "verify"
	}
	r, e := c.runner.Run(ctx, c.command(mode))
	if e != nil {
		return fmt.Errorf("%s: %w: %s", id, e, r.Stderr)
	}
	return nil
}
func (c checks) Observe(ctx context.Context, _ journal.Record) (journal.Observation, error) {
	r, e := c.runner.Run(ctx, c.command("verify"))
	if e != nil {
		return journal.Observation{State: journal.Unknown}, e
	}
	return journal.Observation{State: journal.Satisfied, Evidence: hash([]byte(r.Stdout))}, nil
}
func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func run() error {
	mode := flag.String("mode", "plan", "plan or apply")
	scenario := flag.String("scenario", "media", "media or separately authorized reboot checkpoint")
	profile := flag.String("target-profile", "", "explicit ALM-29 clean-host media or reboot profile")
	stateDirectory := flag.String("journal-directory", "", "existing copied journal for reconciliation; never executes effects")
	evidence := flag.String("evidence", "", "fresh protected experiment directory")
	approved := flag.String("approved-plan", "", "reviewed plan digest")
	authorized := flag.Bool("authorize-disposable-lab", false, "authorize the fixed ALM-27 laboratory target")
	flag.Parse()
	repo, e := os.Getwd()
	if e != nil {
		return e
	}
	if !filepath.IsAbs(*evidence) {
		return fmt.Errorf("absolute protected evidence path required")
	}
	runner, e := execution.NewRunner(nil)
	if e != nil {
		return e
	}
	if *scenario != "media" && *scenario != "reboot" && *scenario != "agent" {
		return fmt.Errorf("unsupported bounded scenario")
	}
	hostID := "0fffa9a4683f35b696f0b6ee7330734b"
	var profileBytes []byte
	if *profile != "" {
		if (*scenario != "media" && *scenario != "reboot") || !filepath.IsAbs(*profile) {
			return fmt.Errorf("target profile requires media/reboot scenario and absolute path")
		}
		profileBytes, e = os.ReadFile(*profile)
		if e != nil {
			return e
		}
		var target struct {
			Machine string `json:"machine"`
			Task    string `json:"task"`
		}
		if json.Unmarshal(profileBytes, &target) != nil || target.Task != "xu7wxvch94ii2iw38hpb8d58" || len(target.Machine) != 32 {
			return fmt.Errorf("invalid ALM-29 target profile")
		}
		if _, e := hex.DecodeString(target.Machine); e != nil {
			return e
		}
		hostID = target.Machine
	}
	c := checks{runner, repo, *evidence, *scenario, *profile}
	ctx := context.Background()
	if e = c.Check(ctx, "media.preflight"); e != nil {
		return e
	}
	actionID := "media.00.synthetic"
	description := "Confined two-browser synthetic conference with mandatory cleanup"
	if *scenario == "reboot" {
		actionID, description = "media.01.reboot", "Separately authorized single reboot; no active calls or temporary media policy"
	}
	if *scenario == "agent" {
		actionID, description = "media.02.agent", "Isolated real agent controls; clone database, local calls, mandatory cleanup"
	}
	graph, e := execution.NewGraph([]execution.Action{{ID: actionID, StageID: "MediaAcceptance", Description: description, Preconditions: []string{"media.preflight"}, Command: c.command("experiment"), Postconditions: []string{"media.cleaned"}}})
	if e != nil {
		return e
	}
	// Bind every executable fixture, the manifest, the immutable graph and the
	// current read-only host observation before acquiring execution ownership.
	sources := map[string]string{}
	for _, p := range []string{"cmd/almadial-media-lab/main.go", "scripts/lab/m3_05_agent.py", "scripts/lab/m3_05_agent_guest.py", "scripts/lab/m3_05_agent_browser.cjs", "scripts/lab/m0_07_assets.py", "internal/provision/phone_native.py", "scripts/lab/m3_05_reboot.py", "scripts/lab/m3_05_run.py", "scripts/lab/m3_05_native.py", "scripts/lab/m3_05_browser.cjs", "scripts/lab/m3_05_tunnel.py", "scripts/lab/m0_05_capture.js", "scripts/lab/m0_03_signal.py", "manifests/stack.lock.yaml"} {
		b, err := os.ReadFile(filepath.Join(repo, p))
		if err != nil {
			return err
		}
		sources[p] = hash(b)
	}
	if len(profileBytes) != 0 {
		sources["target-profile"] = hash(profileBytes)
	}
	b, _ := json.Marshal(sources)
	r, e := runner.Run(ctx, c.command("inspect"))
	if e != nil {
		return e
	}
	hostState := hash([]byte(r.Stdout))
	policy := "ALM27 synthetic LAN; snapshot alm25-pre-firewall-20261111; expire source-scoped RTP in 600 seconds; cleanup"
	if len(profileBytes) != 0 {
		policy = "ALM29 disposable clean-host qualification; snapshot alm29-pristine; bounded synthetic LAN media or idle reboot; cleanup; profile " + hash(profileBytes)
	}
	identity := journal.Identity{Host: hash([]byte(hostID)), HostState: hostState, Config: hash(b), Manifest: sources["manifests/stack.lock.yaml"], Plan: hash(append(append(b, graph.JSON()...), []byte(hostState)...)), Policy: hash([]byte(policy)), Graph: graph.Digest()}
	state := filepath.Join(*evidence, "execution")
	if *stateDirectory != "" {
		if *mode != "reconcile" {
			return fmt.Errorf("alternate journal is limited to reconcile mode")
		}
		state = *stateDirectory
	}
	previous, readErr := journal.Inspect(state)
	complete := false
	if readErr == nil {
		if previous.Identity != identity {
			return journal.ErrIdentity
		}
		observation, err := c.Observe(ctx, journal.Record{})
		if err != nil || observation.State != journal.Satisfied {
			return fmt.Errorf("recorded experiment no longer satisfies fresh checks")
		}
		complete = len(previous.Actions) == 1 && previous.Actions[0].ID == actionID && previous.Actions[0].State == journal.Succeeded
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return readErr
	}
	actions := graph.Actions()
	if complete {
		actions = []execution.Action{}
	}
	out, _ := json.Marshal(map[string]any{"identity": identity, "actions": actions, "no_op": complete})
	fmt.Println(string(out))
	if *mode == "plan" {
		return nil
	}
	if (*mode != "apply" && *mode != "reconcile") || !*authorized || *approved != identity.Plan {
		return fmt.Errorf("explicit laboratory authorization and matching reviewed plan required")
	}
	ownership, e := journal.Acquire(filepath.Join(filepath.Dir(*evidence), "media-execution-ownership"))
	if e != nil {
		return e
	}
	defer ownership.Close()
	session, e := journal.Acquire(state)
	if e != nil {
		return e
	}
	defer session.Close()
	doc, e := journal.New(identity, graph, map[string]journal.Retry{actionID: journal.Never})
	if e != nil {
		return e
	}
	if readErr == nil {
		if e = session.Resume(ctx, doc, c); e != nil {
			return e
		}
		reconciled, err := session.Snapshot()
		if err != nil || len(reconciled.Actions) != 1 || reconciled.Actions[0].State != journal.Succeeded {
			return journal.ErrReconcile
		}
		fmt.Println(`{"no_op":true,"runner_mutations":0,"reconciled":true}`)
		return nil
	}
	if *mode == "reconcile" {
		return fmt.Errorf("reconciliation requires an existing journal")
	}
	if e = session.Initialize(doc); e != nil {
		return e
	}
	if e = session.Begin(actionID); e != nil {
		return e
	}
	traces, execErr := graph.Execute(ctx, runner, c)
	for _, t := range traces {
		raw, _ := json.Marshal(t)
		fmt.Println(string(raw))
		if e = session.Finish(ctx, t, c); e != nil {
			return e
		}
	}
	return execErr
}
