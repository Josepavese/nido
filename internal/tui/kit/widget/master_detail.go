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
	Sidebar *SidebarList
	Pages   *PageManager
	Split   *SplitView

	// AutoSwitch determines if selection messages automatically trigger page switches.
	// True by default.
	AutoSwitch bool
}

// NewMasterDetail creates a new MasterDetail orchestrator.
func NewMasterDetail(sidebar *SidebarList, pages *PageManager, borderStyle lipgloss.Style) *MasterDetail {
	m := &MasterDetail{
		Sidebar:    sidebar,
		Pages:      pages,
		AutoSwitch: true,
	}
	m.Split = NewSplitView(sidebar, pages, borderStyle)
	return m
}

// Init initializes children.
func (m *MasterDetail) Init() tea.Cmd {
	return tea.Batch(m.Sidebar.Init(), m.Pages.Init())
}

// Update handles orchestration and propagates messages.
func (m *MasterDetail) Update(msg tea.Msg) (viewlet.Viewlet, tea.Cmd) {
	var cmds []tea.Cmd

	// Orchestration: Listen for SelectionMsg from Sidebar (or anyone)
	if m.AutoSwitch {
		if selMsg, ok := msg.(viewlet.SelectionMsg); ok {
			// We use the item's FilterValue or a fallback as the key for the PageManager.
			// Standard Nido SidebarItem defines Title() etc.
			// We check if the item implements a specific Keyed interface or just use Title.
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

	// Delegation to SplitView (which delegates to Sidebar and Pages)
	newSplit, cmd := m.Split.Update(msg)
	m.Split = newSplit.(*SplitView)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
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
	return m.Split.HandleMouse(x, y, msg)
}

// Shortcuts delegates to active page.
func (m *MasterDetail) Shortcuts() []viewlet.Shortcut {
	return m.Pages.Shortcuts()
}

func (m *MasterDetail) Focus() tea.Cmd {
	return m.Pages.Focus()
}

func (m *MasterDetail) Blur() {
	m.Pages.Blur()
}

func (m *MasterDetail) Focused() bool {
	return m.Pages.Focused()
}
