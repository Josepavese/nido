package config

import (
	"bufio"
	"os"
	"strings"
)

// Config defines the DNA of a Nido nest. It controls where life is archived,
// default hatching species, and how disk cloning should evolve.
type Config struct {
	BackupDir       string
	TemplateDefault string
	SSHUser         string
	ImageDir        string // Directory for downloaded images (default: ~/.nido/images)
	LinkedClones    bool   // Whether to use Copy-on-Write linked clones (default: true)
}

// LoadConfig reads the genetic configuration from a file.
// If a key is missing, it falls back to historical defaults.
func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cfg := &Config{
		BackupDir:       "/tmp/libvirt-pool/backups",
		TemplateDefault: "template-headless",
		SSHUser:         "vmuser",
		ImageDir:        "",   // Will be set to ~/.nido/images if not specified
		LinkedClones:    true, // Default to true (space saving)
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
		val := parts[1]

		switch key {
		case "BACKUP_DIR":
			cfg.BackupDir = val
		case "TEMPLATE_DEFAULT":
			cfg.TemplateDefault = val
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
		// Backwards compatibility for old config
		case "CACHE_IMAGES":
			if val == "false" || val == "0" {
				cfg.LinkedClones = false
			} else {
				cfg.LinkedClones = true
			}
		}
	}

	return cfg, nil
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
