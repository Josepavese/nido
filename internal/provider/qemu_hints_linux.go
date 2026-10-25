package provider

import (
	"github.com/Josepavese/nido/v5/internal/machine"
	"github.com/Josepavese/nido/v5/internal/pkg/sysutil"
)

func qemuInstallHints() (string, string) {
	pkg := "qemu-system-x86"
	if sysutil.HostArchitecture() == machine.ARM64 {
		pkg = "qemu-system-arm qemu-efi-aarch64"
	}
	return "(Try: sudo apt update && sudo apt install " + pkg + " qemu-utils)", "(Included in qemu-utils)"
}
