package widget

import (
	"github.com/Josepavese/nido/v5/internal/tui/kit/layout"
	"github.com/Josepavese/nido/v5/internal/tui/kit/theme"
	"github.com/Josepavese/nido/v5/internal/tui/kit/view"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ListModal is a modal that displays a scrollable list of items.
type ListModal struct {
	Title       string
	List        list.Model
	OnSelect    func(list.Item) tea.Cmd
	OnHighlight func(list.Item) // Optional Live Preview
	OnCancel    func() tea.Cmd
	Active      bool
	Width       int
	Height      int
	BorderColor lipgloss.TerminalColor // Optional override
	listRect    layout.Rect
}

// NewListModal creates a new modal with a list.
func NewListModal(title string, items []list.Item, width, height int, onSelect func(list.Item) tea.Cmd, onCancel func() tea.Cmd) *ListModal {
	// Configure List to match Sidebar styling
	t := theme.Current()
	styles := SidebarStyles{
		Normal:   t.Styles.SidebarItem,
		Selected: t.Styles.SidebarItemSelected,
		Dim:      lipgloss.NewStyle().Foreground(t.Palette.TextDim),
		Action:   t.Styles.SidebarItemSelected,
	}

	// Use SidebarDelegate for consistent rendering
	// We use "  " as padding for items without icons to align nicely
	d := NewSidebarDelegate(styles, "  ")

	l := list.New(items, d, width-4, height-4)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false) // Disable filtering for simple selection, or enable if needed

	return &ListModal{
		Title:    title,
		List:     l,
		Width:    width,
		Height:   height,
		OnSelect: onSelect,
		OnCancel: onCancel,
	}
}

func (m *ListModal) Show() {
	m.Active = true
}

func (m *ListModal) Hide() {
	m.Active = false
}

func (m *ListModal) IsActive() bool {
	return m.Active
}

func (m *ListModal) Update(msg tea.Msg) (*ListModal, tea.Cmd) {
	if !m.Active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.Hide()
			if m.OnCancel != nil {
				return m, m.OnCancel()
			}
			return m, nil
		case "enter":
			selected := m.List.SelectedItem()
			if selected != nil {
				m.Hide()
				if m.OnSelect != nil {
					return m, m.OnSelect(selected)
				}
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	prevIdx := m.List.Index()
	m.List, cmd = m.List.Update(msg)

	// Live Preview: If selection changed, apply theme immediately
	if m.List.Index() != prevIdx {
		if m.OnHighlight != nil {
			if item := m.List.SelectedItem(); item != nil {
				m.OnHighlight(item)
			}
		}
	}

	return m, cmd
}

func (m *ListModal) HandleMouse(x, y int, msg tea.MouseMsg) (tea.Cmd, bool) {
	if !m.Active || !m.listRect.Contains(x, y) {
		return nil, m.Active
	}
	previous := m.List.Index()
	adapter := &SidebarList{ListView: ListView{Model: &m.List}, focused: true}
	_, _, handled := adapter.HandleMouse(x-m.listRect.X, y-m.listRect.Y, msg)
	if m.List.Index() != previous && m.OnHighlight != nil {
		m.OnHighlight(m.List.SelectedItem())
	}
	return nil, handled

}

// View replicates the BoxedSidebar layout: Header Card + Styled List Box
// View renders the modal with a standard single frame.
func (m *ListModal) View(parentWidth, parentHeight int) string {
	if !m.Active {
		return ""
	}

	t := theme.Current()

	// 1. Refresh Styles (Dynamic Theme)
	// We must recreate the delegate styles because the theme might have changed.
	sidebarStyles := SidebarStyles{
		Normal: t.Styles.SidebarItem.UnsetBackground().Foreground(t.Palette.Text), // Ensure no background for normal items and force text color
		Selected: t.Styles.SidebarItemSelected.
			UnsetBackground().            // Remove background
			Foreground(t.Palette.Accent). // Use Active/Accent color for text
			Bold(true),
		Dim:    lipgloss.NewStyle().Foreground(t.Palette.TextDim),
		Action: t.Styles.SidebarItemSelected,
	}

	// Update the delegate
	d := NewSidebarDelegate(sidebarStyles, "  ")
	m.List.SetDelegate(d)

	// 2. Define Frame Style matching Widget.Modal logic
	// Default to Accent color to match standard Modals (unless overridden)
	var borderColor lipgloss.TerminalColor = t.Palette.Accent
	if m.BorderColor != nil {
		borderColor = m.BorderColor
	}

	naturalHeight := max(1, len(m.List.Items())) + layout.DialogHeadingHeight + layout.DialogBorder + 2*layout.DialogPaddingY
	dbox := measureDialog(parentWidth, parentHeight, m.Width, min(m.Height, naturalHeight))
	title := layout.Fit(dbox.inner.Width, 1, t.Styles.Title.Render(m.Title))
	resizeList(&m.List, dbox.inner.Width, max(1, dbox.inner.Height-layout.DialogHeadingHeight))
	m.listRect = layout.NewRect(dbox.inner.X, dbox.inner.Y+layout.DialogHeadingHeight, dbox.inner.Width, dbox.inner.Height-layout.DialogHeadingHeight)
	content := title + "\n" + renderList(&m.List, dbox.inner.Width, m.listRect.Height)
	return dbox.render(parentWidth, parentHeight, content, borderColor)
}

// SimpleListItem is a helper for simple string lists
type SimpleListItem string

func (i SimpleListItem) FilterValue() string { return string(i) }
func (i SimpleListItem) Title() string       { return string(i) }
func (i SimpleListItem) Description() string { return "" }
func (i SimpleListItem) Icon() string        { return "" }
func (i SimpleListItem) IsAction() bool      { return false }

func (m *ListModal) Shortcuts() []view.Shortcut {
	return []view.Shortcut{{Key: "↑/↓", Label: "choose"}, {Key: "enter", Label: "select"}, {Key: "esc", Label: "cancel"}}
}
