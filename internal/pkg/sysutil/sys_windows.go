package sysutil

import (
	"os/exec"
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

// TerminalCommand returns the command parts to open a terminal with a command.
func TerminalCommand(cmd string) (string, []string) {
	// Priority: Windows Terminal > cmd.exe start
	if _, err := exec.LookPath("wt.exe"); err == nil {
		return "wt.exe", []string{"-w", "0", "nt", "sh", "-c", cmd}
	}

	return "cmd.exe", []string{"/c", "start", "sh", "-c", cmd}
}

// VNCCommand returns the command to open a VNC viewer.
func VNCCommand(addr string) (string, []string) {
	return "cmd.exe", []string{"/c", "start", "vnc://" + addr}
}
