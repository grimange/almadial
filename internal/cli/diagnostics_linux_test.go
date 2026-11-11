//go:build linux

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/grimange/almadial/internal/diagnostics"
)

func TestDiagnosticRoutingAndUsage(t *testing.T) {
	for _, tc := range []struct {
		args []string
		code int
	}{
		{[]string{"status", "--help"}, 0}, {[]string{"doctor", "--help"}, 0}, {[]string{"evidence", "export", "--help"}, 0}, {[]string{"support-bundle", "--help"}, 0},
		{[]string{"status", "--repair"}, 2}, {[]string{"doctor", "--output", "/tmp/should-not-create"}, 2}, {[]string{"evidence", "export"}, 2}, {[]string{"evidence", "other"}, 2}, {[]string{"status", "--journal", "relative-secret"}, 2}, {[]string{"support-bundle", "--output", "relative-secret"}, 2},
	} {
		var out, err bytes.Buffer
		code := RunContext(context.Background(), tc.args, strings.NewReader(""), &out, &err)
		if code != tc.code {
			t.Fatalf("%v: code %d", tc.args, code)
		}
		if strings.Contains(out.String()+err.String(), "relative-secret") {
			t.Fatal("argument disclosure")
		}
	}
}
func TestDiagnosticRendering(t *testing.T) {
	for _, state := range []string{"OK", "WARN", "FAIL", "UNKNOWN"} {
		r := diagnostics.Report{ReadOnly: true, Checks: []diagnostics.Check{{ID: "fixture", State: state, Observation: "fixed observation", Expected: "healthy", Guidance: "inspect locally", Blocking: true}}}
		want := 10
		if state == "OK" {
			want = 0
		}
		for _, mode := range []string{"status", "doctor"} {
			for _, machine := range []bool{false, true} {
				var out bytes.Buffer
				if code := renderDiagnosticReport(mode, machine, r, &out); code != want {
					t.Fatal(code, want)
				}
				if machine {
					var actual diagnostics.Report
					if json.Unmarshal(out.Bytes(), &actual) != nil || actual.Checks[0].State != state {
						t.Fatal("invalid JSON")
					}
				} else if mode == "doctor" && !strings.Contains(out.String(), "Expected: healthy\n  Guidance: inspect locally") {
					t.Fatal("missing actionable guidance")
				}
			}
		}
		if renderDiagnosticReport("status", true, r, brokenWriter{}) != Failure {
			t.Fatal("output failure")
		}
	}
}
