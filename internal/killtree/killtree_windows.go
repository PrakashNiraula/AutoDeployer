//go:build windows

package killtree

import (
	"fmt"
	"os/exec"
	"strconv"
)

// KillProcessGroup uses taskkill /T /F (B.3 primary).
func KillProcessGroup(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid pid %d", pid)
	}
	cmd := exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T", "/F")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("taskkill: %w: %s", err, string(out))
	}
	return nil
}
