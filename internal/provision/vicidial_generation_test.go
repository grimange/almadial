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

type vicidialGenerationProbeRunner struct {
	output string
	result execution.Result
	err    error
	calls  int
}

func (r *vicidialGenerationProbeRunner) RunInput(_ context.Context, _ execution.Command, _ []byte) (execution.Result, error) {
	r.calls++
	v := r.result
	v.Stdout = r.output
	return v, r.err
}
func TestVICIdialGenerationReceiptInspection(t *testing.T) {
	binding, inputs := hash("binding"), hash("inputs")
	for _, tc := range []struct {
		name, output   string
		valid, wrapped bool
	}{
		{"absent", `{"State":"absent","Wrapped":false}`, true, false},
		{"published", `{"State":"` + hash("published") + `","Wrapped":true}`, true, true},
		{"absent but wrapped", `{"State":"absent","Wrapped":true}`, false, false},
		{"published but unwrapped", `{"State":"` + hash("published") + `","Wrapped":false}`, false, false},
		{"partial publication", `{"State":"partial","Wrapped":true}`, false, false},
		{"unknown field", `{"State":"absent","Wrapped":false,"extra":true}`, false, false},
		{"trailing payload", `{"State":"absent","Wrapped":false} {}`, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &vicidialGenerationProbeRunner{output: tc.output, result: execution.Result{Started: true}}
			state, wrapped, err := inspectVICIdialGeneration(context.Background(), r, binding, inputs)
			if (err == nil) != tc.valid || (tc.valid && (wrapped != tc.wrapped || state == "")) {
				t.Fatal(state, wrapped, err)
			}
		})
	}
	for _, result := range []execution.Result{{}, {Started: true, ExitCode: 1}, {Started: true, StdoutTruncated: true}, {Started: true, StderrTruncated: true}} {
		r := &vicidialGenerationProbeRunner{output: `{"State":"absent","Wrapped":false}`, result: result}
		if _, _, err := inspectVICIdialGeneration(context.Background(), r, binding, inputs); err == nil {
			t.Fatal("failed observation accepted")
		}
	}
	r := &vicidialGenerationProbeRunner{result: execution.Result{Started: true}}
	if _, _, err := inspectVICIdialGeneration(context.Background(), r, "bad", inputs); err == nil || r.calls != 0 {
		t.Fatal("invalid binding dispatched")
	}
}
func TestVICIdialBridgePostconditions(t *testing.T) {
	rows := func(change int) string {
		out := []string{}
		for i := 0; i < 300; i++ {
			n, server, extension, leave := 9600000+i, "127.0.0.1", "", "0"
			var date any
			switch change {
			case 1:
				if i == 0 {
					n++
				}
			case 2:
				server = "192.0.2.1"
			case 3:
				extension = "live-call"
			case 4:
				leave = "1"
			case 5:
				date = "2026-01-01 00:00:00"
			}
			b, _ := json.Marshal([]any{n, server, extension, leave, date})
			out = append(out, string(b))
		}
		return strings.Join(out, "\n")
	}
	for _, tc := range []struct {
		name, text    string
		absent, valid bool
	}{
		{"empty before", "", true, true}, {"empty after", "", false, false}, {"complete", rows(0), false, true},
		{"duplicate slot", rows(1), false, false}, {"foreign server", rows(2), false, false},
		{"active extension", rows(3), false, false}, {"active leave", rows(4), false, false},
		{"active date", rows(5), false, false}, {"missing slot", strings.Join(strings.Split(rows(0), "\n")[:299], "\n"), false, false},
		{"extra slot", rows(0) + "\n[9600300,\"127.0.0.1\",\"\",\"0\",null]", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &vicidialGenerationProbeRunner{output: tc.text, result: execution.Result{Started: true}}
			if err := inspectVICIdialBridges(context.Background(), r, tc.absent); (err == nil) != tc.valid {
				t.Fatal(err)
			}
		})
	}
	r := &vicidialGenerationProbeRunner{err: errors.New("bounded runner failure")}
	if inspectVICIdialBridges(context.Background(), r, true) == nil {
		t.Fatal("SQL failure treated as absent")
	}
}
func TestVICIdialRegistrationPreimagePublication(t *testing.T) {
	row := vicidialSeedRegistration()
	good := `{"Preserved":"` + rawSHA(row.protectedPreimage()) + `"}`
	for _, output := range []string{good, `{"Preserved":"wrong"}`, `{}`, good + " {}"} {
		r := &vicidialGenerationProbeRunner{output: output, result: execution.Result{Started: true}}
		if err := preserveVICIdialRegistration(context.Background(), r, row); (err == nil) != (output == good) {
			t.Fatal(err)
		}
	}
}
