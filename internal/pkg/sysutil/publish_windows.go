package sysutil

import "golang.org/x/sys/windows"

func publishFile(source, destination string) error {
	from, err := windows.UTF16PtrFromString(nativeWindowsPath(source))
	if err != nil {
		return err
	}
	to, err := windows.UTF16PtrFromString(nativeWindowsPath(destination))
	if err != nil {
		return err
	}
	return windows.MoveFileEx(from, to, windows.MOVEFILE_WRITE_THROUGH)
}
