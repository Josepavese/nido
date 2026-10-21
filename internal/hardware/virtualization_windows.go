package hardware

import (
	"golang.org/x/sys/windows"
	"unsafe"
)

// WHvGetCapability is the Windows Hypervisor Platform device API. Failure or an
// absent platform is software emulation, never a claim of hardware support.
// ABI: https://learn.microsoft.com/en-us/virtualization/api/hypervisor-platform/funcs/whvgetcapability
func (*nativeHost) Virtualization() Virtualization {
	dll := windows.NewLazySystemDLL("WinHvPlatform.dll")
	if err := dll.Load(); err != nil {
		return Virtualization{CPU: "qemu64"}
	}
	defer windows.FreeLibrary(windows.Handle(dll.Handle()))
	proc := dll.NewProc("WHvGetCapability")
	if proc.Find() != nil {
		return Virtualization{CPU: "qemu64"}
	}
	var present uint64
	var written uint32
	const hypervisorPresentCapability = 0
	const hypervisorPresentBytes = 4
	status, _, _ := proc.Call(hypervisorPresentCapability, uintptr(unsafe.Pointer(&present)), unsafe.Sizeof(present), uintptr(unsafe.Pointer(&written)))
	if status == 0 && written >= hypervisorPresentBytes && written <= uint32(unsafe.Sizeof(present)) && uint32(present) != 0 {
		return Virtualization{Accelerator: "whpx", CPU: "host"}
	}
	return Virtualization{CPU: "qemu64"}
}
