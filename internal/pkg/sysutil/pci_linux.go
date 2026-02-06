//go:build linux

package sysutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// PreparePassthrough prepares a PCI device for VFIO passthrough.
// It unbinds the device from its current driver, saves the driver name for later restoration,
// and binds it to vfio-pci.
func PreparePassthrough(pciID string, stateDir string) error {
	id := strings.TrimSpace(pciID)
	if len(strings.Split(id, ":")) == 2 {
		id = "0000:" + id
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
		if err := WriteFile(driverFile, []byte(driverName), 0644); err != nil {
			fmt.Printf("⚠️  Warning: Failed to save original driver info: %v\n", err)
		}

		fmt.Printf("⚡ [Zero-Config] Unbinding %s from %s...\n", id, driverName)
		unbindCmd := fmt.Sprintf("echo '%s' > /sys/bus/pci/devices/%s/driver/unbind", id, id)
		if err := ExecPrivileged(unbindCmd); err != nil {
			return fmt.Errorf("failed to unbind device: %w", err)
		}
	}

	// 2. Bind to vfio-pci
	fmt.Printf("⚡ [Zero-Config] Binding %s to vfio-pci...\n", id)
	if err := ExecPrivileged("modprobe vfio-pci"); err != nil {
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

	bindCmd := fmt.Sprintf("echo '%s %s' > /sys/bus/pci/drivers/vfio-pci/new_id", vendor, device)
	_ = ExecPrivileged(bindCmd)

	bindDevCmd := fmt.Sprintf("echo '%s' > /sys/bus/pci/drivers/vfio-pci/bind", id)
	if err := ExecPrivileged(bindDevCmd); err != nil {
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
	limitCmd := fmt.Sprintf("prlimit --pid %d --memlock=unlimited:unlimited", pid)
	if err := ExecPrivileged(limitCmd); err != nil {
		fmt.Printf("⚠️  Warning: Failed to boost memory limits. (%v)\n", err)
	}

	return nil
}

// RestorePassthrough reverses the prepare operation.
// It explicitly attempts to rebind the device to its original driver.
func RestorePassthrough(pciID string, stateDir string) error {
	id := strings.TrimSpace(pciID)
	if len(strings.Split(id, ":")) == 2 {
		id = "0000:" + id
	}

	devPath := fmt.Sprintf("/sys/bus/pci/devices/%s", id)
	driverPath := filepath.Join(devPath, "driver")

	if _, err := os.Stat(driverPath); err == nil {
		driverLink, _ := os.Readlink(driverPath)
		if filepath.Base(driverLink) == "vfio-pci" {
			fmt.Printf("⚡ [Zero-Config] Releasing %s from vfio-pci...\n", id)

			unbindCmd := fmt.Sprintf("echo '%s' > /sys/bus/pci/drivers/vfio-pci/unbind", id)
			if err := ExecPrivileged(unbindCmd); err != nil {
				return fmt.Errorf("failed to unbind from vfio-pci: %w", err)
			}

			// Restore Original Driver (Memory)
			driverFile := filepath.Join(stateDir, fmt.Sprintf("pci-%s.driver", id))
			if data, err := os.ReadFile(driverFile); err == nil {
				originalDriver := strings.TrimSpace(string(data))
				fmt.Printf("⚡ [Zero-Config] Rebinding %s to original driver '%s'...\n", id, originalDriver)

				bindCmd := fmt.Sprintf("echo '%s' > /sys/bus/pci/drivers/%s/bind", id, originalDriver)
				if err := ExecPrivileged(bindCmd); err == nil {
					os.Remove(driverFile)
					return nil
				}
				fmt.Printf("⚠️  Warning: Rebind failed %v, falling back to rescan.\n", err)
			}

			// Fallback: Rescan
			fmt.Printf("⚡ [Zero-Config] Restoring host driver for %s (Rescan)...\n", id)
			ExecPrivileged(fmt.Sprintf("echo 1 > /sys/bus/pci/devices/%s/remove", id))
			time.Sleep(100 * time.Millisecond)
			ExecPrivileged("echo 1 > /sys/bus/pci/rescan")
		}
	}
	return nil
}
