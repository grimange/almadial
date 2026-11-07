package execution

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Command contains non-secret values only. No shell, PATH lookup, inherited
// environment or credential transport is provided by this foundation.
type Command struct {
	Executable  string        `json:"executable"`
	Args        []string      `json:"args"`
	Timeout     time.Duration `json:"timeout_ns"`
	StdoutLimit int           `json:"stdout_limit"`
	StderrLimit int           `json:"stderr_limit"`
}

func (c Command) validate() error {
	if !filepath.IsAbs(c.Executable) || filepath.Clean(c.Executable) != c.Executable || strings.ContainsAny(c.Executable, "\x00\r\n") {
		return contract("executable must be an absolute clean path without NUL or line breaks")
	}
	if c.Timeout <= 0 || c.Timeout > 24*time.Hour {
		return contract("timeout must be positive and at most 24 hours")
	}
	if c.StdoutLimit < 1 || c.StderrLimit < 1 || c.StdoutLimit > 16<<20 || c.StderrLimit > 16<<20 {
		return contract("each diagnostic limit must be between 1 byte and 16 MiB")
	}
	switch filepath.Base(c.Executable) {
	case "sh", "bash", "dash", "zsh", "ksh", "fish":
		return contract("shell executables are unsupported; provide direct executable and argv")
	}
	if len(c.Args) > 4096 {
		return contract("argv exceeds 4096 arguments")
	}
	n := 0
	for _, a := range c.Args {
		n += len(a)
		if strings.ContainsRune(a, 0) {
			return contract("argv must not contain NUL bytes")
		}
	}
	if n > 1<<20 {
		return contract("argv exceeds 1 MiB")
	}
	return nil
}

type Result struct {
	Started         bool   `json:"started"`
	ExitCode        int    `json:"exit_code"`
	Stdout          string `json:"stdout"`
	Stderr          string `json:"stderr"`
	StdoutTruncated bool   `json:"stdout_truncated"`
	StderrTruncated bool   `json:"stderr_truncated"`
}

// Runner owns a copied, bounded set of exact sensitive byte patterns. It can be
// shared between calls; output buffers and processes are per invocation.
type Runner struct{ secrets []string }

func NewRunner(sensitive []string) (*Runner, error) {
	if len(sensitive) > 64 {
		return nil, failure(Contract)
	}
	r := &Runner{}
	total := 0
	for _, s := range sensitive {
		total += len(s)
		if s == "" || len(s) > 4096 || total > 65536 {
			return nil, failure(Contract)
		}
		r.secrets = append(r.secrets, s)
	}
	return r, nil
}
func (r *Runner) forbidden(c Command) bool {
	for _, s := range r.secrets {
		if strings.Contains(c.Executable, s) {
			return true
		}
		for _, a := range c.Args {
			if strings.Contains(a, s) {
				return true
			}
		}
	}
	return false
}

// redact masks full patterns and incomplete suffixes which could have been cut
// by capture limits or process termination. Check all patterns against the
// original bytes first, so overlapping patterns cannot expose another token.
func (r *Runner) redact(raw []byte, limit int) string {
	masked := make([]bool, len(raw))
	for _, s := range r.secrets {
		token := []byte(s)
		for offset := 0; offset < len(raw); {
			i := bytes.Index(raw[offset:], token)
			if i < 0 {
				break
			}
			start := offset + i
			for j := start; j < start+len(token); j++ {
				masked[j] = true
			}
			offset = start + 1
		}
		for n := 1; n < len(token) && n <= len(raw); n++ {
			if bytes.Equal(raw[len(raw)-n:], token[:n]) {
				for j := len(raw) - n; j < len(raw); j++ {
					masked[j] = true
				}
			}
		}
	}
	var b strings.Builder
	for i := 0; i < len(raw); {
		if masked[i] {
			b.WriteString("[REDACTED]")
			for i < len(raw) && masked[i] {
				i++
			}
		} else {
			b.WriteByte(raw[i])
			i++
		}
	}
	out := b.String()
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

type capture struct {
	mu       sync.Mutex
	data     []byte
	limit    int
	overflow bool
}

func (b *capture) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := len(p)
	remaining := b.limit - len(b.data)
	if len(p) > remaining {
		b.overflow = true
		p = p[:remaining]
	}
	b.data = append(b.data, p...)
	return n, nil
}

// Validate checks the complete static command policy without launching anything.
func (r *Runner) Validate(c Command) error {
	if r == nil {
		return failure(Contract)
	}
	if err := c.validate(); err != nil {
		return err
	}
	if r.forbidden(c) {
		return contract("registered sensitive content is forbidden in executable and argv")
	}
	return nil
}

// Run drains and discards excess bytes until exit/deadline, then reports overflow
// as failure. Timeout/cancellation take priority over overflow and process exit.
func (r *Runner) Run(ctx context.Context, c Command) (Result, error) {
	out := Result{ExitCode: -1}
	if r == nil || ctx == nil {
		return out, failure(Contract)
	}
	if err := r.Validate(c); err != nil {
		return out, err
	}
	if err := ctx.Err(); err != nil {
		return out, contextFailure(err)
	}
	bounded, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()
	cmd := exec.CommandContext(bounded, c.Executable, append([]string(nil), c.Args...)...)
	cmd.Env = []string{"LANG=C", "LC_ALL=C"}
	stdout := &capture{limit: c.StdoutLimit}
	stderr := &capture{limit: c.StderrLimit}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := prepareProcess(cmd); err != nil {
		return out, err
	}
	err := cmd.Start()
	if err != nil {
		if bounded.Err() != nil {
			return out, contextFailure(bounded.Err())
		}
		return out, failure(Start)
	}
	out.Started = true
	err = cmd.Wait()
	cleanupProcess(cmd)
	out.ExitCode = cmd.ProcessState.ExitCode()
	out.Stdout = r.redact(stdout.data, c.StdoutLimit)
	out.Stderr = r.redact(stderr.data, c.StderrLimit)
	out.StdoutTruncated = stdout.overflow
	out.StderrTruncated = stderr.overflow
	if bounded.Err() != nil {
		e := contextFailure(bounded.Err())
		e.ExitCode = out.ExitCode
		return out, e
	}
	if stdout.overflow || stderr.overflow {
		e := failure(OutputLimit)
		e.ExitCode = out.ExitCode
		return out, e
	}
	if err != nil {
		kind := Internal
		if _, ok := err.(*exec.ExitError); ok {
			kind = Exit
		}
		e := failure(kind)
		e.ExitCode = out.ExitCode
		return out, e
	}
	return out, nil
}
func contextFailure(err error) *Error {
	k := Canceled
	if err == context.DeadlineExceeded {
		k = Timeout
	}
	e := failure(k)
	e.cause = err
	return e
}
