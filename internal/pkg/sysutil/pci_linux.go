//go:build linux

package sysutil

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var pciIDPattern = regexp.MustCompile(`^(?:[0-9a-fA-F]{4}:)?[0-9a-fA-F]{2}:[0-9a-fA-F]{2}\.[0-7]$`)

func normalizePCIID(pciID string) (string, error) {
	id := strings.TrimSpace(pciID)
	if !pciIDPattern.MatchString(id) {
		return "", fmt.Errorf("invalid PCI id %q", pciID)
	}
	if len(strings.Split(id, ":")) == 2 {
		id = "0000:" + id
	}
	return strings.ToLower(id), nil
}

func runPrivileged(name string, args ...string) error {
	cmdArgs := args
	if os.Getuid() != 0 {
		cmdArgs = append([]string{name}, args...)
		name = "sudo"
	}
	cmd := exec.Command(name, cmdArgs...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s failed: %s (%w)", name, string(out), err)
	}
	return nil
}

func writeSysfsPrivileged(path, value string) error {
	data := []byte(value + "\n")
	if os.Getuid() == 0 {
		return os.WriteFile(path, data, 0644)
	}
	cmd := exec.Command("sudo", "tee", path)
	cmd.Stdin = bytes.NewReader(data)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("sudo tee %s failed: %s (%w)", path, string(out), err)
	}
	return nil
}

// PreparePassthrough prepares a PCI device for VFIO passthrough.
// It unbinds the device from its current driver, saves the driver name for later restoration,
// and binds it to vfio-pci.
func PreparePassthrough(pciID string, stateDir string) error {
	id, err := normalizePCIID(pciID)
	if err != nil {
		return err
	}

	devPath := fmt.Sprintf("/sys/bus/pci/devices/%s", id)
	driverPath := filepath.Join(devPath, "driver")

	// 1. Check if bound and save state
	if _, err := os.Stat(driverPath); err == nil {
		driverLink, _ := os.Readlink(driverPath)
		driverName := filepath.Base(driverLink)

		if driverName == "vfio-pci" {
			return nil
		}

		// Save original driver for restoration
		driverFile := filepath.Join(stateDir, fmt.Sprintf("pci-%s.driver", id))
		_ = WriteFile(driverFile, []byte(driverName), 0644)
		if err := writeSysfsPrivileged(filepath.Join(devPath, "driver", "unbind"), id); err != nil {
			return fmt.Errorf("failed to unbind device: %w", err)
		}
	}

	// 2. Bind to vfio-pci
	if err := runPrivileged("modprobe", "vfio-pci"); err != nil {
		return fmt.Errorf("failed to load vfio-pci module: %w", err)
	}

	out, err := exec.Command("lspci", "-n", "-s", id).Output()
	if err != nil {
		return fmt.Errorf("failed to get device info: %w", err)
	}
	parts := strings.Fields(string(out))
	if len(parts) < 3 {
		return fmt.Errorf("unexpected lspci output: %s", string(out))
	}
	vendorDevice := parts[2]
	vdParts := strings.Split(vendorDevice, ":")
	if len(vdParts) != 2 {
		return fmt.Errorf("failed to parse vendor:device: %s", vendorDevice)
	}
	vendor, device := vdParts[0], vdParts[1]

	_ = writeSysfsPrivileged("/sys/bus/pci/drivers/vfio-pci/new_id", fmt.Sprintf("%s %s", vendor, device))

	if err := writeSysfsPrivileged("/sys/bus/pci/drivers/vfio-pci/bind", id); err != nil {
		if _, err := os.Stat(filepath.Join(devPath, "driver")); err == nil {
			driverLink, _ := os.Readlink(filepath.Join(devPath, "driver"))
			if filepath.Base(driverLink) == "vfio-pci" {
				return nil
			}
		}
		return fmt.Errorf("failed to bind to vfio-pci: %w", err)
	}

	// 3. Memory Limits
	pid := os.Getpid()
	_ = runPrivileged("prlimit", "--pid", fmt.Sprintf("%d", pid), "--memlock=unlimited:unlimited")

	return nil
}

// RestorePassthrough reverses the prepare operation.
// It explicitly attempts to rebind the device to its original driver.
func RestorePassthrough(pciID string, stateDir string) error {
	id, err := normalizePCIID(pciID)
	if err != nil {
		return err
	}

	devPath := fmt.Sprintf("/sys/bus/pci/devices/%s", id)
	driverPath := filepath.Join(devPath, "driver")

	if _, err := os.Stat(driverPath); err == nil {
		driverLink, _ := os.Readlink(driverPath)
		if filepath.Base(driverLink) == "vfio-pci" {
			if err := writeSysfsPrivileged("/sys/bus/pci/drivers/vfio-pci/unbind", id); err != nil {
				return fmt.Errorf("failed to unbind from vfio-pci: %w", err)
			}

			// Restore Original Driver (Memory)
			driverFile := filepath.Join(stateDir, fmt.Sprintf("pci-%s.driver", id))
			if data, err := os.ReadFile(driverFile); err == nil {
				originalDriver := strings.TrimSpace(string(data))
				if err := writeSysfsPrivileged(filepath.Join("/sys/bus/pci/drivers", originalDriver, "bind"), id); err == nil {
					os.Remove(driverFile)
					return nil
				}
			}

			// Fallback: Rescan
			_ = writeSysfsPrivileged(filepath.Join(devPath, "remove"), "1")
			time.Sleep(100 * time.Millisecond)
			_ = writeSysfsPrivileged("/sys/bus/pci/rescan", "1")
		}
	}
	return nil
}
