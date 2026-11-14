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

//go:embed phone_native.py
var phoneAdapterSource string

type PhoneNative struct {
	Network NetworkNative
	Request PhoneRequest
	Source  PhoneSource
}

// Preserve the bounded, redacted subprocess result through the stage runner.
type phoneCommandError struct {
	result execution.Result
	cause  error
}

func (e *phoneCommandError) Error() string {
	return fmt.Sprintf("%v: S10 native adapter (exit %d); stdout: %s; stderr: %s", errPhone, e.result.ExitCode, strings.TrimSpace(e.result.Stdout), strings.TrimSpace(e.result.Stderr))
}
func (e *phoneCommandError) Unwrap() error { return errPhone }

func (n PhoneNative) adapter(ctx context.Context, mode, expected string) (PhoneSnapshot, error) {
	var s PhoneSnapshot
	b, e := json.Marshal(map[string]any{"Request": n.Request, "Network": n.Network.Request, "Files": n.Source.files, "Source": n.Source.Digest(), "ExpectedCurrent": expected})
	if e != nil {
		return s, e
	}
	r, e := n.Network.TLS.Web.Runtime.Application.input()
	if e != nil {
		return s, e
	}
	result, e := r.RunInput(ctx, dbCommand("/usr/bin/python3", "-B", "-c", phoneAdapterSource, mode), b)
	if !result.Started || result.ExitCode != 0 || result.StdoutTruncated || result.StderrTruncated {
		return s, &phoneCommandError{result: result, cause: e}
	}
	if e != nil {
		return s, e
	}
	if decode(result.Stdout, &s) != nil {
		return s, errPhone
	}
	return s, nil
}
func (n PhoneNative) InspectPhone(ctx context.Context) (PhoneSnapshot, error) {
	parentNative := n.Network
	parentNative.TLS.Web.Runtime.Application.PhoneHandoff = true
	parentNative.TLS.Web.Runtime.Application.Asterisk.PhoneHandoff = true
	parent, e := parentNative.InspectNetwork(ctx)
	if e != nil {
		return PhoneSnapshot{}, fmt.Errorf("S10 S09 prerequisite: %w", e)
	}
	if !networkSatisfied(parent) {
		return PhoneSnapshot{}, errPhone
	}
	s, e := n.adapter(ctx, "phone-inspect", "")
	if e != nil {
		return s, e
	}
	s.Host, s.Foundation = parent.Host, parent.Digest()
	return s, nil
}

type phoneRunner struct {
	native PhoneNative
	stage  PhoneStage
}

func (n PhoneNative) Runner(s PhoneStage) execution.CommandRunner { return phoneRunner{n, s} }
func (r phoneRunner) Validate(c execution.Command) error {
	d := r.stage.doc()
	if d.PhoneSourceSHA256 != r.native.Source.Digest() || d.Policy != PhonePolicyDigest() || d.Request != r.native.Request || d.Source != r.native.Network.TLS.Web.Runtime.Application.Source.Digest() {
		return errPhone
	}
	for _, a := range r.stage.Graph().Actions() {
		if jsonString(a.Command) == jsonString(c) {
			return nil
		}
	}
	return errPhone
}
func (r phoneRunner) Run(ctx context.Context, c execution.Command) (execution.Result, error) {
	if r.Validate(c) != nil || len(c.Args) != 2 || c.Args[1] != "phone.00.activate" {
		return execution.Result{}, errPhone
	}
	_, e := r.native.adapter(ctx, "phone-activate", r.stage.doc().Before.Current)
	result := execution.Result{Started: true}
	if e != nil {
		result.ExitCode = 1
		var nativeError *phoneCommandError
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
