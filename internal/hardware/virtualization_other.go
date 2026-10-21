//go:build !linux && !darwin && !windows

package hardware

func (*nativeHost) Virtualization() Virtualization { return Virtualization{CPU: "qemu64"} }
