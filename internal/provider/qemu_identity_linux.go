//go:build linux

package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

func inspectVMProcess(pid int, identity vmProcessIdentity) error {
	root := filepath.Join("/proc", strconv.Itoa(pid))
	exe, err := os.Readlink(filepath.Join(root, "exe"))
	if err != nil {
		return err
	}
	cmd, err := os.ReadFile(filepath.Join(root, "cmdline"))
	if err != nil {
		return err
	}
	if !qemuExecutable(exe) || !matchesVMCommand(strings.Split(string(cmd), "\x00"), identity) {
		return fmt.Errorf("PID %d does not belong to VM %q", pid, identity.name)
	}
	return nil
}

func stopManagedProcess(pid int, identity vmProcessIdentity, _ bool) error {
	// Open before inspection. The kernel handle pins this process even if it
	// exits and its numeric PID is reused before a signal is sent.
	fd, err := unix.PidfdOpen(pid, 0)
	if err == unix.ESRCH {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open VM process handle: %w", err)
	}
	defer unix.Close(fd)
	if err := inspectVMProcess(pid, identity); err != nil {
		return err
	}
	if err := unix.PidfdSendSignal(fd, unix.SIGINT, nil, 0); err != nil && err != unix.ESRCH {
		return err
	}
	poll := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
	_, err = unix.Poll(poll, int(vmStopTimeout.Milliseconds()))
	if err != nil {
		return err
	}
	if poll[0].Revents&unix.POLLIN == 0 {
		if err := unix.PidfdSendSignal(fd, unix.SIGKILL, nil, 0); err != nil && err != unix.ESRCH {
			return err
		}
		if _, err := unix.Poll(poll, int(vmStopTimeout.Milliseconds())); err != nil {
			return err
		}
		if poll[0].Revents&unix.POLLIN == 0 {
			return fmt.Errorf("VM %q did not exit", identity.name)
		}
	}
	return nil
}
