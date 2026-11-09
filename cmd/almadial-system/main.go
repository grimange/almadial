// almadial-system qualifies only S02 on an explicitly authorized disposable VM.
// It neither enables normal installation nor installs/configures applications.
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
	"strconv"
	"strings"
	"syscall"

	"github.com/grimange/almadial/internal/artifacts"
	"github.com/grimange/almadial/internal/config"
	"github.com/grimange/almadial/internal/operatorlog"
	"github.com/grimange/almadial/internal/preflight"
	"github.com/grimange/almadial/internal/provision"
)

const usage = `Usage: almadial-system OPERATION [flags]
  inventory
      Read accounts, RPM trust, approved paths, persistent SELinux defaults and storage
  assess --input FILE
      Pure assessment of saved SystemInputs, including explicit recording budget
  inspect --config FILE --requirements FILE --budget FILE
      Read host state into immutable planning inputs; no metadata refresh or writes
  plan --input FILE --manifest FILE
      Pure S02 action graph, stage identity and findings
  lab-run --input FILE --manifest FILE --authorization FILE --state-dir DIR
  lab-resume (same flags)
      Execute only the reviewed S02 graph with ALM-13/14 locking and durable intent

No production installation authorization is granted. The manifest remains CANDIDATE.
Only fixed approved identities/directories are supported. No file replacement,
recursive ownership/permission changes, custom labels, service changes or packages.
Existing incompatible metadata is a conflict. Capture private host output securely.
Use a new private journal directory per reviewed plan; resume uses original inputs.
`

var errInput = errors.New("invalid qualification input; review required files and flags")

func readFile(path string) (*os.File, error) {
	f, e := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if e != nil {
		return nil, errInput
	}
	s, e := f.Stat()
	if e != nil || !s.Mode().IsRegular() || s.Size() > 4<<20 {
		f.Close()
		return nil, errInput
	}
	return f, nil
}
func load(path string, v any) error {
	f, e := readFile(path)
	if e != nil {
		return e
	}
	defer f.Close()
	d := json.NewDecoder(io.LimitReader(f, 4<<20))
	d.DisallowUnknownFields()
	if d.Decode(v) != nil {
		return errInput
	}
	if _, e = d.Token(); e != io.EOF {
		return errInput
	}
	return nil
}
func run(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "help" {
		_, e := io.WriteString(out, usage)
		return e
	}
	n, e := provision.NewNative(nil)
	if e != nil {
		return e
	}
	// This fixed child helper requires its live S02 parent, retained ALM-14
	// ownership and matching durable running intent. It accepts no arbitrary
	// destination or command and performs no shell interpretation.
	if args[0] == "__directory" {
		if len(args) != 3 {
			return errInput
		}
		index, e := strconv.Atoi(args[1])
		if e != nil {
			return errInput
		}
		var before provision.SystemDirectoryRequest
		d := json.NewDecoder(strings.NewReader(args[2]))
		d.DisallowUnknownFields()
		if d.Decode(&before) != nil {
			return errInput
		}
		if _, e = d.Token(); e != io.EOF {
			return errInput
		}
		return provision.ApplySystemDirectory(ctx, index, before, n)
	}
	mode := args[0]
	switch mode {
	case "inventory", "assess", "inspect", "plan", "lab-run", "lab-resume":
	default:
		return errInput
	}
	fs := flag.NewFlagSet("almadial-system", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	in := fs.String("input", "", "")
	mf := fs.String("manifest", "", "")
	cf := fs.String("config", "", "")
	rf := fs.String("requirements", "", "")
	bf := fs.String("budget", "", "")
	af := fs.String("authorization", "", "")
	sd := fs.String("state-dir", "", "")
	if fs.Parse(args[1:]) != nil || fs.NArg() != 0 {
		return errInput
	}
	emit := func(v any) error { return json.NewEncoder(out).Encode(v) }
	if mode == "inventory" {
		if fs.NFlag() != 0 {
			return errInput
		}
		s, e := n.InspectSystem(ctx)
		if e != nil {
			return e
		}
		return emit(s)
	}
	if mode == "inspect" {
		if fs.NFlag() != 3 || *cf == "" || *rf == "" || *bf == "" {
			return errInput
		}
		f, e := readFile(*cf)
		if e != nil {
			return e
		}
		c, e := config.Load(f)
		f.Close()
		if e != nil {
			return e
		}
		var r preflight.Requirements
		var b provision.RecordingBudget
		if load(*rf, &r) != nil || load(*bf, &b) != nil {
			return errInput
		}
		s, e := n.InspectSystem(ctx)
		if e != nil {
			return e
		}
		return emit(provision.SystemInputs{Parent: provision.Inputs{Config: c, Preflight: preflight.Collect(ctx, preflight.Linux{}), Requirements: r, Inventory: s.RPM}, Snapshot: s, Budget: b})
	}
	if *in == "" {
		return errInput
	}
	var i provision.SystemInputs
	if load(*in, &i) != nil {
		return errInput
	}
	if mode == "assess" {
		if fs.NFlag() != 1 {
			return errInput
		}
		return emit(provision.AssessSystem(i.Snapshot, i.Budget))
	}
	if *mf == "" || *cf != "" || *rf != "" || *bf != "" {
		return errInput
	}
	f, e := readFile(*mf)
	if e != nil {
		return e
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
		if fs.NFlag() != 2 {
			return errInput
		}
		return emit(struct {
			Stage       json.RawMessage           `json:"stage"`
			StageSHA256 string                    `json:"stage_sha256"`
			Graph       json.RawMessage           `json:"graph"`
			Findings    []provision.SystemFinding `json:"findings"`
		}{s.JSON(), s.Digest(), s.Graph().JSON(), provision.AssessSystem(i.Snapshot, i.Budget)})
	}
	if fs.NFlag() != 4 || *af == "" || *sd == "" {
		return errInput
	}
	var a provision.SystemLabAuthorization
	if load(*af, &a) != nil {
		return errInput
	}
	redactor, _ := operatorlog.NewRedactor(nil)
	log, e := operatorlog.New(io.Discard, os.Stderr, nil, redactor)
	if e != nil {
		return e
	}
	result, e := provision.ExecuteSystemLab(ctx, s, m, a, *sd, n, n.Runner, mode == "lab-resume", log)
	if ee := emit(result); ee != nil {
		return ee
	}
	return e
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if e := run(ctx, os.Args[1:], os.Stdout); e != nil {
		fmt.Fprintln(os.Stderr, "almadial-system:", e)
		os.Exit(1)
	}
}
