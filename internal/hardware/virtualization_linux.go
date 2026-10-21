package hardware

// KVM_GET_API_VERSION and KVM_API_VERSION from linux/kvm.h.
const kvmGetAPIVersion = 0xAE00
const kvmAPIVersion = 12

func (h *nativeHost) Virtualization() Virtualization {
	if version, err := h.io.DeviceValue("/dev/kvm", kvmGetAPIVersion); err == nil && version == kvmAPIVersion {
		return Virtualization{Accelerator: "kvm", CPU: "host"}
	}
	return Virtualization{CPU: "qemu64"}
}
