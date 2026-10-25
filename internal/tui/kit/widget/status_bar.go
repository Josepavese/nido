package widget

import (
	"github.com/Josepavese/nido/v4/internal/tui/kit/layout"
	"github.com/Josepavese/nido/v4/internal/tui/kit/theme"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"strings"
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
	if s.width <= 0 {
		return ""
	}
	render := func(item StatusBarItem) string {
		return s.styles.Key.Render(item.Key) + " " + s.styles.Label.Render(item.Label)
	}
	gap := strings.Repeat(" ", theme.Space.SM)
	// The last hint is the persistent escape route. Earlier hints are contextual.
	var exit string
	items := s.items
	if len(items) > 0 {
		exit = render(items[len(items)-1])
		items = items[:len(items)-1]
	}
	available := max(0, s.width-lipgloss.Width(exit)-lipgloss.Width(gap))
	var hints []string
	for _, item := range items {
		candidate := strings.Join(append(append([]string{}, hints...), render(item)), gap)
		if lipgloss.Width(candidate) > available {
			continue
		}
		hints = append(hints, render(item))
	}
	left := strings.Join(hints, gap)
	if s.status != "" {
		space := available - lipgloss.Width(left) - lipgloss.Width(gap)
		if space > 0 {
			left += gap + s.styles.Status.Render(ansi.Truncate(s.status, space, "…"))
		}
	}
	spacer := strings.Repeat(" ", max(0, s.width-lipgloss.Width(left)-lipgloss.Width(exit)))
	return layout.Fit(s.width, 1, left+spacer+exit)
}
