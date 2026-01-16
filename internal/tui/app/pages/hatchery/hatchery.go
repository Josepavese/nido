package hatchery

import (
	"fmt"
	"strings"

	"github.com/Josepavese/nido/internal/provider"
	"github.com/Josepavese/nido/internal/tui/app/ops"
	"github.com/Josepavese/nido/internal/tui/kit/layout"
	"github.com/Josepavese/nido/internal/tui/kit/theme"
	fv "github.com/Josepavese/nido/internal/tui/kit/view"
	widget "github.com/Josepavese/nido/internal/tui/kit/widget"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- Data Models ---

// SourceItem adapts the string-based source list to the SidebarItem interface.
type SourceItem struct {
	Raw   string
	Type  string // "TEMPLATE", "FLAVOUR", "CLOUD"
	Label string
}

func (i SourceItem) Title() string       { return i.Label }
func (i SourceItem) Description() string { return i.Type }
func (i SourceItem) FilterValue() string { return i.Raw }
func (i SourceItem) String() string      { return i.Label }
func (i SourceItem) Icon() string {
	// Use centralized theme icons
	return theme.IconForType(i.Type)
}
func (i SourceItem) IsAction() bool { return false }

// --- Viewlets ---

// Incubator is the configuration form for the new VM.
type Incubator struct {
	fv.BaseViewlet
	Parent *Hatchery

	// State
	SelectedSource *SourceItem
	Form           *widget.Form

	// Accessors for dynamic updates
	header *widget.Card
	input  *widget.Input
	toggle *widget.Toggle

	// Styles
	LabelStyle  lipgloss.Style
	ToggleStyle lipgloss.Style
	ButtonStyle lipgloss.Style
}

func NewIncubator(parent *Hatchery) *Incubator {
	t := theme.Current()

	inc := &Incubator{
		Parent:      parent,
		LabelStyle:  lipgloss.NewStyle().Foreground(t.Palette.TextDim).Width(15),
		ToggleStyle: lipgloss.NewStyle().Foreground(t.Palette.Success),
		ButtonStyle: lipgloss.NewStyle().
			Background(t.Palette.Accent).
			Foreground(t.Palette.Background).
			Bold(true).
			Padding(0, 2),
	}

	// 1. Header (Placeholder, updated on selection)
	inc.header = widget.NewCard(theme.IconTemplate, "Select Source", "TEMPLATE")

	// 2. Name Input
	inc.input = widget.NewInput("Name", "bird-name", func(s string) error {
		if len(s) < 3 {
			return fmt.Errorf("too short")
		}
		return nil
	})
	// Real-time filtering for valid VM name characters
	inc.input.Filter = func(r rune) bool {
		return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.'
	}

	// 3. Toggle
	inc.toggle = widget.NewToggle("GUI Mode", true)

	// 4. Action Button
	// "Il testo del bottone puo essere Spawn" -> Content="SPAWN"
	// "allineato a destra come le label" -> Label="Action" (Right aligned by kit)
	btn := widget.NewSubmitButton("Action", "SPAWN", func() tea.Cmd {
		return inc.submitSpawn()
	})

	inc.Form = widget.NewForm(
		inc.header,
		inc.input,
		inc.toggle,
		btn,
	)
	// Enable configurable spacing if needed, defaults to 0
	inc.Form.Spacing = 0

	return inc
}

func (i *Incubator) submitSpawn() tea.Cmd {
	if i.SelectedSource == nil {
		return nil
	}

	// Validate
	if err := i.input.Validator(i.input.Value()); err != nil {
		return nil // Form handles visual error state
	}

	// Construct Msg
	req := ops.RequestSpawnMsg{
		Name:     i.input.Value(),
		Source:   i.SelectedSource.Title(),
		GUI:      i.toggle.Checked,
		UserData: "",
	}

	// Reset
	i.input.SetValue("")

	// Return command
	return func() tea.Msg { return req }
}

func (i *Incubator) Init() tea.Cmd { return nil }
func (i *Incubator) Focus() tea.Cmd {
	i.BaseViewlet.Focus()
	return i.Form.Focus()
}
func (i *Incubator) Blur() {
	i.BaseViewlet.Blur()
	i.Form.Blur()
}

func (i *Incubator) Update(msg tea.Msg) (fv.Viewlet, tea.Cmd) {
	var cmd tea.Cmd

	// Only handle if focused
	if !i.Focused() {
		return i, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			return i, i.Form.NextField()
		case "shift+tab", "up":
			return i, i.Form.PrevField()
		}

	case tea.MouseMsg:
		// Mouse handling is tricky with generic form elements without hit testing logic in Kit.
		// For now, improved declarative form sacrifices custom mouse regions unless added to Kit.
		// User didn't strictly request mouse support for this refactor, but we should preserve it if possible.
		// Given complexity, let's rely on Keyboard for this iteration or simple click-to-focus if Form supported it.
		// Disabling custom mouse logic for this pass to ensure cleanliness.
	}

	// Delegate rest to Form
	newForm, cmd := i.Form.Update(msg)
	i.Form = newForm

	return i, cmd
}

func (i *Incubator) View() string {
	if i.SelectedSource == nil {
		return layout.Center(i.Width(), "Select a source to begin incubation.")
	}

	w := i.Width()
	padding := theme.Current().Layout.ContainerPadding
	safeWidth := w - (2 * padding)

	// Constrain max width (User Request: "fallo piu corto")
	if safeWidth > 60 {
		safeWidth = 60
	}
	if safeWidth < 40 {
		safeWidth = 40
	}

	// Pass width to Declarative Form and ActionStack
	i.Form.Width = safeWidth

	// Standardized Left Alignment (matches Registry)
	return i.Form.View(safeWidth)
}

func (i *Incubator) SetSource(item *SourceItem) {
	i.SelectedSource = item

	// Update Header Component
	i.header.Icon = item.Icon()
	i.header.Title = item.Title()
	i.header.Subtitle = string(item.Type)

	// Reset form focus to Input?
	// The Form.FocusIndex might be 0 (Header is -1/unfocusable).
	// Let's reset input.
	i.input.SetValue("")
}

func (i *Incubator) Shortcuts() []fv.Shortcut {
	return []fv.Shortcut{
		{Key: "tab", Label: "glide"},
		{Key: "enter", Label: "spawn"},
	}
}

func (i *Incubator) IsModalActive() bool {
	return false
}

func (i *Incubator) HasActiveTextInput() bool {
	return i.Form != nil && i.Form.HasActiveTextInput()
}

func (i *Incubator) HasActiveFocus() bool {
	return i.Form != nil && i.Form.HasActiveFocus()
}

// --- Main Container ---

// Hatchery implements the Wizard viewlet.
type Hatchery struct {
	fv.BaseViewlet

	// Components
	Sidebar       *widget.SidebarList
	Incubator     *Incubator
	MasterDetail  *widget.MasterDetail
	Pages         *widget.PageManager
	ConfirmDelete *widget.Modal

	prov               provider.VMProvider
	pendingDeleteForce bool
}

// NewHatchery returns a new Hatchery viewlet.
func NewHatchery(prov provider.VMProvider) *Hatchery {
	h := &Hatchery{
		prov: prov,
	}

	// 1. Sidebar (Sources)
	t := theme.Current()
	styles := widget.SidebarStyles{
		Normal:   t.Styles.SidebarItem,
		Selected: t.Styles.SidebarItemSelected,
		Dim:      lipgloss.NewStyle().Foreground(t.Palette.TextDim),
		Action:   t.Styles.SidebarItemSelected.Copy(),
	}
	h.Sidebar = widget.NewSidebarList([]widget.SidebarItem{
		SourceItem{Raw: "LOADING", Type: "INFO", Label: "Loading..."},
	}, theme.Width.Sidebar, styles, theme.RenderIcon(theme.IconHatchery))

	// 2. Incubator (Detail)
	h.Incubator = NewIncubator(h)

	// 3. Pages Wrapper
	h.Pages = widget.NewPageManager()
	h.Pages.AddPage("INCUBATOR", h.Incubator)
	h.Pages.SwitchTo("INCUBATOR")

	// 4. MasterDetail
	border := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(t.Palette.SurfaceSubtle)

	h.MasterDetail = widget.NewMasterDetail(
		widget.NewBoxedSidebar(
			widget.NewCard(theme.IconHatchery, "Sources", "Catalog"),
			h.Sidebar,
		),
		h.Pages, border,
	)
	h.MasterDetail.AutoSwitch = false

	// 5. Modal
	h.ConfirmDelete = widget.NewModal(
		"Delete Template",
		"Are you sure?", // Text updated dynamically
		func() tea.Cmd {
			if item := h.Sidebar.SelectedItem(); item != nil {
				if srcItem, ok := item.(SourceItem); ok && srcItem.Type == "TEMPLATE" {
					return func() tea.Msg {
						// Pass the force flag determined by the check
						return ops.RequestDeleteTemplateMsg{Name: srcItem.Label, Force: h.pendingDeleteForce}
					}
				}
			}
			return nil
		},
		nil,
	)

	return h
}

func (h *Hatchery) Init() tea.Cmd {
	return tea.Batch(
		h.MasterDetail.Init(),
		h.Sidebar.Focus(), // Initially focus Sidebar
		// 1. Fast Load (Cache Only) -> Shows data instantly
		ops.FetchSources(h.prov, ops.SourceActionSpawn, true, false),
		// 2. Background Refresh (Force Remote) -> Updates UI later
		ops.FetchSources(h.prov, ops.SourceActionSpawn, false, true),
	)
}

func (h *Hatchery) Update(msg tea.Msg) (fv.Viewlet, tea.Cmd) {
	// 1. Modal Interception (Blocking)
	if h.ConfirmDelete.IsActive() {
		newModal, cmd := h.ConfirmDelete.Update(msg)
		h.ConfirmDelete = newModal
		return h, cmd
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case ops.TemplateUsageMsg:
		if msg.Err != nil {
			// If check failed, assume not used but show error? Or just proceed with standard modal?
			// Let's safe-fail to standard modal but warn in logs if we had them.
			h.pendingDeleteForce = false
			h.ConfirmDelete.Message = fmt.Sprintf("Delete template '%s'?\n(Usage check failed: %v)", msg.Name, msg.Err)
			h.ConfirmDelete.Show()
			return h, nil
		}

		if msg.InUse {
			h.pendingDeleteForce = true
			h.ConfirmDelete.Title = "CRITICAL WARNING"
			h.ConfirmDelete.Message = fmt.Sprintf("Template '%s' is IN USE by %d VM(s)!\n\nDeleting it will BREAK those VMs.\nAre you absolutely sure?", msg.Name, len(msg.UsedBy))
			// Ideally we'd color this red or something, but the widget.Modal is simple.
			// The content is enough.
		} else {
			h.pendingDeleteForce = false
			h.ConfirmDelete.Title = "Delete Template"
			h.ConfirmDelete.Message = fmt.Sprintf("Delete template '%s'?\nThis cannot be undone.", msg.Name)
		}
		h.ConfirmDelete.Show()
		return h, nil

	// 1. Data Loaded
	case ops.SourcesLoadedMsg:
		if msg.Err != nil {
			// Show error if we are in loading state OR empty
			items := h.Sidebar.Items()
			isLoading := false
			if len(items) > 0 {
				if s, ok := items[0].(SourceItem); ok {
					if s.Raw == "LOADING" {
						isLoading = true
					}
				}
			} else {
				// Empty list means we can show error safely
				isLoading = true
			}

			if isLoading {
				errItem := SourceItem{
					Raw:   fmt.Sprintf("[ERROR] %v", msg.Err),
					Type:  "ERROR",
					Label: fmt.Sprintf("Error: %v", msg.Err),
				}
				h.Sidebar.SetItems([]widget.SidebarItem{errItem})
			}
		} else {
			// We got data!
			items := make([]widget.SidebarItem, 0)
			for _, s := range msg.Sources {
				// Parse "TYPE Name" or "[TYPE] Name"
				// Clean up prefixes for display, keeping raw type for Icon()
				parts := strings.SplitN(s, " ", 2)
				if len(parts) == 2 {
					sType := strings.Trim(parts[0], "[]")
					sName := parts[1]
					items = append(items, SourceItem{Raw: s, Type: sType, Label: sName})
				}
			}
			if len(items) > 0 {
				h.Sidebar.SetItems(items)
				// Initial Selection: Auto-select first item
				if first, ok := items[0].(SourceItem); ok {
					h.Incubator.SetSource(&first)
				}
			} else {
				emptyItem := SourceItem{
					Raw:   "No sources found",
					Type:  "INFO",
					Label: "No sources found",
				}
				h.Sidebar.SetItems([]widget.SidebarItem{emptyItem})
			}
		}

	// 2. Sidebar Selection
	case fv.SelectionMsg:
		if item, ok := msg.Item.(SourceItem); ok {
			h.Incubator.SetSource(&item)
			// Remove focus switching: Navigation (arrows) should ONLY update content.
			// Focus switch is explicit via Tab/Click.
		}

	// 3. Spawn Request

	case tea.KeyMsg:
		// Focus Management: Tab / Shift+Tab / Esc
		if h.Sidebar.Focused() && msg.String() == "enter" {
			cmds = append(cmds, h.MasterDetail.SetFocus(widget.FocusDetail))
		}

		// Delete Template Action - NOW CHECKS USAGE FIRST
		if h.Sidebar.Focused() && (msg.String() == "delete" || msg.String() == "backspace") {
			if item := h.Sidebar.SelectedItem(); item != nil {
				// Only delete templates
				srcItem, ok := item.(SourceItem)
				if ok && srcItem.Type == "TEMPLATE" {
					// Async check
					return h, ops.CheckTemplateUsage(h.prov, srcItem.Label)
				}
			}
		}
	}

	// Delegate to MasterDetail
	newMD, cmd := h.MasterDetail.Update(msg)
	h.MasterDetail = newMD.(*widget.MasterDetail)
	cmds = append(cmds, cmd)

	return h, tea.Batch(cmds...)
}

func (h *Hatchery) View() string {
	// If Modal is Active, we show it INSTEAD of the content (Blocking Overlay)
	if h.ConfirmDelete.IsActive() {
		return h.ConfirmDelete.View(h.Width(), h.Height())
	}
	return h.MasterDetail.View()
}

func (h *Hatchery) Resize(r layout.Rect) {
	h.BaseViewlet.Resize(r)
	h.MasterDetail.Resize(r)
}

func (h *Hatchery) HandleMouse(x, y int, msg tea.MouseMsg) (fv.Viewlet, tea.Cmd, bool) {
	if h.ConfirmDelete.IsActive() {
		return h, nil, true // Consume all mouse events if modal active to block interaction
	}
	return h.MasterDetail.HandleMouse(x, y, msg)
}

func (h *Hatchery) Focus() tea.Cmd {
	return h.MasterDetail.Focus()
}

func (h *Hatchery) Shortcuts() []fv.Shortcut {
	if h.ConfirmDelete.IsActive() {
		return []fv.Shortcut{
			{Key: "enter", Label: "engage"},
			{Key: "esc", Label: "back"},
		}
	}

	shortcuts := []fv.Shortcut{
		{Key: "↑/↓", Label: "glide"},
	}

	// Delegate to MasterDetail for local pane shortcuts
	// But Hatchery.Shortcuts overrides it, so we manually merge or pick.
	// We want to combine Sidebar glide with Incubator actions if possible.
	if h.MasterDetail.ActiveFocus == widget.FocusDetail {
		shortcuts = append(shortcuts, h.Incubator.Shortcuts()...)
		shortcuts = append(shortcuts, fv.Shortcut{Key: "esc", Label: "back"})
	} else {
		shortcuts = append(shortcuts, fv.Shortcut{Key: "enter", Label: "engage"})
	}

	// Contextual Actions
	if item := h.Sidebar.SelectedItem(); item != nil {
		if srcItem, ok := item.(SourceItem); ok && srcItem.Type == "TEMPLATE" {
			shortcuts = append(shortcuts, fv.Shortcut{Key: "delete", Label: "cull"})
		}
	}

	return shortcuts
}

func (h *Hatchery) IsModalActive() bool {
	return h.ConfirmDelete != nil && h.ConfirmDelete.IsActive()
}

func (h *Hatchery) HasActiveTextInput() bool {
	if h.Incubator != nil {
		return h.Incubator.HasActiveTextInput()
	}
	return false
}

func (h *Hatchery) HasActiveFocus() bool {
	if h.Incubator != nil {
		return h.Incubator.HasActiveFocus()
	}
	return false
}

// --- Wrapper for Sidebar with Header ---

// End of Hatchery Viewlet
