package sysutil

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

var ErrResourceBusy = errors.New("resource is owned by another operation")

const resourceMarker = "nido-resource-lock-v1\n"

type ResourceLease interface{ Close() error }
type resourceLease struct {
	file *os.File
	path string
}

// AcquireResource is a nonblocking, cross-process ownership reservation. Kernel
// locks are released on process exit; complete markers can be reused after crashes.
func AcquireResource(path string) (ResourceLease, error) {
	marker, err := os.CreateTemp(filepath.Dir(path), ".nido-lease-*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(marker.Name())
	defer marker.Close()
	if _, err := marker.Write([]byte(resourceMarker)); err != nil {
		return nil, err
	}
	if err := marker.Close(); err != nil {
		return nil, err
	}
	if err := PublishFile(marker.Name(), path); err != nil && !errors.Is(err, fs.ErrExist) {
		return nil, err
	}
	file, err := openResourceFile(path)
	if err != nil {
		return nil, err
	}
	valid := false
	defer func() {
		if !valid {
			file.Close()
		}
	}()
	if err := lockResourceFile(file); err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	named, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("%w: reservation was replaced", ErrResourceBusy)
	}
	links, err := downloadFileLinks(file)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || !named.Mode().IsRegular() || !os.SameFile(info, named) || links != 1 {
		return nil, fmt.Errorf("invalid resource reservation identity")
	}
	data, err := io.ReadAll(io.LimitReader(file, int64(len(resourceMarker)+1)))
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(data, []byte(resourceMarker)) {
		return nil, fmt.Errorf("unrecognized resource reservation; existing bytes preserved")
	}
	valid = true
	return &resourceLease{file: file, path: path}, nil
}
func (l *resourceLease) Close() error {
	if l.file == nil {
		return nil
	}
	info, err := l.file.Stat()
	if err == nil {
		named, statErr := os.Lstat(l.path)
		if statErr == nil && named.Mode().IsRegular() && os.SameFile(info, named) {
			err = os.Remove(l.path)
		} else if statErr != nil && !os.IsNotExist(statErr) {
			err = statErr
		}
	}
	closeErr := l.file.Close()
	l.file = nil
	return errors.Join(err, closeErr)
}
