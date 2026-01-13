package gui

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/Josepavese/nido/internal/tui/services"
	tea "github.com/charmbracelet/bubbletea"
)

// openTerminalCmd opens an external terminal with an SSH session.
func openTerminalCmd(sshCmd string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "linux":
			// Try common terminals
			terms := []string{"x-terminal-emulator", "gnome-terminal", "konsole", "xfce4-terminal", "xterm"}
			var termFound string
			for _, t := range terms {
				if _, err := exec.LookPath(t); err == nil {
					termFound = t
					break
				}
			}

			if termFound != "" {
				// Wrap command to keep terminal open on error
				// Use a more robust bash wrapper
				wrappedCmd := fmt.Sprintf("%s || (echo ''; echo '----------------------------------------'; echo '⚠️  SSH SESSION FAILED'; echo '----------------------------------------'; echo 'Press Enter to close this terminal...'; read)", sshCmd)

				switch termFound {
				case "gnome-terminal", "xfce4-terminal":
					cmd = exec.Command(termFound, "--", "bash", "-c", wrappedCmd)
				case "konsole":
					cmd = exec.Command(termFound, "-e", "bash", "-c", wrappedCmd)
				default:
					cmd = exec.Command(termFound, "-e", "bash", "-c", wrappedCmd)
				}
			} else {
				return services.LogMsg{Text: "No terminal emulator found"}
			}
		case "darwin":
			// macOS: Use osascript to open Terminal
			appleScript := fmt.Sprintf(`tell application "Terminal" to do script "%s"`, sshCmd)
			cmd = exec.Command("osascript", "-e", appleScript)
		case "windows":
			// Windows: Open cmd and run ssh
			cmd = exec.Command("cmd", "/c", "start", "cmd", "/k", sshCmd)
		default:
			return services.LogMsg{Text: fmt.Sprintf("Quick SSH not supported on %s", runtime.GOOS)}
		}

		if cmd != nil {
			err := cmd.Start()
			if err != nil {
				return services.LogMsg{Text: fmt.Sprintf("Failed to open terminal: %v", err)}
			}
		}
		return nil
	}
}

// openVNCCmd opens an external VNC viewer.
func openVNCCmd(vncAddr string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "linux":
			// Use xdg-open for vnc:// or try common viewers
			if _, err := exec.LookPath("xdg-open"); err == nil {
				cmd = exec.Command("xdg-open", "vnc://"+vncAddr)
			} else if _, err := exec.LookPath("gvncviewer"); err == nil {
				cmd = exec.Command("gvncviewer", vncAddr)
			} else if _, err := exec.LookPath("vncviewer"); err == nil {
				cmd = exec.Command("vncviewer", vncAddr)
			}
		case "darwin":
			cmd = exec.Command("open", "vnc://"+vncAddr)
		case "windows":
			// Windows: vnc:// might not be registered by default, but we can try
			cmd = exec.Command("cmd", "/c", "start", "vnc://"+vncAddr)
		}

		if cmd != nil {
			err := cmd.Start()
			if err != nil {
				return services.LogMsg{Text: fmt.Sprintf("Failed to open VNC: %v", err)}
			}
		} else {
			return services.LogMsg{Text: "No VNC viewer or xdg-open found"}
		}
		return nil
	}
}
