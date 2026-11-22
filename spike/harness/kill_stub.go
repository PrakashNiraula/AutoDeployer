//go:build !windows

package main

import (
	"fmt"
	"os"
	"syscall"
)

func killTreeTaskkill(pid int) error {
	// Unix process-group kill approximation for local harness experiments.
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if err := p.Signal(syscall.SIGKILL); err != nil {
		return fmt.Errorf("kill: %w", err)
	}
	return nil
}
