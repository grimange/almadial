//go:build linux

package provision

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/grimange/almadial/internal/execution"
)

//go:embed network_native.py
var networkAdapterSource string

type NetworkNative struct {
	TLS          TLSNative
	Request      NetworkRequest
	Dependencies *NetworkDependencies
}

// Preserve the bounded, redacted subprocess result through the stage runner.
type networkCommandError struct {
	result execution.Result
	cause  error
}

func (e *networkCommandError) Error() string {
	return fmt.Sprintf("%v: S09 native adapter (exit %d); stdout: %s; stderr: %s", errNetwork, e.result.ExitCode, strings.TrimSpace(e.result.Stdout), strings.TrimSpace(e.result.Stderr))
}
func (e *networkCommandError) Unwrap() error { return errNetwork }

func (n NetworkNative) adapter(ctx context.Context, mode, expected string) (NetworkSnapshot, error) {
	var s NetworkSnapshot
	xml, e := NetworkPolicy(n.Request)
	if e != nil {
		return s, e
	}
	b, e := json.Marshal(map[string]any{"Request": n.Request, "XML": xml, "ExpectedCurrent": expected})
	if e != nil {
		return s, e
	}
	r, e := n.TLS.Web.Runtime.Application.input()
	if e != nil {
		return s, e
	}
	result, e := r.RunInput(ctx, dbCommand("/usr/bin/python3", "-B", "-c", networkAdapterSource, mode), b)
	if !result.Started || result.ExitCode != 0 || result.StdoutTruncated || result.StderrTruncated {
		return s, &networkCommandError{result: result, cause: e}
	}
	if e != nil {
		return s, e
	}
	if decode(result.Stdout, &s) != nil {
		return s, errNetwork
	}
	return s, nil
}
func (n NetworkNative) InspectNetwork(ctx context.Context) (NetworkSnapshot, error) {
	parentNative := n.TLS
	if n.Dependencies != nil {
		if !n.Dependencies.valid() {
			return NetworkSnapshot{}, errNetwork
		}
		base := &parentNative.Web.Runtime.Application.Asterisk.Database.Native
		base.Runner = networkFoundationRunner{base.Runner, *n.Dependencies}
	}
	parent, e := parentNative.InspectTLS(ctx)
	if e != nil {
		return NetworkSnapshot{}, fmt.Errorf("S09 S08 prerequisite: %w", e)
	}
	if !tlsSatisfied(parent) {
		return NetworkSnapshot{}, errNetwork
	}
	s, e := n.adapter(ctx, "network-inspect", "")
	if e != nil {
		return s, e
	}
	s.Host, s.Foundation = parent.Host, parent.Digest()
	return s, nil
}

type networkRunner struct {
	native NetworkNative
	stage  NetworkStage
}

func (n NetworkNative) Runner(s NetworkStage) execution.CommandRunner { return networkRunner{n, s} }
func (r networkRunner) Validate(c execution.Command) error {
	d := r.stage.doc()
	if d.DependenciesSHA256 != hash(r.native.Dependencies) || d.Policy != NetworkPolicyDigest() || d.Request != r.native.Request || d.Source != r.native.TLS.Web.Runtime.Application.Source.Digest() {
		return errNetwork
	}
	for _, a := range r.stage.Graph().Actions() {
		if jsonString(a.Command) == jsonString(c) {
			return nil
		}
	}
	return errNetwork
}
func (r networkRunner) Run(ctx context.Context, c execution.Command) (execution.Result, error) {
	if r.Validate(c) != nil || len(c.Args) != 2 || c.Args[1] != "network.00.activate" {
		return execution.Result{}, errNetwork
	}
	_, e := r.native.adapter(ctx, "network-activate", r.stage.doc().Before.Current)
	result := execution.Result{Started: true}
	if e != nil {
		result.ExitCode = 1
		var nativeError *networkCommandError
		if errors.As(e, &nativeError) {
			result = nativeError.result
		}
		result.Stderr = e.Error()
		var failure *execution.Error
		if nativeError != nil && errors.As(nativeError.cause, &failure) {
			return result, failure
		}
	}
	return result, e
}
