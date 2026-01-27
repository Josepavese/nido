package sysutil

import (
	"encoding/binary"
	"fmt"
	"os/exec"
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

// TerminalCommand returns the command parts to open a terminal with a command.
func TerminalCommand(cmd string) (string, []string) {
	// Priority: iTerm2 > Terminal.app
	if _, err := exec.LookPath("iterm2"); err == nil {
		return "osascript", []string{"-e", fmt.Sprintf(`tell application "iTerm" to create window with default profile command "%s"`, cmd)}
	}

	return "osascript", []string{"-e", fmt.Sprintf(`tell application "Terminal" to do script "%s"`, cmd)}
}

// VNCCommand returns the command to open a VNC viewer.
func VNCCommand(addr string) (string, []string) {
	return "open", []string{"vnc://" + addr}
}
