//go:build unix

package procinfo

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// StartKey returns a stable process-start key for pid (Linux/Unix /proc starttime).
func StartKey(pid int) (string, error) {
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return "", err
	}
	return parseStart(string(b))
}

// Alive reports whether pid exists and returns its current start key.
func Alive(pid int) (bool, string, error) {
	key, err := StartKey(pid)
	if err != nil {
		if os.IsNotExist(err) {
			return false, "", nil
		}
		// ESRCH / gone
		if _, statErr := os.Stat(fmt.Sprintf("/proc/%d", pid)); statErr != nil {
			return false, "", nil
		}
		return false, "", err
	}
	return true, key, nil
}

func parseStart(stat string) (string, error) {
	// format: pid (comm) state ... starttime is field 22 (1-based) after comm
	idx := strings.LastIndex(stat, ")")
	if idx < 0 || idx+2 >= len(stat) {
		return "", fmt.Errorf("proc stat: bad format")
	}
	fields := strings.Fields(stat[idx+2:])
	// fields[0] is state (field 3); starttime is field 22 → index 19 in this slice
	if len(fields) < 20 {
		return "", fmt.Errorf("proc stat: too few fields")
	}
	if _, err := strconv.ParseUint(fields[19], 10, 64); err != nil {
		return "", fmt.Errorf("proc stat starttime: %w", err)
	}
	return fields[19], nil
}
