//go:build linux

package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/grimange/almadial/internal/diagnostics"
)

const diagnosticHelp = `Usage:
  almadial status [--json] [--ca-file FILE] [--manifest FILE] [--journal FILE]
  almadial doctor [--json] [--ca-file FILE] [--manifest FILE] [--journal FILE]
  almadial evidence export --output /absolute/new-bundle.tar.gz [same inputs]
  almadial support-bundle --output /absolute/new-bundle.tar.gz [same inputs]

Read-only local inspection. Run as authorized local root for socket/process access.
Only explicit export writes a new mode-0600 archive; no repairs are performed.
--ca-file selects the intended CA (lab: /etc/pki/tls/almadial-s08/current/trust.pem).
Manifest/journal are optional explicit inputs; no journal is selected implicitly.
Exit codes: 0 healthy (or bundle written), 10 degraded status/doctor, 1 output error,
2 usage. Exported report retains health findings even when export exits 0.
`

func runDiagnostics(ctx context.Context, args []string, out, errout io.Writer) int {
	mode := args[0]
	args = args[1:]
	if mode == "evidence" {
		if len(args) == 0 || args[0] != "export" {
			fmt.Fprintln(errout, "almadial: use evidence export --output FILE")
			return Usage
		}
		args = args[1:]
	}
	fs := flag.NewFlagSet("diagnostics", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var o diagnostics.Options
	var output string
	var machine, help bool
	fs.StringVar(&o.CAFile, "ca-file", "", "")
	fs.StringVar(&o.Manifest, "manifest", "", "")
	fs.StringVar(&o.Journal, "journal", "", "")
	fs.StringVar(&output, "output", "", "")
	fs.BoolVar(&machine, "json", false, "")
	fs.BoolVar(&help, "help", false, "")
	if fs.Parse(args) != nil || fs.NArg() != 0 {
		fmt.Fprintln(errout, "almadial: invalid diagnostic options; use status --help")
		return Usage
	}
	if help {
		if _, e := io.WriteString(out, diagnosticHelp); e != nil {
			return Failure
		}
		return Success
	}
	export := mode == "evidence" || mode == "support-bundle"
	if export && output == "" || !export && output != "" {
		fmt.Fprintln(errout, "almadial: explicit --output required only for bundle export")
		return Usage
	}
	for _, p := range []string{o.CAFile, o.Manifest, o.Journal, output} {
		if p != "" && (!filepath.IsAbs(p) || filepath.Clean(p) != p || strings.ContainsAny(p, "\x00\r\n")) {
			fmt.Fprintln(errout, "almadial: diagnostic paths must be absolute, clean paths")
			return Usage
		}
	}
	r := diagnostics.New().Collect(ctx, o)
	if export {
		b, e := diagnostics.Bundle(r)
		if e != nil {
			fmt.Fprintln(errout, "almadial: cannot construct bounded bundle")
			return Failure
		}
		if diagnostics.WriteBundle(output, b) != nil {
			fmt.Fprintln(errout, "almadial: unsafe, existing or unwritable bundle destination")
			return Failure
		}
		_, e = fmt.Fprintln(out, "Sanitized support bundle written; inspect report.json, redaction.json and SHA256SUMS before sharing.")
		if e != nil {
			return Failure
		}
		return Success
	}
	return renderDiagnosticReport(mode, machine, r, out)
}

func renderDiagnosticReport(mode string, machine bool, r diagnostics.Report, out io.Writer) int {
	if machine {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if enc.Encode(r) != nil {
			return Failure
		}
	} else {
		for _, c := range r.Checks {
			if _, e := fmt.Fprintf(out, "%-14s %-28s %s\n", c.State, c.ID, c.Observation); e != nil {
				return Failure
			}
			if mode == "doctor" {
				if _, e := fmt.Fprintf(out, "  Expected: %s\n  Guidance: %s\n", c.Expected, c.Guidance); e != nil {
					return Failure
				}
			}
		}
	}
	return r.ExitCode()
}
