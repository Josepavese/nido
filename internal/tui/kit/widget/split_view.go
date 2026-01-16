package widget

import (
	"github.com/Josepavese/nido/internal/tui/kit/layout"
	"github.com/Josepavese/nido/internal/tui/kit/theme"
	viewlet "github.com/Josepavese/nido/internal/tui/kit/view"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SplitView is a generic viewlet that manages a Sidebar and a Main content area.
type SplitView struct {
	Sidebar viewlet.Viewlet
	Main    viewlet.Viewlet

	// SidebarWidth determins the width of the sidebar.
	// This should be set before calling Resize() or View().
	SidebarWidth int

	// Styles
	BorderStyle lipgloss.Style

	// internal state
	resolvedSidebarWidth int
	height               int
}

// NewSplitView creates a new SplitView with a specific border style for the sidebar separator.
func NewSplitView(sidebar, main viewlet.Viewlet, borderStyle lipgloss.Style) *SplitView {
	return &SplitView{
		Sidebar:     sidebar,
		Main:        main,
		BorderStyle: borderStyle,
	}
}

// Update propagates messages to children.
func (s *SplitView) Update(msg tea.Msg) (viewlet.Viewlet, tea.Cmd) {
	var cmds []tea.Cmd
	var v viewlet.Viewlet

	v, cmdS := s.Sidebar.Update(msg)
	s.Sidebar = v
	cmds = append(cmds, cmdS)

	v, cmdM := s.Main.Update(msg)
	s.Main = v
	cmds = append(cmds, cmdM)

	return s, tea.Batch(cmds...)
}

// View renders the sidebar and main content side-by-side.
func (s *SplitView) View() string {
	sidebarView := s.Sidebar.View()
	mainView := s.Main.View()

	if s.resolvedSidebarWidth <= 0 {
		return mainView
	}

	sidebarStyled := lipgloss.NewStyle().
		Width(s.resolvedSidebarWidth).
		Render(sidebarView)

	sepStyle := theme.Current().Styles.Border.Copy().
		Border(lipgloss.NormalBorder(), false, false, false, true). // Left only
		Height(s.height)

	separator := sepStyle.Render("")

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		sidebarStyled,
		separator,
		mainView,
	)
}

// Resize calculates the layout for sidebar and main content.
func (s *SplitView) Resize(r layout.Rect) {
	s.height = r.Height

	sw := s.SidebarWidth
	if sw <= 0 {
		dim := layout.Calculate(r.Width, r.Height, 30, theme.Width.Sidebar, theme.Width.SidebarWide)
		sw = dim.SidebarWidth
	}

	if sw == 0 && r.Width > 30 {
		sw = 30
	}

	s.resolvedSidebarWidth = sw

	if sw <= 0 {
		s.Main.Resize(r)
		s.Sidebar.Resize(layout.NewRect(0, 0, 0, 0))
		return
	}

	sidebarWidth := sw
	borderW := s.BorderStyle.GetHorizontalFrameSize()
	sidebarRect := layout.NewRect(r.X, r.Y, sidebarWidth, r.Height)

	mainX := r.X + sidebarWidth + borderW
	mainW := r.Width - sidebarWidth - borderW

	if mainW < 0 {
		mainW = 0
	}

	mainRect := layout.NewRect(mainX, r.Y, mainW, r.Height)

	s.Sidebar.Resize(sidebarRect)
	s.Main.Resize(mainRect)
}

func (s *SplitView) Init() tea.Cmd { return nil }

func (s *SplitView) Focus() tea.Cmd { return nil }
func (s *SplitView) Blur()          {}
func (s *SplitView) Focused() bool  { return false }

func (s *SplitView) Shortcuts() []viewlet.Shortcut { return nil }

func (s *SplitView) HandleMouse(x, y int, msg tea.MouseMsg) (viewlet.Viewlet, tea.Cmd, bool) {
	borderW := s.BorderStyle.GetHorizontalFrameSize()

	if x < s.resolvedSidebarWidth {
		_, cmd, handled := s.Sidebar.HandleMouse(x, y, msg)
		return s, cmd, handled
	}

	mainX := s.resolvedSidebarWidth + borderW
	if x >= mainX {
		_, cmd, handled := s.Main.HandleMouse(x-mainX, y, msg)
		return s, cmd, handled
	}

	return s, nil, false
}

func (s *SplitView) IsModalActive() bool {
	if s.Sidebar != nil && s.Sidebar.IsModalActive() {
		return true
	}
	if s.Main != nil && s.Main.IsModalActive() {
		return true
	}
	return false
}

func (s *SplitView) HasActiveTextInput() bool {
	if s.Sidebar != nil && s.Sidebar.HasActiveTextInput() {
		return true
	}
	if s.Main != nil && s.Main.HasActiveTextInput() {
		return true
	}
	return false
}

func (s *SplitView) HasActiveFocus() bool {
	if s.Sidebar != nil && s.Sidebar.HasActiveFocus() {
		return true
	}
	if s.Main != nil && s.Main.HasActiveFocus() {
		return true
	}
	return false
}

func (s *SplitView) Focusable() bool {
	return false
}
