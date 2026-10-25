package ops

import (
	"github.com/Josepavese/nido/v4/internal/events"
	"os"

	"github.com/Josepavese/nido/v4/internal/lifecycle"
	"github.com/Josepavese/nido/v4/internal/pkg/sysutil"
	tea "github.com/charmbracelet/bubbletea"
)

// RequestUninstallMsg triggers the uninstall process from the TUI.
type RequestUninstallMsg struct{}

// UninstallCmd executes the uninstallation.
// If successful, it sends tea.Quit() because the binary is gone/going.
// Returns an error msg if it fails.
func UninstallCmd(publish events.Publisher) tea.Cmd {
	return func() tea.Msg {
		// Calculate paths similar to CLI

		nidoDir := sysutil.NidoHome()

		exe, err := os.Executable()
		if err != nil {
			return OpResultMsg{Op: "uninstall", Err: err}
		}

		if err := lifecycle.Uninstall(nidoDir, exe, publish); err != nil {
			return OpResultMsg{Op: "uninstall", Err: err}
		}

		// Quit immediately on success
		return tea.Quit()
	}
}
