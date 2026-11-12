// Package cli routes commands; preparation performs read-only local inspection.
package cli

import (
	"context"
	"fmt"
	"github.com/grimange/almadial/internal/buildinfo"
	"io"
	"os"
)

const (
	Success = 0
	Failure = 1
	Usage   = 2
)

const help = `AlmaDial — single-server VICIdial installer for AlmaLinux 10

Usage: almadial [help | version | wizard | plan | install | resume | status | doctor | evidence export]
  help, --help, -h       Show this help
  version, --version    Show version and build provenance

  wizard                Prepare configuration, inspect local host and review a plan
                        Use wizard --help for required inputs and approval options

  status, doctor        Read-only operational checks; use status --help
  evidence export       Explicit bounded sanitized support bundle

  plan, install, resume Unified authorized lab installation; use install --help
`

// Run writes successful output to stdout and errors to stderr; it never exits.
// Help/version perform no inspection. Wizard reads local host/configuration and
// creates a bounded diagnostic file only when explicitly requested.
func Run(args []string, stdout, stderr io.Writer) int {
	return RunContext(context.Background(), args, os.Stdin, stdout, stderr)
}

func runBasic(args []string, stdout, stderr io.Writer) int {
	output := help
	if len(args) > 1 {
		fmt.Fprintln(stderr, "almadial: expected one command; use --help")
		return Usage
	}
	if len(args) == 1 {
		switch args[0] {
		case "help", "--help", "-h":
		case "version", "--version":
			output = buildinfo.String()
		default:
			// Do not echo arbitrary user input, which could contain secrets/control codes.
			fmt.Fprintln(stderr, "almadial: unknown command or option; use --help")
			return Usage
		}
	}
	if _, err := io.WriteString(stdout, output); err != nil {
		fmt.Fprintln(stderr, "almadial: cannot write output")
		return Failure
	}
	return Success
}
