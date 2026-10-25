package widget

import (
	"github.com/Josepavese/nido/v4/internal/tui/kit/layout"
	"github.com/Josepavese/nido/v4/internal/tui/kit/theme"
	viewlet "github.com/Josepavese/nido/v4/internal/tui/kit/view"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
)

// ListView adapts a bubbletea list.Model to the Viewlet interface.
// It serves as a generic primitive that can be embedded or used standalone.
type ListView struct {
	Model  *list.Model
	height int
}

// NewListView creates a new ListView adapter.
func NewListView(m *list.Model) *ListView {
	return &ListView{Model: m, height: m.Height()}
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
	return renderList(l.Model, l.Model.Width(), l.height)
}

// Resize updates the list dimensions.
func (l *ListView) Resize(r layout.Rect) {
	l.height = r.Height
	resizeList(l.Model, r.Width, r.Height)
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

// The kit owns pagination, including its row budget. The underlying list adds a
// mandatory blank line above its paginator, which wastes scarce terminal rows.
func resizeList(model *list.Model, width, height int) {
	model.SetShowPagination(false)
	model.Paginator.Type = paginator.Arabic
	contentHeight := height
	if height > layout.ScrollHintHeight && len(model.VisibleItems()) > height {
		contentHeight -= layout.ScrollHintHeight
	}
	model.SetSize(max(0, width), max(0, contentHeight))
}

func renderList(model *list.Model, width, height int) string {
	resizeList(model, width, height)
	content := model.View()
	if model.Paginator.TotalPages > 1 && height > layout.ScrollHintHeight {
		content = layout.VStack(0, content, theme.Current().Styles.TextMuted.Render(model.Paginator.View()))
	}
	return layout.Fit(width, height, content)
}
