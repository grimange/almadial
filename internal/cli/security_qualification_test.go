//go:build linux

package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/grimange/almadial/internal/diagnostics"
	"github.com/grimange/almadial/internal/execution"
	"github.com/grimange/almadial/internal/journal"
	"github.com/grimange/almadial/internal/operatorlog"
)

// The artificial secret arrives over stdin, never in the child process argv.
func TestALM31SentinelProcess(t *testing.T) {
	for _, a := range os.Args {
		if a == "--alm31-sentinel-helper" {
			b, _ := io.ReadAll(os.Stdin)
			os.Stdout.Write(b)
			os.Stderr.Write(b)
			os.Exit(7)
		}
	}
}

type sentinelDiagnosticRunner struct{ value string }

func (r sentinelDiagnosticRunner) Run(context.Context, execution.Command) (execution.Result, error) {
	return execution.Result{Started: true, ExitCode: 7, Stdout: r.value, Stderr: r.value}, errors.New(r.value)
}

// Exercise the same representative secret classes through failure capture,
// operator logs, persisted-state decoding and the allowlisted diagnostic export.
// These are synthetic local fixtures, not claims of live-host fault injection.
func TestALM31CredentialLeakageBoundaries(t *testing.T) {
	for _, tc := range []struct{ name, value string }{
		{"database", "ALM31_FAKE_DATABASE_PASSWORD_57"},
		{"sip", "ALM31_FAKE_SIP_PASSWORD_62"},
		{"ami", "ALM31_FAKE_AMI_SECRET_49"},
		{"administrator", "ALM31_FAKE_ADMIN_PASSWORD_83"},
		{"private_key", "-----BEGIN PRIVATE KEY-----\nALM31_FAKE_KEY\n-----END PRIVATE KEY-----"},
		{"session", "ALM31_FAKE_SESSION_TOKEN_91"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner, err := execution.NewRunner([]string{tc.value})
			if err != nil {
				t.Fatal(err)
			}
			exe, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			cmd := execution.Command{Executable: exe, Args: []string{"-test.run=^TestALM31SentinelProcess$", "--", "--alm31-sentinel-helper"}, Timeout: 5 * time.Second, StdoutLimit: 4096, StderrLimit: 4096}
			result, runErr := runner.RunInput(context.Background(), cmd, []byte(tc.value))
			var typed *execution.Error
			if !errors.As(runErr, &typed) || !result.Started || result.ExitCode != 7 {
				t.Fatal("native failure context lost")
			}
			if result.Stdout != "[REDACTED]" || result.Stderr != "[REDACTED]" || strings.Contains(fmt.Sprint(runErr), tc.value) {
				t.Fatal("captured output leaked")
			}
			cmd.Args = append(cmd.Args, tc.value)
			rejected, rejectErr := runner.Run(context.Background(), cmd)
			if rejectErr == nil || rejected.Started || strings.Contains(rejectErr.Error(), tc.value) {
				t.Fatal("secret-bearing argv not safely rejected")
			}

			var stdout, stderr, log bytes.Buffer
			redactor, _ := operatorlog.NewRedactor([]string{tc.value})
			logger, err := operatorlog.New(&stdout, &stderr, &log, redactor)
			if err != nil {
				t.Fatal(err)
			}
			for _, level := range []operatorlog.Level{operatorlog.Info, operatorlog.Error} {
				if err := logger.Write(operatorlog.Record{Level: level, Stage: "S03", Action: "fixture", Message: tc.value, Next: tc.value}); err != nil {
					t.Fatal(err)
				}
			}
			for _, b := range [][]byte{stdout.Bytes(), stderr.Bytes(), log.Bytes()} {
				if len(b) == 0 || bytes.Contains(b, []byte(tc.value)) {
					t.Fatal("operator sink leakage or missing record")
				}
			}

			input, _ := json.Marshal(map[string]string{"password": tc.value})
			_, decodeErr := journal.Decode(bytes.NewReader(input))
			if !errors.Is(decodeErr, journal.ErrInvalid) || strings.Contains(decodeErr.Error(), tc.value) {
				t.Fatal("state decoder exposed invalid input")
			}
			collector := diagnostics.Collector{Runner: sentinelDiagnosticRunner{tc.value}, ReadFile: func(string, int) ([]byte, error) { return input, nil }, Now: func() time.Time { return time.Unix(0, 0) }}
			report := collector.Collect(context.Background(), diagnostics.Options{CAFile: "/fixture/ca", Journal: "/fixture/state", Manifest: "/fixture/manifest"})
			reportJSON, err := json.Marshal(report)
			if err != nil {
				t.Fatal(err)
			}
			bundle, err := diagnostics.Bundle(report)
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Contains(reportJSON, []byte("ALM31_FAKE")) {
				t.Fatal("diagnostic projection leaked")
			}
			gz, err := gzip.NewReader(bytes.NewReader(bundle))
			if err != nil {
				t.Fatal(err)
			}
			defer gz.Close()
			tr := tar.NewReader(gz)
			members := 0
			for {
				h, err := tr.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				switch h.Name {
				case "SHA256SUMS", "manifest.json", "redaction.json", "report.json":
				default:
					t.Fatal("unexpected export member")
				}
				b, err := io.ReadAll(tr)
				if err != nil {
					t.Fatal(err)
				}
				if bytes.Contains(b, []byte(tc.value)) || bytes.Contains(b, []byte("ALM31_FAKE")) {
					t.Fatal("export member leaked")
				}
				members++
			}
			if members != 4 {
				t.Fatal("incomplete export")
			}

		})
	}
}
