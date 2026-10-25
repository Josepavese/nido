//go:build !linux && !darwin && !windows

package hardware

func (*nativeHost) Virtualization(guest string) Virtualization { return SoftwareVirtualization(guest) }
