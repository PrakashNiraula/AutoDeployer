//go:build windows

package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"syscall"
	"unsafe"
)

// killTreeTaskkill uses taskkill /T /F (fallback / primary documented path).
func killTreeTaskkill(pid int) error {
	cmd := exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T", "/F")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("taskkill: %w (%s)", err, bytesTrim(out))
	}
	return nil
}

func bytesTrim(b []byte) string {
	return string(bytesTrimSpace(b))
}

func bytesTrimSpace(b []byte) []byte {
	for len(b) > 0 && (b[0] == ' ' || b[0] == '\n' || b[0] == '\r' || b[0] == '\t') {
		b = b[1:]
	}
	for len(b) > 0 {
		c := b[len(b)-1]
		if c == ' ' || c == '\n' || c == '\r' || c == '\t' {
			b = b[:len(b)-1]
			continue
		}
		break
	}
	return b
}

// Windows Job Object helpers (spike). We create a job WITHOUT KILL_ON_JOB_CLOSE
// so detach experiments can leave children alive when the parent handle closes.
var (
	modKernel32                  = syscall.NewLazyDLL("kernel32.dll")
	procCreateJobObjectW         = modKernel32.NewProc("CreateJobObjectW")
	procSetInformationJobObject  = modKernel32.NewProc("SetInformationJobObject")
	procAssignProcessToJobObject = modKernel32.NewProc("AssignProcessToJobObject")
	procTerminateJobObject       = modKernel32.NewProc("TerminateJobObject")
	procCloseHandle              = modKernel32.NewProc("CloseHandle")
)

const (
	JobObjectExtendedLimitInformation = 9
	JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE = 0x00002000
)

type JOBOBJECT_BASIC_LIMIT_INFORMATION struct {
	PerProcessUserTimeLimit int64
	PerJobUserTimeLimit     int64
	LimitFlags              uint32
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	Affinity                uintptr
	PriorityClass           uint32
	SchedulingClass         uint32
}

type IO_COUNTERS struct {
	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
}

type JOBOBJECT_EXTENDED_LIMIT_INFORMATION struct {
	BasicLimitInformation JOBOBJECT_BASIC_LIMIT_INFORMATION
	IoInfo                IO_COUNTERS
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}

func createJobObject(killOnClose bool) (syscall.Handle, error) {
	h, _, err := procCreateJobObjectW.Call(0, 0)
	if h == 0 {
		return 0, err
	}
	handle := syscall.Handle(h)
	var info JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	if killOnClose {
		info.BasicLimitInformation.LimitFlags = JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	} else {
		info.BasicLimitInformation.LimitFlags = 0
	}
	r, _, err2 := procSetInformationJobObject.Call(
		uintptr(handle),
		JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		unsafe.Sizeof(info),
	)
	if r == 0 {
		procCloseHandle.Call(uintptr(handle))
		return 0, err2
	}
	return handle, nil
}

func assignToJob(job syscall.Handle, pid uint32) error {
	h, err := syscall.OpenProcess(syscall.PROCESS_ALL_ACCESS, false, pid)
	if err != nil {
		return err
	}
	defer procCloseHandle.Call(uintptr(h))
	r, _, err2 := procAssignProcessToJobObject.Call(uintptr(job), uintptr(h))
	if r == 0 {
		return err2
	}
	return nil
}

func terminateJob(job syscall.Handle) error {
	r, _, err := procTerminateJobObject.Call(uintptr(job), 1)
	if r == 0 {
		return err
	}
	return nil
}

func closeHandle(h syscall.Handle) {
	procCloseHandle.Call(uintptr(h))
}
