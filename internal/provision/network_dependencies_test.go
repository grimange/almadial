//go:build linux

package provision

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/grimange/almadial/internal/execution"
)

type networkRPMFixture struct{ current Snapshot }

func (f networkRPMFixture) Validate(execution.Command) error { return nil }
func (f networkRPMFixture) Run(context.Context, execution.Command) (execution.Result, error) {
	b, _ := json.Marshal(f.current)
	return execution.Result{Started: true, Stdout: string(b)}, nil
}

func TestNetworkClosedDependencyHandoff(t *testing.T) {
	before := fixture()
	var candidates []Candidate
	if e := json.Unmarshal([]byte(networkDependencyCandidates), &candidates); e != nil {
		t.Fatal(e)
	}
	d := NetworkDependencies{Before: before, Transaction: Transaction{BeforeSHA256: before.Digest(), Install: candidates, Remove: []Package{}}}
	after := applied(before, d.Transaction)
	if !d.valid() || d.Verify(after) != nil {
		t.Fatal("reviewed additive handoff rejected")
	}
	for _, defect := range []string{"extra RPM", "wrong signer", "unfinished history", "missing dependency", "host"} {
		t.Run(defect, func(t *testing.T) {
			bad := applied(before, d.Transaction)
			switch defect {
			case "extra RPM":
				bad.Installed = append(bad.Installed, Installed{Name: "unexpected", EVR: "0:1-1", Arch: "noarch", Signer: before.Installed[0].Signer})
			case "wrong signer":
				bad.Installed[0].Signer = "unknown"
			case "unfinished history":
				bad.HistorySHA256 = before.HistorySHA256
			case "missing dependency":
				bad.Installed = bad.Installed[1:]
			case "host":
				bad.Host = hash("other host")
			}
			if d.Verify(bad) == nil {
				t.Fatal("invalid physical inventory accepted")
			}
		})
	}
	runner := networkFoundationRunner{networkRPMFixture{after}, d}
	c := command("inspect", nil)
	c.Timeout = 30 * time.Second
	r, e := runner.Run(context.Background(), c)
	var retained Snapshot
	if e != nil || decode(r.Stdout, &retained) != nil || retained.Digest() != before.Digest() {
		t.Fatal("accepted receipt not retained after complete physical verification", e)
	}
	d.Transaction.Install[0].SHA256 = hash("unreviewed payload")
	if d.valid() {
		t.Fatal("unreviewed package hash accepted")
	}
}
