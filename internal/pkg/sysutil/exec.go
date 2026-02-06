package sysutil

import (
	"fmt"
	"os"
	"os/exec"
)

// ExecPrivileged executes a shell command with root privileges.
// If already root, runs directly. If not, uses sudo.
// This is a Logic-Layer-Agnostic way to request elevated operations.
func ExecPrivileged(cmdStr string) error {
	var cmd *exec.Cmd
	if os.Getuid() == 0 { // Getuid is POSIX, might need conditional for Windows?
		// Windows doesn't really have sudo in the same way, but this helper is primarily for Linux ops
		// For cross-platform safety, we should stick to standard 'sudo' usage or 'runas' on Windows?
		// Actually, standard Go os.Getuid() is supported on Linux/Darwin. Windows?
		// On Windows os.Getuid() returns -1.
		cmd = exec.Command("sh", "-c", cmdStr)
	} else {
		cmd = exec.Command("sudo", "sh", "-c", cmdStr)
	}

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("privileged command failed: %s (%w)", string(out), err)
	}
	return nil
}
