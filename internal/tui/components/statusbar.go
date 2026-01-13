package components

import (
	"github.com/Josepavese/nido/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// StatusBar renders the footer bar with keymap hints and status.
type StatusBar struct {
	width  int
	items  []StatusBarItem
	status string
}

// StatusBarItem represents a single keymap hint.
type StatusBarItem struct {
	Key   string
	Label string
}

// NewStatusBar creates a status bar with the given width.
func NewStatusBar(width int) StatusBar {
	return StatusBar{
		width: width,
	}
}

// SetItems updates the keymap hints.
func (s *StatusBar) SetItems(items []StatusBarItem) {
	s.items = items
}

// SetStatus sets the right-aligned status text.
func (s *StatusBar) SetStatus(status string) {
	s.status = status
}

// SetWidth updates the status bar width.
func (s *StatusBar) SetWidth(w int) {
	s.width = w
}

// View renders the status bar.
func (s StatusBar) View() string {
	t := theme.Current()

	keyStyle := lipgloss.NewStyle().
		Foreground(t.Palette.Accent).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(t.Palette.TextDim)

	// Build keymap section
	var keymap string
	for i, item := range s.items {
		if i > 0 {
			keymap += "  "
		}
		keymap += keyStyle.Render(item.Key) + " " + labelStyle.Render(item.Label)
	}

	// Build status section
	statusRendered := labelStyle.Render(s.status)

	// Calculate spacing
	keymapWidth := lipgloss.Width(keymap)
	statusWidth := lipgloss.Width(statusRendered)
	spacerWidth := s.width - keymapWidth - statusWidth - 2
	if spacerWidth < 1 {
		spacerWidth = 1
	}

	spacer := lipgloss.NewStyle().Width(spacerWidth).Render("")

	return lipgloss.JoinHorizontal(lipgloss.Top, keymap, spacer, statusRendered)
}
