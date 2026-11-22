//go:build unix

package killtree

import (
	"fmt"
	"syscall"
)

// KillProcessGroup sends SIGKILL to the process group (negative pgid).
func KillProcessGroup(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid pid %d", pid)
	}
	// Prefer process group; fall back to the single process.
	if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
		if err2 := syscall.Kill(pid, syscall.SIGKILL); err2 != nil {
			return fmt.Errorf("kill pgid/pid %d: %v / %v", pid, err, err2)
		}
	}
	return nil
}
