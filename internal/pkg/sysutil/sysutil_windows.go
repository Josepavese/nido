//go:build windows

package sysutil

// FixPermissions on Windows is a no-op as file ownership model via sudo/chown isn't applicable.
func FixPermissions(path string) error {
	return nil
}
