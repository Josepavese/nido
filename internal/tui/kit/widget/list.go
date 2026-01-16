package widget

import (
	"github.com/Josepavese/nido/internal/tui/kit/layout"
	viewlet "github.com/Josepavese/nido/internal/tui/kit/view"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// ListView adapts a bubbletea list.Model to the Viewlet interface.
// It serves as a generic primitive that can be embedded or used standalone.
type ListView struct {
	Model *list.Model
}

// NewListView creates a new ListView adapter.
func NewListView(m *list.Model) *ListView {
	return &ListView{Model: m}
}

// Update handles messages and delegates to the underlying bubbles model.
func (l *ListView) Update(msg tea.Msg) (viewlet.Viewlet, tea.Cmd) {
	var cmd tea.Cmd
	newModel, cmd := l.Model.Update(msg)
	*l.Model = newModel
	return l, cmd
}

// View returns the rendered list.
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
func (l *ListView) HasActiveTextInput() bool      { return false }
func (l *ListView) HasActiveFocus() bool          { return false }
func (l *ListView) Focusable() bool               { return true }

func (l *ListView) HandleMouse(x, y int, msg tea.MouseMsg) (viewlet.Viewlet, tea.Cmd, bool) {
	// Simple passthrough for generic ListView;
	// specialized lists (like SidebarList) can override with hit-testing logic.
	msg.X = x
	msg.Y = y
	newV, cmd := l.Update(msg)
	return newV, cmd, true
}
