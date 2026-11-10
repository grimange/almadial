//go:build linux

package provision

import (
	"context"
	_ "embed"
	"encoding/json"
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
