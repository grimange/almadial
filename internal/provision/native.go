package provision

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"io"
	"time"

	"github.com/grimange/almadial/internal/execution"
)

//go:embed native.py
var nativeAdapter string

// Native's runner is ALM-13's bounded direct-argv runner. It is injectable only
// at this internal package boundary for deterministic tests, never via user argv.
type Native struct{ Runner execution.CommandRunner }

func NewNative(sensitive []string) (Native, error) {
	r, e := execution.NewRunner(sensitive)
	return Native{r}, e
}
func command(mode string, payload any) execution.Command {
	args := []string{"-B", "-c", nativeAdapter, mode}
	if payload != nil {
		b, _ := json.Marshal(payload)
		args = append(args, string(b))
	}
	return execution.Command{Executable: "/usr/bin/python3", Args: args, Timeout: 20 * time.Minute, StdoutLimit: 4 << 20, StderrLimit: 64 << 10}
}
func decode(raw string, v any) error {
	d := json.NewDecoder(bytes.NewBufferString(raw))
	d.DisallowUnknownFields()
	if d.Decode(v) != nil {
		return errPolicy
	}
	if _, e := d.Token(); e != io.EOF {
		return errPolicy
	}
	return nil
}
func (n Native) Inspect(ctx context.Context) (Snapshot, error) {
	var s Snapshot
	if ctx == nil || n.Runner == nil {
		return s, errPolicy
	}
	c := command("inspect", nil)
	c.Timeout = 30 * time.Second
	r, e := n.Runner.Run(ctx, c)
	if e != nil {
		return s, e
	}
	if !r.Started || r.ExitCode != 0 || r.StdoutTruncated || r.StderrTruncated {
		return s, errPolicy
	}
	if decode(r.Stdout, &s) != nil {
		return Snapshot{}, errPolicy
	}
	return canonical(s), nil
}

type request struct {
	Mode         string       `json:"mode"`
	Packages     []Package    `json:"packages"`
	Before       Snapshot     `json:"before"`
	BeforeSHA256 string       `json:"before_sha256"`
	Transaction  *Transaction `json:"transaction,omitempty"`
	PublicKey    string       `json:"public_key,omitempty"`
}

func nativeCommand(mode string, s Snapshot, t *Transaction) execution.Command {
	r := request{Mode: mode, Packages: Packages(), Before: canonical(s), BeforeSHA256: s.Digest(), Transaction: t}
	if mode == "trust" {
		r.PublicKey = epelPublicKey
	}
	return command(mode, r)
}
