package sysutil

import (
	"context"
	"errors"

	"golang.org/x/sys/unix"
)

// WaitProcessExit holds a pidfd identity, so PID reuse cannot change its target.
func WaitProcessExit(ctx context.Context, pid int) error {
	if pid <= 0 {
		return errInvalidProcessIdentity
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	fd, err := unix.PidfdOpen(pid, 0)
	if errors.Is(err, unix.ESRCH) {
		return nil
	}
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	poll := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
	for ctx.Err() == nil {
		_, err := unix.Poll(poll, int(processWaitCancelInterval.Milliseconds()))
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if err != nil {
			return err
		}
		if poll[0].Revents&unix.POLLIN != 0 {
			return nil
		}
	}
	return ctx.Err()
}
