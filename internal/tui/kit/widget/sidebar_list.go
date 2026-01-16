package widget

import (
	"fmt"
	"io"

	"github.com/Josepavese/nido/internal/tui/kit/layout"
	"github.com/Josepavese/nido/internal/tui/kit/theme"
	viewlet "github.com/Josepavese/nido/internal/tui/kit/view"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SidebarItem is the interface that items in the SidebarList must implement.
type SidebarItem interface {
	list.Item
	Title() string
	// Icon returns a short emoji or symbol for the item. Return empty string if none.
	Icon() string
	// IsAction returns true if this item represents an action (e.g. "Create New") rather than a data item.
	// Action items may be rendered with different styling (e.g. highlighted background).
	IsAction() bool
}

// SidebarItemString is a simple wrapper for using strings as SidebarItems.
type SidebarItemString string

func (s SidebarItemString) Title() string       { return string(s) }
func (s SidebarItemString) Description() string { return "" }
func (s SidebarItemString) FilterValue() string { return string(s) }
func (s SidebarItemString) Icon() string        { return "" }
func (s SidebarItemString) IsAction() bool      { return false }

// SidebarStyles defines the customization for the sidebar list.
type SidebarStyles struct {
	Normal   lipgloss.Style
	Selected lipgloss.Style
	Dim      lipgloss.Style
	Action   lipgloss.Style // Highlighted style for action items (e.g. "Create New")
}

// SidebarList is a standardized list component for sidebars.
// It enforces strict layout but accepts framework-agnostic styling.
type SidebarList struct {
	ListView // Embeds adapter logic
	focused  bool
}

// NewSidebarList creates a new SidebarList with provided configuration.
// It is the caller's responsibility to pass the standard theme values.
func NewSidebarList(items []SidebarItem, width int, styles SidebarStyles, noIconPadding string) *SidebarList {
	// Convert typed items to generic list.Item
	listItems := make([]list.Item, len(items))
	for i, it := range items {
		listItems[i] = it
	}

	d := NewSidebarDelegate(styles, noIconPadding)
	l := list.New(listItems, d, width, 0)

	// Enforce Framework Standards
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(true)
	l.SetFilteringEnabled(false) // Sidebars usually don't filter unless focused specifically

	// Initial Resize
	l.SetSize(width, 10)

	sl := &SidebarList{}
	sl.Model = new(list.Model)
	*sl.Model = l
	return sl
}

// SidebarDelegate handles rendering of sidebar items.
type SidebarDelegate struct {
	Styles        SidebarStyles
	NoIconPadding string
	height        int
	spacing       int
}

func NewSidebarDelegate(s SidebarStyles, noIconPadding string) SidebarDelegate {
	return SidebarDelegate{
		Styles:        s,
		NoIconPadding: noIconPadding,
		height:        1,
		spacing:       0,
	}
}

func (d SidebarDelegate) Height() int                             { return d.height }
func (d SidebarDelegate) Spacing() int                            { return d.spacing }
func (d SidebarDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d SidebarDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(SidebarItem)
	if !ok {
		return
	}

	// Logic: Icon + Title
	// Padding: 0 (Let lipgloss handle any internal margin if needed, but we want flush left for strict grid)
	// Actually, visually we want: "ICON NAME"

	str := item.Title()
	icon := item.Icon()

	if icon != "" {
		// Use centralized RenderIcon for robust alignment (4 chars)
		// This handles feathers, eggs, and DNA with consistent spacing.
		str = fmt.Sprintf("%s%s", theme.RenderIcon(icon), str)
	} else {
		// Use configurable padding for non-icon items
		if d.NoIconPadding != "" {
			str = d.NoIconPadding + str
		}
	}

	// Truncate to fit width
	// Use Width-3 for safety against wide emojis pushing the border
	maxW := m.Width() - 3
	if maxW > 0 && lipgloss.Width(str) > maxW {
		str = lipgloss.NewStyle().MaxWidth(maxW).Render(str)
	}

	if index == m.Index() {
		// Selected State
		if item.IsAction() {
			// Full width background for Action items (High Contrast)
			// Apply width to ensure background fills the line
			style := d.Styles.Action.Copy().Width(m.Width())
			fmt.Fprint(w, style.Render(str))
		} else {
			fmt.Fprint(w, d.Styles.Selected.Render(str))
		}
	} else {
		// Unselected State
		if item.IsAction() {
			// User wants background even when unselected to signify it's a button.
			// We'll use a subtle dimmed background.
			style := d.Styles.Normal.Copy().
				Width(m.Width()).
				Background(d.Styles.Dim.GetForeground()).   // Use Dim color for subtle background
				Foreground(d.Styles.Action.GetBackground()) // Contrast text (Background color of selected action)
			fmt.Fprint(w, style.Render(str))
		} else {
			fmt.Fprint(w, d.Styles.Normal.Render(str))
		}
	}
}

// Helper for type assertion
func (s *SidebarList) SelectedItem() SidebarItem {
	if s.Model.SelectedItem() == nil {
		return nil
	}
	return s.Model.SelectedItem().(SidebarItem)
}

func (s *SidebarList) SetItems(items []SidebarItem) {
	listItems := make([]list.Item, len(items))
	for i, it := range items {
		listItems[i] = it
	}
	s.Model.SetItems(listItems)
}

// Shortcuts returns standard navigation shortcuts
func (s *SidebarList) Shortcuts() []viewlet.Shortcut {
	return []viewlet.Shortcut{
		{Key: "↑/↓", Label: "glide"},
	}
}

// Passthrough methods for compatibility and convenience

func (s *SidebarList) Index() int {
	return s.Model.Index()
}

func (s *SidebarList) Select(index int) {
	s.Model.Select(index)
}

func (s *SidebarList) Items() []list.Item {
	return s.Model.Items()
}

// SetItemsGeneric allows setting items from generic list.Item slice,
func (s *SidebarList) SetItemsGeneric(items []list.Item) {
	s.Model.SetItems(items)
}

func (s *SidebarList) Paginator() *paginator.Model {
	return &s.Model.Paginator
}

func (s *SidebarList) Update(msg tea.Msg) (viewlet.Viewlet, tea.Cmd) {
	// Gate Key Handling: Only process keys if focused
	if _, ok := msg.(tea.KeyMsg); ok && !s.Focused() {
		return s, nil
	}
	// Gate Mouse Handling: Ignore mouse in Update (rely on HandleMouse or ignore completely)
	if _, ok := msg.(tea.MouseMsg); ok {
		return s, nil
	}

	oldIdx := s.Index()
	var cmd tea.Cmd
	newModel, cmd := s.Model.Update(msg)
	*s.Model = newModel

	if s.Index() != oldIdx {
		// Selection changed!
		cmd = tea.Batch(cmd, func() tea.Msg {
			return viewlet.SelectionMsg{Index: s.Index(), Item: s.SelectedItem()}
		})
	}
	return s, cmd
}

func (s *SidebarList) HandleMouse(x, y int, msg tea.MouseMsg) (viewlet.Viewlet, tea.Cmd, bool) {
	// Robust manual hit testing to bypass bubbles/list complexity
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return s, nil, false
	}

	// 1. Account for bubbles/list Title line
	if s.Model.Title != "" && s.Model.ShowTitle() {
		y--
	}

	// 2. Account for bubbles/list internal overhead
	if s.Model.FilteringEnabled() {
		y -= 2 // Standard filter bar
	}
	if s.Model.ShowStatusBar() {
		y--
	}

	// Avoid negative coordinates after offset compensation
	if y < 0 {
		return s, nil, false
	}

	// bubbles/list Pagination logic
	perPage := s.Model.Paginator.PerPage
	page := s.Model.Paginator.Page
	targetIdx := (page * perPage) + y

	if targetIdx >= 0 && targetIdx < len(s.Model.Items()) {
		// Auto-Focus on Click
		s.Focus()
		s.Model.Select(targetIdx)
		item := s.Model.SelectedItem()

		return s, func() tea.Msg {
			return viewlet.SelectionMsg{Index: targetIdx, Item: item}
		}, true
	}

	return s, nil, false
}

// Dummy methods to satisfy Viewlet interface if needed
func (s *SidebarList) Resize(r layout.Rect)     { s.Model.SetSize(r.Width, r.Height) }
func (s *SidebarList) Focus() tea.Cmd           { s.focused = true; return nil }
func (s *SidebarList) Blur()                    { s.focused = false }
func (s *SidebarList) Focused() bool            { return s.focused }
func (s *SidebarList) View() string             { return s.Model.View() }
func (s *SidebarList) Init() tea.Cmd            { return nil }
func (s *SidebarList) IsModalActive() bool      { return false }
func (s *SidebarList) HasActiveTextInput() bool { return false }
func (s *SidebarList) HasActiveFocus() bool     { return s.focused }
func (s *SidebarList) Focusable() bool          { return true }
