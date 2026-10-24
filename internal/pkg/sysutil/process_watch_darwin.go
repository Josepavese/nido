package sysutil

import (
	"context"
	"errors"

	"golang.org/x/sys/unix"
)

// WaitProcessExit observes one native process identity through EVFILT_PROC.
func WaitProcessExit(ctx context.Context, pid int) error {
	if pid <= 0 {
		return errInvalidProcessIdentity
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	fd, err := unix.Kqueue()
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	unix.CloseOnExec(fd)
	change := unix.Kevent_t{Ident: uint64(pid), Filter: unix.EVFILT_PROC, Flags: unix.EV_ADD | unix.EV_ONESHOT, Fflags: unix.NOTE_EXIT}
	if _, err := unix.Kevent(fd, []unix.Kevent_t{change}, nil, nil); err != nil {
		if errors.Is(err, unix.ESRCH) {
			return nil
		}
		return err
	}
	result := make([]unix.Kevent_t, 1)
	for ctx.Err() == nil {
		timeout := unix.NsecToTimespec(processWaitCancelInterval.Nanoseconds())
		n, err := unix.Kevent(fd, nil, result, &timeout)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if err != nil {
			return err
		}
		if n > 0 {
			if result[0].Flags&unix.EV_ERROR != 0 {
				return unix.Errno(result[0].Data)
			}
			return nil
		}
	}
	return ctx.Err()
}
