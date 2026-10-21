//go:build !windows

package sysutil

import (
	"fmt"
	"golang.org/x/sys/unix"
	"os"
)

// DeviceValue performs a no-payload device control query. The HAL supplies the
// request code and interprets the value; no native descriptor escapes the PAL.
func (NativeHostIO) DeviceValue(path string, request uintptr) (uintptr, error) {
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	value, _, errno := unix.Syscall(unix.SYS_IOCTL, file.Fd(), request, 0)
	if errno != 0 {
		return 0, fmt.Errorf("query device value: %w", errno)
	}
	return value, nil
}
