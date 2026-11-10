//go:build linux

package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grimange/almadial/internal/execution"
)

func vicidialAdapter(ctx context.Context, runner vicidialInputRunner, mode string, payload any, result any) error {
	input, e := json.Marshal(payload)
	if e != nil || runner == nil || ctx == nil {
		return errVICIdial
	}
	r, e := runner.RunInput(ctx, vicidialFilesCommand(mode), input)
	if e != nil || !r.Started || r.ExitCode != 0 || r.StdoutTruncated || r.StderrTruncated || decode(r.Stdout, result) != nil {
		return errVICIdial
	}
	return nil
}
func preserveVICIdialRegistration(ctx context.Context, runner vicidialInputRunner, row vicidialRegistration) error {
	b := row.protectedPreimage()
	var result struct{ Preserved string }
	if vicidialAdapter(ctx, runner, "registration-preimage", struct{ Preimage []byte }{b}, &result) != nil || result.Preserved != rawSHA(b) {
		return errVICIdial
	}
	return nil
}
func inspectVICIdialGeneration(ctx context.Context, runner vicidialInputRunner, binding, inputs string) (string, bool, error) {
	var result struct {
		State   string
		Wrapped bool
	}
	if !shaPattern.MatchString(binding) || !shaPattern.MatchString(inputs) || vicidialAdapter(ctx, runner, "generation-inspect", struct{ Binding, Inputs string }{binding, inputs}, &result) != nil {
		return "", false, errVICIdial
	}
	if result.State == "absent" && !result.Wrapped || shaPattern.MatchString(result.State) && result.Wrapped {
		return result.State, result.Wrapped, nil
	}
	return "", false, errVICIdial
}

const vicidialBridgesQuery = `SELECT JSON_ARRAY(conf_exten,server_ip,extension,leave_3way,leave_3way_datetime) FROM asterisk.vicidial_confbridges WHERE server_ip='127.0.0.1' ORDER BY conf_exten;`
const vicidialRebuildSQL = `UPDATE asterisk.servers SET rebuild_conf_files='Y' WHERE server_id='TESTast' AND server_ip='127.0.0.1' AND rebuild_conf_files='N' AND conf_engine='CONFBRIDGE' AND asterisk_version='22.12.0';`

func vicidialReloadCommands() []string {
	return []string{"module reload res_pjsip.so", "module reload res_pjsip_config_wizard.so", "dialplan reload", "module reload app_confbridge.so", "manager reload"}
}

func inspectVICIdialBridges(ctx context.Context, runner vicidialInputRunner, allowAbsent bool) error {
	text, e := vicidialSQL(ctx, runner, vicidialBridgesQuery)
	if e != nil {
		return e
	}
	if text == "" && allowAbsent {
		return nil
	}
	lines := strings.Split(text, "\n")
	if len(lines) != 300 {
		return errVICIdial
	}
	for i, line := range lines {
		var row []json.RawMessage
		if json.Unmarshal([]byte(line), &row) != nil || len(row) != 5 {
			return errVICIdial
		}
		// These slots are generated application configuration, never an initial
		// schema import. S06 is inactive, so no live conference state is admissible.
		var n int
		var server, extension, leave string
		if json.Unmarshal(row[0], &n) != nil || json.Unmarshal(row[1], &server) != nil || json.Unmarshal(row[2], &extension) != nil || json.Unmarshal(row[3], &leave) != nil || n != 9600000+i || server != "127.0.0.1" || extension != "" || leave != "0" || string(row[4]) != "null" {
			return errVICIdial
		}
	}
	return nil
}
func generateVICIdial(ctx context.Context, runner vicidialInputRunner, commands execution.CommandRunner, source VICIdialSource, binding, expectedInputs, ami string) (string, error) {
	program, _, e := source.Generator()
	if e != nil || commands == nil {
		return "", errVICIdial
	}
	before, state, e := inspectVICIdialRegistration(ctx, runner, ami)
	if e != nil || !shaPattern.MatchString(state) || before.Rebuild != "Y" || inspectVICIdialBridges(ctx, runner, true) != nil {
		return "", errVICIdial
	}
	inputs, preimage, e := inspectVICIdialGeneratorInputs(ctx, runner)
	if e != nil || inputs != expectedInputs {
		return "", errVICIdial
	}
	var prepared struct{ Prepared bool }
	if vicidialAdapter(ctx, runner, "generation-prepare", struct{ Program, Preimage []byte }{program, preimage}, &prepared) != nil || !prepared.Prepared {
		return "", errVICIdial
	}
	run := func() error {
		r, e := commands.Run(ctx, dbCommand("/usr/bin/perl", vicidialWork+"/generator.pl"))
		if e != nil || !r.Started || r.ExitCode != 0 || r.StdoutTruncated || r.StderrTruncated {
			return errVICIdial
		}
		after, post, e := inspectVICIdialRegistration(ctx, runner, ami)
		if e != nil || post != state || after.Rebuild != "N" || inspectVICIdialBridges(ctx, runner, false) != nil {
			return errVICIdial
		}
		fresh, _, e := inspectVICIdialGeneratorInputs(ctx, runner)
		if e != nil || fresh != inputs {
			return errVICIdial
		}
		return nil
	}
	if run() != nil {
		return "", errVICIdial
	}
	first := map[string]string{}
	if vicidialAdapter(ctx, runner, "generated-identity", struct{}{}, &first) != nil || len(first) != 4 {
		return "", errVICIdial
	}
	// Qualify unchanged regeneration once during initial installation, under the
	// same durable generation intent. Fresh NO_OP inspection never executes this.
	if _, e = vicidialSQL(ctx, runner, vicidialRebuildSQL); e != nil || run() != nil {
		return "", errVICIdial
	}
	second := map[string]string{}
	if vicidialAdapter(ctx, runner, "generated-identity", struct{}{}, &second) != nil || hash(first) != hash(second) {
		return "", errVICIdial
	}
	var published struct {
		State   string
		Wrapped bool
	}
	if vicidialAdapter(ctx, runner, "generation-publish", struct{ Binding, Inputs string }{binding, inputs}, &published) != nil || !published.Wrapped || !shaPattern.MatchString(published.State) {
		return "", errVICIdial
	}
	fresh, wrapped, e := inspectVICIdialGeneration(ctx, runner, binding, inputs)
	if e != nil || !wrapped || fresh != published.State {
		return "", errVICIdial
	}
	return fresh, nil
}

// Fixed native reloads are checked against capabilities separately. A CLI exit
// status alone cannot prove that Asterisk accepted its generated configuration.
func reloadVICIdialAsterisk(ctx context.Context, n AsteriskNative) error {
	for _, command := range vicidialReloadCommands() {
		out, e := n.cli(ctx, command)
		if e != nil || strings.Contains(strings.ToLower(out), "no such command") || strings.Contains(strings.ToLower(out), "failed") {
			return fmt.Errorf("%w: native configuration reload", errVICIdial)
		}
	}
	return nil
}
