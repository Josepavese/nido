package sysutil

import "golang.org/x/sys/unix"

func publishFile(source, destination string) error {
	return unix.RenamexNp(source, destination, unix.RENAME_EXCL)
}
