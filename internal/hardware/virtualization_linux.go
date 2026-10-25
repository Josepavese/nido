package hardware

import "github.com/Josepavese/nido/v4/internal/machine"

// KVM_GET_API_VERSION and KVM_API_VERSION from linux/kvm.h.
const kvmGetAPIVersion = 0xAE00
const kvmAPIVersion = 12

func (h *nativeHost) Virtualization(guest string) Virtualization {
	if guest != h.io.Architecture() {
		return SoftwareVirtualization(guest)
	}
	if version, err := h.io.DeviceValue("/dev/kvm", kvmGetAPIVersion); err == nil && version == kvmAPIVersion {
		return Virtualization{Accelerator: machine.AcceleratorKVM, CPU: machine.CPUHost}
	}
	return SoftwareVirtualization(guest)
}
