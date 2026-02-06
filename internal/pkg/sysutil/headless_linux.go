//go:build linux

package sysutil

import (
	"os"
	"path/filepath"
	"strings"
)

// IsHeadless checks if the system appears to be running without a physical display attached or active GUI session.
// Used to determine if it's safe to passthrough the primary GPU.
func IsHeadless() bool {
	// 1. Check for Active Display Server Environment Variables
	// If these are set, someone is likely using the display.
	if os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != "" {
		return false
	}

	// 2. Check for DRM Connectors (Linux specific check)
	// If we find any connector with status "connected", we assume a monitor is plugged in.
	// /sys/class/drm/card*-*/status
	// Note: IPMI/BMC virtual displays might show as connected, so this is a heuristic.
	globPattern := "/sys/class/drm/card*-*/status"
	matches, _ := filepath.Glob(globPattern)
	for _, statusFile := range matches {
		data, err := os.ReadFile(statusFile)
		if err == nil {
			status := strings.TrimSpace(string(data))
			if status == "connected" {
				// Special exception for virtual connectors?
				// e.g. Virtual-1. But for now, if it's connected, strict safety says NOT headless.
				return false
			}
		}
	}

	// 3. Check runlevel or systemd target?
	// Too complex. The Environment check + DRM check is usually sufficient for "Is a user logged in physically?".

	return true
}
