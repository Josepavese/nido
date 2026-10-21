package sysutil

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// WriteFile replaces metadata atomically on its own filesystem. Existing mode,
// writable-file semantics and explicit file symlinks are preserved; errors leave
// the previous bytes intact, including short writes or storage exhaustion.
func WriteFile(path string, data []byte, mode os.FileMode) error {
	return writeFileAtomically(path, mode, func(writer io.Writer) error {
		n, err := writer.Write(data)
		if err == nil && n != len(data) {
			return io.ErrShortWrite
		}
		return err
	})
}
func writeFileAtomically(path string, mode os.FileMode, write func(io.Writer) error) error {
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			resolved, err := filepath.EvalSymlinks(path)
			if err != nil {
				return err
			}
			path = resolved
		}
		info, err = os.Stat(path)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("metadata target is not a regular file")
		}
		check, err := os.OpenFile(path, os.O_WRONLY, 0)
		if err != nil {
			return err
		}
		if err := check.Close(); err != nil {
			return err
		}
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return err
	}
	stage, err := os.CreateTemp(filepath.Dir(path), ".nido-metadata-*")
	if err != nil {
		return err
	}
	defer os.Remove(stage.Name())
	defer stage.Close()
	if err := write(stage); err != nil {
		return err
	}
	if err := stage.Sync(); err != nil {
		return err
	}
	if err := stage.Chmod(mode); err != nil {
		return err
	}
	if err := stage.Close(); err != nil {
		return err
	}
	if err := FixPermissions(stage.Name()); err != nil {
		return err
	}
	return os.Rename(stage.Name(), path)
}
