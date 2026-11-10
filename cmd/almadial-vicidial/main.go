// almadial-vicidial verifies S05 source and file state without mutations.
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
	"github.com/grimange/almadial/internal/provision"
)

const usage = `Usage: almadial-vicidial OPERATION --manifest FILE --source ZIP
  source   Verify pinned r4037 archive and canonical inventory; emit file manifest
  files    Verify source, then inspect installed files and metadata read-only
  prerequisites  Verify source, then inspect approved Perl/PHP dependencies read-only
  perl-source  Verify pinned Net::Telnet 3.05 tarball passed as --source
  phone-fixture  Execute isolated upstream PJSIP generation cases; no database or endpoint

The source/files/prerequisites/perl-source/phone-fixture commands never install files.
  inspect --accepted-asterisk FILE: read-only full S05 inspection
  plan --input FILE: pure immutable S05 graph
  lab-run / lab-resume --input FILE --authorization FILE --state-dir DIR
Native operations additionally require --manifest, --source, --perl-source,
--asterisk-source, --signature, --keyring, --scratch and --pjproject.
Only lab-run/resume can mutate an explicitly authorized disposable VM.
No command authorizes production.
Application, database, generator and runtime verification remain separate gates.
Protect inspection output as local laboratory evidence.
`

var errInput = errors.New("invalid S05 verification request; review operation, manifest and source")

func run(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "help" {
		_, e := io.WriteString(out, usage)
		return e
	}
	if args[0] == "inspect" || args[0] == "plan" || args[0] == "lab-run" || args[0] == "lab-resume" {
		return runNative(ctx, args, out)
	}
	if args[0] != "source" && args[0] != "files" && args[0] != "perl-source" && args[0] != "prerequisites" && args[0] != "phone-fixture" {
		return errInput
	}
	f := flag.NewFlagSet("almadial-vicidial", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	mf := f.String("manifest", "", "")
	archive := f.String("source", "", "")
	if f.Parse(args[1:]) != nil || f.NArg() != 0 || f.NFlag() != 2 || *mf == "" || *archive == "" {
		return errInput
	}
	in, e := os.OpenFile(*mf, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if e != nil {
		return errInput
	}
	defer in.Close()
	stat, e := in.Stat()
	if e != nil || !stat.Mode().IsRegular() || stat.Size() > 4<<20 {
		return errInput
	}
	m, e := artifacts.Load(in)
	if e != nil {
		return errInput
	}
	if args[0] == "perl-source" {
		dependency, e := provision.LoadVICIdialPerlSource(ctx, m, *archive)
		if e != nil {
			return e
		}
		return json.NewEncoder(out).Encode(struct{ Component, Version, Identity string }{"Net::Telnet", "3.05", dependency.Digest()})
	}
	s, e := provision.LoadVICIdialSource(ctx, m, *archive)
	if e != nil {
		return e
	}
	if args[0] == "phone-fixture" {
		result, e := provision.QualifyVICIdialPhoneFixture(ctx, s)
		if e != nil {
			return e
		}
		return json.NewEncoder(out).Encode(result)
	}
	if args[0] == "prerequisites" {
		p, e := provision.ReadVICIdialPrerequisites(ctx)
		if e != nil {
			return e
		}
		return json.NewEncoder(out).Encode(p)
	}
	if args[0] == "files" {
		a, e := provision.ReadVICIdialFiles(ctx, s)
		if e != nil {
			return e
		}
		return json.NewEncoder(out).Encode(a)
	}
	d, e := s.Deployment()
	if e != nil {
		return e
	}
	_, generator, e := s.Generator()
	if e != nil {
		return e
	}
	return json.NewEncoder(out).Encode(struct {
		Revision, Identity string
		Inventory          []provision.VICIdialSourceFile
		Deployment         []provision.VICIdialFile
		Generator          provision.VICIdialGenerator
	}{provision.VICIdialRevision, s.Digest(), s.Inventory(), d, generator})
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if e := run(ctx, os.Args[1:], os.Stdout); e != nil {
		// Never export input paths, file contents or dependency diagnostics.
		var check *provision.VICIdialCheckError
		if errors.As(e, &check) {
			_, _ = io.WriteString(os.Stderr, check.Error()+"\n")
		} else {
			_, _ = io.WriteString(os.Stderr, "S05 verification failed; preserve existing state and review inputs\n")
		}
		os.Exit(1)
	}
}
