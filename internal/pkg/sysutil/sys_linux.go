package sysutil

import (
	"bufio"
	"os"
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
