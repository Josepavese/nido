//go:build !linux && !darwin && !windows

package provider

func qemuInstallHints() (string, string) { return "(Install QEMU)", "(Included in QEMU)" }
