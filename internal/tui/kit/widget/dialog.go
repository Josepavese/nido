package widget

import (
	"github.com/Josepavese/nido/v4/internal/tui/kit/layout"
	"github.com/Josepavese/nido/v4/internal/tui/kit/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// dialogGeometry is shared by every overlay: outer dimensions include its frame.
type dialogGeometry struct {
	outer    layout.Rect
	inner    layout.Rect
	paddingY int
}

func measureDialog(parentW, parentH, preferredW, preferredH int) dialogGeometry {
	w := max(0, min(preferredW, parentW-layout.DialogMargin))
	h := max(0, min(preferredH, parentH-layout.DialogMargin))
	padY := layout.DialogPaddingY
	if parentH < layout.MinHeight {
		padY = 0
	}
	outer := layout.NewRect((parentW-w)/2, (parentH-h)/2, w, h)
	return dialogGeometry{outer: outer, inner: outer.Inner(layout.DialogBorder/2+padY, layout.DialogBorder/2+layout.DialogPaddingX, layout.DialogBorder/2+padY, layout.DialogBorder/2+layout.DialogPaddingX), paddingY: padY}
}
func (d dialogGeometry) render(parentW, parentH int, content string, color lipgloss.TerminalColor) string {
	frame := theme.Current().Styles.Border.BorderForeground(color).Padding(d.paddingY, layout.DialogPaddingX).Width(max(0, d.outer.Width-layout.DialogBorder))
	return layout.PlaceOverlay(parentW, parentH, frame.Render(layout.Fit(d.inner.Width, d.inner.Height, content)))
}

// Overlay is a blocking kit surface. Its parent draws it and routes mouse hits
// using the same parent rectangle, independently of any panes behind it.
type Overlay interface {
	IsActive() bool
	View(int, int) string
	HandleMouse(int, int, tea.MouseMsg) (tea.Cmd, bool)
}
