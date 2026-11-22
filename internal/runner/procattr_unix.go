//go:build unix

package runner

import (
	"os/exec"
	"syscall"
)

// applyProcAttr sets Unix process-group membership so timeout/cancel can kill the tree.
func applyProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
