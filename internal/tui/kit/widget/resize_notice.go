package widget

import (
	"fmt"
	"github.com/Josepavese/nido/v4/internal/tui/kit/layout"
	"github.com/Josepavese/nido/v4/internal/tui/kit/theme"
	"github.com/charmbracelet/lipgloss"
)

// RenderResizeNotice is the kit's resting state when meaningful interaction
// cannot fit. The application retains drafts, selection and background tasks.
func RenderResizeNotice(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	t := theme.Current()
	title := t.Styles.AccentStrong.Render(theme.IconNest + " A little room to hatch")
	measure := fmt.Sprintf("%d×%d → %d×%d", width, height, layout.MinWidth, layout.MinHeight)
	text := lipgloss.JoinVertical(lipgloss.Center, title, t.Styles.TextMuted.Render("Stretch the nest. Your place is saved."), measure, t.Styles.TextMuted.Render("ctrl+c quit"))
	if height < lipgloss.Height(text) || width < lipgloss.Width(title) {
		text = measure + "\n^C quit"
	}
	return layout.PlaceOverlay(width, height, text)
}
