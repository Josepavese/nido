package widget

import (
	"github.com/charmbracelet/lipgloss"
)

// StatusBarStyles defines the appearance of the status bar.
type StatusBarStyles struct {
	Key      lipgloss.Style
	Label    lipgloss.Style
	Status   lipgloss.Style
	Inactive lipgloss.Style
}

// StatusBar renders the footer bar with keymap hints and status.
type StatusBar struct {
	width  int
	items  []StatusBarItem
	status string
	styles StatusBarStyles
}

// StatusBarItem represents a single keymap hint.
type StatusBarItem struct {
	Key   string
	Label string
}

// NewStatusBar creates a status bar with the given width and styles.
func NewStatusBar(width int, styles StatusBarStyles) StatusBar {
	return StatusBar{
		width:  width,
		styles: styles,
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
	// Build keymap section
	var keymap string
	for i, item := range s.items {
		if i > 0 {
			keymap += "  "
		}
		keymap += s.styles.Key.Render(item.Key) + " " + s.styles.Label.Render(item.Label)
	}

	// Build status section
	statusRendered := s.styles.Status.Render(s.status)

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
