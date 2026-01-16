package widget

import (
	"github.com/Josepavese/nido/internal/tui/kit/layout"
	viewlet "github.com/Josepavese/nido/internal/tui/kit/view"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MasterDetail is a high-level component that orchestrates a Sidebar and multiple Pages.
// It automatically wires SelectionMsg from the sidebar to switch pages in the PageManager.
type MasterDetail struct {
	viewlet.BaseViewlet
	Sidebar viewlet.Viewlet
	Pages   *PageManager
	Split   *SplitView

	// ActiveFocus tracks which side has input focus (0=Sidebar, 1=Detail)
	ActiveFocus int

	// AutoSwitch determines if selection messages automatically trigger page switches.
	// True by default.
	AutoSwitch bool
}

const (
	FocusSidebar = 0
	FocusDetail  = 1
)

// NewMasterDetail creates a new MasterDetail orchestrator.
func NewMasterDetail(sidebar viewlet.Viewlet, pages *PageManager, borderStyle lipgloss.Style) *MasterDetail {
	m := &MasterDetail{
		Sidebar:     sidebar,
		Pages:       pages,
		AutoSwitch:  true,
		ActiveFocus: FocusSidebar, // Default to Sidebar navigation
	}
	m.Split = NewSplitView(sidebar, pages, borderStyle)
	return m
}

// Init initializes children and sets initial focus.
func (m *MasterDetail) Init() tea.Cmd {
	return tea.Batch(
		m.Sidebar.Init(),
		m.Pages.Init(),
		m.SetFocus(m.ActiveFocus), // Apply initial focus state
	)
}

// Update handles orchestration and propagates messages.
func (m *MasterDetail) Update(msg tea.Msg) (viewlet.Viewlet, tea.Cmd) {
	var cmds []tea.Cmd

	// Orchestration: Listen for SelectionMsg
	if m.AutoSwitch {
		if selMsg, ok := msg.(viewlet.SelectionMsg); ok {
			key := ""
			if keyed, ok := selMsg.Item.(interface{ Key() string }); ok {
				key = keyed.Key()
			} else if sidebarItem, ok := selMsg.Item.(SidebarItem); ok {
				key = sidebarItem.Title()
			}

			if key != "" && key != m.Pages.Active {
				cmds = append(cmds, m.Pages.SwitchTo(key))
			}
		}
	}

	// Delegation to SplitView
	newSplit, cmd := m.Split.Update(msg)
	m.Split = newSplit.(*SplitView)
	cmds = append(cmds, cmd)

	// Focus Management via keyboard
	if kmsg, ok := msg.(tea.KeyMsg); ok && !m.IsModalActive() {
		switch kmsg.String() {
		case "tab", "right":
			if m.ActiveFocus == FocusSidebar && m.Pages.Focusable() {
				cmds = append(cmds, m.SetFocus(FocusDetail))
			}
		case "shift+tab", "left", "esc":
			if m.ActiveFocus == FocusDetail {
				cmds = append(cmds, m.SetFocus(FocusSidebar))
			}
		case "enter":
			if m.ActiveFocus == FocusSidebar {
				// Optionally jump to detail on select
				// cmds = append(cmds, m.SetFocus(FocusDetail))
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *MasterDetail) SetFocus(side int) tea.Cmd {
	m.ActiveFocus = side
	if side == FocusSidebar {
		m.Pages.Blur()
		return m.Sidebar.Focus()
	} else {
		m.Sidebar.Blur()
		return m.Pages.Focus()
	}
}

// View renders the layout.
func (m *MasterDetail) View() string {
	return m.Split.View()
}

// Resize propagates dimensions.
func (m *MasterDetail) Resize(r layout.Rect) {
	m.BaseViewlet.Resize(r)
	m.Split.Resize(r)
}

// HandleMouse delegates.
func (m *MasterDetail) HandleMouse(x, y int, msg tea.MouseMsg) (viewlet.Viewlet, tea.Cmd, bool) {
	// Mouse click might change focus.
	// SplitView HandleMouse calls HandleMouse on children.
	// We rely on children to set their own focus internal flags?
	// But MasterDetail needs to know ActiveFocus for Shortcuts().
	// We should intercept results.
	// But SplitView doesn't return which child handled it easily without tracking.
	// Simplified: Let SplitView handle it, and we might be out of sync on ActiveFocus
	// unless we check Focused() status of children.

	v, cmd, handled := m.Split.HandleMouse(x, y, msg)
	if handled {
		// Update ActiveFocus based on who claims focus
		if m.Sidebar.Focused() {
			m.ActiveFocus = FocusSidebar
		} else if m.Pages.Focused() {
			m.ActiveFocus = FocusDetail
		}
	}
	return v, cmd, handled
}

// Shortcuts delegates to active pane.
func (m *MasterDetail) Shortcuts() []viewlet.Shortcut {
	if m.ActiveFocus == FocusSidebar {
		return m.Sidebar.Shortcuts()
	}
	return m.Pages.Shortcuts()
}

func (m *MasterDetail) Focus() tea.Cmd {
	return m.SetFocus(m.ActiveFocus)
}

func (m *MasterDetail) Blur() {
	m.Sidebar.Blur()
	m.Pages.Blur()
}

func (m *MasterDetail) Focused() bool {
	return m.Sidebar.Focused() || m.Pages.Focused()
}

func (m *MasterDetail) IsModalActive() bool {
	return (m.Sidebar != nil && m.Sidebar.IsModalActive()) || (m.Pages != nil && m.Pages.IsModalActive())
}

func (m *MasterDetail) HasActiveInput() bool {
	res := (m.Sidebar != nil && m.Sidebar.HasActiveInput()) || (m.Pages != nil && m.Pages.HasActiveInput())
	return res
}
