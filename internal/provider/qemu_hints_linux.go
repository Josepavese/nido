package provider

import "runtime"

func qemuInstallHints() (string, string) {
	pkg := "qemu-system-x86"
	if runtime.GOARCH == "arm64" {
		pkg = "qemu-system-arm"
	}
	return "(Try: sudo apt update && sudo apt install " + pkg + " qemu-utils)", "(Included in qemu-utils)"
}
