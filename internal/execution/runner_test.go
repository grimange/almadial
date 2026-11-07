package execution

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Helpers never use a shell or install software. Mode and test-only output are
// non-secret argv. The artificial sensitive output is generated inside the child.
func TestProcessHelper(t *testing.T) {
	for i, a := range os.Args {
		if a == "--alm13-helper" {
			mode := os.Args[i+1]
			switch mode {
			case "ok":
				fmt.Print("hello")
				fmt.Fprint(os.Stderr, "diagnostic")
			case "empty":
			case "exit":
				fmt.Fprint(os.Stderr, "failure")
				os.Exit(7)
			case "sleep":
				time.Sleep(30 * time.Second)
			case "overflow":
				fmt.Print(strings.Repeat("x", 1<<20))
			case "exact":
				fmt.Print("1234")
				fmt.Fprint(os.Stderr, "1234")
			case "beyond":
				fmt.Print("12345")
			case "binary":
				os.Stdout.Write([]byte{0, 255, 254, 1})
			case "echo":
				fmt.Print(os.Args[i+2])
			case "secret", "secret-exit", "secret-sleep":
				// Split writes exercise capture boundaries without leaking into test logs.
				fmt.Print("api-")
				fmt.Print("token-test")
				fmt.Fprint(os.Stderr, "api-token-test")
				if mode == "secret-exit" {
					os.Exit(3)
				}
				if mode == "secret-sleep" {
					time.Sleep(30 * time.Second)
				}
			case "environment":
				fmt.Print(os.Getenv("ALM13_PARENT_ONLY"))
			case "descendant":
				child := exec.Command(os.Args[0], "-test.run=^TestProcessHelper$", "--", "--alm13-helper", "sleep")
				child.Stdout = os.Stdout
				child.Stderr = os.Stderr
				if err := child.Start(); err != nil {
					os.Exit(9)
				}
				fmt.Println(child.Process.Pid)
				time.Sleep(30 * time.Second)
			case "parent-exits":
				child := exec.Command(os.Args[0], "-test.run=^TestProcessHelper$", "--", "--alm13-helper", "sleep")
				child.Stdout = os.Stdout
				child.Stderr = os.Stderr
				if err := child.Start(); err != nil {
					os.Exit(9)
				}
				fmt.Println(child.Process.Pid)
			default:
				os.Exit(99)
			}
			os.Exit(0)
		}
	}
}
func helper(t *testing.T, mode string) Command {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return Command{exe, []string{"-test.run=^TestProcessHelper$", "--", "--alm13-helper", mode}, 5 * time.Second, 4096, 4096}
}
func category(t *testing.T, err error, want Category) *Error {
	t.Helper()
	var e *Error
	if !errors.As(err, &e) || !errors.Is(err, &Error{Category: want}) {
		t.Fatalf("category mismatch: got %v want %s", err, want)
	}
	return e
}
func TestRunnerResults(t *testing.T) {
	r, _ := NewRunner(nil)
	for _, tt := range []struct {
		mode        string
		limit       int
		kind        Category
		exit        int
		out, stderr string
		truncated   bool
	}{
		{"ok", 4096, "", 0, "hello", "diagnostic", false}, {"empty", 4, "", 0, "", "", false},
		{"exit", 4096, Exit, 7, "", "failure", false}, {"exact", 4, "", 0, "1234", "1234", false},
		{"beyond", 4, OutputLimit, 0, "1234", "", true}, {"overflow", 4, OutputLimit, 0, "xxxx", "", true},
		{"binary", 4, "", 0, string([]byte{0, 255, 254, 1}), "", false},
	} {
		t.Run(tt.mode, func(t *testing.T) {
			c := helper(t, tt.mode)
			c.StdoutLimit = tt.limit
			c.StderrLimit = tt.limit
			got, err := r.Run(context.Background(), c)
			if tt.kind != "" {
				category(t, err, tt.kind)
			} else if err != nil {
				t.Fatal(err)
			}
			if !got.Started || got.ExitCode != tt.exit || got.Stdout != tt.out || got.Stderr != tt.stderr || got.StdoutTruncated != tt.truncated {
				t.Fatal("unexpected bounded result")
			}
		})
	}
}
func TestRunnerContracts(t *testing.T) {
	r, _ := NewRunner([]string{"sensitive-value"})
	cases := []struct {
		name   string
		change func(*Command)
	}{
		{"relative", func(c *Command) { c.Executable = "go" }}, {"shell", func(c *Command) { c.Executable = "/bin/sh"; c.Args = []string{"-c", "true"} }},
		{"timeout", func(c *Command) { c.Timeout = 0 }}, {"huge-timeout", func(c *Command) { c.Timeout = 25 * time.Hour }},
		{"zero-output", func(c *Command) { c.StdoutLimit = 0 }}, {"negative-output", func(c *Command) { c.StderrLimit = -1 }},
		{"huge-output", func(c *Command) { c.StdoutLimit = 17 << 20 }}, {"nul", func(c *Command) { c.Args = []string{"bad\x00arg"} }},
		{"secret-argv", func(c *Command) { c.Args = append(c.Args, "sensitive-value") }},
		{"secret-path", func(c *Command) { c.Executable = "/sensitive-value" }},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			c := helper(t, "ok")
			tt.change(&c)
			result, err := r.Run(context.Background(), c)
			category(t, err, Contract)
			if result.Started {
				t.Fatal("invalid command started")
			}
			if strings.Contains(fmt.Sprint(err), "sensitive-value") {
				t.Fatal("unsafe diagnostic")
			}
		})
	}
	c := helper(t, "ok")
	c.Executable = "/alm13-nonexistent-executable"
	result, err := r.Run(context.Background(), c)
	category(t, err, Start)
	if result.Started {
		t.Fatal("missing executable started")
	}
	for _, secrets := range [][]string{{""}, {strings.Repeat("x", 4097)}, make([]string, 65)} {
		if _, err := NewRunner(secrets); err == nil {
			t.Fatal("invalid redaction accepted")
		}
	}
}
func TestLiteralArgumentsAndEnvironment(t *testing.T) {
	r, _ := NewRunner(nil)
	c := helper(t, "echo")
	literal := "$(touch /never-created); `echo unsafe` * > file"
	c.Args = append(c.Args, literal)
	got, err := r.Run(context.Background(), c)
	if err != nil || got.Stdout != literal {
		t.Fatal("argv was not literal")
	}
	t.Setenv("ALM13_PARENT_ONLY", "not-inherited")
	got, err = r.Run(context.Background(), helper(t, "environment"))
	if err != nil || got.Stdout != "" {
		t.Fatal("environment inherited")
	}
}
func TestCancellation(t *testing.T) {
	r, _ := NewRunner(nil)
	t.Run("before", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		cancel()
		got, err := r.Run(ctx, helper(t, "sleep"))
		category(t, err, Canceled)
		if !errors.Is(err, context.Canceled) || got.Started {
			t.Fatal("cancellation not respected")
		}
	})
	t.Run("deadline", func(t *testing.T) {
		c := helper(t, "sleep")
		c.Timeout = 80 * time.Millisecond
		start := time.Now()
		_, err := r.Run(context.Background(), c)
		category(t, err, Timeout)
		if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > 3*time.Second {
			t.Fatal("deadline not bounded")
		}
	})
	t.Run("during", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		timer := time.AfterFunc(80*time.Millisecond, cancel)
		defer timer.Stop()
		_, err := r.Run(ctx, helper(t, "sleep"))
		category(t, err, Canceled)
	})
}
func TestRedaction(t *testing.T) {
	r, _ := NewRunner([]string{"api-token-test"})
	for _, mode := range []string{"secret", "secret-exit", "secret-sleep"} {
		t.Run(mode, func(t *testing.T) {
			c := helper(t, mode)
			if mode == "secret-sleep" {
				c.Timeout = 150 * time.Millisecond
			}
			got, err := r.Run(context.Background(), c)
			if mode == "secret-exit" {
				category(t, err, Exit)
			}
			if mode == "secret-sleep" {
				category(t, err, Timeout)
			}
			if strings.Contains(got.Stdout, "api-") || strings.Contains(got.Stderr, "api-") || strings.Contains(fmt.Sprint(err), "api-") {
				t.Fatal("redaction failed")
			}
			if mode != "secret-sleep" && got.Stdout != "[REDACTED]" {
				t.Fatal("missing redaction")
			}
		})
	}
	for limit := 1; limit < 14; limit++ {
		c := helper(t, "secret")
		c.StdoutLimit = limit
		c.StderrLimit = limit
		got, _ := r.Run(context.Background(), c)
		if strings.ContainsAny(got.Stdout, "apikntos") || len(got.Stdout) > limit {
			t.Fatal("sensitive truncation suffix exposed")
		}
	}
	for _, test := range []struct {
		patterns []string
		raw      string
		want     string
	}{
		{[]string{"abcde", "cdefg"}, "abcdefg", "[REDACTED]"}, {[]string{"abcde"}, "xxabc", "xx[REDACTED]"},
	} {
		r, _ := NewRunner(test.patterns)
		if r.redact([]byte(test.raw), 100) != test.want {
			t.Fatal("overlap or prefix leak")
		}
	}
}
func TestProcessGroupCleanup(t *testing.T) {
	r, _ := NewRunner(nil)
	for _, mode := range []string{"descendant", "parent-exits"} {
		t.Run(mode, func(t *testing.T) {
			c := helper(t, mode)
			c.Timeout = 400 * time.Millisecond
			if mode == "parent-exits" {
				c.Timeout = 5 * time.Second
			}
			got, err := r.Run(context.Background(), c)
			if mode == "descendant" {
				category(t, err, Timeout)
			} else {
				category(t, err, Internal)
			}
			pid, parseErr := strconv.Atoi(strings.TrimSpace(got.Stdout))
			if parseErr != nil {
				t.Fatal("helper failed to report child")
			}
			defer syscall.Kill(pid, syscall.SIGKILL)
			// A killed orphan may briefly remain a zombie owned by PID 1; it cannot run.
			deadline := time.Now().Add(2 * time.Second)
			for {
				b, e := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
				if os.IsNotExist(e) || strings.Contains(string(b), ") Z ") {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("descendant still running")
				}
				time.Sleep(10 * time.Millisecond)
			}
		})
	}
}
