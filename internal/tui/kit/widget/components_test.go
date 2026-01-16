package widget

import (
	"testing"

	"github.com/Josepavese/nido/internal/tui/kit/layout"
	viewlet "github.com/Josepavese/nido/internal/tui/kit/view"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Mock Viewlet for testing
type mockViewlet struct {
	width  int
	height int
}

func (m *mockViewlet) Init() tea.Cmd                                 { return nil }
func (m *mockViewlet) Update(msg tea.Msg) (viewlet.Viewlet, tea.Cmd) { return m, nil }
func (m *mockViewlet) View() string                                  { return "" }
func (m *mockViewlet) Resize(r layout.Rect) {
	m.width = r.Width
	m.height = r.Height
}
func (m *mockViewlet) Shortcuts() []viewlet.Shortcut { return nil }
func (m *mockViewlet) Focus() tea.Cmd                { return nil }
func (m *mockViewlet) Blur()                         {}
func (m *mockViewlet) Focused() bool                 { return false }
func (m *mockViewlet) HandleMouse(x, y int, msg tea.MouseMsg) (viewlet.Viewlet, tea.Cmd, bool) {
	return m, nil, false
}
func (m *mockViewlet) IsModalActive() bool  { return false }
func (m *mockViewlet) HasActiveInput() bool { return false }
func (m *mockViewlet) Focusable() bool      { return true }

func TestSplitView_Resize(t *testing.T) {
	sidebar := &mockViewlet{}
	main := &mockViewlet{}
	sv := NewSplitView(sidebar, main, lipgloss.NewStyle())
	sv.SidebarWidth = 20

	// Test 1: Standard Resize
	// Width 100. Sidebar 20. Border 1 (default). Main = 100 - 20 - 1 = 79.
	// Wait, lipgloss style might vary. Let's assume 1 for Right Border only.
	// We need to verify what GetHorizontalFrameSize returns for that style.

	// Force style for deterministic test?
	// The NewSplitView uses theme which might be auto-detected.
	// But in test environment, theme.Current() should return defaults.

	rect := layout.NewRect(0, 0, 100, 50)
	sv.Resize(rect)

	if sidebar.width != 20 {
		t.Errorf("expected sidebar width 20, got %d", sidebar.width)
	}

	// Allow for border variation (0, 1, or 2)
	expectedMain := 100 - 20 - sv.BorderStyle.GetHorizontalFrameSize()
	if main.width != expectedMain {
		t.Errorf("expected main width %d, got %d (border frame size: %d)", expectedMain, main.width, sv.BorderStyle.GetHorizontalFrameSize())
	}
}

func TestSplitView_Resize_Narrow(t *testing.T) {
	sidebar := &mockViewlet{}
	main := &mockViewlet{}
	sv := NewSplitView(sidebar, main, lipgloss.NewStyle())

	// Simulate hiding sidebar
	sv.SidebarWidth = 0

	rect := layout.NewRect(0, 0, 80, 24)
	sv.Resize(rect)

	if sidebar.width != 25 {
		t.Errorf("expected narrow sidebar width 25, got %d", sidebar.width)
	}
	// 80 - 25 = 55 (border is 0 for NewStyle)
	if main.width != 55 {
		t.Errorf("expected main width 55, got %d", main.width)
	}
}

func TestListView_Adapter(t *testing.T) {
	// Setup list model
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 10, 10)
	lv := NewListView(&l)

	// Test Resize updates internal model
	lv.Resize(layout.NewRect(0, 0, 50, 20))

	if l.Width() != 50 {
		t.Errorf("expected list width 50, got %d", l.Width())
	}
	if l.Height() != 20 {
		t.Errorf("expected list height 20, got %d", l.Height())
	}
}
