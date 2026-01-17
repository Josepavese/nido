package widget

import (
	"github.com/Josepavese/nido/internal/tui/kit/layout"
	"github.com/Josepavese/nido/internal/tui/kit/theme"
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
}

// NewListModal creates a new modal with a list.
func NewListModal(title string, items []list.Item, width, height int, onSelect func(list.Item) tea.Cmd, onCancel func() tea.Cmd) *ListModal {
	// Configure List to match Sidebar styling
	t := theme.Current()
	styles := SidebarStyles{
		Normal:   t.Styles.SidebarItem,
		Selected: t.Styles.SidebarItemSelected,
		Dim:      lipgloss.NewStyle().Foreground(t.Palette.TextDim),
		Action:   t.Styles.SidebarItemSelected.Copy(),
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
	// Mouse support for list not fully implemented in this simple modal wrapper
	// but could be passed through.
	return nil, false // Consume click if in bounds?
}

// View replicates the BoxedSidebar layout: Header Card + Styled List Box
// View renders the modal with a standard single frame.
func (m *ListModal) View() string {
	if !m.Active {
		return ""
	}

	t := theme.Current()

	// 1. Refresh Styles (Dynamic Theme)
	// We must recreate the delegate styles because the theme might have changed.
	sidebarStyles := SidebarStyles{
		Normal: t.Styles.SidebarItem.Copy().UnsetBackground().Foreground(t.Palette.Text), // Ensure no background for normal items and force text color
		Selected: t.Styles.SidebarItemSelected.Copy().
			UnsetBackground().            // Remove background
			Foreground(t.Palette.Accent). // Use Active/Accent color for text
			Bold(true),
		Dim:    lipgloss.NewStyle().Foreground(t.Palette.TextDim),
		Action: t.Styles.SidebarItemSelected.Copy(),
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

	// 2.5 Update Paginator Styles (Reactive)
	m.List.Paginator.ActiveDot = lipgloss.NewStyle().Foreground(t.Palette.Accent).Render("•")
	m.List.Paginator.InactiveDot = lipgloss.NewStyle().Foreground(t.Palette.SurfaceHighlight).Render("•")

	// Calculate correct dimensions accounting for border and padding

	// ... logic remains same ...
	styleHeight := m.Height - 4
	styleWidth := m.Width - 6

	if styleHeight < 1 {
		styleHeight = 1
	}
	if styleWidth < 1 {
		styleWidth = 1
	}

	borderStyle := t.Styles.Border.Copy().
		BorderForeground(borderColor).
		Width(styleWidth).
		// Height(styleHeight). // Removed to let border wrap content naturally (prevents bottom clipping)
		Padding(1, 2).
		Align(lipgloss.Left) // Left align content (User request)

	// 3. Inner Content Layout
	const titleH = 1
	const spacerH = 1

	listH := styleHeight - titleH - spacerH
	listW := styleWidth

	if listH < 1 {
		listH = 1
	}

	// 4. Resize List to fit exactly
	m.List.SetSize(listW, listH)

	// 5. Render Components
	// Title needs to be cleanly rendered without background
	titleStyle := t.Styles.Title.Copy().
		Padding(0, 1) // Keep padding but remove background

	// Center title within the box width manually since we are Left aligning the container
	title := layout.Center(listW, titleStyle.Render(m.Title))

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"", // Spacer
		m.List.View(),
	)

	return borderStyle.Render(content)
}

func (m *ListModal) Resize(w, h int) {
	// Optional: responsive resize logic
	// m.Width = w / 2
	// m.Height = h / 2
	// m.List.SetSize(m.Width-4, m.Height-4)
}

// SimpleListItem is a helper for simple string lists
type SimpleListItem string

func (i SimpleListItem) FilterValue() string { return string(i) }
func (i SimpleListItem) Title() string       { return string(i) }
func (i SimpleListItem) Description() string { return "" }
func (i SimpleListItem) Icon() string        { return "" }
func (i SimpleListItem) IsAction() bool      { return false }
