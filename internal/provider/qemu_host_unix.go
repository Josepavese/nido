//go:build !windows

package provider

import (
	"bytes"
	"fmt"
	"github.com/Josepavese/nido/v4/internal/pkg/sysutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func (p *QemuProvider) launchQEMU(args []string) (int, error) {
	binary, err := sysutil.QemuSystemBinary()
	if err != nil {
		return 0, err
	}
	cmd := exec.Command(binary, args...)
	cmd.SysProcAttr = detachedQemuSysProcAttr()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("QEMU failed to start: %w (stderr: %s)", err, stderr.String())
	}
	return 0, nil
}
func readLaunchedQemuPID(runDir, name string, _ int) int {
	var pid int
	for i := 0; i < 10; i++ {
		data, err := os.ReadFile(filepath.Join(runDir, name+".pid"))
		if err == nil {
			fmt.Sscanf(string(data), "%d", &pid)
			if pid > 0 {
				return pid
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return pid
}
func qemuDaemonArgs(runDir, name string) []string {
	return []string{"-daemonize", "-pidfile", filepath.Join(runDir, name+".pid")}
}
func qemuQMPPath(runDir, name string) string { return filepath.Join(runDir, name+".qmp") }
func qemuQMPEndpoint(runDir, name string) string {
	return "unix:" + qemuQMPPath(runDir, name) + ",server,nowait"
}
func virtualGPUDevice() string { return "virtio-gpu-pci" }
