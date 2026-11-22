//go:build unix

package main

import "syscall"

func sysSetpgid() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}
