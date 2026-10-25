package provider

import (
	"bytes"
	"fmt"
	"github.com/Josepavese/nido/v5/internal/pkg/sysutil"
	"os/exec"
	"slices"
	"time"
)

func (p *QemuProvider) launchQEMU(args []string, arch string) (int, error) {
	pid, err := launchWindowsQEMU(args, arch)
	if err == nil {
		return pid, nil
	}
	fallback := windowsTCGFallbackArgs(args, arch)
	if slices.Equal(args, fallback) {
		return 0, err
	}
	return launchWindowsQEMU(fallback, arch)
}
func launchWindowsQEMU(args []string, arch string) (int, error) {
	binary, err := sysutil.QemuSystemBinary(arch)
	if err != nil {
		return 0, err
	}
	cmd := exec.Command(binary, args...)
	cmd.SysProcAttr = detachedQemuSysProcAttr()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("QEMU failed to start: %w (stderr: %s)", err, stderr.String())
	}
	pid := cmd.Process.Pid
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return 0, fmt.Errorf("QEMU exited during startup: %v (stderr: %s)", err, stderr.String())
	case <-time.After(1500 * time.Millisecond):
		return pid, nil
	}
}
func readLaunchedQemuPID(_ string, _ string, pid int) int { return pid }
func qemuDaemonArgs(string, string) []string              { return nil }
func qemuQMPPath(string, string) string                   { return "" }
func qemuQMPEndpoint(string, string) string               { return "tcp:127.0.0.1:0,server,nowait" }
func virtualGPUDevice() string                            { return "virtio-vga" }
func qemuInstallHints() (string, string) {
	return "(Install QEMU and add it to PATH)", "(Included in QEMU)"
}
