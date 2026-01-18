package ops

import (
	"os"
	"path/filepath"

	"github.com/Josepavese/nido/internal/lifecycle"
	tea "github.com/charmbracelet/bubbletea"
)

// RequestUninstallMsg triggers the uninstall process from the TUI.
type RequestUninstallMsg struct{}

// UninstallCmd executes the uninstallation.
// If successful, it sends tea.Quit() because the binary is gone/going.
// Returns an error msg if it fails.
func UninstallCmd() tea.Cmd {
	return func() tea.Msg {
		// Calculate paths similar to CLI
		home, _ := os.UserHomeDir()
		nidoDir := filepath.Join(home, ".nido")

		exe, err := os.Executable()
		if err != nil {
			return OpResultMsg{Op: "uninstall", Err: err}
		}

		if err := lifecycle.Uninstall(nidoDir, exe); err != nil {
			return OpResultMsg{Op: "uninstall", Err: err}
		}

		// Quit immediately on success
		return tea.Quit
	}
}
