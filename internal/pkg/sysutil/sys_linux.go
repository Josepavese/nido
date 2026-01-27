package sysutil

import (
	"bufio"
	"os"
	"os/exec"
	"strconv"
	"strings"
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
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				kb, _ := strconv.Atoi(parts[1])
				return kb / 1024
			}
		}
	}
	return 0
}

// TerminalCommand returns the command parts to open a terminal with a command.
// It avoids Snap-based terminals (like gnome-terminal snap) which have GLIBC conflicts.
func TerminalCommand(cmd string) (string, []string) {
	// Priority list for Linux terminals
	terminals := []struct {
		bin  string
		args []string
	}{
		{"kitty", []string{"-e", "sh", "-c", cmd}},
		{"alacritty", []string{"-e", "sh", "-c", cmd}},
		{"wezterm", []string{"start", "--", "sh", "-c", cmd}},
		{"xterm", []string{"-e", cmd}},
		{"konsole", []string{"-e", cmd}},
		{"gnome-terminal", []string{"--", "sh", "-c", cmd}},
	}

	for _, t := range terminals {
		if path, err := exec.LookPath(t.bin); err == nil {
			// Avoid Snap-based gnome-terminal which causes GLIBC lookup errors
			if t.bin == "gnome-terminal" && strings.Contains(path, "/snap/") {
				continue
			}
			return t.bin, t.args
		}
	}

	// Ultimate fallback
	return "sh", []string{"-c", cmd}
}

// VNCCommand returns the command to open a VNC viewer.
func VNCCommand(addr string) (string, []string) {
	viewers := []string{"vncviewer", "remote-viewer", "vinagre", "gvncviewer"}
	for _, v := range viewers {
		if _, err := exec.LookPath(v); err == nil {
			return v, []string{addr}
		}
	}
	return "xdg-open", []string{"vnc://" + addr}
}
