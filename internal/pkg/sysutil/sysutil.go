package sysutil

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// DefaultMemory calculated as min(2048MB, 50% system RAM).
func DefaultMemory() int {
	return calculateDefaultMemory()
}

// DefaultVCPUs returns the default number of vCPUs.
func DefaultVCPUs() int {
	return 1
}

// CopyFile performs a Go-native file copy, replacing the need for 'cp'.
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}

// TempDir returns a cross-platform safe temporary directory for Nido.
func TempDir() string {
	dir := filepath.Join(os.TempDir(), "nido")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

// EnsureDir makes sure a directory exists and returns its absolute path.
func EnsureDir(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(abs, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", abs, err)
	}
	return abs, nil
}
