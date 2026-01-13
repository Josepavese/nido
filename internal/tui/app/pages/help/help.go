package help

import (
	"github.com/Josepavese/nido/internal/tui/kit/layout"
	view "github.com/Josepavese/nido/internal/tui/kit/view"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// Help implements the Help viewlet.
type Help struct {
	view.BaseViewlet
	viewport viewport.Model
}

// NewHelp returns a new Help viewlet.
func NewHelp() *Help {
	h := &Help{}
	h.viewport = viewport.New(0, 0)
	h.viewport.SetContent(`Manual placeholder`)
	return h
}

// Init initializes the viewlet.
func (h *Help) Init() tea.Cmd {
	return nil
}

// Shortcuts returns the list of keyboard shortcuts available.
func (h *Help) Shortcuts() []view.Shortcut {
	return view.DefaultShortcuts()
}

func (h *Help) HandleMouse(x, y int, msg tea.MouseMsg) (view.Viewlet, tea.Cmd, bool) {
	h.viewport, _ = h.viewport.Update(msg)
	return h, nil, true
}

func (h *Help) Update(msg tea.Msg) (view.Viewlet, tea.Cmd) {
	return h, nil
}

// View renders the help view.
func (h *Help) View() string {
	return h.viewport.View()
}

// Resize updates the viewlet dimensions.
func (h *Help) Resize(r layout.Rect) {
	h.BaseViewlet.Resize(r)
	h.viewport.Width = r.Width
	h.viewport.Height = r.Height
}
