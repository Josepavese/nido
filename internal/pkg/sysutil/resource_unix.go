//go:build !windows

package sysutil

import (
	"errors"
	"golang.org/x/sys/unix"
	"os"
)

func openResourceFile(path string) (*os.File, error) { return os.OpenFile(path, os.O_RDWR, 0600) }
func lockResourceFile(file *os.File) error {
	err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
		return ErrResourceBusy
	}
	return err
}
