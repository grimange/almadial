package execution

import (
	"os"
	"os/exec"
	"syscall"
	"time"
)

// A private process group contains ordinary descendants. No shell or supervisor
// is started. Escaping the group with setsid is outside this trusted-code API.
func prepareProcess(c *exec.Cmd) error {
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	c.Cancel = func() error {
		err := syscall.Kill(-c.Process.Pid, syscall.SIGKILL)
		if err == syscall.ESRCH {
			return os.ErrProcessDone
		}
		return err
	}
	c.WaitDelay = 200 * time.Millisecond
	return nil
}
func cleanupProcess(c *exec.Cmd) { _ = syscall.Kill(-c.Process.Pid, syscall.SIGKILL) }
