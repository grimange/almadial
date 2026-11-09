// almadial-rpm is an explicit S01 disposable-lab qualification entry point.
// It does not implement or authorize the normal multi-stage installer.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/grimange/almadial/internal/artifacts"
	"github.com/grimange/almadial/internal/config"
	"github.com/grimange/almadial/internal/preflight"
	"github.com/grimange/almadial/internal/provision"
)

const usage = `Usage: almadial-rpm OPERATION [flags]
  inventory                         Read local RPM/repository/key/history state
  assess --input FILE               Assess a saved inventory without host probes
  inspect --config FILE --requirements FILE
                                    Read inventory and M1 host snapshot
  plan --input FILE --manifest FILE  Pure S01 specialization of saved M1 inputs
  lab-run --input FILE --manifest FILE --authorization FILE --state-dir DIR
                                    Execute the exact separately authorized stage
  lab-resume (same flags)            Reconcile; uncertain RPM actions never replay

No operation writes repository files. plan performs no host probes or mutations.
lab-run without an input transaction is an explicit DNF metadata/cache operation.
Input prepare_repository_trust=true instead plans only the pinned EPEL key import.
Its returned transaction must be reviewed, placed in input.transaction, replanned
and separately authorized before package installation. A CANDIDATE manifest is
never authorized for normal installation. Output may contain private host data;
capture it only in protected evidence. No network endpoint or arbitrary command
can be supplied. Credentials are not read. Ctrl-C cancels the bounded action.
`

func regularInput(path string) (*os.File, error) {
	f, e := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if e != nil {
		return nil, errors.New("cannot open input")
	}
	s, e := f.Stat()
	if e != nil || !s.Mode().IsRegular() || s.Size() > 4<<20 {
		f.Close()
		return nil, errors.New("invalid input")
	}
	return f, nil
}
func input(path string, v any) error {
	f, e := regularInput(path)
	if e != nil {
		return e
	}
	defer f.Close()
	d := json.NewDecoder(io.LimitReader(f, 4<<20))
	d.DisallowUnknownFields()
	if d.Decode(v) != nil {
		return errors.New("invalid JSON input")
	}
	if _, e = d.Token(); e != io.EOF {
		return errors.New("trailing input")
	}
	return nil
}
func run(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "help" {
		_, e := io.WriteString(out, usage)
		return e
	}
	mode := args[0]
	if mode != "inventory" && mode != "assess" && mode != "inspect" && mode != "plan" && mode != "lab-run" && mode != "lab-resume" {
		return errors.New("unknown operation")
	}
	fs := flag.NewFlagSet("almadial-rpm", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	in := fs.String("input", "", "")
	mf := fs.String("manifest", "", "")
	cf := fs.String("config", "", "")
	rf := fs.String("requirements", "", "")
	af := fs.String("authorization", "", "")
	sd := fs.String("state-dir", "", "")
	if fs.Parse(args[1:]) != nil || fs.NArg() != 0 {
		return errors.New("invalid flags")
	}
	n, e := provision.NewNative(nil)
	if e != nil {
		return e
	}
	emit := func(v any) error { return json.NewEncoder(out).Encode(v) }
	if mode == "assess" {
		if *in == "" || fs.NFlag() != 1 {
			return errors.New("assess requires only input")
		}
		var s provision.Snapshot
		if e = input(*in, &s); e != nil {
			return e
		}
		return emit(provision.Assess(s, nil, false))
	}
	if mode == "inventory" {
		if fs.NFlag() != 0 {
			return errors.New("inventory accepts no flags")
		}
		s, e := n.Inspect(ctx)
		if e != nil {
			return e
		}
		return emit(s)
	}
	if mode == "inspect" {
		if *cf == "" || *rf == "" || fs.NFlag() != 2 {
			return errors.New("inspect requires only config and requirements")
		}
		var c config.Config
		var r preflight.Requirements
		f, e := regularInput(*cf)
		if e != nil {
			return errors.New("cannot open config")
		}
		c, e = config.Load(f)
		f.Close()
		if e != nil {
			return e
		}
		if e = input(*rf, &r); e != nil {
			return e
		}
		s, e := n.Inspect(ctx)
		if e != nil {
			return e
		}
		return emit(provision.Inputs{Config: c, Requirements: r, Preflight: preflight.Collect(ctx, preflight.Linux{}), Inventory: s})
	}
	if *in == "" || *mf == "" || *cf != "" || *rf != "" {
		return errors.New("input and manifest required")
	}
	var i provision.Inputs
	if e = input(*in, &i); e != nil {
		return e
	}
	f, e := regularInput(*mf)
	if e != nil {
		return errors.New("cannot open manifest")
	}
	m, e := artifacts.Load(f)
	f.Close()
	if e != nil {
		return e
	}
	s, e := i.Prepare(m)
	if e != nil {
		return e
	}
	if mode == "plan" {
		if *af != "" || *sd != "" {
			return errors.New("plan accepts no authorization or state directory")
		}
		return emit(struct {
			Stage       json.RawMessage `json:"stage"`
			StageSHA256 string          `json:"stage_sha256"`
			Graph       json.RawMessage `json:"graph"`
		}{s.JSON(), s.Digest(), s.Graph().JSON()})
	}
	if *af == "" || *sd == "" {
		return errors.New("exact lab authorization and private state directory required")
	}
	var a provision.LabAuthorization
	if e = input(*af, &a); e != nil {
		return e
	}
	result, e := provision.ExecuteLab(ctx, s, m, a, *sd, n, mode == "lab-resume", nil)
	if ee := emit(result); ee != nil {
		return ee
	}
	return e
}
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if e := run(ctx, os.Args[1:], os.Stdout); e != nil {
		fmt.Fprintln(os.Stderr, "almadial-rpm:", e)
		os.Exit(1)
	}
}
