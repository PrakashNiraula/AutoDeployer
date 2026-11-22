//go:build unix

package runner

import (
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/PrakashNiraula/autodeployer/internal/killtree"
)

func TestKillProcessGroup(t *testing.T) {
	cmd := exec.Command("sleep", "30")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	time.Sleep(50 * time.Millisecond)
	if err := killtree.KillProcessGroup(pid); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
}
