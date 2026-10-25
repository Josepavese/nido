package widget

import (
	"github.com/Josepavese/nido/v4/internal/tui/kit/layout"
	"github.com/Josepavese/nido/v4/internal/tui/kit/theme"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// RenderEmptyState gives unselected and empty details the same measured invitation.
// It contains no focusable controls and never invents placeholder domain values.
func RenderEmptyState(icon, title, message string, width, height int) string {
	measure := layout.FormWidth(width, theme.Inset.TotalContent)
	heading := NewCard(icon, title, "").View(measure)
	body := theme.Current().Styles.TextDim.Width(measure).Align(lipgloss.Center).Render(ansi.Wrap(message, max(1, measure), ""))
	return layout.PlaceOverlay(width, height, layout.VStack(theme.Space.SM, heading, body))
}
