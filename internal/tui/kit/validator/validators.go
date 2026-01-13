// Package validators provides input validation functions for the Nido TUI.
// All validators return nil if valid, or an error describing the issue.
package validator

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// FilePath validates that a path exists and is readable.
func FilePath(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil // Empty is allowed (optional field)
	}

	info, err := os.Stat(s)
	if os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", s)
	}
	if err != nil {
		return fmt.Errorf("cannot access file: %v", err)
	}
	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file")
	}

	return nil
}

// DirPath validates that a path is a directory.
func DirPath(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil // Empty is allowed (optional field)
	}

	info, err := os.Stat(s)
	if os.IsNotExist(err) {
		return fmt.Errorf("directory not found: %s", s)
	}
	if err != nil {
		return fmt.Errorf("cannot access directory: %v", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is a file, not a directory")
	}

	return nil
}

// Port validates a port number.
// Must be between 1 and 65535.
func Port(n int) error {
	if n < 1 || n > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	return nil
}

// PortString validates a port number from string input.
func PortString(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil // Empty is allowed
	}

	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	if err != nil {
		return fmt.Errorf("invalid port number")
	}

	return Port(n)
}

// NotEmpty validates that a string is not empty.
func NotEmpty(s string) error {
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("value cannot be empty")
	}
	return nil
}

// ConfigKey validates a configuration key name.
func ConfigKey(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("key cannot be empty")
	}

	// Keys are uppercase with underscores
	pattern := regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
	if !pattern.MatchString(s) {
		return fmt.Errorf("key must be uppercase with underscores (e.g., MY_KEY)")
	}

	return nil
}
