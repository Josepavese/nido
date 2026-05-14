//go:build windows

package runner

import (
	"os"
	"os/exec"
	"strconv"
)

func prepareCommand(cmd *exec.Cmd) {}

func terminateProcessTree(process *os.Process) {
	if process == nil {
		return
	}
	_ = exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(process.Pid)).Run()
	_ = process.Kill()
}
