// +build !windows

package supervisor

import (
	"os/exec"
	"syscall"
)

// setSysProcAttr sets Unix-specific process attributes
func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}
