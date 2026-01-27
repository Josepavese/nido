package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_Defaults(t *testing.T) {
	// Create a temp config file
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.env")

	err := os.WriteFile(cfgPath, []byte(""), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Check defaults (hardcoded in LoadConfig or logic)
	// Based on LoadConfig implementation:
	if cfg.SSHUser != "vmuser" {
		t.Errorf("Expected default SSHUser 'vmuser', got '%s'", cfg.SSHUser)
	}
}

func TestLoadConfig_Values(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.env")

	backups := filepath.Join(os.TempDir(), "backups")
	content := "BACKUP_DIR=" + backups + "\nSSH_USER=testuser\nTEMPLATE_DEFAULT=my-tpl\n"
	err := os.WriteFile(cfgPath, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.BackupDir != backups {
		t.Errorf("Expected BackupDir %q, got %q", backups, cfg.BackupDir)
	}
	if cfg.SSHUser != "testuser" {
		t.Errorf("Expected SSHUser 'testuser', got '%s'", cfg.SSHUser)
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join(os.TempDir(), "non-existent-nido-config.env"))
	if err == nil {
		t.Error("Expected error for missing file, got nil")
	}
	if cfg != nil {
		t.Error("Expected nil config for missing file")
	}
}
