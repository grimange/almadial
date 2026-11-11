//go:build linux

package provision

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/grimange/almadial/internal/execution"
)

//go:embed runtime_native.py
var runtimeAdapterSource string

func runtimeAdapter() string {
	// Reuse the accepted secure file primitives without changing their S05
	// identity, authorization surface or receipt digest.
	return strings.TrimSuffix(vicidialFilesAdapter, "if __name__ == '__main__':\n    sys.exit(main())\n") + runtimeAdapterSource
}

//go:embed runtime.go
var runtimeDerivationSource string

type RuntimeNative struct{ Application VICIdialNative }

func (n RuntimeNative) files() ([]vicidialOwnedFile, error) {
	secrets, e := databaseSecrets()
	if e != nil || len(secrets) != 2 {
		return nil, errRuntime
	}
	ami, e := vicidialAMISecret()
	if e != nil || !vicidialHex(ami, 20) {
		return nil, errRuntime
	}
	app, e := vicidialConfiguration(secrets[0], ami)
	if e != nil {
		return nil, e
	}
	config := string(app[0].content) + "VARfastagi_log_min_servers => 1\nVARfastagi_log_max_servers => 2\nVARfastagi_log_min_spare_servers => 1\nVARfastagi_log_max_spare_servers => 1\n"
	files := []vicidialOwnedFile{
		{"/etc/cron.d/almadial-s06", "root", "root", "system_cron_spool_t", 0644, []byte(RuntimeCron())},
		{"/etc/astguiclient-s06.conf", "root", "asterisk", "etc_t", 0640, []byte(config)},
		{vicidialHome + "/almadial-s06-ready.pl", "root", "root", "usr_t", 0644, []byte(runtimeReadiness)},
	}
	for _, p := range []struct {
		id, script string
		active     bool
	}{{"fastagi", "FastAGI_log.pl", true}, {"updater", "AST_update.pl", false}, {"listener", "AST_manager_listen.pl", false}} {
		script, e := n.Application.Source.runtimeScript(p.script)
		if e != nil {
			return nil, e
		}
		files = append(files, vicidialOwnedFile{vicidialHome + "/almadial-s06-" + p.script, "root", "root", "usr_t", 0644, script}, vicidialOwnedFile{"/etc/systemd/system/almadial-s06-" + p.id + ".service", "root", "root", "systemd_unit_file_t", 0644, []byte(runtimeUnit(p.id, p.script, p.active))})
	}
	return files, nil
}

func (n RuntimeNative) adapter(ctx context.Context, mode string) (RuntimeSnapshot, error) {
	var out RuntimeSnapshot
	files, e := n.files()
	if e != nil {
		return out, e
	}
	type file struct {
		Target, Owner, Group, Label, SHA256, PriorSHA256 string
		Mode                                             uint32
		Content                                          []byte
	}
	payload := struct {
		Files      []file
		TLSHandoff bool
	}{TLSHandoff: n.Application.Asterisk.TLSHandoff && mode == "runtime-inspect"}
	for _, f := range files {
		prior := ""
		if f.path == "/etc/astguiclient-s06.conf" {
			prior = rawSHA([]byte(strings.ReplaceAll(string(f.content), "VARDB_server => localhost", "VARDB_server => 127.0.0.1")))
		}
		if f.path == vicidialHome+"/almadial-s06-ready.pl" {
			prior = rawSHA([]byte(runtimeReadinessPrevious))
		}
		payload.Files = append(payload.Files, file{f.path, f.owner, f.group, f.label, rawSHA(f.content), prior, f.mode, f.content})
	}
	input, e := json.Marshal(payload)
	if e != nil {
		return out, errRuntime
	}
	r, e := n.Application.input()
	if e != nil {
		return out, e
	}
	c := dbCommand("/usr/bin/python3", "-c", runtimeAdapter(), mode)
	c.Timeout = 90 * time.Second
	result, e := r.RunInput(ctx, c, input)
	if e != nil || !result.Started || result.ExitCode != 0 || result.StdoutTruncated || result.StderrTruncated {
		// The adapter never includes raw command output in diagnostics.
		return out, fmt.Errorf("%w: %s", errRuntime, strings.TrimSpace(result.Stderr))
	}
	if mode == "runtime-inspect" && decode(result.Stdout, &out) != nil {
		return out, errRuntime
	}
	return out, nil
}

func (n RuntimeNative) InspectRuntime(ctx context.Context) (RuntimeSnapshot, error) {
	s, e := n.adapter(ctx, "runtime-inspect")
	if e != nil {
		return s, e
	}
	app, e := n.Application.inspectVICIdial(ctx, true)
	if e != nil || !vicidialSatisfied(app) {
		return s, fmt.Errorf("%w: accepted S05 foundation: %v", errRuntime, e)
	}
	s.Host, s.Foundation = app.Host, app.Digest()
	return s, nil
}

type runtimeRunner struct {
	native RuntimeNative
	stage  RuntimeStage
}

func (n RuntimeNative) Runner(s RuntimeStage) execution.CommandRunner { return runtimeRunner{n, s} }
func (r runtimeRunner) Validate(c execution.Command) error {
	if r.stage.doc().Policy != RuntimePolicyDigest() || r.stage.doc().Source != r.native.Application.Source.Digest() {
		return errRuntime
	}
	for _, a := range r.stage.Graph().Actions() {
		if jsonString(a.Command) == jsonString(c) {
			return nil
		}
	}
	return errRuntime
}
func (r runtimeRunner) Run(ctx context.Context, c execution.Command) (execution.Result, error) {
	if r.Validate(c) != nil || len(c.Args) != 2 {
		return execution.Result{}, errRuntime
	}
	mode := ""
	switch c.Args[1] {
	case "runtime.00.files":
		mode = "runtime-files"
	case "runtime.01.services":
		mode = "runtime-services"
	case "runtime.02.verify":
		mode = "runtime-inspect"
	default:
		return execution.Result{}, errRuntime
	}
	_, e := r.native.adapter(ctx, mode)
	if e == nil && mode == "runtime-services" {
		deadline := time.NewTimer(25 * time.Second)
		defer deadline.Stop()
		for {
			s, ee := r.native.adapter(ctx, "runtime-inspect")
			if ee != nil {
				e = ee
				break
			}
			if shaPattern.MatchString(s.Health) {
				break
			}
			select {
			case <-ctx.Done():
				e = ctx.Err()
			case <-deadline.C:
				e = errRuntime
			case <-time.After(100 * time.Millisecond):
				continue
			}
			break
		}
	}
	result := execution.Result{Started: true, ExitCode: 0}
	if e != nil {
		result.ExitCode = 1
	}
	return result, e
}
