package sysutil

import (
	"fmt"
	"io"
	"os"
	"os/user"
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

// DefaultDiskSize is the default size for new VM root disks (20GB).
const DefaultDiskSize = "20G"

// DefaultDiskBytes returns the default disk size in bytes (20GB).
const DefaultDiskBytes = 20 * 1024 * 1024 * 1024

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

	if err := os.Chmod(dst, sourceInfo.Mode()); err != nil {
		return err
	}
	return FixPermissions(dst)
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
	if err := FixPermissions(abs); err != nil {
		return "", fmt.Errorf("failed to set permissions on %s: %w", abs, err)
	}
	return abs, nil
}

// UserHome returns the current user's home directory.
// If running under sudo, it attempts to return the original user's home directory.
func UserHome() (string, error) {
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
		u, err := user.Lookup(sudoUser)
		if err == nil {
			return u.HomeDir, nil
		}
	}
	return os.UserHomeDir()
}

// FixPermissions and GetTargetUIDGID are implemented in platform-specific files.

// WriteFile writes data to a file named by filename and enforces correct ownership.
// If the file does not exist, WriteFile creates it with permissions perm.
func WriteFile(filename string, data []byte, perm os.FileMode) error {
	if err := os.WriteFile(filename, data, perm); err != nil {
		return err
	}
	return FixPermissions(filename)
}

// ProvisionFile wraps a file creation operation (like an external command)
// and ensures the resulting file has the correct ownership.
func ProvisionFile(path string, generator func() error) error {
	if err := generator(); err != nil {
		return err
	}
	return FixPermissions(path)
}
