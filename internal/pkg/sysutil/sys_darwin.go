package sysutil

import (
	"encoding/binary"
	"syscall"
)

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
	out, err := syscall.Sysctl("hw.memsize")
	if err != nil {
		return 0
	}

	if len(out) >= 8 {
		bytes := uint64(binary.LittleEndian.Uint64([]byte(out)))
		return int(bytes / 1024 / 1024)
	} else if len(out) >= 4 {
		bytes := uint32(binary.LittleEndian.Uint32([]byte(out)))
		return int(bytes / 1024 / 1024)
	}

	return 0
}
