package hardware

import (
	"fmt"
	"regexp"
	"strings"
)

const AutoAcceleratorID = "auto"
const VirtualGPUAcceleratorID = "virtual:gpu"

var pciIDPattern = regexp.MustCompile(`^(?:[0-9a-fA-F]{4}:)?[0-9a-fA-F]{2}:[0-9a-fA-F]{2}\.[0-7]$`)

// ValidateAcceleratorID validates the public accelerator tokens accepted by
// CLI/MCP before any privileged sysfs operation is attempted.
func ValidateAcceleratorID(id string) error {
	id = strings.TrimSpace(id)
	if id == AutoAcceleratorID || id == VirtualGPUAcceleratorID {
		return nil
	}
	if !pciIDPattern.MatchString(id) {
		return fmt.Errorf("invalid accelerator %q: use PCI id like 0000:01:00.0, auto, or virtual:gpu", id)
	}
	return nil
}

func ValidateAccelerators(ids []string) error {
	for _, id := range ids {
		if err := ValidateAcceleratorID(id); err != nil {
			return err
		}
	}
	return nil
}

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
