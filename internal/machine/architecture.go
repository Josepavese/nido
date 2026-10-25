// Package machine owns guest instruction sets and their QEMU platform profiles.
// It contains no host I/O; platform discovery belongs to sysutil and hardware.
package machine

import (
	"fmt"
	"strings"
)

const (
	Auto           = "auto"
	AMD64          = "amd64"
	ARM64          = "arm64"
	AcceleratorTCG = "tcg"
	CPUHost        = "host"
	FirmwareARM64  = "edk2-aarch64-code.fd"
)

type Profile struct {
	Architecture string
	Binary       string
	Machine      string
	SoftwareCPU  string
	Console      string
}

var profiles = map[string]Profile{
	AMD64: {AMD64, "qemu-system-x86_64", "pc", "qemu64", "ttyS0"},
	ARM64: {ARM64, "qemu-system-aarch64", "virt", "max", "ttyAMA0"},
}

func Architectures() []string { return []string{AMD64, ARM64} }

// Normalize accepts conventional platform spellings only at the boundary.
func Normalize(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case AMD64, "x86_64", "x64":
		return AMD64, nil
	case ARM64, "aarch64":
		return ARM64, nil
	default:
		return "", fmt.Errorf("unsupported architecture %q; choose amd64 or arm64", value)
	}
}

func Resolve(request, host string) (string, error) {
	if request == "" || request == Auto {
		request = host
	}
	return Normalize(request)
}

func For(arch string) (Profile, error) {
	canonical, err := Normalize(arch)
	if err != nil {
		return Profile{}, err
	}
	return profiles[canonical], nil
}

// FirmwareArgs attaches immutable UEFI code and caller-owned writable variables.
func FirmwareArgs(code, variables string) []string {
	return []string{"-drive", "if=pflash,format=raw,readonly=on,file=" + code, "-drive", "if=pflash,format=raw,file=" + variables}
}
