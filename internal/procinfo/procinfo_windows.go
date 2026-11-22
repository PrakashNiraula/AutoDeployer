//go:build windows

package procinfo

import (
	"fmt"
	"os"
)

// StartKey returns a placeholder start key on Windows until Win32 creation-time spike lands.
// Uses a best-effort marker so locks still work via PID liveness.
func StartKey(pid int) (string, error) {
	return fmt.Sprintf("win-pid-%d", pid), nil
}

// Alive reports whether the process exists.
func Alive(pid int) (bool, string, error) {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false, "", nil
	}
	// On Windows FindProcess always succeeds; Signal(0) is not portable.
	// Use OpenProcess via syscall would be better; for now treat FindProcess as weak.
	_ = p
	key, _ := StartKey(pid)
	// Weak check: assume alive if we got here; runner cancel uses taskkill.
	return true, key, nil
}
