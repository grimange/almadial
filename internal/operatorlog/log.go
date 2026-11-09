// Package operatorlog renders bounded records after registered-secret redaction.
// Callers must register secrets before submitting diagnostics. No secret sources
// are resolved here, and unknown/encoded secrets cannot be recognized.
package operatorlog

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"unicode"
)

const MaxMessage = 16384
const MaxOutput = 1 << 20

var ErrLimit = errors.New("operator log: output limit reached; select a new private log destination")
var ErrWrite = errors.New("operator log: write failed; check destination capacity and permissions")

type Redactor struct{ secrets []string }

func NewRedactor(secrets []string) (*Redactor, error) {
	if len(secrets) > 64 {
		return nil, errors.New("redaction: registration limit exceeded")
	}
	r := &Redactor{}
	total := 0
	for _, s := range secrets {
		total += len(s)
		if s == "" || len(s) > 4096 || total > 65536 {
			return nil, errors.New("redaction: invalid registration")
		}
		r.secrets = append(r.secrets, s)
	}
	return r, nil
}

// Text treats fragments as ONE ordered diagnostic record, without separators.
// This supports secrets spanning stdout/stderr fragments and chunk boundaries.
// Truncation happens before masking; incomplete token suffixes are masked too.
// Across independent records no fragment correlation is promised. Do not stream
// unredacted fragments to a sink; assemble one bounded record before publication.
func (r *Redactor) Text(fragments ...string) string {
	if r == nil {
		return "[diagnostic unavailable]"
	}
	raw := make([]byte, 0, MaxMessage)
	truncated := false
	for _, s := range fragments {
		n := min(len(s), MaxMessage-len(raw))
		raw = append(raw, s[:n]...)
		if n < len(s) {
			truncated = true
		}
	}
	masked := make([]bool, len(raw))
	for _, s := range r.secrets {
		token := []byte(s)
		for off := 0; off < len(raw); {
			i := bytes.Index(raw[off:], token)
			if i < 0 {
				break
			}
			start := off + i
			for j := start; j < start+len(token); j++ {
				masked[j] = true
			}
			off = start + 1
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
	// Strip terminal controls AFTER matching original secret bytes.
	out := strings.Map(func(c rune) rune {
		if unicode.IsControl(c) || unicode.Is(unicode.Cf, c) {
			return ' '
		}
		return c
	}, b.String())
	if len(out) > MaxMessage {
		out = out[:MaxMessage]
		truncated = true
	}
	if truncated {
		out += " [TRUNCATED]"
	}
	clear(raw)
	return out
}

type Level string

const (
	Info    Level = "INFO"
	Warning Level = "WARNING"
	Error   Level = "ERROR"
)

type Record struct {
	Level   Level  `json:"level"`
	Stage   string `json:"stage"`
	Action  string `json:"action"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Next    string `json:"next"`
}

// Logger serializes whole records. INFO uses stdout; WARNING/ERROR use stderr.
// Optional file output receives the exact same safe JSON lines. A sink failure
// poisons the logger; continuing would hide an incomplete diagnostic record.
type Logger struct {
	mu             sync.Mutex
	out, err, file io.Writer
	redactor       *Redactor
	used           int
	failed         error
}

func New(out, err, file io.Writer, r *Redactor) (*Logger, error) {
	if out == nil || err == nil || r == nil {
		return nil, errors.New("operator log: missing sink or redactor")
	}
	return &Logger{out: out, err: err, file: file, redactor: r}, nil
}
func (l *Logger) Write(rec Record, fragments ...string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.failed != nil {
		return l.failed
	}
	if rec.Level != Info && rec.Level != Warning && rec.Level != Error {
		return errors.New("operator log: invalid severity")
	}
	r := l.redactor
	rec.Stage = r.Text(rec.Stage)
	rec.Action = r.Text(rec.Action)
	rec.Code = r.Text(rec.Code)
	rec.Next = r.Text(rec.Next)
	all := append([]string{rec.Message}, fragments...)
	rec.Message = r.Text(all...)
	b, e := json.Marshal(rec)
	if e != nil {
		return ErrWrite
	}
	b = append(b, '\n')
	if len(b) > MaxOutput-l.used {
		l.failed = ErrLimit
		return l.failed
	}
	l.used += len(b)
	sinks := []io.Writer{l.out}
	if rec.Level != Info {
		sinks[0] = l.err
	}
	if l.file != nil {
		sinks = append(sinks, l.file)
	}
	for _, w := range sinks {
		n, e := w.Write(b)
		if e != nil || n != len(b) {
			l.failed = ErrWrite
			return l.failed
		}
	}
	return nil
}
