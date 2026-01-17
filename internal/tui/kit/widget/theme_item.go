package widget

import (
	"github.com/charmbracelet/lipgloss"
)

// ThemeItem is a SidebarItem that represents a UI theme.
// It displays a colored block icon matching the theme's primary color.
type ThemeItem struct {
	Name         string
	PrimaryColor lipgloss.TerminalColor
}

func (i ThemeItem) Title() string       { return i.Name }
func (i ThemeItem) Description() string { return "" }
func (i ThemeItem) FilterValue() string { return i.Name }
func (i ThemeItem) IsAction() bool      { return false }

// Icon returns a colored block character using the theme's primary color key.
// Since Icon() returns a string, we return the glyph itself.
// The SidebarDelegate handles color if it's just text, but here we want
// specific coloring per item. SidebarDelegate usually colors ALL icons same.
// WAIT. SidebarDelegate (viewed earlier) just does: fmt.Sprintf("%s%s", theme.RenderIcon(icon), str)
// RenderIcon just pads it.
// It does NOT apply color to the icon.
// The color comes from `d.Styles.Normal` or `d.Styles.Selected`.
// To have per-item icon colors, we need to embed the ANSI escape codes in the Icon string itself.
func (i ThemeItem) Icon() string {
	if i.PrimaryColor == nil {
		return "🎨"
	}
	// return a block colored with the primary color
	return lipgloss.NewStyle().Foreground(i.PrimaryColor).Render("█")
}
