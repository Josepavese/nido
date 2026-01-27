package sysutil

import (
	"syscall"
	"unsafe"
)

var (
	kernel32                 = syscall.NewLazyDLL("kernel32.dll")
	procGlobalMemoryStatusEx = kernel32.NewProc("GlobalMemoryStatusEx")
)

type memoryStatusEx struct {
	cbSize                  uint32
	dwMemoryLoad            uint32
	ullTotalPhys            uint64
	ullAvailPhys            uint64
	ullTotalPageFile        uint64
	ullAvailPageFile        uint64
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtendedVirtual uint64
}

func calculateDefaultMemory() int {
	totalRAM := getTotalSystemMemoryMB()
	if totalRAM == 0 {
		return 2048
	}

	halfRAM := totalRAM / 2
	if halfRAM < 2048 {
		return halfRAM
	}
	return 2048
}

func getTotalSystemMemoryMB() int {
	var mse memoryStatusEx
	mse.cbSize = uint32(unsafe.Sizeof(mse))

	ret, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&mse)))
	if ret == 0 {
		return 0
	}

	return int(mse.ullTotalPhys / 1024 / 1024)
}
