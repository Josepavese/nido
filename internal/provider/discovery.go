package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/Josepavese/nido/internal/pkg/sysutil"
)

// Accelerator represents a PCI device candidate for passthrough.
type Accelerator struct {
	ID         string // PCI ID (e.g., "0000:00:1f.3")
	Vendor     string // Vendor Name/ID
	Device     string // Device Name/ID
	Class      string // Device Class (e.g., "VGA compatible controller")
	IOMMUGroup string // IOMMU Group ID
	IsIsolated bool   // True if this is the only device in the group (or others are bridges)
	IsSafe     bool   // True if Isolated AND not the primary GPU
	Warning    string // Reason why it might not be safe
}

// ListAccelerators scans the host PCI bus for potential accelerators.
// It analyzes IOMMU groups to determine safety.
func (p *QemuProvider) ListAccelerators() ([]Accelerator, error) {
	if runtime.GOOS != "linux" {
		// Passthrough is only supported on Linux
		return []Accelerator{}, nil
	}

	sysBusPCI := "/sys/bus/pci/devices"
	entries, err := os.ReadDir(sysBusPCI)
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
		if groupDest, err := os.Readlink(groupPath); err == nil {
			groupID := filepath.Base(groupDest)
			groupMembers[groupID] = append(groupMembers[groupID], pciID)
		}
	}

	// Identify Boot VGA to mark as unsafe
	bootVGA := ""
	// Try to find boot_vga file
	// Usually at /sys/bus/pci/devices/<id>/boot_vga containing "1"
	for _, entry := range entries {
		pciID := entry.Name()
		if content, err := os.ReadFile(filepath.Join(sysBusPCI, pciID, "boot_vga")); err == nil {
			if strings.TrimSpace(string(content)) == "1" {
				bootVGA = pciID
				break
			}
		}
	}

	for _, entry := range entries {
		pciID := entry.Name()
		devPath := filepath.Join(sysBusPCI, pciID)

		// Get Class
		classBytes, _ := os.ReadFile(filepath.Join(devPath, "class"))
		// class is 0xCCSSPP (Class, Subclass, ProgIF)
		classStr := strings.TrimPrefix(strings.TrimSpace(string(classBytes)), "0x")

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
		vendorBytes, _ := os.ReadFile(filepath.Join(devPath, "vendor"))
		deviceBytes, _ := os.ReadFile(filepath.Join(devPath, "device"))
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
		if groupDest, err := os.Readlink(groupPath); err == nil {
			groupID = filepath.Base(groupDest)
		}

		isIsolated := true
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

		if pciID == bootVGA {
			// If we are Headless, taking the Boot VGA is technically safe(r).
			if sysutil.IsHeadless() {
				acc.Warning = "Primary Display (Headless Mode Detected)"
				// We still warn, but we leave IsSafe=true if it was isolated
			} else {
				acc.IsSafe = false
				acc.Warning = "Primary Display (Active Session Detected)"
			}
		} else if !isIsolated {
			acc.IsSafe = false
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
