package sysutil

import (
	"context"
	"errors"

	"golang.org/x/sys/windows"
)

// WaitProcessExit retains a SYNCHRONIZE handle until exit or cancellation.
func WaitProcessExit(ctx context.Context, pid int) error {
	if pid <= 0 {
		return errInvalidProcessIdentity
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	handle, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
		return nil
	}
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)
	for ctx.Err() == nil {
		result, err := windows.WaitForSingleObject(handle, uint32(processWaitCancelInterval.Milliseconds()))
		if err != nil {
			return err
		}
		if result == windows.WAIT_OBJECT_0 {
			return nil
		}
	}
	return ctx.Err()
}
