// Package buildinfo exposes controlled link-time metadata without host inspection.
package buildinfo

import (
	"fmt"
	"runtime"
)

// Version and Commit may be supplied with go build -ldflags -X.
var Version = "dev"
var Commit = "unknown"

func String() string {
	return fmt.Sprintf("almadial %s\ncommit: %s\ntarget: %s/%s\ngo: %s\n", Version, Commit, runtime.GOOS, runtime.GOARCH, runtime.Version())
}
