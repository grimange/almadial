package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"os"
	"syscall"

	"github.com/grimange/almadial/internal/artifacts"
	"github.com/grimange/almadial/internal/operatorlog"
	"github.com/grimange/almadial/internal/provision"
)

// The accepted S04 input is retained as an immutable parent, including its
// foundation observations. S05 never re-executes an earlier stage.
func runNative(ctx context.Context, args []string, out io.Writer) error {
	mode := args[0]
	fs := flag.NewFlagSet("almadial-vicidial", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	input := fs.String("input", "", "")
	accepted := fs.String("accepted-asterisk", "", "")
	manifest := fs.String("manifest", "", "")
	source := fs.String("source", "", "")
	perl := fs.String("perl-source", "", "")
	ast := fs.String("asterisk-source", "", "")
	signature := fs.String("signature", "", "")
	keyring := fs.String("keyring", "", "")
	scratch := fs.String("scratch", "", "")
	pj := fs.String("pjproject", "", "")
	auth := fs.String("authorization", "", "")
	state := fs.String("state-dir", "", "")
	if fs.Parse(args[1:]) != nil || fs.NArg() != 0 {
		return errInput
	}
	for _, value := range []string{*manifest, *source, *perl, *ast, *signature, *keyring, *scratch, *pj} {
		if value == "" {
			return errInput
		}
	}
	switch mode {
	case "inspect":
		if fs.NFlag() != 9 || *accepted == "" || *input != "" {
			return errInput
		}
	case "plan":
		if fs.NFlag() != 9 || *input == "" || *accepted != "" {
			return errInput
		}
	case "lab-run", "lab-resume":
		if fs.NFlag() != 11 || *input == "" || *accepted != "" || *auth == "" || *state == "" {
			return errInput
		}
	default:
		return errInput
	}
	f, e := readNativeFile(*manifest)
	if e != nil {
		return e
	}
	m, e := artifacts.Load(f)
	f.Close()
	if e != nil {
		return e
	}
	s, e := provision.LoadVICIdialSource(ctx, m, *source)
	if e != nil {
		return e
	}
	p, e := provision.LoadVICIdialPerlSource(ctx, m, *perl)
	if e != nil {
		return e
	}
	a, e := provision.LoadAsteriskSource(ctx, m, *ast, *signature, *keyring, *scratch, *pj)
	if e != nil {
		return e
	}
	database, e := provision.LoadDatabaseSource(ctx, m, *source)
	if e != nil {
		return e
	}
	var i provision.VICIdialInputs
	if mode == "inspect" {
		if loadNative(*accepted, &i.Parent) != nil {
			return errInput
		}
		// Older S04 inspection files omitted the read-only S03 source annotation.
		// The source is freshly verified above; populated mismatches remain invalid.
		if i.Parent.Parent.SourceSHA256 == "" {
			i.Parent.Parent.SourceSHA256 = database.Digest()
		}
	} else if loadNative(*input, &i) != nil {
		return errInput
	}
	if _, e = i.Parent.Prepare(m, a); e != nil {
		return e
	}
	n, e := provision.NewNative(nil)
	if e != nil {
		return e
	}
	native := provision.VICIdialNative{Asterisk: provision.AsteriskNative{Database: provision.DatabaseNative{Native: n, Budget: i.Parent.Parent.Parent.Budget, Source: database}, Source: a}, Accepted: i.Parent.Snapshot, DatabaseAccepted: i.Parent.Parent.Snapshot, Source: s, Dependency: p}
	emit := func(v any) error { return json.NewEncoder(out).Encode(v) }
	if mode == "inspect" {
		i.Snapshot, e = native.InspectVICIdial(ctx)
		if e != nil {
			return e
		}
		i.SourceSHA256 = s.Digest()
		i.DependencySHA256 = p.Digest()
		return emit(i)
	}
	stage, e := i.Prepare(m, s, p, a, database)
	if e != nil {
		return e
	}
	if mode == "plan" {
		return emit(struct {
			Stage       json.RawMessage            `json:"stage"`
			StageSHA256 string                     `json:"stage_sha256"`
			Graph       json.RawMessage            `json:"graph"`
			State       provision.VICIdialSnapshot `json:"state"`
		}{stage.JSON(), stage.Digest(), stage.Graph().JSON(), i.Snapshot})
	}
	var authorization provision.VICIdialLabAuthorization
	if loadNative(*auth, &authorization) != nil {
		return errInput
	}
	redactor, _ := operatorlog.NewRedactor(nil)
	log, e := operatorlog.New(io.Discard, os.Stderr, nil, redactor)
	if e != nil {
		return e
	}
	result, e := provision.ExecuteVICIdialLab(ctx, stage, m, authorization, *state, native, native.Runner(stage), mode == "lab-resume", log)
	if ee := emit(result); ee != nil {
		return ee
	}
	return e
}

func readNativeFile(path string) (*os.File, error) {
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
func loadNative(path string, v any) error {
	f, e := readNativeFile(path)
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
