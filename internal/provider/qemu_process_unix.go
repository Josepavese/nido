//go:build !windows

package provider

import (
	"errors"
	"os"
	"syscall"
)

func detachedQemuSysProcAttr() *syscall.SysProcAttr {
	return nil
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil || errors.Is(err, syscall.EPERM)
}
