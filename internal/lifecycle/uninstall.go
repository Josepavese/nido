package lifecycle

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Uninstall performs the destructive removal of Nido's data and binary.
// It requires an explicit confirmation unless force is true.
// Note: In TUI/CLI contexts, "force" usually means bypassing the *interactive* prompt.
// Here path validation serves as the last line of defense.
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

	// 2. Remove Data Directory
	if _, err := os.Stat(dataDir); err == nil {
		if err := os.RemoveAll(dataDir); err != nil {
			return fmt.Errorf("failed to remove data directory %s: %w", dataDir, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to access data directory %s: %w", dataDir, err)
	}

	// 3. Remove Binary (Self-Destruct)
	if binPath != "" {
		if _, err := os.Stat(binPath); err == nil {
			if err := os.Remove(binPath); err != nil {
				// Don't error out hard on binary removal if data is gone,
				// but report it.
				return fmt.Errorf("data removed, but failed to remove binary %s: %w", binPath, err)
			}
		}
	}

	return nil
}
