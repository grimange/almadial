package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"syscall"
	"unsafe"

	"github.com/grimange/almadial/internal/artifacts"
	"github.com/grimange/almadial/internal/config"
	"github.com/grimange/almadial/internal/operatorlog"
	"github.com/grimange/almadial/internal/operatorui"
	"github.com/grimange/almadial/internal/preflight"
)

const wizardHelp = `Usage: almadial wizard --manifest FILE --requirements FILE [--config FILE]
  --review-only             Review preparation without requesting approval
  --unattended              Never prompt; requires --config
  --approve                 Explicit unattended approval intent
  --config-sha256 DIGEST     Effective configuration identity being approved
  --plan-sha256 DIGEST       Immutable plan identity being approved
  --log-file FILE            New diagnostic file in an existing owned 0700 directory
  --help                    Show wizard usage

Interactive input and output must both be terminals. Credential prompts accept
source references only, never passwords. No installation actions are implemented.
A CANDIDATE manifest always blocks approval. Review-only can display blocked plans.
Requirements JSON uses the existing preflight.Requirements fields and must supply
an explicitly reviewed resource/ISA budget; no production sizing is assumed.
`

func terminal(v any) bool {
	f, ok := v.(*os.File)
	if !ok {
		return false
	}
	var term syscall.Termios
	_, _, e := syscall.Syscall6(syscall.SYS_IOCTL, f.Fd(), syscall.TCGETS, uintptr(unsafe.Pointer(&term)), 0, 0, 0)
	return e == 0
}

// Open bounded regular input without following the final symlink or blocking on
// FIFOs/devices. Error text never includes the supplied private path.
func regular(path string, limit int64) (*os.File, error) {
	f, e := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0)
	if e != nil {
		return nil, errors.New("cannot open input file; use a readable regular file")
	}
	s, e := f.Stat()
	if e != nil || !s.Mode().IsRegular() || s.Size() > limit {
		f.Close()
		return nil, errors.New("input must be a bounded regular file")
	}
	return f, nil
}
func runWizard(ctx context.Context, args []string, in io.Reader, out, errout io.Writer) (code int) {
	fs := flag.NewFlagSet("wizard", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	cfg := fs.String("config", "", "")
	manifest := fs.String("manifest", "", "")
	budget := fs.String("requirements", "", "")
	unattended := fs.Bool("unattended", false, "")
	review := fs.Bool("review-only", false, "")
	approve := fs.Bool("approve", false, "")
	cd := fs.String("config-sha256", "", "")
	pd := fs.String("plan-sha256", "", "")
	logpath := fs.String("log-file", "", "")
	help := fs.Bool("help", false, "")
	fail := func(e error, c int) int {
		category := "contract"
		if c == Failure {
			category = "precondition"
		}
		if errors.Is(e, operatorui.ErrCanceled) {
			category = "canceled"
		}
		fmt.Fprintf(errout, "almadial: stage preparation code %s status NOT STARTED: %s\nAction: correct the reported input or prerequisite and rerun wizard; no installation started.\n", category, e.Error())
		return c
	}
	if fs.Parse(args) != nil || fs.NArg() != 0 {
		return fail(errors.New("invalid wizard arguments; use wizard --help"), Usage)
	}
	if *help {
		if _, e := io.WriteString(out, wizardHelp); e != nil {
			return Failure
		}
		return Success
	}
	if *manifest == "" || *budget == "" {
		return fail(errors.New("--manifest and --requirements are required; use wizard --help"), Usage)
	}
	interactive := terminal(in) && terminal(out) && !*unattended
	if !interactive && !*unattended {
		return fail(errors.New("no interactive terminal; use --unattended --config FILE and explicit approval, or --review-only"), Usage)
	}
	if *unattended && *cfg == "" {
		return fail(errors.New("--unattended requires --config FILE; stdin is never used for configuration"), Usage)
	}
	if !*unattended && (*approve || *cd != "" || *pd != "") {
		return fail(errors.New("approval flags require --unattended"), Usage)
	}
	if *review && (*approve || *cd != "" || *pd != "") {
		return fail(errors.New("--review-only cannot be combined with approval flags"), Usage)
	}
	var c config.Config
	if *cfg != "" {
		f, e := regular(*cfg, 1<<20)
		if e != nil {
			return fail(e, Usage)
		}
		c, e = config.Load(f)
		f.Close()
		if e != nil {
			return fail(e, Usage)
		}
	}
	f, e := regular(*manifest, 64<<10)
	if e != nil {
		return fail(e, Usage)
	}
	m, e := artifacts.Load(f)
	f.Close()
	if e != nil {
		return fail(e, Usage)
	}
	f, e = regular(*budget, 64<<10)
	if e != nil {
		return fail(e, Usage)
	}
	var r preflight.Requirements
	dec := json.NewDecoder(io.LimitReader(f, (64<<10)+1))
	dec.DisallowUnknownFields()
	e = dec.Decode(&r)
	var extra any
	if e == nil && dec.Decode(&extra) != io.EOF {
		e = errors.New("extra JSON")
	}
	f.Close()
	if e != nil {
		return fail(errors.New("invalid requirements JSON; use reviewed preflight.Requirements fields"), Usage)
	}
	var prompts *operatorui.Prompts
	promptCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	if interactive {
		prompts = operatorui.NewPrompts(promptCtx, in, out)
		c, e = operatorui.Configure(prompts, c)
		if e != nil {
			return fail(e, Failure)
		}
	}
	if ctx.Err() != nil {
		return fail(operatorui.ErrCanceled, Failure)
	}
	// Read-only local inspection only. No remote targets, privilege escalation,
	// package mutation, journal initialization or installation is called.
	s := preflight.Collect(ctx, preflight.Linux{})
	if ctx.Err() != nil {
		return fail(operatorui.ErrCanceled, Failure)
	}
	prep, e := operatorui.Prepare(c, s, r, m)
	if e != nil {
		return fail(e, Usage)
	}
	if e = prep.Present(out); e != nil {
		return fail(errors.New("cannot write plan presentation"), Failure)
	}
	var logFile *os.File
	if *logpath != "" {
		logFile, e = operatorlog.OpenFile(*logpath)
		if e != nil {
			return fail(e, Failure)
		}
		defer func() {
			syncErr := logFile.Sync()
			closeErr := logFile.Close()
			if syncErr != nil || closeErr != nil {
				fmt.Fprintln(errout, "almadial: operator log finalization failed; check storage")
				code = Failure
			}
		}()
	}
	redactor, _ := operatorlog.NewRedactor(nil)
	var fileSink io.Writer
	if logFile != nil {
		fileSink = logFile
	}
	logger, _ := operatorlog.New(out, errout, fileSink, redactor)
	emit := func(level operatorlog.Level, code, message, next string) int {
		if e := logger.Write(operatorlog.Record{Level: level, Stage: "preparation", Code: code, Message: message, Next: next}); e != nil {
			return fail(e, Failure)
		}
		if level == operatorlog.Error {
			return Failure
		}
		return Success
	}
	if *review {
		return emit(operatorlog.Info, "review_only", "Plan review completed; approval NOT REQUESTED; execution NOT STARTED.", "Resolve all blocking findings before requesting approval.")
	}
	var decision operatorui.Decision
	if *unattended {
		decision = operatorui.Decision{Approved: *approve, ConfigSHA256: *cd, PlanSHA256: *pd}
		e = decision.Validate(prep)
	} else {
		decision, e = prep.Approve(ctx, prompts)
	}
	if ctx.Err() != nil {
		e = operatorui.ErrCanceled
	}
	if e != nil {
		return emit(operatorlog.Error, "approval_rejected", e.Error(), "Correct configuration/approval inputs, resolve mandatory findings and obtain an authorized manifest; review a fresh plan.")
	}
	return emit(operatorlog.Info, "approval_recorded", "Transient approval decision recorded for config "+decision.ConfigSHA256+" plan "+decision.PlanSHA256+"; execution NOT STARTED.", "Future executor must revalidate all authorization and acquire exclusive journal ownership before mutation.")
}

// RunContext adds preparation to the accepted help/version router.
func RunContext(ctx context.Context, args []string, in io.Reader, out, err io.Writer) int {
	if len(args) > 0 && (args[0] == "plan" || args[0] == "install" || args[0] == "resume" || args[0] == "__directory") {
		return runInstall(ctx, args, out, err)
	}
	if len(args) > 0 && (args[0] == "status" || args[0] == "doctor" || args[0] == "evidence" || args[0] == "support-bundle") {
		return runDiagnostics(ctx, args, out, err)
	}
	if len(args) > 0 && args[0] == "wizard" {
		return runWizard(ctx, args[1:], in, out, err)
	}
	return runBasic(args, out, err)
}
