//go:build linux

package provision

import (
	"bytes"
	"compress/gzip"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/grimange/almadial/internal/execution"
)

//go:embed native_runtime.py
var nativeRuntimeAdapter string

//go:embed r4037_runtime.py
var nativeRuntimeDerivations string

func nativeRuntimeProgram() string {
	return strings.Split(phoneAdapterSource, "def rows(")[0] + nativeRuntimeDerivations + "\n" + nativeRuntimeAdapter
}
func NativeRuntimePolicyDigest() string { return rawSHA([]byte(nativeRuntimeProgram())) }

// NativeRuntimeHealthProgram performs only authenticated read-only readiness probes.
// It never exports credentials or database contents.
func NativeRuntimeHealthProgram() string { return nativeRuntimeProgram() }
func NativeRuntimeCron() string {
	return "# AlmaDial remote native VICIdial maintenance; systemd owns long-running workers.\nSHELL=/bin/bash\nPATH=/usr/sbin:/usr/bin:/sbin:/bin\nCRON_TZ=UTC\nMAILTO=\"\"\n" + nativeMaintenanceCron()
}
func nativeMaintenanceCron() string {
	rows := [][3]string{
		{"*/3 * * * *", "AST_CRON_audio_1_move_VDonly.pl", "--HTTPS"},
		{"* * * * *", "AST_manager_kill_hung_congested.pl", ""}, {"* * * * *", "AST_vm_update.pl", ""}, {"* * * * *", "AST_conf_update.pl", ""}, {"11 * * * *", "AST_flush_DBqueue.pl", "-q"}, {"33 * * * *", "AST_cleanup_agent_log.pl", ""}, {"50 0 * * *", "AST_cleanup_agent_log.pl", "--last-24hours"}, {"25 0 * * *", "AST_DB_dead_cb_purge.pl", "--purge-non-cb --quiet"}, {"* * * * *", "AST_VDhopper.pl", "-q"}, {"1 1,7 * * *", "ADMIN_adjust_GMTnow_on_leads.pl", "--list-settings"}, {"2 1 * * *", "AST_reset_mysql_vars.pl", ""}, {"3 1 * * *", "AST_DB_optimize.pl", ""}, {"2 0 * * 0", "AST_agent_week.pl", ""},
	}
	var out strings.Builder
	out.WriteString("* * * * * root /usr/bin/systemctl start almadial-native-maintenance.service\n")
	for _, row := range rows {
		fmt.Fprintf(&out, "%s asterisk /usr/bin/flock -n /run/asterisk/almadial-%s.lock /usr/bin/perl /usr/share/astguiclient/%s", row[0], row[1], row[1])
		if row[2] != "" {
			fmt.Fprint(&out, " "+row[2])
		}
		out.WriteByte('\n')
	}
	return out.String()
}

// NativeRuntime completes remote S06 after the fresh S01-S10 ownership checks.
// It uses the same action graph, exclusive lock and durable stage journal.
// baseline.json is an immutable file-ownership handoff, not execution state.
type NativeRuntime struct {
	Native  Native
	Sources InstallSources
	Plan    InstallPlan
}

func (n NativeRuntime) payload() (map[string]any, error) {
	d, e := n.Plan.document()
	if e != nil || d.Request.Network.Mode != "remote" {
		return nil, errRuntime
	}
	sources := map[string][]byte{}
	pins := map[string]string{}
	deployment := map[string]string{}
	metadata := map[string]VICIdialFile{}
	required := []string{"bin/ADMIN_keepalive_ALL.pl", "bin/AST_CRON_audio_1_move_VDonly.pl", "bin/AST_CRON_audio_1_move_mix.pl", "bin/AST_CRON_audio_2_compress.pl", "www/agc/vicidial.php", "www/agc/vdc_db_query.php", "www/agc/conf_exten_check.php", "www/agc/manager_send.php", "www/agc/calendar_db.js", "agi/park_CID.agi", "bin/AST_update.pl", "bin/AST_manager_listen.pl", "bin/AST_manager_send.pl", "bin/AST_VDauto_dial.pl", "bin/AST_send_action_child.pl", "bin/AST_manager_kill_hung_congested.pl", "bin/AST_vm_update.pl", "bin/AST_conf_update.pl", "bin/AST_flush_DBqueue.pl", "bin/AST_cleanup_agent_log.pl", "bin/AST_DB_dead_cb_purge.pl", "bin/AST_VDhopper.pl", "bin/ADMIN_adjust_GMTnow_on_leads.pl", "bin/AST_reset_mysql_vars.pl", "bin/AST_DB_optimize.pl", "bin/AST_agent_week.pl"}
	for _, name := range required {
		b, ok := n.Sources.Application.files[name]
		if !ok {
			return nil, fmt.Errorf("native runtime pinned input missing: %s", name)
		}
		var compressed bytes.Buffer
		zw := gzip.NewWriter(&compressed)
		if _, e = zw.Write(b); e != nil {
			return nil, e
		}
		if e = zw.Close(); e != nil {
			return nil, e
		}
		sources[name] = compressed.Bytes()
		pins[name] = rawSHA(b)
	}
	files, e := n.Sources.Application.Deployment()
	if e != nil {
		return nil, e
	}
	for _, f := range files {
		deployment[f.Target] = f.SHA256
		metadata[f.Target] = f
	}
	return map[string]any{"Binding": hash(struct{ Host, Plan, Manifest, Source, Contract string }{d.Request.Authorization.Host, n.Plan.Digest(), d.Manifest, n.Sources.Application.Digest(), "remote-native-ownership/v1"}), "Policy": NativeRuntimePolicyDigest(), "Hostname": d.Request.Phone.Hostname, "WSSURL": d.Request.Phone.WSSURL, "Network": d.Request.Network, "Sources": sources, "SourcePins": pins, "DeploymentPins": deployment, "DeploymentMetadata": metadata}, nil
}
func (n NativeRuntime) adapter(ctx context.Context, mode, baseline string) (RuntimeSnapshot, error) {
	var s RuntimeSnapshot
	input, e := n.payload()
	if e != nil {
		return s, e
	}
	input["ExpectedBaseline"] = baseline
	b, e := json.Marshal(input)
	if e != nil {
		return s, e
	}
	runner, ok := n.Native.Runner.(vicidialInputRunner)
	if !ok {
		return s, errRuntime
	}
	c := dbCommand("/usr/bin/python3", "-B", "-c", nativeRuntimeProgram(), mode)
	c.Timeout = 2 * time.Minute
	c.StdoutLimit = 1 << 20
	result, e := runner.RunInput(ctx, c, b)
	if e != nil || !result.Started || result.ExitCode != 0 || result.StdoutTruncated || result.StderrTruncated {
		return s, fmt.Errorf("native VICIdial runtime: %w (%s)", errRuntime, strings.TrimSpace(result.Stderr)+fmt.Sprintf("; execution=%v; input_bytes=%d", e, len(b)))
	}
	if decode(result.Stdout, &s) != nil {
		return s, errRuntime
	}
	return s, nil
}
func (n NativeRuntime) InspectRuntime(ctx context.Context) (RuntimeSnapshot, error) {
	d, e := n.Plan.document()
	if e != nil {
		return RuntimeSnapshot{}, e
	}
	inventory, e := n.Native.Inspect(ctx)
	if e != nil || inventory.Host != d.Request.Authorization.Host || inventory.ValidateTrust() != nil {
		return RuntimeSnapshot{}, errRuntime
	}
	s, e := n.adapter(ctx, "native-inspect", "")
	s.Host = inventory.Host
	s.Foundation = n.Plan.Digest()
	return s, e
}
func prepareNativeRuntime(p InstallPlan, sources InstallSources, before RuntimeSnapshot) (RuntimeStage, error) {
	d, e := p.document()
	if e != nil || d.Request.Network.Mode != "remote" || !runtimeReady(before) || before.Host != d.Request.Authorization.Host || before.Foundation != p.Digest() {
		return RuntimeStage{}, errRuntime
	}
	actions := []execution.Action{}
	for _, v := range []struct{ id, description, state string }{
		{"runtime.00.files", "Reconcile pinned native VICIdial derivations, local AMI, modules, dialplan and maintenance", before.Files},
		{"runtime.01.services", "Supervise upstream updater, listener, manager sender and autodial with systemd", before.Services},
		{"runtime.02.verify", "Verify native AMI, modules, campaign dialplan, maintenance and updater heartbeat", before.Health},
	} {
		if shaPattern.MatchString(v.state) && len(actions) == 0 {
			continue
		}
		deps := []string{}
		if len(actions) > 0 {
			deps = append(deps, actions[len(actions)-1].ID)
		}
		actions = append(actions, execution.Action{ID: v.id, StageID: "S06-native", Description: v.description, Dependencies: deps, Preconditions: []string{"runtime.fresh"}, Command: runtimeCommand(v.id), Postconditions: []string{v.id + ".verified"}})
	}
	g, e := execution.NewGraph(actions)
	if e != nil {
		return RuntimeStage{}, e
	}
	return RuntimeStage{jsonString(runtimeStageData{p.Digest(), hash(d.Request.Config), d.Manifest, sources.Application.Digest(), NativeRuntimePolicyDigest(), before}), g}, nil
}

type nativeRuntimeRunner struct {
	native NativeRuntime
	stage  RuntimeStage
}

func (r nativeRuntimeRunner) Validate(c execution.Command) error {
	d := r.stage.doc()
	if d.Policy != NativeRuntimePolicyDigest() || d.Parent != r.native.Plan.Digest() || d.Source != r.native.Sources.Application.Digest() {
		return errRuntime
	}
	for _, a := range r.stage.Graph().Actions() {
		if jsonString(c) == jsonString(a.Command) {
			return nil
		}
	}
	return errRuntime
}
func (r nativeRuntimeRunner) Run(ctx context.Context, c execution.Command) (execution.Result, error) {
	if r.Validate(c) != nil || len(c.Args) != 2 {
		return execution.Result{}, errRuntime
	}
	modes := map[string]string{"runtime.00.files": "native-files", "runtime.01.services": "native-services", "runtime.02.verify": "native-inspect"}
	mode := modes[c.Args[1]]
	if mode == "" {
		return execution.Result{}, errRuntime
	}
	_, e := r.native.adapter(ctx, mode, r.stage.doc().Before.Foreign)
	out := execution.Result{Started: true}
	if e != nil {
		out.ExitCode = 1
		out.Stderr = e.Error()
	}
	return out, e
}
func nativeHandoffExists() (bool, error) {
	st, e := os.Lstat("/var/lib/almadial/native-runtime/baseline.json")
	if errors.Is(e, os.ErrNotExist) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	if !st.Mode().IsRegular() {
		return false, errRuntime
	}
	return true, nil
}

// nativeRuntimePlan is read-only. An absent handoff must first pass the old
// recursive provisioning checks; an existing handoff verifies immutable files
// plus live application state instead of freezing mutable agent/campaign rows.
func (r *installRun) nativeRuntimePlan(phone PhoneInputs) (NativeRuntime, RuntimeStage, error) {
	n := NativeRuntime{r.native, r.sources, r.plan}
	exists, e := nativeHandoffExists()
	if e != nil {
		return n, RuntimeStage{}, e
	}
	if !exists {
		if e = r.verifyComplete(phone); e != nil {
			return n, RuntimeStage{}, e
		}
	}
	fresh, e := n.InspectRuntime(r.ctx)
	if e != nil {
		return n, RuntimeStage{}, e
	}
	stage, e := prepareNativeRuntime(r.plan, r.sources, fresh)
	return n, stage, e
}
func (r *installRun) completeRuntime(phone PhoneInputs) (bool, error) {
	if r.d.Request.Network.Mode != "remote" {
		return true, r.verifyComplete(phone)
	}
	n, stage, e := r.nativeRuntimePlan(phone)
	if e != nil {
		return false, e
	}
	if len(stage.Graph().Actions()) == 0 {
		return true, nil
	}
	// Repair attempts are ordinary existing stage journals. Retain every prior
	// accepted attempt; resume the first unfinished one, never overwrite history.
	for generation := 1; generation <= 100; generation++ {
		name := fmt.Sprintf("S06-native-%s-%03d", NativeRuntimePolicyDigest()[:12], generation)
		var accepted RuntimeSnapshot
		if e = r.read(name+"-accepted", &accepted); e == nil {
			continue
		} else if !errors.Is(e, os.ErrNotExist) {
			return false, e
		}
		_, e = driveInstallStage(r, name, func() (RuntimeSnapshot, error) { return n.InspectRuntime(r.ctx) }, func(before RuntimeSnapshot, resume bool) error {
			prepared, e := prepareNativeRuntime(r.plan, r.sources, before)
			if e != nil {
				return e
			}
			_, e = executeRuntimeStage(r.ctx, prepared, r.stageState(name), n, nativeRuntimeRunner{n, prepared}, resume, r.log, hash(struct {
				Plan          string
				Authorization InstallAuthorization
			}{r.plan.Digest(), r.d.Request.Authorization}))
			return e
		})
		if e != nil {
			return false, e
		}
		fresh, e := n.InspectRuntime(r.ctx)
		if e != nil || !runtimeSatisfied(fresh) {
			return false, errRuntime
		}
		return false, nil
	}
	return false, errors.New("native runtime repair history exhausted; preserve journals for review")
}
