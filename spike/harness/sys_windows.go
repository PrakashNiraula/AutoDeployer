//go:build windows

package main

import "syscall"

func sysSetpgid() *syscall.SysProcAttr {
	return nil
}
