//go:build !linux

package sysutil

// PreparePassthrough is a no-op on non-Linux systems.
func PreparePassthrough(pciID string, stateDir string) error {
	return nil
}

// RestorePassthrough is a no-op on non-Linux systems.
func RestorePassthrough(pciID string, stateDir string) error {
	return nil
}
