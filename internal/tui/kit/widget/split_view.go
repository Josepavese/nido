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
// Note: In Nido's architecture, specific tabs often handle their own updates
// in the model loop, so this might be a no-op depending on usage.
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

	// Use resolved width from Resize
	if s.resolvedSidebarWidth <= 0 {
		return mainView
	}

	// 1. Sidebar (Content Only)
	sidebarStyled := lipgloss.NewStyle().
		Width(s.resolvedSidebarWidth).
		Render(sidebarView)

	// 2. Separator Column
	// We create a standalone column that is 0-width but has a Left Border.
	// We force height to s.height to ensure the line goes all the way down.
	// Note: Border(Left) adds 1 char width.
	sepStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true). // Left only
		BorderForeground(theme.Current().Palette.SurfaceHighlight).
		Height(s.height)

	separator := sepStyle.Render("")

	// 3. Main Content
	// No border on main content correctly.

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		sidebarStyled,
		separator,
		mainView,
	)
}

// Resize calculates the layout for sidebar and main content.
func (s *SplitView) Resize(r layout.Rect) {
	s.height = r.Height // Persist height for View/Separator

	sw := s.SidebarWidth
	if sw <= 0 {
		// Use standard Nido logic if no explicit width provided
		// Narrow (25), Regular (Sidebar=30), Wide (SidebarWide=38)
		// We set narrow to 25 to satisfy the user's min-width request.
		dim := layout.Calculate(r.Width, r.Height, 25, theme.Width.Sidebar, theme.Width.SidebarWide)
		sw = dim.SidebarWidth
	}

	// Force minimum safety width to update persisted s.resolvedSidebarWidth
	// If the user complains about "missing lists", we should err on the side of showing them.
	// Only hide if strictly 0 requested?
	// But Calculate returns 0 for Narrow.
	// We want to force it to be visible if possible?
	// User moved to Phase 3 "Ensure Min-Width of 25 chars for Narrow terminals".
	// So Calculate passed 25. It should return 25.
	// Unless r.Width < 25.

	if sw == 0 && r.Width > 30 {
		// Fallback for safety calculation error
		sw = 30
	}

	// Persist the calculated width for View() to use in an internal field.
	// This preserves s.SidebarWidth as 0 (Auto) for future Resize calls.
	s.resolvedSidebarWidth = sw

	if sw <= 0 {
		// Full width for Main
		s.Main.Resize(r)
		// Hide Sidebar (size 0)
		s.Sidebar.Resize(layout.NewRect(0, 0, 0, 0))
		return
	}

	// Calculate widths using the effective width
	sidebarWidth := sw
	// Sidebar takes its configured width.
	// The Border adds separate width (usually 1 char).
	borderW := s.BorderStyle.GetHorizontalFrameSize()
	if borderW == 0 {
		// Just in case style changes (GetHorizontalBorder returns left+right)
		// For right border only, it should be 1.
		// Let's force check or assume standard border.
		// Using GetHorizontalBorder() is safer if style is robust.
		// If style is empty, it returns 0.
	}

	// We resize the inner sidebar component to the requested width.
	// The wrapper in View() adds the border.
	sidebarRect := layout.NewRect(r.X, r.Y, sidebarWidth, r.Height)

	// Main content starts after sidebar + border
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

// Focus delegates focus to Main view by default, or manages internal focus state.
func (s *SplitView) Focus() tea.Cmd { return nil }
func (s *SplitView) Blur()          {}
func (s *SplitView) Focused() bool  { return false }

func (s *SplitView) Shortcuts() []viewlet.Shortcut { return nil }

func (s *SplitView) HandleMouse(x, y int, msg tea.MouseMsg) (viewlet.Viewlet, tea.Cmd, bool) {
	// Translate coordinates and delegate
	borderW := s.BorderStyle.GetHorizontalFrameSize()

	// Check Sidebar
	if x < s.resolvedSidebarWidth {
		_, cmd, handled := s.Sidebar.HandleMouse(x, y, msg)
		return s, cmd, handled
	}

	// Check Main
	mainX := s.resolvedSidebarWidth + borderW
	if x >= mainX {
		_, cmd, handled := s.Main.HandleMouse(x-mainX, y, msg)
		return s, cmd, handled
	}

	return s, nil, false
}
