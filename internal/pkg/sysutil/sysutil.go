package sysutil

import (
	"fmt"
	"github.com/Josepavese/nido/v4/internal/machine"
	"io"
	"os"
	"os/exec"
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

// ProvisionFile wraps a file creation operation (like an external command)
// and ensures the resulting file has the correct ownership.
func ProvisionFile(path string, generator func() error) error {
	if err := generator(); err != nil {
		return err
	}
	return FixPermissions(path)
}

// FindExecutable resolves an executable from PATH, with Windows fallbacks for
// installers that place binaries in a standard directory without updating PATH.
func FindExecutable(names ...string) (string, error) {
	var lastErr error
	for _, name := range names {
		path, err := exec.LookPath(name)
		if err == nil {
			return path, nil
		}
		lastErr = err
	}

	if path, err := fallbackExecutable(names); err == nil {
		return path, nil
	}

	if lastErr != nil {
		return "", lastErr
	}
	return "", exec.ErrNotFound
}

// QemuSystemBinary resolves the QEMU system emulator used by Nido.
func QemuSystemBinary(arch string) (string, error) {
	profile, err := machine.For(arch)
	if err != nil {
		return "", err
	}
	return FindExecutable(profile.Binary)
}

// QemuImgBinary resolves qemu-img.
func QemuImgBinary() (string, error) {
	return FindExecutable("qemu-img")
}

// MakeExecutable applies native executable permissions to an installed artifact.
func MakeExecutable(path string) error { return os.Chmod(path, 0755) }

// NidoHome owns the application state root. An explicit root isolates test and
// container instances without changing the operating-system user's home.
func NidoHome() string {
	if root := os.Getenv("NIDO_HOME"); root != "" {
		if abs, err := filepath.Abs(root); err == nil {
			return abs
		}
	}
	home, _ := UserHome()
	return filepath.Join(home, ".nido")
}
