package config

import (
	"bufio"
	"os"
	"strings"
)

type Config struct {
	BackupDir       string
	TemplateDefault string
	SSHUser         string
	ImageDir        string // Directory for downloaded images (default: ~/.nido/images)
}

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
		ImageDir:        "", // Will be set to ~/.nido/images if not specified
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
		}
	}

	return cfg, nil
}
