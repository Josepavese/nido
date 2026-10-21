//go:build !windows

package sysutil

import (
	"golang.org/x/sys/unix"
	"os"
)

func downloadFileLinks(file *os.File) (uint64, error) {
	var stat unix.Stat_t
	err := unix.Fstat(int(file.Fd()), &stat)
	return uint64(stat.Nlink), err
}
