package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config defines the DNA of a Nido nest. It controls where life is archived,
// default hatching species, and how disk cloning should evolve.
type Config struct {
	BackupDir      string
	SSHUser        string
	ImageDir       string // Directory for downloaded images (default: ~/.nido/images)
	LinkedClones   bool   // Whether to use Copy-on-Write linked clones (default: true)
	Theme          string // Active UI theme name (default: auto)
	PortRangeStart int    // Start of custom port range (default: 30000)
	PortRangeEnd   int    // End of custom port range (default: 32767)
	TUI            TUIConfig
}

// parseInt attempts to parse an integer string, returning the value and a flag.
func parseInt(val string) (int, bool) {
	v, err := strconv.Atoi(strings.TrimSpace(val))
	if err != nil {
		return 0, false
	}
	return v, true
}

// TUIConfig defines runtime overrides for the TUI layout.
type TUIConfig struct {
	SidebarWidth     int
	SidebarWideWidth int
	InsetContent     int
	TabMinWidth      int
	ExitZoneWidth    int
	GapScale         int
}

// LoadConfig reads the genetic configuration from a file.
// If a key is missing, it falls back to historical defaults.
func LoadConfig(path string) (*Config, error) {
	home, _ := os.UserHomeDir()
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cfg := &Config{
		BackupDir:      filepath.Join(home, ".nido", "backups"),
		SSHUser:        "vmuser",
		ImageDir:       filepath.Join(home, ".nido", "images"),
		LinkedClones:   true, // Default to true (space saving)
		PortRangeStart: 30000,
		PortRangeEnd:   32767,
		TUI: TUIConfig{
			SidebarWidth:     30,
			SidebarWideWidth: 38,
			InsetContent:     4,
			TabMinWidth:      6,
			ExitZoneWidth:    4,
			GapScale:         1,
		},
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		val := strings.TrimSpace(os.ExpandEnv(parts[1]))

		if val == "" && key != "THEME" {
			continue // Skip empty overrides to allow defaults to persist
		}

		switch key {
		case "THEME":
			cfg.Theme = val
		case "BACKUP_DIR":
			cfg.BackupDir = val
		case "SSH_USER":
			cfg.SSHUser = val
		case "IMAGE_DIR":
			cfg.ImageDir = val
		case "LINKED_CLONES":
			if val == "false" || val == "0" {
				cfg.LinkedClones = false
			} else {
				cfg.LinkedClones = true
			}
		case "TUI_SIDEBAR_WIDTH":
			if parsed, ok := parseInt(val); ok {
				cfg.TUI.SidebarWidth = parsed
			}
		case "TUI_SIDEBAR_WIDE_WIDTH":
			if parsed, ok := parseInt(val); ok {
				cfg.TUI.SidebarWideWidth = parsed
			}
		case "TUI_INSET_CONTENT":
			if parsed, ok := parseInt(val); ok {
				cfg.TUI.InsetContent = parsed
			}
		case "TUI_TAB_MIN_WIDTH":
			if parsed, ok := parseInt(val); ok {
				cfg.TUI.TabMinWidth = parsed
			}
		case "TUI_EXIT_ZONE_WIDTH":
			if parsed, ok := parseInt(val); ok {
				cfg.TUI.ExitZoneWidth = parsed
			}
		case "TUI_GAP_SCALE":
			if parsed, ok := parseInt(val); ok {
				cfg.TUI.GapScale = parsed
			}
		case "PORT_RANGE_START":
			if parsed, ok := parseInt(val); ok {
				cfg.PortRangeStart = parsed
			}
		case "PORT_RANGE_END":
			if parsed, ok := parseInt(val); ok {
				cfg.PortRangeEnd = parsed
			}
		}
	}
	return cfg, nil
}

// ApplyEnvOverrides updates TUI-related settings from environment variables.
func (c *Config) ApplyEnvOverrides() {
	if v, ok := parseInt(os.Getenv("NIDO_TUI_SIDEBAR_WIDTH")); ok {
		c.TUI.SidebarWidth = v
	}
	if v, ok := parseInt(os.Getenv("NIDO_TUI_SIDEBAR_WIDE_WIDTH")); ok {
		c.TUI.SidebarWideWidth = v
	}
	if v, ok := parseInt(os.Getenv("NIDO_TUI_INSET_CONTENT")); ok {
		c.TUI.InsetContent = v
	}
	if v, ok := parseInt(os.Getenv("NIDO_TUI_TAB_MIN_WIDTH")); ok {
		c.TUI.TabMinWidth = v
	}
	if v, ok := parseInt(os.Getenv("NIDO_TUI_EXIT_ZONE_WIDTH")); ok {
		c.TUI.ExitZoneWidth = v
	}
	if v, ok := parseInt(os.Getenv("NIDO_TUI_GAP_SCALE")); ok && v > 0 {
		c.TUI.GapScale = v
	}
	if v, ok := parseInt(os.Getenv("NIDO_PORT_RANGE_START")); ok {
		c.PortRangeStart = v
	}
	if v, ok := parseInt(os.Getenv("NIDO_PORT_RANGE_END")); ok {
		c.PortRangeEnd = v
	}
}

// UpdateConfig modifies a single genetic sequence in the configuration file.
// It performs an atomic-like update by reading the whole genome (file) first.
func UpdateConfig(path, key, value string) error {
	// 1. Read all lines
	var lines []string
	if _, err := os.Stat(path); err == nil {
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		lines = strings.Split(string(content), "\n")
	}

	// 2. Update or Append
	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, key+"=") {
			lines[i] = key + "=" + value
			found = true
			break
		}
	}

	if !found {
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			lines = append(lines, "")
		}
		lines = append(lines, key+"="+value)
	}

	// 3. Write back
	output := strings.Join(lines, "\n")
	// Ensure newline at end
	if !strings.HasSuffix(output, "\n") {
		output += "\n"
	}

	return os.WriteFile(path, []byte(output), 0644)
}

// UpdateConfigMany modifies multiple settings atomically.
func UpdateConfigMany(path string, updates map[string]string) error {
	// 1. Read all lines
	var lines []string
	if _, err := os.Stat(path); err == nil {
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		lines = strings.Split(string(content), "\n")
	}

	// 2. Update existing keys
	processedKeys := make(map[string]bool)
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		for key, val := range updates {
			if strings.HasPrefix(trimmed, key+"=") {
				lines[i] = key + "=" + val
				processedKeys[key] = true
				break
			}
		}
	}

	// 3. Append new keys
	for key, val := range updates {
		if !processedKeys[key] {
			if len(lines) > 0 && lines[len(lines)-1] != "" {
				lines = append(lines, "")
			}
			lines = append(lines, key+"="+val)
		}
	}

	// 4. Write back
	output := strings.Join(lines, "\n")
	if !strings.HasSuffix(output, "\n") {
		output += "\n"
	}

	return os.WriteFile(path, []byte(output), 0644)
}
