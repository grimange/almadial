//go:build !linux

package execution

import "os/exec"

func prepareProcess(*exec.Cmd) error { return failure(Contract) }
func cleanupProcess(*exec.Cmd)       {}
