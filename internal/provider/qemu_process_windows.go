//go:build windows

package provider

import (
	"os"
	"syscall"
)

const (
	windowsDetachedProcess        = 0x00000008
	windowsCreateBreakawayFromJob = 0x01000000
	windowsCreateNewProcessGroup  = 0x00000200
)

func detachedQemuSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: windowsDetachedProcess | windowsCreateBreakawayFromJob | windowsCreateNewProcessGroup,
	}
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	handle, err := syscall.OpenProcess(0x00100000, false, uint32(pid)) // SYNCHRONIZE
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(handle)
	status, err := syscall.WaitForSingleObject(handle, 0)
	return err == nil && status == 0x00000102 // WAIT_TIMEOUT means still running.
}

func stopQemuProcess(process *os.Process, graceful bool) error {
	return process.Kill()
}
