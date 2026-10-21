//go:build linux

package hardware

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// ListAccelerators scans the host PCI bus for potential accelerators.
// It analyzes IOMMU groups to determine safety.
func (h *nativeHost) ListAccelerators() ([]Accelerator, error) {
	sysBusPCI := pciDevicesPath
	entries, err := h.io.ReadDir(sysBusPCI)
	if err != nil {
		return nil, fmt.Errorf("failed to list pci devices: %w", err)
	}

	var devices []Accelerator

	// Pre-calculate IOMMU group membership
	// Map: GroupID -> []DevicePCI_ID
	groupMembers := make(map[string][]string)

	for _, entry := range entries {
		pciID := entry.Name()
		groupPath := filepath.Join(sysBusPCI, pciID, "iommu_group")
		if groupDest, err := h.io.Readlink(groupPath); err == nil {
			groupID := filepath.Base(groupDest)
			groupMembers[groupID] = append(groupMembers[groupID], pciID)
		}
	}

	// Identify Boot VGA to mark as unsafe
	bootVGA := map[string]bool{}
	displayIdentity := map[string]bool{}
	// Try to find boot_vga file
	// Usually at /sys/bus/pci/devices/<id>/boot_vga containing "1"
	for _, entry := range entries {
		pciID := entry.Name()
		if content, err := h.io.ReadFile(filepath.Join(sysBusPCI, pciID, "boot_vga")); err == nil {
			displayIdentity[pciID] = true
			if strings.TrimSpace(string(content)) == "1" {
				bootVGA[pciID] = true
			}
		}
	}

	for _, entry := range entries {
		pciID := entry.Name()
		devPath := filepath.Join(sysBusPCI, pciID)

		// Get Class
		classBytes, err := h.io.ReadFile(filepath.Join(devPath, "class"))
		if err != nil {
			return nil, fmt.Errorf("read PCI class for %s: %w", pciID, err)
		}
		// class is 0xCCSSPP (Class, Subclass, ProgIF)
		classStr := strings.TrimPrefix(strings.TrimSpace(string(classBytes)), "0x")
		if len(classStr) != 6 {
			return nil, fmt.Errorf("invalid PCI class for %s", pciID)
		}

		// Filter: We primarily care about:
		// 03xx (Display controller)
		// 04xx (Multimedia controller - e.g. Audio)
		// 0Cxx (Serial bus controller - e.g. USB/Thunderbolt)
		// 12xx (Processing accelerators)
		// But for now, let's list interesting ones.
		if len(classStr) >= 2 {
			prefix := classStr[0:2]
			// Skip Bridges (06), Storage (01), Network (02) unless requested?
			// User might want NIC passthrough (02).
			// Let's whitelist interesting classes for "Accelerator" discovery to avoid noise.
			// 03: Display
			// 04: Multimedia
			// 02: Network (often requested)
			// 12: Processing accelerators (TPUs etc)
			// 0c03: USB (often requested)
			isInteresting := false
			if prefix == ClassDisplay || prefix == ClassProcessing || prefix == ClassNetwork || prefix == ClassMultimedia {
				isInteresting = true
			}
			if strings.HasPrefix(classStr, ClassSerialBusUSB) { // USB
				isInteresting = true
			}

			if !isInteresting {
				continue
			}
		}

		// Get Vendor/Device
		vendorBytes, _ := h.io.ReadFile(filepath.Join(devPath, "vendor"))
		deviceBytes, _ := h.io.ReadFile(filepath.Join(devPath, "device"))
		vendorID := strings.TrimPrefix(strings.TrimSpace(string(vendorBytes)), "0x")
		deviceID := strings.TrimPrefix(strings.TrimSpace(string(deviceBytes)), "0x")

		// Resolve names (simple lookup or just IDs)
		// For now, we use IDs unless we want to parse /usr/share/misc/pci.ids (too heavy?)
		// Let's use lspci output if available? No, keep it pure Go for speed.
		// We will just return the IDs and maybe a generic class name.

		className := getClassName(classStr)

		// IOMMU Group Analysis
		groupID := ""
		groupPath := filepath.Join(devPath, "iommu_group")
		if groupDest, err := h.io.Readlink(groupPath); err == nil {
			groupID = filepath.Base(groupDest)
		}

		isIsolated := groupID != ""
		siblings := groupMembers[groupID]
		// Check siblings
		for _, siblingID := range siblings {
			if siblingID == pciID {
				continue
			}
			// If sibling is a Bridge (06xx), it often doesn't count against isolation for endpoints
			// But strict isolation requires checking.
			// For simplicity/safety: Isolated = singleton group OR siblings are just PCIe Root Ports.
			// Let's stick to Singleton for "Strict Safety".
			isIsolated = false
			break
		}

		acc := Accelerator{
			ID:         pciID,
			Vendor:     vendorID,
			Device:     deviceID,
			Class:      className,
			IOMMUGroup: groupID,
			IsIsolated: isIsolated,
			IsSafe:     isIsolated,
		}

		if groupID == "" {
			acc.IsSafe = false
			acc.Warning = "No IOMMU group; passthrough isolation unavailable"
		} else if className == DescDisplay && !displayIdentity[pciID] {
			acc.IsSafe = false
			acc.Warning = "Primary display identity is unavailable"
		} else if bootVGA[pciID] {
			acc.IsSafe = false
			acc.Warning = "Primary Display; host device must remain attached"
		} else if !isIsolated {
			acc.Warning = fmt.Sprintf("Shared IOMMU Group %s (Siblings: %s)", groupID, strings.Join(siblings, ", "))
		}

		devices = append(devices, acc)
	}

	// Sort by ID
	sort.Slice(devices, func(i, j int) bool {
		return devices[i].ID < devices[j].ID
	})

	return devices, nil
}

func getClassName(classHex string) string {
	if len(classHex) < 4 {
		return "Unknown"
	}
	// Simple lookup
	prefix := classHex[0:2]
	switch prefix {
	case ClassDisplay:
		return DescDisplay
	case ClassNetwork:
		return DescNetwork
	case ClassMultimedia:
		return DescMultimedia
	case ClassSerialBus:
		if strings.HasPrefix(classHex, ClassSerialBusUSB) {
			return DescUSB
		}
		return DescSerial
	case ClassProcessing:
		return DescProcessing
	}
	return classHex
}
