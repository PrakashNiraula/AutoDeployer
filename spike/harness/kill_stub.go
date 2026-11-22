//go:build !windows

package main

import (
	"fmt"
	"os"
	"syscall"
)

func killTreeTaskkill(pid int) error {
	// Prefer process-group kill (matches production killtree on Unix).
	if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
		p, err2 := os.FindProcess(pid)
		if err2 != nil {
			return fmt.Errorf("kill pgid/pid: %v / %v", err, err2)
		}
		if err3 := p.Signal(syscall.SIGKILL); err3 != nil {
			return fmt.Errorf("kill: %w", err3)
		}
	}
	return nil
}
