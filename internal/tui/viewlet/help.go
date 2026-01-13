package viewlet

import (
	"github.com/Josepavese/nido/internal/tui/layout"
	"github.com/Josepavese/nido/internal/tui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Help implements the Help viewlet displaying keyboard shortcuts.
type Help struct {
	BaseViewlet
	width  int
	height int
}

// NewHelp creates a new Help viewlet.
func NewHelp() *Help {
	return &Help{}
}

// Init initializes the Help viewlet.
func (h *Help) Init() tea.Cmd {
	return nil
}

// Update handles messages for the Help viewlet.
func (h *Help) Update(msg tea.Msg) (Viewlet, tea.Cmd) {
	return h, nil
}

// View renders the Help viewlet.
func (h *Help) View() string {
	t := theme.Current()

	keyStyle := lipgloss.NewStyle().Foreground(t.Palette.AccentStrong).Width(7)
	descStyle := lipgloss.NewStyle().Foreground(t.Palette.TextDim)
	sectionStyle := lipgloss.NewStyle().Foreground(t.Palette.Accent).Bold(true)

	// Compact row helper
	row := func(key, desc string) string {
		return keyStyle.Render(key) + descStyle.Render(desc)
	}

	// Build sections as columns
	col1 := layout.VStack(0,
		sectionStyle.Render("NAVIGATION"),
		row("1-5", "Switch tabs"),
		row("←/→", "Cycle tabs"),
		"",
		sectionStyle.Render("FLEET"),
		row("↑/↓", "Select VM"),
		row("↵", "Start/Stop"),
		row("Del", "Delete"),
		row("s", "SSH"),
		row("i", "Info"),
	)

	col2 := layout.VStack(0,
		sectionStyle.Render("HATCHERY"),
		row("Tab", "Next field"),
		row("Space", "Select src"),
		row("↵", "Spawn"),
		"",
		sectionStyle.Render("CONFIG"),
		row("↑/↓", "Select key"),
		row("↵", "Edit"),
		row("Esc", "Cancel"),
	)

	col3 := layout.VStack(0,
		sectionStyle.Render("GLOBAL"),
		row("q", "Quit"),
		row("Ctrl+C", "Force quit"),
	)

	// Arrange columns horizontally with gap
	colStyle := lipgloss.NewStyle().Width(25)

	return layout.HStack(2,
		colStyle.Render(col1),
		colStyle.Render(col2),
		colStyle.Render(col3),
	)
}

// Resize updates the viewlet dimensions.
func (h *Help) Resize(width, height int) {
	h.width = width
	h.height = height
}

// Shortcuts returns Help-specific shortcuts.
func (h *Help) Shortcuts() []Shortcut {
	return []Shortcut{{Key: "q", Label: "quit"}}
}
