//go:build linux

package provision

import (
	"context"
	_ "embed"
	"encoding/json"
	"strconv"
	"time"

	"github.com/grimange/almadial/internal/execution"
)

//go:embed vicidial_files.py
var vicidialFilesAdapter string

type vicidialInputRunner interface {
	RunInput(context.Context, execution.Command, []byte) (execution.Result, error)
}

// ReadVICIdialFiles performs read-only native inspection through the same
// bounded runner as the other stages. It neither creates a staging area nor
// installs dependencies; source verification must already have succeeded.
func ReadVICIdialFiles(ctx context.Context, source VICIdialSource) (VICIdialFilesAssessment, error) {
	runner, e := execution.NewRunner(nil)
	if e != nil {
		return VICIdialFilesAssessment{}, e
	}
	return readVICIdialFiles(ctx, source, runner)
}

func vicidialFilesCommand(mode string) execution.Command {
	return execution.Command{Executable: "/usr/bin/python3", Args: []string{"-B", "-c", vicidialFilesAdapter, mode}, Timeout: 5 * time.Minute, StdoutLimit: 4 << 20, StderrLimit: 64 << 10}
}

func (s VICIdialSource) retainedArchiveVerified() bool {
	// Only LoadVICIdialSource can supply a verified alternative representation.
	// Keep the historical archive valid without changing its source identity.
	digest := s.archiveSHA
	if digest == "" {
		digest = databaseArchiveSHA
	}
	return shaPattern.MatchString(digest) && rawSHA(s.archive) == digest
}

func (s VICIdialSource) archiveInstallCommand() execution.Command {
	c := vicidialFilesCommand("install")
	if s.archiveSHA != "" && s.archiveSHA != databaseArchiveSHA && s.retainedArchiveVerified() {
		// Execute the unchanged, policy-bound adapter with the representation
		// verified by acquisition. Its ownership, archive hash, per-file source
		// hashes and derived-content checks all remain mandatory. The canonical
		// tree identity, historical policy and saved journal identities stay exact.
		c.Args[2] = "ns={'__name__':'almadial_verified_archive'}\nexec(" + strconv.Quote(vicidialFilesAdapter) + ",ns)\nns['ARCHIVE_SHA']=" + strconv.Quote(s.archiveSHA) + "\nraise SystemExit(ns['main']())\n"
	}
	return c
}

func readVICIdialFiles(ctx context.Context, source VICIdialSource, runner vicidialInputRunner) (VICIdialFilesAssessment, error) {
	f, e := source.Deployment()
	if e != nil || runner == nil || ctx == nil {
		return VICIdialFilesAssessment{}, errVICIdial
	}
	input, e := json.Marshal(struct{ Files []VICIdialFile }{f})
	if e != nil {
		return VICIdialFilesAssessment{}, errVICIdial
	}
	r, e := runner.RunInput(ctx, vicidialFilesCommand("inspect"), input)
	if e != nil || !r.Started || r.ExitCode != 0 || r.StdoutTruncated || r.StderrTruncated {
		return VICIdialFilesAssessment{}, errVICIdial
	}
	var observed []VICIdialFileObservation
	if decode(r.Stdout, &observed) != nil {
		return VICIdialFilesAssessment{}, errVICIdial
	}
	return AssessVICIdialFiles(f, observed)
}
