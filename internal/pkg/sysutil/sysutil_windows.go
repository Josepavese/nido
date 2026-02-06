//go:build windows

package sysutil

// GetTargetUIDGID on Windows returns 0,0 (root/admin effectively) or similar,
// but since ownership isn't mapped the same way, we just return nil error and dummy values.
// We could return -1, -1 but 0, 0 is safer default if used blindly.
func GetTargetUIDGID() (int, int, error) {
	return 0, 0, nil
}

// FixPermissions on Windows is a no-op as file ownership model via sudo/chown isn't applicable.
func FixPermissions(path string) error {
	return nil
}
