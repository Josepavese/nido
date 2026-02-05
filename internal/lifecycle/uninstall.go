package lifecycle

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Josepavese/nido/internal/pkg/sysutil"
)

// Uninstall performs the destructive removal of Nido's data and binary.
func Uninstall(dataDir, binPath string) error {
	// 1. Safety Checks
	if dataDir == "" || dataDir == "/" || dataDir == "." {
		return fmt.Errorf("refusing to delete unsafe data directory: %s", dataDir)
	}
	// Sanity check: Ensure dataDir looks like a nido dir (ends in .nido or nido-test)
	base := filepath.Base(dataDir)
	if !strings.Contains(base, "nido") {
		return fmt.Errorf("safety trigger: data directory '%s' does not look like a nido folder", dataDir)
	}

	// 2. Shell Cleanup
	CleanShellConfig()

	// 3. Desktop Integration Cleanup
	CleanDesktopIntegration()

	// 4. Remove Data Directory
	if _, err := os.Stat(dataDir); err == nil {
		if err := os.RemoveAll(dataDir); err != nil {
			return fmt.Errorf("failed to remove data directory %s: %w", dataDir, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to access data directory %s: %w", dataDir, err)
	}

	// 4. Remove Binary (Self-Destruct)
	if binPath != "" {
		if _, err := os.Stat(binPath); err == nil {
			if err := os.Remove(binPath); err != nil {
				// Don't error out hard on binary removal if data is gone, but report it.
				return fmt.Errorf("data removed, but failed to remove binary %s: %w", binPath, err)
			}
		}
	}

	return nil
}

// CleanShellConfig attempts to remove Nido-related entries from shell configuration files.
func CleanShellConfig() {
	home, err := sysutil.UserHome()
	if err != nil {
		return
	}

	targets := []string{
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".zshrc"),
		filepath.Join(home, ".bash_profile"),
		filepath.Join(home, ".profile"),
	}

	for _, path := range targets {
		if _, err := os.Stat(path); err == nil {
			_ = removeNidoLines(path)
		}
	}
}

func removeNidoLines(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	changed := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		// Skip lines added by Nido installers:
		// # Nido v3
		// # Nido VM Manager
		// export PATH="$PATH:/home/user/.nido/bin"
		// source ".../nido.bash"
		// source <(nido completion ...)
		if strings.Contains(line, "# Nido") ||
			strings.Contains(line, ".nido/bin") ||
			strings.Contains(line, "nido completion") {
			changed = true
			continue
		}
		newLines = append(newLines, line)
	}

	if changed {
		return os.WriteFile(path, []byte(strings.Join(newLines, "\n")), 0644)
	}
	return nil
}

// CleanDesktopIntegration removes launcher entries and shortcuts across platforms.
func CleanDesktopIntegration() {
	home, err := sysutil.UserHome()
	if err != nil {
		return
	}

	// 1. Linux (.desktop)
	linuxEntry := filepath.Join(home, ".local/share/applications/nido.desktop")
	if _, err := os.Stat(linuxEntry); err == nil {
		_ = os.Remove(linuxEntry)
	}
	// Also remove the launcher script if it exists
	launcherScript := filepath.Join(home, ".nido/bin/nido-launcher")
	if _, err := os.Stat(launcherScript); err == nil {
		_ = os.Remove(launcherScript)
	}

	// 2. macOS (.app)
	macEntry := filepath.Join(home, "Applications/Nido.app")
	if _, err := os.Stat(macEntry); err == nil {
		_ = os.RemoveAll(macEntry)
	}

	// 3. Windows (Start Menu .lnk)
	// On Windows, we try to find the Programs folder.
	// This is a best effort since we are likely on Linux during dev.
	if appData := os.Getenv("APPDATA"); appData != "" {
		winEntry := filepath.Join(appData, "Microsoft/Windows/Start Menu/Programs/Nido.lnk")
		if _, err := os.Stat(winEntry); err == nil {
			_ = os.Remove(winEntry)
		}
	}
}
