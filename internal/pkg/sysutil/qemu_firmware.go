package sysutil

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Josepavese/nido/v4/internal/machine"
)

// QEMUFirmware pairs matching immutable code and an initial variables image.
// Callers copy Variables for each VM; the package's original is never writable.
type QEMUFirmware struct{ Code, Variables string }

func FindQEMUFirmware(arch string) (QEMUFirmware, error) {
	if arch != machine.ARM64 {
		return QEMUFirmware{}, fmt.Errorf("no UEFI profile for %s", arch)
	}
	binary, err := QemuSystemBinary(arch)
	if err != nil {
		return QEMUFirmware{}, err
	}
	if resolved, err := filepath.EvalSymlinks(binary); err == nil {
		binary = resolved
	}
	binDir := filepath.Dir(binary)
	qemuDirs := []string{filepath.Join(binDir, "..", "share", "qemu"), filepath.Join(binDir, "share"), "/usr/share/qemu"}
	pairs := []QEMUFirmware{{"/usr/share/AAVMF/AAVMF_CODE.fd", "/usr/share/AAVMF/AAVMF_VARS.fd"}, {filepath.Join(binDir, "..", "share", "AAVMF", "AAVMF_CODE.fd"), filepath.Join(binDir, "..", "share", "AAVMF", "AAVMF_VARS.fd")}}
	for _, dir := range qemuDirs {
		pairs = append(pairs, QEMUFirmware{filepath.Join(dir, machine.FirmwareARM64), filepath.Join(dir, "edk2-arm-vars.fd")})
	}
	for _, pair := range pairs {
		code, e1 := os.Stat(pair.Code)
		vars, e2 := os.Stat(pair.Variables)
		if e1 == nil && e2 == nil && code.Mode().IsRegular() && vars.Mode().IsRegular() {
			return pair, nil
		}
	}
	return QEMUFirmware{}, fmt.Errorf("ARM64 UEFI code/variables not found; install the QEMU EDK2 or qemu-efi-aarch64 firmware package")
}
