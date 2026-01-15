package widget

import (
	"github.com/Josepavese/nido/internal/tui/kit/layout"
	viewlet "github.com/Josepavese/nido/internal/tui/kit/view"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// ListView adapts a bubbletea list.Model to the Viewlet interface.
type ListView struct {
	Model *list.Model
}

// NewListView creates a new ListView adapter.
func NewListView(m *list.Model) *ListView {
	return &ListView{Model: m}
}

// Update handles messages. Note: The list update logic is often handled
// by the parent model in complex apps, but this provides a default pass-through.
func (l *ListView) Update(msg tea.Msg) (viewlet.Viewlet, tea.Cmd) {
	var cmd tea.Cmd
	*l.Model, cmd = l.Model.Update(msg)
	return l, cmd
}

// View returns the list view.
func (l *ListView) View() string {
	return l.Model.View()
}

// Resize updates the list dimensions.
func (l *ListView) Resize(r layout.Rect) {
	l.Model.SetSize(r.Width, r.Height)
}

func (l *ListView) Init() tea.Cmd                 { return nil }
func (l *ListView) Focus() tea.Cmd                { return nil }
func (l *ListView) Blur()                         {}
func (l *ListView) Focused() bool                 { return false }
func (l *ListView) Shortcuts() []viewlet.Shortcut { return nil }
func (l *ListView) IsModalActive() bool           { return false }

func (l *ListView) HandleMouse(x, y int, msg tea.MouseMsg) (viewlet.Viewlet, tea.Cmd, bool) {
	msg.X = x
	msg.Y = y
	newV, cmd := l.Update(msg)
	return newV, cmd, true
}
