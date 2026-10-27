package registry

import (
	"fmt"

	"github.com/Josepavese/nido/v4/internal/provider"
	"github.com/Josepavese/nido/v4/internal/tui/app/ops"
	"github.com/Josepavese/nido/v4/internal/tui/kit/layout"
	"github.com/Josepavese/nido/v4/internal/tui/kit/theme"
	view "github.com/Josepavese/nido/v4/internal/tui/kit/view"
	widget "github.com/Josepavese/nido/v4/internal/tui/kit/widget"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// RegistryItem represents an image in the sidebar (either remote or local).
type RegistryItem struct {
	Arch          string
	Name          string
	Version       string
	Provider      string // "nido", "official", or "blueprint"
	IsLocal       bool   // True = Cache, False = Remote Catalog
	DeleteName    string
	DeleteVersion string
}

func (i RegistryItem) Title() string {
	return fmt.Sprintf("%s:%s [%s]", i.Name, i.Version, i.Arch)
}

func (i RegistryItem) Description() string {
	if i.IsLocal {
		if i.Provider == "blueprint" {
			return fmt.Sprintf("Blueprint cache • %s", i.Version)
		}
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
		if i.Provider == "blueprint" {
			return theme.IconBlueprint
		}
		return theme.IconCache
	}
	if i.Provider == "blueprint" {
		return theme.IconBlueprint
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
	pendingDelete *RegistryItem
	localItems    []RegistryItem
	remoteItems   []RegistryItem
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
		Action:   t.Styles.SidebarItemSelected.Foreground(t.Palette.Accent).Bold(true),
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
			if r.pendingDelete != nil {
				item := *r.pendingDelete
				deleteName := item.DeleteName
				deleteVersion := item.DeleteVersion
				if deleteName == "" {
					deleteName = item.Name
					deleteVersion = item.Version
				}
				return func() tea.Msg {
					return ops.RequestDeleteImageMsg{Name: deleteName, Version: deleteVersion, Arch: item.Arch}
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
	selected, _ := r.Sidebar.SelectedItem().(RegistryItem)
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

	for index, item := range items {
		if candidate, ok := item.(RegistryItem); ok && candidate == selected {
			r.Sidebar.Select(index)
			r.DetailView.UpdateItem(candidate)
			return
		}
	}
	if selected.Name == "" {
		for index, item := range items {
			if candidate, ok := item.(RegistryItem); ok {
				r.Sidebar.Select(index)
				r.DetailView.UpdateItem(candidate)
				return
			}
		}
	}
	r.DetailView.UpdateItem(RegistryItem{})
}

func (r *Registry) Update(msg tea.Msg) (view.Viewlet, tea.Cmd) {
	// 1. Modal Interception
	if view.IsInputMessage(msg) && r.ConfirmDelete.IsActive() {
		newModal, cmd := r.ConfirmDelete.Update(msg)
		r.ConfirmDelete = newModal
		return r, cmd
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	// Selection commands may arrive after later input or a catalog refresh.
	case view.SelectionMsg:
		if item, ok := r.Sidebar.SelectedItem().(RegistryItem); ok {
			r.DetailView.UpdateItem(item)
		} else {
			// Header selected? Clear detail
			r.DetailView.UpdateItem(RegistryItem{})
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if !r.MasterDetail.Split.SinglePane() && r.MasterDetail.ActiveFocus == widget.FocusSidebar {
				return r, r.DetailView.pullSelected()
			}

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
				item := r.DetailView.Item
				r.pendingDelete = &item
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
				Arch:     img.Arch,
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
				Arch:          img.Arch,
				Name:          img.Name,
				Version:       img.Version,
				Provider:      img.Kind,
				IsLocal:       true,
				DeleteName:    img.DeleteName,
				DeleteVersion: img.DeleteVersion,
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
		cmd, handled := r.ConfirmDelete.HandleMouse(x, y, msg)
		return r, cmd, handled
	}
	_, cmd, handled := r.MasterDetail.HandleMouse(x, y, msg)
	return r, cmd, handled
}

func (r *Registry) Shortcuts() []view.Shortcut {
	if r.ConfirmDelete.IsActive() {
		return r.ConfirmDelete.Shortcuts()
	}

	s := r.MasterDetail.Shortcuts()

	// Contextual Shortcuts
	if item := r.Sidebar.SelectedItem(); item != nil {
		if regItem, ok := item.(RegistryItem); ok && regItem.Name != "" {
			if regItem.IsLocal {
				s = append(s, view.Shortcut{Key: "p", Label: "purge"})
				s = append(s, view.Shortcut{Key: "delete", Label: "cull"})
			} else {
				if !r.MasterDetail.Split.SinglePane() || r.MasterDetail.ActiveFocus == widget.FocusDetail {
					s = append(s, view.Shortcut{Key: "enter", Label: "pull"})
				}
			}
		}
	}

	if r.MasterDetail.ActiveFocus == widget.FocusDetail && !r.MasterDetail.Split.SinglePane() {
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
	if r.MasterDetail != nil && r.MasterDetail.ActiveFocus == widget.FocusDetail {
		return true
	}
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
		if item.Provider == "blueprint" {
			d.Header.Icon = theme.IconBlueprint
			d.SourceInput.SetValue("Blueprint Cache")
		} else {
			d.Header.Icon = theme.IconCache
			d.SourceInput.SetValue("Local Cache")
		}
	} else {
		// Use IconPackage for consistency with the list view icons
		d.Header.Icon = theme.IconPackage
		d.SourceInput.SetValue("Remote Registry")
	}
}

func (d *RegistryDetail) Init() tea.Cmd { return nil }

func (d *RegistryDetail) Update(msg tea.Msg) (view.Viewlet, tea.Cmd) {
	if !d.Focused() {
		return d, nil
	}
	if k, ok := msg.(tea.KeyMsg); ok && k.String() == "enter" && !d.Item.IsLocal && d.Item.Name != "" {
		return d, d.pullSelected()
	}
	_, cmd := d.Form.Update(msg)
	return d, cmd
}
func (d *RegistryDetail) HandleMouse(x, y int, msg tea.MouseMsg) (view.Viewlet, tea.Cmd, bool) {
	cmd, handled := d.Form.HandleMouse(x, y, msg)
	return d, cmd, handled
}

func (d *RegistryDetail) View() string {
	if d.Item.Name == "" {
		return widget.RenderEmptyState(theme.IconRegistry, "Explore the library", "Select a cached image or a remote image to see its details.", d.Width(), d.Height())
	}
	return d.Form.Viewport(layout.FormWidth(d.Width(), theme.Inset.TotalContent), d.Height())
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

func (d *RegistryDetail) pullSelected() tea.Cmd {
	if d.Item.IsLocal || d.Item.Name == "" {
		return nil
	}
	ref := fmt.Sprintf("%s:%s", d.Item.Name, d.Item.Version)
	return func() tea.Msg { return ops.RequestPullMsg{Image: ref, Arch: d.Item.Arch} }
}

func (SectionHeader) Selectable() bool { return false }
