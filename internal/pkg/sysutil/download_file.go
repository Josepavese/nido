package sysutil

import (
	"fmt"
	"io"
	"os"
)

// OpenDownloadFile refuses symlinks, hard-linked aliases and changed identities
// before it truncates or writes. The descriptor remains owned behind this port.
func OpenDownloadFile(path string, resume bool, expectedSize int64) (io.WriteCloser, error) {
	flags := os.O_CREATE | os.O_WRONLY
	if resume {
		flags |= os.O_APPEND
	}
	file, err := os.OpenFile(path, flags, 0600)
	if err != nil {
		return nil, err
	}
	valid := false
	defer func() {
		if !valid {
			file.Close()
		}
	}()
	opened, err := file.Stat()
	if err != nil {
		return nil, err
	}
	named, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !opened.Mode().IsRegular() || !named.Mode().IsRegular() || !os.SameFile(opened, named) {
		return nil, fmt.Errorf("partial download is not a private regular file")
	}
	links, err := downloadFileLinks(file)
	if err != nil {
		return nil, err
	}
	if links != 1 {
		return nil, fmt.Errorf("partial download has shared hard links")
	}
	if resume && opened.Size() != expectedSize {
		return nil, fmt.Errorf("partial download changed during resume")
	}
	if !resume {
		if err := file.Truncate(0); err != nil {
			return nil, err
		}
	}
	valid = true
	return file, nil
}
