package sysutil

import (
	"golang.org/x/sys/windows"
	"os"
)

const eventLockLength = 1

func privateEventDirectory(os.FileInfo) error { return nil }
func lockEventOwner(path string) (func(), error) {
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return nil, os.ErrPermission
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, eventFileMode)
	if err != nil {
		return nil, err
	}
	var overlapped windows.Overlapped
	if err := windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, eventLockLength, 0, &overlapped); err != nil {
		f.Close()
		return nil, err
	}
	return func() { windows.UnlockFileEx(windows.Handle(f.Fd()), 0, eventLockLength, 0, &overlapped); f.Close() }, nil
}
