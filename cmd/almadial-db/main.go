// almadial-db qualifies only S03 on an explicitly authorized disposable VM.
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
	"github.com/grimange/almadial/internal/operatorlog"
	"github.com/grimange/almadial/internal/preflight"
	"github.com/grimange/almadial/internal/provision"
)

const usage = `Usage: almadial-db OPERATION [flags]
  inspect --config FILE --requirements FILE --budget FILE --manifest FILE --source ZIP
      Read-only S03 inspection; no metadata refresh or writes
  plan --input FILE --manifest FILE --source ZIP
      Deterministic graph from saved inputs and verified pinned SQL
  lab-run --input FILE --manifest FILE --source ZIP --authorization FILE --state-dir DIR
  lab-resume (same flags)
      Only the explicitly authorized disposable-lab S03 graph, using ALM-13/14

The CANDIDATE manifest never authorizes production. A fresh recovery snapshot and
exact target/plan/source approval are mandatory. Partial or unknown DDL requires
review. No destructive recovery or schema replay. Protect host output.
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
	mode := args[0]
	switch mode {
	case "inspect", "plan", "lab-run", "lab-resume":
	default:
		return errInput
	}
	fs := flag.NewFlagSet("almadial-db", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	in := fs.String("input", "", "")
	mf := fs.String("manifest", "", "")
	source := fs.String("source", "", "")
	cf := fs.String("config", "", "")
	rf := fs.String("requirements", "", "")
	bf := fs.String("budget", "", "")
	af := fs.String("authorization", "", "")
	sd := fs.String("state-dir", "", "")
	if fs.Parse(args[1:]) != nil || fs.NArg() != 0 || *mf == "" || *source == "" {
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
	sql, e := provision.LoadDatabaseSource(ctx, m, *source)
	if e != nil {
		return e
	}
	emit := func(v any) error { return json.NewEncoder(out).Encode(v) }
	n, e := provision.NewNative(nil)
	if e != nil {
		return e
	}
	var i provision.DatabaseInputs
	if mode == "inspect" {
		if fs.NFlag() != 5 || *cf == "" || *rf == "" || *bf == "" {
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
		sys, e := n.InspectSystem(ctx)
		if e != nil {
			return e
		}
		db := provision.DatabaseNative{Native: n, Budget: b, Source: sql}
		snap, e := db.InspectDatabase(ctx)
		if e != nil {
			return e
		}
		i = provision.DatabaseInputs{Parent: provision.SystemInputs{Parent: provision.Inputs{Config: c, Preflight: preflight.Collect(ctx, preflight.Linux{}), Requirements: r, Inventory: sys.RPM}, Snapshot: sys, Budget: b}, Snapshot: snap, SourceSHA256: sql.Digest()}
		return emit(i)
	}
	if *in == "" || *cf != "" || *rf != "" || *bf != "" || load(*in, &i) != nil {
		return errInput
	}
	stage, e := i.Prepare(m, sql)
	if e != nil {
		return e
	}
	if mode == "plan" {
		if fs.NFlag() != 3 {
			return errInput
		}
		return emit(struct {
			Stage       json.RawMessage         `json:"stage"`
			StageSHA256 string                  `json:"stage_sha256"`
			Graph       json.RawMessage         `json:"graph"`
			State       provision.DatabaseState `json:"state"`
		}{stage.JSON(), stage.Digest(), stage.Graph().JSON(), i.Snapshot.State()})
	}
	if fs.NFlag() != 5 || *af == "" || *sd == "" {
		return errInput
	}
	var a provision.DatabaseLabAuthorization
	if load(*af, &a) != nil {
		return errInput
	}
	redactor, _ := operatorlog.NewRedactor(nil)
	log, e := operatorlog.New(io.Discard, os.Stderr, nil, redactor)
	if e != nil {
		return e
	}
	db := provision.DatabaseNative{Native: n, Budget: i.Parent.Budget, Source: sql}
	result, e := provision.ExecuteDatabaseLab(ctx, stage, m, a, *sd, db, db.Runner(stage), mode == "lab-resume", log)
	if ee := emit(result); ee != nil {
		return ee
	}
	return e
}
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if e := run(ctx, os.Args[1:], os.Stdout); e != nil {
		fmt.Fprintln(os.Stderr, "almadial-db:", e)
		os.Exit(1)
	}
}
