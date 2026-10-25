package widget

import (
	"github.com/Josepavese/nido/v5/internal/tui/kit/layout"
	"github.com/Josepavese/nido/v5/internal/tui/kit/theme"
	viewlet "github.com/Josepavese/nido/v5/internal/tui/kit/view"
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
	width                int
	singlePane           bool
	ShowMain             bool
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
	if s.singlePane {
		if s.ShowMain {
			return layout.Fit(s.width, s.height, s.Main.View())
		}
		return layout.Fit(s.width, s.height, s.Sidebar.View())
	}
	sidebarView := layout.Fit(s.resolvedSidebarWidth, s.height, s.Sidebar.View())
	mainW := s.width - s.resolvedSidebarWidth - layout.PaneDividerWidth
	separator := s.BorderStyle.Border(lipgloss.NormalBorder(), false, false, false, true).Height(s.height).Render("")
	return layout.Fit(s.width, s.height, lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, separator, layout.Fit(mainW, s.height, s.Main.View())))
}

func (s *SplitView) Resize(r layout.Rect) {
	s.width, s.height = r.Width, r.Height
	sw := layout.SidebarWidth(r.Width, s.SidebarWidth, theme.Width.Sidebar, theme.Width.SidebarWide)
	s.singlePane = layout.SinglePane(r.Width, sw)
	s.resolvedSidebarWidth = sw
	if s.singlePane {
		s.Sidebar.Resize(r)
		s.Main.Resize(r)
		return
	}
	s.Sidebar.Resize(layout.NewRect(r.X, r.Y, sw, r.Height))
	s.Main.Resize(layout.NewRect(r.X+sw+layout.PaneDividerWidth, r.Y, r.Width-sw-layout.PaneDividerWidth, r.Height))
}

func (s *SplitView) SinglePane() bool { return s.singlePane }

func (s *SplitView) Init() tea.Cmd { return nil }

func (s *SplitView) Focus() tea.Cmd { return nil }
func (s *SplitView) Blur()          {}
func (s *SplitView) Focused() bool  { return false }

func (s *SplitView) Shortcuts() []viewlet.Shortcut { return nil }

func (s *SplitView) HandleMouse(x, y int, msg tea.MouseMsg) (viewlet.Viewlet, tea.Cmd, bool) {
	if s.singlePane {
		target := s.Sidebar
		if s.ShowMain {
			target = s.Main
		}
		_, cmd, handled := target.HandleMouse(x, y, msg)
		return s, cmd, handled
	}
	borderW := layout.PaneDividerWidth

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

// MainAt resolves mouse focus with the same geometry used by View/HandleMouse.
func (s *SplitView) MainAt(x int) bool {
	if s.singlePane {
		return s.ShowMain
	}
	return x >= s.resolvedSidebarWidth+layout.PaneDividerWidth
}
