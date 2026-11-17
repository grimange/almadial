//go:build linux

package provision

import (
	"context"
	"encoding/json"
	"github.com/grimange/almadial/internal/execution"
	"time"
)

// NetworkDependencies is the closed, reviewed additive RPM handoff. Every
// physical inspection verifies the complete inventory delta before retaining
// the accepted S01-S08 receipt identity. No receipt is rewritten or trusted
// instead of current host inspection.
type NetworkDependencies struct {
	Before      Snapshot
	Transaction Transaction
}

const networkDependencyCandidates = `[{"name":"firewalld","evr":"0:2.4.3-4.el10_2","arch":"noarch","repository":"baseos","sha256":"f5f4a3dde4648d63c0788338b29d659529a82ad20b6b847a4d2435f0936d52d2"},{"name":"firewalld-filesystem","evr":"0:2.4.3-4.el10_2","arch":"noarch","repository":"baseos","sha256":"e06163c486be0d18b840a03a604ca734b588e22687a876bf7c906423da1266c2"},{"name":"libnftnl","evr":"0:1.3.0-3.el10","arch":"x86_64","repository":"baseos","sha256":"34a5277e8131c943f84d0323a28227e7445d456606f6f2250764e1f8908ff26e"},{"name":"nftables","evr":"1:1.1.5-6.el10_2","arch":"x86_64","repository":"baseos","sha256":"0d9db44ee3e3d0f74a04e97cf9cef2dd12ed26e9afd2be91b28b928763f6ed65"},{"name":"python3-firewall","evr":"0:2.4.3-4.el10_2","arch":"noarch","repository":"baseos","sha256":"c008b6b62ec34fbcbae74fdd41f8a8e6deb2b742f38670141c120adf18ed290b"},{"name":"python3-nftables","evr":"1:1.1.5-6.el10_2","arch":"x86_64","repository":"baseos","sha256":"68fcc65881baf3884ec5216bd7499f3f375eb817f6cf76a708ccf2df9dc784bb"}]`

func (d NetworkDependencies) valid() bool {
	if d.Before.Release == "10.3" {
		return len(d.Transaction.Install) == 0 && len(d.Transaction.Remove) == 0 && d.Transaction.Validate(d.Before) == nil
	}
	var want []Candidate
	if json.Unmarshal([]byte(networkDependencyCandidates), &want) != nil {
		return false
	}
	return len(d.Transaction.Remove) == 0 && jsonString(normalize(d.Transaction).Install) == jsonString(want) && d.Transaction.Validate(d.Before) == nil
}
func (d NetworkDependencies) Verify(after Snapshot) error {
	if !d.valid() {
		return errNetwork
	}
	return Verify(d.Before, after, d.Transaction)
}

type networkFoundationRunner struct {
	runner       execution.CommandRunner
	dependencies NetworkDependencies
}

func (r networkFoundationRunner) Validate(c execution.Command) error { return r.runner.Validate(c) }
func (r networkFoundationRunner) Run(ctx context.Context, c execution.Command) (execution.Result, error) {
	result, e := r.runner.Run(ctx, c)
	if e != nil {
		return result, e
	}
	inspect := command("inspect", nil)
	inspect.Timeout = 30 * time.Second
	if jsonString(c) == jsonString(inspect) {
		var current Snapshot
		if !result.Started || result.ExitCode != 0 || result.StdoutTruncated || decode(result.Stdout, &current) != nil || r.dependencies.Verify(current) != nil {
			return execution.Result{}, errNetwork
		}
		b, _ := json.Marshal(canonical(r.dependencies.Before))
		result.Stdout = string(b)
	}
	return result, nil
}
func (r networkFoundationRunner) RunInput(ctx context.Context, c execution.Command, input []byte) (execution.Result, error) {
	runner, ok := r.runner.(vicidialInputRunner)
	if !ok {
		return execution.Result{}, errNetwork
	}
	return runner.RunInput(ctx, c, input)
}
