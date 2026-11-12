//go:build linux

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/grimange/almadial/internal/artifacts"
	"github.com/grimange/almadial/internal/buildinfo"
	"github.com/grimange/almadial/internal/diagnostics"
	"github.com/grimange/almadial/internal/operatorlog"
	"github.com/grimange/almadial/internal/preflight"
	"github.com/grimange/almadial/internal/provision"
)

const installationHelp = `Usage:
  almadial plan --request FILE --manifest FILE
  almadial plan --plan FILE --manifest FILE
  almadial install --plan FILE --manifest FILE --approve-sha256 SHA256
  almadial resume --plan FILE --manifest FILE --approve-sha256 SHA256

Optional --pause-after S03 stops only after verified database completion; resume
uses the same plan and does not repeat completed effects.

The plan command inspects the local host without provisioning. Save its JSON
output and review all twelve stage responsibilities before approving its digest.
Install/resume require local root and the same approved, host-bound plan.
State: /var/lib/almadial-install (existing stage journals and immutable inputs).
The candidate manifest supports authorized disposable laboratory qualification;
it does not authorize production installation. Browser/media acceptance remains
an explicit functional qualification after provisioning.
`

func installRead(path string) ([]byte, error) {
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	st, e := f.Stat()
	if e != nil {
		return nil, e
	}
	if !st.Mode().IsRegular() || st.Size() > 32<<20 {
		return nil, errors.New("invalid installation input file")
	}
	return io.ReadAll(io.LimitReader(f, 32<<20+1))
}
func installDecode(b []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return e
	}
	if _, e := d.Token(); e != io.EOF {
		return errors.New("trailing installation input")
	}
	return nil
}
func runInstall(ctx context.Context, args []string, out, errout io.Writer) int {
	if len(args) == 2 && args[1] == "--help" {
		io.WriteString(out, installationHelp)
		return Success
	}
	fail := func(e error) int { fmt.Fprintln(errout, "almadial installation:", e); return Failure }
	// The accepted S02 helper verifies the live parent lock and running intent.
	if args[0] == "__directory" {
		if len(args) != 3 {
			return Usage
		}
		index, e := strconv.Atoi(args[1])
		if e != nil {
			return Usage
		}
		var r provision.SystemDirectoryRequest
		if installDecode([]byte(args[2]), &r) != nil {
			return Usage
		}
		n, e := provision.NewNative(nil)
		if e != nil {
			return fail(e)
		}
		if e = provision.ApplySystemDirectory(ctx, index, r, n); e != nil {
			return fail(e)
		}
		return Success
	}
	fs := flag.NewFlagSet("installation", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	request := fs.String("request", "", "")
	planFile := fs.String("plan", "", "")
	manifest := fs.String("manifest", "", "")
	approval := fs.String("approve-sha256", "", "")
	pauseAfter := fs.String("pause-after", "", "")
	if fs.Parse(args[1:]) != nil || fs.NArg() != 0 || *manifest == "" {
		fmt.Fprintln(errout, "almadial: installation inputs required; use install --help")
		return Usage
	}
	planning := args[0] == "plan"
	assessing := planning && *planFile != "" && *request == "" && *approval == ""
	if *pauseAfter != "" && (planning || *pauseAfter != "S03") {
		return Usage
	}
	if planning && !assessing && (*request == "" || *planFile != "" || *approval != "") || !planning && (*request != "" || *planFile == "" || *approval == "") {
		return Usage
	}
	raw, e := installRead(*manifest)
	if e != nil {
		return fail(e)
	}
	m, e := artifacts.Load(bytes.NewReader(raw))
	if e != nil {
		return fail(e)
	}
	var r provision.InstallRequest
	var planRaw []byte
	if planning && !assessing {
		raw, e = installRead(*request)
		if e != nil {
			return fail(e)
		}
		if e = installDecode(raw, &r); e != nil {
			return fail(e)
		}
	} else {
		planRaw, e = installRead(*planFile)
		if e != nil {
			return fail(e)
		}
		r, e = provision.InstallPlanRequest(planRaw)
		if e != nil {
			return fail(e)
		}
	}
	sources, e := provision.LoadInstallSources(ctx, m, r.Sources)
	if e != nil {
		return fail(e)
	}
	n, e := provision.NewNative(nil)
	if e != nil {
		return fail(e)
	}
	var p provision.InstallPlan
	if assessing {
		p, e = provision.DecodeInstallPlan(bytes.TrimSpace(planRaw), m, sources)
		if e != nil {
			return fail(e)
		}
		if e = provision.AssessInstall(ctx, p, m, sources, n, "/var/lib/almadial-install"); e != nil {
			return fail(e)
		}
		if e = json.NewEncoder(out).Encode(struct {
			PlanSHA256 string   `json:"plan_sha256"`
			Actions    []string `json:"actions"`
			NoOp       bool     `json:"no_op"`
		}{p.Digest(), []string{}, true}); e != nil {
			return fail(e)
		}
		return Success
	}
	if planning {
		inventory, e := n.Inspect(ctx)
		if e != nil {
			return fail(e)
		}
		cert, e := (provision.TLSNative{Request: r.TLS}).Material()
		if e != nil {
			return fail(e)
		}
		p, e = provision.PrepareInstall(r, provision.Inputs{Config: r.Config, Requirements: r.Requirements, Preflight: preflight.Collect(ctx, preflight.Linux{}), Inventory: inventory}, m, sources, cert)
		if e != nil {
			return fail(e)
		}
		fmt.Fprintln(errout, "Review installation plan; approval SHA-256:", p.Digest())
		if _, e = out.Write(append(p.JSON(), '\n')); e != nil {
			return fail(e)
		}
		return Success
	}
	if os.Geteuid() != 0 {
		return fail(errors.New("install/resume require authorized local root"))
	}
	p, e = provision.DecodeInstallPlan(bytes.TrimSpace(planRaw), m, sources)
	if e != nil {
		return fail(e)
	}
	redactor, e := operatorlog.NewRedactor(nil)
	if e != nil {
		return fail(e)
	}
	log, e := operatorlog.New(errout, errout, nil, redactor)
	if e != nil {
		return fail(e)
	}
	result, e := provision.ExecuteInstallCheckpoint(ctx, p, m, sources, n, "/var/lib/almadial-install", *approval, log, *pauseAfter)
	if errors.Is(e, provision.ErrInstallPaused) {
		fmt.Fprintln(errout, provision.ErrInstallPaused)
		if json.NewEncoder(out).Encode(struct {
			provision.InstallResult
			Paused bool `json:"paused"`
		}{result, true}) != nil {
			return Failure
		}
		return Success
	}
	if e != nil {
		return fail(e)
	}
	health := diagnostics.New().Collect(ctx, diagnostics.Options{CAFile: "/etc/pki/tls/almadial-s08/current/trust.pem", Manifest: *manifest})
	result.Completed = append(result.Completed, "S12 read-only diagnostics")
	report := struct {
		provision.InstallResult
		Build       string             `json:"build"`
		Diagnostics diagnostics.Report `json:"diagnostics"`
		Endpoints   []string           `json:"endpoints"`
	}{result, buildinfo.String(), health, []string{"https://localhost/vicidial/admin.php", "https://localhost/agc/vicidial.php", "https://localhost/almadial-phone/", "wss://localhost:8089/ws"}}
	if e = json.NewEncoder(out).Encode(report); e != nil {
		return fail(e)
	}
	for _, check := range health.Checks {
		if check.Blocking && (check.State == "FAIL" || check.State == "UNKNOWN") {
			return fail(errors.New("installed host requires attention: " + check.ID + "; run almadial doctor"))
		}
	}
	return Success
}
