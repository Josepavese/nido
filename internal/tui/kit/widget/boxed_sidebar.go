package widget

import (
	"github.com/Josepavese/nido/internal/tui/kit/layout"
	"github.com/Josepavese/nido/internal/tui/kit/theme"
	viewlet "github.com/Josepavese/nido/internal/tui/kit/view"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// BoxedSidebar wraps a SidebarList with a Card header, enforcing consistent geometry.
type BoxedSidebar struct {
	viewlet.BaseViewlet
	Header  *Card
	Sidebar *SidebarList
}

// NewBoxedSidebar creates a wrapper for consistent sidebar layout.
func NewBoxedSidebar(header *Card, sidebar *SidebarList) *BoxedSidebar {
	return &BoxedSidebar{
		Header:  header,
		Sidebar: sidebar,
	}
}

func (s *BoxedSidebar) Init() tea.Cmd {
	return s.Sidebar.Init()
}

func (s *BoxedSidebar) Resize(r layout.Rect) {
	s.BaseViewlet.Resize(r)
	s.Header.SetWidth(r.Width)
	hh := lipgloss.Height(s.Header.View(r.Width))

	// Match Card geometry: Border(1) + Padding(1) on each side -> Width - 4.
	// We make the Sidebar a separate Rounded Box below the Header.
	// Height consumes 2 lines for borders.
	innerW := r.Width - 4
	innerH := r.Height - hh - 2
	if innerW < 0 {
		innerW = 0
	}
	if innerH < 0 {
		innerH = 0
	}

	sr := layout.NewRect(r.X, r.Y+hh, innerW, innerH)
	s.Sidebar.Resize(sr)
}

func (s *BoxedSidebar) View() string {
	t := theme.Current()
	// Match Card style: Rounded Border, SurfaceHighlight color, Padding(0,1)
	style := t.Styles.Border.
		Padding(0, 1).
		Width(s.Width() - 2)

	return lipgloss.JoinVertical(lipgloss.Left,
		s.Header.View(s.Width()),
		style.Render(s.Sidebar.View()),
	)
}

func (s *BoxedSidebar) Update(msg tea.Msg) (viewlet.Viewlet, tea.Cmd) {
	// Delegate update to Sidebar
	newSidebar, cmd := s.Sidebar.Update(msg)
	s.Sidebar = newSidebar.(*SidebarList)
	return s, cmd
}

func (s *BoxedSidebar) Shortcuts() []viewlet.Shortcut {
	return s.Sidebar.Shortcuts()
}

func (s *BoxedSidebar) HandleMouse(x, y int, msg tea.MouseMsg) (viewlet.Viewlet, tea.Cmd, bool) {
	hh := lipgloss.Height(s.Header.View(s.Width()))
	if y >= hh {
		// Inside Sidebar Box area
		// Account for Sidebar Border(1) and Padding(1)
		// localX = x - 2
		// localY = y - hh - 1 (Top Border)
		localX := x - 2
		localY := y - hh - 1

		if localX < 0 {
			return s, nil, false
		}

		v, cmd, handled := s.Sidebar.HandleMouse(localX, localY, msg)
		if sl, ok := v.(*SidebarList); ok {
			s.Sidebar = sl
		}
		return s, cmd, handled
	}
	// Header click? (Ignored for now)
	return s, nil, false
}

func (s *BoxedSidebar) Focus() tea.Cmd {
	return s.Sidebar.Focus()
}

func (s *BoxedSidebar) Blur() {
	s.Sidebar.Blur()
}

func (s *BoxedSidebar) Focused() bool {
	return s.Sidebar.Focused()
}
