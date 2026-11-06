// Package cli implements side-effect-free command routing.
package cli

import (
	"fmt"
	"github.com/grimange/almadial/internal/buildinfo"
	"io"
)

const (
	Success = 0
	Failure = 1
	Usage   = 2
)

const help = `AlmaDial — single-server VICIdial installer for AlmaLinux 10

Usage: almadial [help | version]
  help, --help, -h       Show this help
  version, --version    Show version and build provenance

This foundation provides help and version only.
Installation and host diagnostics are not implemented yet.
`

// Run writes successful output to stdout and errors to stderr; it never exits.
// No filesystem, network, database or installation operation is performed.
func Run(args []string, stdout, stderr io.Writer) int {
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
