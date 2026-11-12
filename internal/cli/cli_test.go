package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestCommands(t *testing.T) {
	for _, tc := range []struct {
		name     string
		args     []string
		code     int
		fragment string
	}{
		{"empty", nil, 0, "Usage:"},
		{"help", []string{"help"}, 0, "Usage:"},
		{"long help", []string{"--help"}, 0, "Usage:"},
		{"short help", []string{"-h"}, 0, "Usage:"},
		{"version", []string{"version"}, 0, "almadial dev"},
		{"long version", []string{"--version"}, 0, "commit: unknown"},
		{"unknown", []string{"secret-value"}, 2, "unknown command"},
		{"flag", []string{"--bad"}, 2, "unknown command"},
		{"install", []string{"install"}, 2, "installation inputs required"},
		{"check", []string{"check"}, 2, "unknown command"},
		{"verify", []string{"verify"}, 2, "unknown command"},
		{"extra", []string{"version", "extra"}, 2, "expected one command"},
		{"empty argument", []string{""}, 2, "unknown command"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, err bytes.Buffer
			if code := Run(tc.args, &out, &err); code != tc.code {
				t.Fatalf("code %d != %d", code, tc.code)
			}
			if tc.code == 0 {
				if err.Len() != 0 || !strings.Contains(out.String(), tc.fragment) {
					t.Fatalf("stdout=%q stderr=%q", out.String(), err.String())
				}
			} else if out.Len() != 0 || !strings.Contains(err.String(), tc.fragment) || strings.Contains(err.String(), "secret-value") {
				t.Fatalf("stdout=%q stderr=%q", out.String(), err.String())
			}
		})
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errors.New("synthetic write failure") }
func TestOutputFailure(t *testing.T) {
	var err bytes.Buffer
	if code := Run(nil, brokenWriter{}, &err); code != Failure || !strings.Contains(err.String(), "cannot write output") {
		t.Fatalf("code=%d stderr=%q", code, err.String())
	}
}
