package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/grimange/almadial/internal/artifacts"
	"github.com/grimange/almadial/internal/operatorlog"
	"github.com/grimange/almadial/internal/provision"
)

// The accepted S08 input is retained as an immutable parent. S10 observes
// current foundation state without re-executing earlier provisioning stages.
func run(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "help" {
		_, e := io.WriteString(out, usage)
		return e
	}
	mode := args[0]
	fs := flag.NewFlagSet("almadial-phone", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	input := fs.String("input", "", "")
	request := fs.String("request", "", "")
	phoneArchive := fs.String("phone-source", "", "")
	accepted := fs.String("accepted-network", "", "")
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
		if fs.NFlag() != 11 || *request == "" || *accepted == "" || *input != "" || *auth != "" || *state != "" {
			return errInput
		}
	case "plan":
		if fs.NFlag() != 10 || *input == "" || *accepted != "" || *request != "" || *phoneArchive == "" || *auth != "" || *state != "" {
			return errInput
		}
	case "lab-run", "lab-resume":
		if fs.NFlag() != 12 || *input == "" || *accepted != "" || *auth == "" || *state == "" || *request != "" || *phoneArchive == "" {
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
	phone, e := provision.LoadPhoneSource(ctx, m, *phoneArchive)
	if e != nil {
		return e
	}
	var i provision.PhoneInputs
	if mode == "inspect" {
		if loadNative(*accepted, &i.Parent) != nil {
			return errInput
		}
		if loadNative(*request, &i.Request) != nil {
			return errInput
		}

	} else if loadNative(*input, &i) != nil {
		return errInput
	}
	if _, e = i.Parent.Parent.Parent.Parent.Parent.Parent.Prepare(m, a); e != nil {
		return e
	}
	n, e := provision.NewNative(nil)
	if e != nil {
		return e
	}
	application := provision.VICIdialNative{Asterisk: provision.AsteriskNative{Database: provision.DatabaseNative{Native: n, Budget: i.Parent.Parent.Parent.Parent.Parent.Parent.Parent.Parent.Budget, Source: database}, Source: a}, Accepted: i.Parent.Parent.Parent.Parent.Parent.Parent.Snapshot, DatabaseAccepted: i.Parent.Parent.Parent.Parent.Parent.Parent.Parent.Snapshot, Source: s, Dependency: p}
	native := provision.PhoneNative{Network: provision.NetworkNative{TLS: provision.TLSNative{Web: provision.WebNative{Runtime: provision.RuntimeNative{Application: application}}, Request: i.Parent.Parent.Request, Certificate: i.Parent.Parent.Certificate}, Request: i.Parent.Request, Dependencies: i.Parent.Dependencies}, Request: i.Request, Source: phone}
	emit := func(v any) error { return json.NewEncoder(out).Encode(v) }
	if mode == "inspect" {
		i.PhoneSourceSHA256 = phone.Digest()
		i.Snapshot, e = native.InspectPhone(ctx)
		if e != nil {
			return e
		}
		return emit(i)
	}
	stage, e := i.Prepare(m, s, p, a, database, phone)
	if e != nil {
		return e
	}
	if mode == "plan" {
		return emit(struct {
			Stage       json.RawMessage         `json:"stage"`
			StageSHA256 string                  `json:"stage_sha256"`
			Graph       json.RawMessage         `json:"graph"`
			State       provision.PhoneSnapshot `json:"state"`
		}{stage.JSON(), stage.Digest(), stage.Graph().JSON(), i.Snapshot})
	}
	var authorization provision.PhoneLabAuthorization
	if loadNative(*auth, &authorization) != nil {
		return errInput
	}
	redactor, _ := operatorlog.NewRedactor(nil)
	log, e := operatorlog.New(io.Discard, os.Stderr, nil, redactor)
	if e != nil {
		return e
	}
	result, e := provision.ExecutePhoneLab(ctx, stage, m, authorization, *state, native, native.Runner(stage), mode == "lab-resume", log)
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

const usage = `Usage: almadial-phone OPERATION --manifest FILE --source ZIP --perl-source FILE --asterisk-source FILE --signature FILE --keyring FILE --scratch DIR --pjproject FILE --phone-source TGZ
  inspect --accepted-network FILE --request FILE: fresh read-only S10 and accepted S01-S09 checks
  plan --input FILE: pure immutable plan; zero host mutations
  lab-run / lab-resume --input FILE --authorization FILE --state-dir DIR
Only explicitly authorized lab-run/resume can deploy the pinned self-hosted phone,
one protected synthetic identity and a registration-only local WSS PJSIP include.
No calls, media, firewall changes or public bindings are authorized.
`

var errInput = errors.New("invalid S10 request; review operation and pinned input flags")

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if e := run(ctx, os.Args[1:], os.Stdout); e != nil {
		_, _ = io.WriteString(os.Stderr, e.Error()+"\n")
		os.Exit(1)
	}
}
