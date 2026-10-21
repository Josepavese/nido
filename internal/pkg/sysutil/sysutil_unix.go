//go:build !windows

package sysutil

import (
	"os"
	"strconv"
)

// targetUIDGID returns the UID and GID of the SUDO_USER,
// or the current process's UID/GID if not running under sudo.
func targetUIDGID() (int, int, error) {
	uidStr := os.Getenv("SUDO_UID")
	gidStr := os.Getenv("SUDO_GID")

	if uidStr != "" && gidStr != "" {
		uid, err := strconv.Atoi(uidStr)
		if err != nil {
			return 0, 0, err
		}
		gid, err := strconv.Atoi(gidStr)
		if err != nil {
			return 0, 0, err
		}
		return uid, gid, nil
	}

	return os.Getuid(), os.Getgid(), nil
}

// FixPermissions restores the ownership of a file to the original user
// if running under sudo.
func FixPermissions(path string) error {
	uid, gid, err := targetUIDGID()
	if err != nil {
		return err
	}

	return os.Chown(path, uid, gid)
}
