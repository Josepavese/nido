package ops

import (
	"fmt"
	"regexp"
)

// VMName validates a virtual machine name.
// Must be alphanumeric, dashes, or underscores. 1-64 chars.
func VMName(s string) error {
	if len(s) == 0 {
		return fmt.Errorf("name cannot be empty")
	}
	if len(s) > 64 {
		return fmt.Errorf("name too long (max 64)")
	}
	match, _ := regexp.MatchString("^[a-zA-Z0-9_-]+$", s)
	if !match {
		return fmt.Errorf("must be alphanumeric, '-', or '_'")
	}
	return nil
}

// TemplateName validates a template name.
// Same rules as VMName.
func TemplateName(s string) error {
	return VMName(s) // Same validation rules
}
