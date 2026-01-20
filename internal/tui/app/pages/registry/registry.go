package registry

import (
	"fmt"

	"github.com/Josepavese/nido/internal/provider"
	"github.com/Josepavese/nido/internal/tui/app/ops"
	"github.com/Josepavese/nido/internal/tui/kit/layout"
	"github.com/Josepavese/nido/internal/tui/kit/theme"
	view "github.com/Josepavese/nido/internal/tui/kit/view"
	widget "github.com/Josepavese/nido/internal/tui/kit/widget"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// RegistryItem represents an image in the sidebar (either remote or local).
type RegistryItem struct {
	Name     string
	Version  string
	Provider string // "nido" or "official"
	IsLocal  bool   // True = Cache, False = Remote Catalog
}

func (i RegistryItem) Title() string {
	return fmt.Sprintf("%s:%s", i.Name, i.Version)
}

func (i RegistryItem) Description() string {
	if i.IsLocal {
		return fmt.Sprintf("Cached • %s", i.Version)
	}
	// "nido" => Flavour, "official" => Cloud
	if i.Provider == "nido" {
		return fmt.Sprintf("Flavour • %s", i.Version)
	}
	return fmt.Sprintf("Cloud • %s", i.Version)
}

func (i RegistryItem) FilterValue() string { return i.Name }
func (i RegistryItem) String() string      { return i.Name }
func (i RegistryItem) Icon() string {
	if i.IsLocal {
		return theme.IconCache
	}
	if i.Provider == "nido" {
		return theme.IconFlavour
	}
	return theme.IconPackage
}
func (i RegistryItem) IsAction() bool { return false }

// SectionHeader for splitting the list
type SectionHeader struct {
	TitleStr string
}

func (s SectionHeader) Title() string       { return s.TitleStr }
func (s SectionHeader) Description() string { return "" }
func (s SectionHeader) FilterValue() string { return "" }
func (s SectionHeader) String() string      { return s.TitleStr }
func (s SectionHeader) Icon() string        { return "" }
func (s SectionHeader) IsAction() bool      { return false } // Render as unselectable/dimmed in custom delegate if needed, or just normal item

// Registry implements the Registry Viewlet
type Registry struct {
	view.BaseViewlet

	prov provider.VMProvider

	// Components
	// Components
	Sidebar       *widget.SidebarList
	SidebarHeader *widget.Card
	DetailView    *RegistryDetail
	Pages         *widget.PageManager
	MasterDetail  *widget.MasterDetail
	ConfirmDelete *widget.Modal

	// Data
	localItems  []RegistryItem
	remoteItems []RegistryItem
}

func NewRegistry(prov provider.VMProvider) *Registry {
	r := &Registry{
		prov: prov,
	}

	// 1. Sidebar
	t := theme.Current()
	styles := widget.SidebarStyles{
		Normal:   t.Styles.SidebarItem,
		Selected: t.Styles.SidebarItemSelected,
		Dim:      lipgloss.NewStyle().Foreground(t.Palette.TextDim),
		Action:   t.Styles.SidebarItemSelected.Copy().Foreground(t.Palette.Accent).Bold(true),
	}
	r.Sidebar = widget.NewSidebarList([]widget.SidebarItem{
		SectionHeader{TitleStr: "Loading..."},
	}, theme.Width.Sidebar, styles, "") // Flush left for headers

	// 2. Detail View
	r.DetailView = NewRegistryDetail(r)
	r.Pages = widget.NewPageManager()
	r.Pages.AddPage("DETAIL", r.DetailView)
	r.Pages.SwitchTo("DETAIL")

	// 3. MasterDetail
	border := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(t.Palette.SurfaceSubtle)

	r.SidebarHeader = widget.NewCard(theme.IconRegistry, "Registry", "Manager")
	r.MasterDetail = widget.NewMasterDetail(
		widget.NewBoxedSidebar(
			r.SidebarHeader,
			r.Sidebar,
		),
		r.Pages,
		border,
	)
	r.MasterDetail.AutoSwitch = false

	// 4. Modal
	r.ConfirmDelete = widget.NewModal(
		"Delete Image",
		"Are you sure?",
		func() tea.Cmd {
			if r.DetailView.Item.Name != "" && r.DetailView.Item.IsLocal {
				item := r.DetailView.Item
				return func() tea.Msg {
					return ops.RequestDeleteImageMsg{Name: item.Name, Version: item.Version}
				}
			}
			return nil
		},
		nil,
	)
	r.ConfirmDelete.SetLevel(widget.ModalLevelDanger)

	return r
}

func (r *Registry) Init() tea.Cmd {
	// Trigger parallel load
	return tea.Batch(
		r.MasterDetail.Init(),
		r.Sidebar.Focus(),
		ops.ListCache(r.prov),                  // Load Cache
		ops.FetchRegistryImages(r.prov, false), // Load Remote
	)
}

func (r *Registry) rebuildSidebar() {
	var items []widget.SidebarItem

	// 1. Local Cache Section
	items = append(items, SectionHeader{TitleStr: "── LOCAL CACHE ──"})
	if len(r.localItems) > 0 {
		for _, it := range r.localItems {
			items = append(items, it)
		}
	} else {
		items = append(items, SectionHeader{TitleStr: "  (empty)"})
	}

	items = append(items, SectionHeader{TitleStr: ""}) // Spacer

	// 2. Remote Catalog Section
	items = append(items, SectionHeader{TitleStr: "── REMOTE CATALOG ──"})
	if len(r.remoteItems) > 0 {
		for _, it := range r.remoteItems {
			items = append(items, it)
		}
	} else {
		items = append(items, SectionHeader{TitleStr: "  loading..."})
	}

	r.Sidebar.SetItems(items)

	// Preserve selection or select first relevant?
	// For now, if nothing selected/empty, select first.
	// But rebuilding shifts indices.
	// Simplest: If nothing selected, select first item that is NOT a header (if possible) or just index 1.
	if r.Sidebar.SelectedItem() == nil && len(items) > 1 {
		r.Sidebar.Select(1) // Usually index 0 is Header
	}
}

func (r *Registry) Update(msg tea.Msg) (view.Viewlet, tea.Cmd) {
	// 1. Modal Interception
	if r.ConfirmDelete.IsActive() {
		newModal, cmd := r.ConfirmDelete.Update(msg)
		r.ConfirmDelete = newModal
		return r, cmd
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	// Sidebar Selection
	case view.SelectionMsg:
		if item, ok := msg.Item.(RegistryItem); ok {
			r.DetailView.UpdateItem(item)
		} else {
			// Header selected? Clear detail
			r.DetailView.UpdateItem(RegistryItem{})
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "p":
			// Prune Action (Global for Cache)
			// Always allowed if valid? Contextual?
			// Let's allow it if we have cache items.
			if len(r.localItems) > 0 {
				return r, func() tea.Msg { return ops.RequestPruneMsg{UnusedOnly: true} }
			}
		case "backspace", "delete":
			// Delete Action (Contextual)
			if r.DetailView.Item.Name != "" && r.DetailView.Item.IsLocal {
				r.ConfirmDelete.Message = fmt.Sprintf("Delete image '%s'?\nThis cannot be undone.", r.DetailView.Item.Name)
				r.ConfirmDelete.Show()
				return r, nil
			}
		}

	case ops.RegistryListMsg:
		if msg.Err != nil {
			return r, nil
		}
		var items []RegistryItem
		for _, img := range msg.Images {
			items = append(items, RegistryItem{
				Name:     img.Name,
				Version:  img.Version,
				Provider: img.Registry, // "nido" or "official"
				IsLocal:  false,
			})
		}
		r.remoteItems = items
		r.rebuildSidebar()

	case ops.CacheListMsg:
		if msg.Err != nil {
			return r, nil
		}
		var items []RegistryItem
		for _, img := range msg.Items {
			items = append(items, RegistryItem{
				Name:    img.Name,
				Version: img.Version,
				IsLocal: true,
			})
		}
		r.localItems = items
		r.rebuildSidebar()

	case ops.CachePruneMsg:
		// Reload cache list after prune/delete
		cmds = append(cmds, ops.ListCache(r.prov))
	}

	newMD, mdCmd := r.MasterDetail.Update(msg)
	r.MasterDetail = newMD.(*widget.MasterDetail)
	cmds = append(cmds, mdCmd)

	return r, tea.Batch(cmds...)
}

func (r *Registry) View() string {
	if r.ConfirmDelete.IsActive() {
		return r.ConfirmDelete.View(r.Width(), r.Height())
	}
	return r.MasterDetail.View()
}

func (r *Registry) Focus() tea.Cmd {
	r.BaseViewlet.Focus()
	return r.MasterDetail.Focus()
}

func (r *Registry) Resize(rect layout.Rect) {
	r.BaseViewlet.Resize(rect)
	r.MasterDetail.Resize(rect)
}

func (r *Registry) HandleMouse(x, y int, msg tea.MouseMsg) (view.Viewlet, tea.Cmd, bool) {
	if r.ConfirmDelete.IsActive() {
		return r, nil, true // Consume all mouse events if modal active
	}
	return r.MasterDetail.HandleMouse(x, y, msg)
}

func (r *Registry) Shortcuts() []view.Shortcut {
	if r.ConfirmDelete.IsActive() {
		return []view.Shortcut{
			{Key: "enter", Label: "engage"},
			{Key: "esc", Label: "back"},
		}
	}

	s := []view.Shortcut{
		{Key: "↑/↓", Label: "glide"},
	}

	// Contextual Shortcuts
	if item := r.Sidebar.SelectedItem(); item != nil {
		if regItem, ok := item.(RegistryItem); ok && regItem.Name != "" {
			if regItem.IsLocal {
				s = append(s, view.Shortcut{Key: "p", Label: "purge"})
				s = append(s, view.Shortcut{Key: "delete", Label: "cull"})
			} else {
				s = append(s, view.Shortcut{Key: "enter", Label: "pull"})
			}
		}
	}

	if r.MasterDetail.ActiveFocus == widget.FocusDetail {
		s = append(s, view.Shortcut{Key: "esc", Label: "back"})
	}

	return s
}

func (r *Registry) IsModalActive() bool {
	return r.ConfirmDelete != nil && r.ConfirmDelete.IsActive()
}

func (r *Registry) HasActiveTextInput() bool {
	if r.DetailView != nil {
		return r.DetailView.HasActiveTextInput()
	}
	return false
}

func (r *Registry) HasActiveFocus() bool {
	if r.DetailView != nil {
		return r.DetailView.HasActiveFocus()
	}
	return false
}

// --- Detail ---

type RegistryDetail struct {
	view.BaseViewlet
	Parent *Registry
	Item   RegistryItem

	// Components
	Header *widget.Card
	Form   *widget.Form

	// Fields
	NameInput    *widget.Input
	VersionInput *widget.Input
	SourceInput  *widget.Input
}

func NewRegistryDetail(parent *Registry) *RegistryDetail {
	d := &RegistryDetail{Parent: parent}
	d.Header = widget.NewCard(theme.IconPackage, "Select Image", "")

	d.NameInput = widget.NewInput("Name", "", nil)
	d.NameInput.Disabled = true
	d.VersionInput = widget.NewInput("Version", "", nil)
	d.VersionInput.Disabled = true
	d.SourceInput = widget.NewInput("Source", "", nil)
	d.SourceInput.Disabled = true

	d.Form = widget.NewForm(d.Header, d.NameInput, d.VersionInput, d.SourceInput)

	return d
}

func (d *RegistryDetail) UpdateItem(item RegistryItem) {
	d.Item = item
	d.Header.Title = item.Name
	d.Header.Subtitle = item.Version

	if item.Name == "" {
		// Empty State
		d.NameInput.SetValue("-")
		d.VersionInput.SetValue("-")
		d.SourceInput.SetValue("-")
		d.Header.Icon = theme.IconUnknown
		d.Header.Title = "No Image Selected"
		d.Header.Subtitle = ""
		return
	}

	d.NameInput.SetValue(item.Name)
	d.VersionInput.SetValue(item.Version)

	if item.IsLocal {
		d.Header.Icon = theme.IconCache
		d.SourceInput.SetValue("Local Cache")
	} else {
		// Use IconPackage for consistency with the list view icons
		d.Header.Icon = theme.IconPackage
		d.SourceInput.SetValue("Remote Registry")
	}
}

func (d *RegistryDetail) Init() tea.Cmd { return nil }

func (d *RegistryDetail) Update(msg tea.Msg) (view.Viewlet, tea.Cmd) {
	// Handle Key Input for Actions
	// Note: Shortcuts are displayed by Parent, but keys are trapped here because Detail has focus (likely)
	// Actually MasterDetail delegates focus. If Sidebar is focused, Detail gets keys? No.
	// We rely on standard propagation.

	if kmsg, ok := msg.(tea.KeyMsg); ok {
		switch kmsg.String() {
		case "enter":
			if !d.Item.IsLocal && d.Item.Name != "" {
				// Trigger Pull
				ref := fmt.Sprintf("%s:%s", d.Item.Name, d.Item.Version)
				return d, func() tea.Msg {
					return ops.RequestPullMsg{Image: ref}
				}
			}
		}
	}
	return d, nil
}

func (d *RegistryDetail) View() string {
	w := d.Width()
	// Use consistent constraints (Min 40, Max 60)
	padding := theme.Current().Layout.ContainerPadding
	safeWidth := w - (2 * padding)

	if safeWidth > 60 {
		safeWidth = 60
	}
	if safeWidth < 40 {
		safeWidth = 40
	}

	d.Form.Width = safeWidth

	return d.Form.View(safeWidth)
}

func (d *RegistryDetail) Resize(r layout.Rect) {
	d.BaseViewlet.Resize(r)
	// Resize form elements if needed? Form.View takes width.
}

func (d *RegistryDetail) IsModalActive() bool {
	return false
}

func (d *RegistryDetail) HasActiveTextInput() bool {
	return d.Form != nil && d.Form.HasActiveTextInput()
}

func (d *RegistryDetail) HasActiveFocus() bool {
	return d.Form != nil && d.Form.HasActiveFocus()
}

func (d *RegistryDetail) Focusable() bool {
	if d.Form == nil {
		return false
	}
	return d.Form.Focusable()
}
