package widget

import (
	"github.com/Josepavese/nido/v5/internal/tui/kit/layout"
	"github.com/Josepavese/nido/v5/internal/tui/kit/theme"
	"github.com/Josepavese/nido/v5/internal/tui/kit/view"
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
	marginY := layout.DialogMargin
	if parentH < layout.MinHeight {
		marginY = 0
	}
	h := max(0, min(preferredH, parentH-marginY))
	padY := layout.DialogPaddingY
	if parentH < layout.CompactHeight {
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
	Shortcuts() []view.Shortcut
	View(int, int) string
	HandleMouse(int, int, tea.MouseMsg) (tea.Cmd, bool)
}

// ActiveOverlay resolves the same priority for rendering, mouse input and hints.
func ActiveOverlay(overlays ...Overlay) Overlay {
	for _, overlay := range overlays {
		if overlay != nil && overlay.IsActive() {
			return overlay
		}
	}
	return nil
}
