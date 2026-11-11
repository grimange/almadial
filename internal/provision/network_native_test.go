//go:build linux

package provision

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/grimange/almadial/internal/execution"
)

type networkDiagnosticRunner struct{ vicidialGenerationProbeRunner }

func (*networkDiagnosticRunner) Validate(execution.Command) error { return nil }
func (*networkDiagnosticRunner) Run(context.Context, execution.Command) (execution.Result, error) {
	return execution.Result{}, errors.New("unexpected command without stdin")
}

// A nonzero bounded subprocess returns both its captured result and an error.
// Keep the adapter's safe diagnostic instead of returning only the generic error.
func TestNetworkNativeRetainsFailedSubprocessDiagnostic(t *testing.T) {
	r := &networkDiagnosticRunner{vicidialGenerationProbeRunner: vicidialGenerationProbeRunner{
		output: "distinct stdout",
		result: execution.Result{Started: true, ExitCode: 7, Stdout: "distinct stdout", Stderr: "distinct stderr: foreign runtime-only zone state\n"},
		err:    errors.New("generic nonzero subprocess error"),
	}}
	n := NetworkNative{Request: NetworkRequest{Mode: "isolated", Interface: "eth0", ManagementAddress: "192.168.1.2", SSHPort: 22}}
	n.TLS.Web.Runtime.Application.Asterisk.Database.Native.Runner = r
	_, err := n.adapter(context.Background(), "network-activate", hash("before"))
	if !errors.Is(err, errNetwork) || !strings.Contains(err.Error(), "foreign runtime-only zone state") || r.calls != 1 {
		t.Fatalf("lost native diagnostic: %v; calls %d", err, r.calls)
	}
	var nativeError *networkCommandError
	if !errors.As(err, &nativeError) || nativeError.result.ExitCode != 7 || !strings.Contains(err.Error(), "exit 7") || !strings.Contains(err.Error(), "distinct stdout") || !strings.Contains(err.Error(), "distinct stderr") {
		t.Fatalf("lost native status or streams: %v", err)
	}
}
