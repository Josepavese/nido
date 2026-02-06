//go:build !linux

package sysutil

// IsHeadless checks if the system appears to be running without a physical display attached.
// On non-Linux systems, we currently default to false or rely on basic logic as passthrough is not supported.
func IsHeadless() bool {
	// For Windows/Darwin, we don't support passthrough, so this check determines
	// nothing critical for now. We return false to be conservative.
	return false
}
