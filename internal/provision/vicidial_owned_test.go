//go:build linux

package provision

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/grimange/almadial/internal/execution"
)

type vicidialOwnedProbe struct {
	t        *testing.T
	results  []execution.Result
	err      error
	commands []execution.Command
	secret   string
}

func (r *vicidialOwnedProbe) RunInput(_ context.Context, c execution.Command, input []byte) (execution.Result, error) {
	r.t.Helper()
	if strings.Contains(c.Executable+strings.Join(c.Args, " "), r.secret) {
		r.t.Fatal("credential in command arguments")
	}
	var payload struct {
		Files []struct {
			Target, SHA256 string
			Content        []byte
		}
	}
	if json.Unmarshal(input, &payload) != nil || len(payload.Files) != 6 {
		r.t.Fatal("invalid transient owned payload")
	}
	for _, f := range payload.Files {
		if f.SHA256 != rawSHA(f.Content) {
			r.t.Fatal("unbound content")
		}
	}
	r.commands = append(r.commands, c)
	if len(r.results) == 0 {
		r.t.Fatal("unexpected extra command")
	}
	x := r.results[0]
	r.results = r.results[1:]
	return x, r.err
}

func TestVICIdialOwnedIndependentPostcondition(t *testing.T) {
	secret := strings.Repeat("b", 20)
	files, _ := vicidialConfiguration(strings.Repeat("a", 48), secret)
	missing := []string{}
	for _, f := range files {
		missing = append(missing, f.path)
	}
	result := func(paths []string, count int) execution.Result {
		b, _ := json.Marshal(struct {
			Missing  []string
			Verified int
		}{paths, count})
		return execution.Result{Started: true, Stdout: string(b)}
	}
	for _, tc := range []struct {
		name        string
		observation execution.Result
		valid       bool
	}{
		{"complete", result(nil, 6), true},
		{"still absent", result(missing, 0), false},
		{"still partial", result(missing[:1], 5), false},
		{"unstarted", execution.Result{Stdout: `{"Missing":[],"Verified":6}`}, false},
		{"failed despite stdout", execution.Result{Started: true, ExitCode: 1, Stdout: `{"Missing":[],"Verified":6}`}, false},
		{"truncated", execution.Result{Started: true, StdoutTruncated: true, Stdout: `{"Missing":[],"Verified":6}`}, false},
		{"foreign missing", result([]string{"/etc/passwd"}, 5), false},
		{"duplicate missing", result([]string{missing[0], missing[0]}, 4), false},
		{"wrong count", result(nil, 5), false},
		{"unknown fields", execution.Result{Started: true, Stdout: `{"Missing":[],"Verified":6,"ignored":true}`}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &vicidialOwnedProbe{t: t, secret: secret, results: []execution.Result{result(missing, 0), tc.observation}}
			identity, err := vicidialOwned(context.Background(), r, "configuration", files, true)
			if (err == nil) != tc.valid {
				t.Fatal(err)
			}
			if len(r.commands) != 2 || r.commands[0].Args[3] != "owned-install" || r.commands[1].Args[3] != "owned-inspect" {
				t.Fatal("missing independent readback")
			}
			if tc.valid && (!shaPattern.MatchString(identity) || strings.Contains(identity, secret)) {
				t.Fatal("unsafe identity")
			}
		})
	}
	r := &vicidialOwnedProbe{t: t, secret: secret, err: errors.New("synthetic failure"), results: []execution.Result{result(nil, 6)}}
	if _, err := vicidialOwned(context.Background(), r, "configuration", files, true); err == nil || len(r.commands) != 1 {
		t.Fatal("failed installation continued")
	}
}
