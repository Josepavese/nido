//go:build !windows

package provider

import "syscall"
import "os"

func detachedQemuSysProcAttr() *syscall.SysProcAttr {
	return nil
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	return err == nil && process.Signal(syscall.Signal(0)) == nil
}

func stopQemuProcess(process *os.Process, graceful bool) error {
	return process.Signal(os.Interrupt)
}
