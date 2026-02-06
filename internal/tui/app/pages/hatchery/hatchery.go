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
	"github.com/charmbracelet/bubbles/list"
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

// AccelItem adapts provider.Accelerator to the List interface.
type AccelItem struct {
	Acc provider.Accelerator
}

func (i AccelItem) Title() string {
	// [ID] Class (Vendor:Device)
	return fmt.Sprintf("[%s] %s", i.Acc.ID, i.Acc.Class)
}
func (i AccelItem) Description() string {
	status := "Safe"
	if !i.Acc.IsSafe {
		status = "UNSAFE: " + i.Acc.Warning
	}
	return fmt.Sprintf("%s | Grp: %s", status, i.Acc.IOMMUGroup)
}
func (i AccelItem) FilterValue() string { return i.Acc.ID + " " + i.Acc.Class }
func (i AccelItem) String() string      { return i.Title() }
func (i AccelItem) Icon() string        { return theme.IconHatchery } // Generic chip icon?
func (i AccelItem) IsAction() bool      { return false }

// --- Viewlets ---

// Incubator is the configuration form for the new VM.
type Incubator struct {
	fv.BaseViewlet
	Parent *Hatchery

	// State
	SelectedSource *SourceItem
	Form           *widget.Form
	PendingPorts   []provider.PortForward
	PendingAccel   string // ID of selected accelerator

	// Accessors for dynamic updates
	header       *widget.Card
	input        *widget.Input
	memInput     *widget.Input
	cpuInput     *widget.Input
	rawArgsInput *widget.Input
	accelSelect  *widget.Select // New widget
	addPortBtn   *widget.Button
	toggle       *widget.Toggle
	spawnBtn     *widget.Button

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
		if parent.ExistingVMs[s] {
			return fmt.Errorf("already exists")
		}
		return nil
	})
	// Real-time filtering for valid VM name characters
	inc.input.Filter = widget.FilterHostName

	// 3. Resource Inputs (Memory & CPUs)
	inc.memInput = widget.NewInput("Memory (MB)", "2048", nil)
	inc.memInput.Filter = widget.FilterNumber
	inc.cpuInput = widget.NewInput("vCPUs", "2", nil)
	inc.cpuInput.Filter = widget.FilterNumber

	// 3b. Raw QEMU Args (Advanced)
	// 3b. Raw QEMU Args (Advanced)
	inc.rawArgsInput = widget.NewInput("QEMU args", "-device usb-host,...", nil)

	// 3c. Accelerator Select
	inc.accelSelect = widget.NewSelect("Accelerator", "None", func() tea.Cmd {
		return parent.OpenAcceleratorModal()
	})

	// 4. Ports List (Read Only) - INTEGRATED INTO FORM
	// ... (no changes to addPortBtn)
	inc.addPortBtn = widget.NewButton("Ports", "Add Forwarding", func() tea.Cmd {
		return inc.Parent.OpenAddPortModal()
	})
	inc.addPortBtn.Centered = true

	// 5. Toggle
	inc.toggle = widget.NewToggle("GUI Mode", true)

	// 5. Action Button
	inc.spawnBtn = widget.NewSubmitButton("Action", "SPAWN", func() tea.Cmd {
		return inc.submitSpawn()
	})

	inc.rebuildForm()

	return inc
}

func (i *Incubator) rebuildForm() {
	var elements []widget.Element

	// 1. Fixed Top
	elements = append(elements, i.header)
	elements = append(elements, i.input)

	// 2. Resources (Memory & CPUs) - 50/50 split
	elements = append(elements, widget.NewRow(i.memInput, i.cpuInput))

	// 2c. Accelerator (Full Width)
	elements = append(elements, i.accelSelect)

	// 2b. Raw Args (Full Width)
	elements = append(elements, i.rawArgsInput)

	// 3. GUI Toggle + Add Port Btn (Weighted 1:1 for equal 50/50 split)
	elements = append(elements, widget.NewRowWithWeights([]widget.Element{i.toggle, i.addPortBtn}, []int{1, 1}))

	// 3. Dynamic Ports (if any)
	if len(i.PendingPorts) > 0 {
		for idx, p := range i.PendingPorts {
			// Col 1: Label + Proto
			lbl := p.Label
			if lbl == "" {
				lbl = "-"
			}
			proto := strings.ToUpper(p.Protocol)
			if proto != "" {
				lbl = fmt.Sprintf("%s (%s)", lbl, proto)
			}
			btnInfo := widget.NewButton("", lbl, nil)
			btnInfo.Disabled = true
			btnInfo.Centered = true

			// Col 2: Host:Guest
			hostVal := "Auto"
			if p.HostPort > 0 {
				hostVal = fmt.Sprint(p.HostPort)
			}
			val := fmt.Sprintf("%s : %d", hostVal, p.GuestPort)
			btnValue := widget.NewButton("", val, nil)
			btnValue.Disabled = true
			btnValue.Centered = true

			// Col 3: Delete Action
			id := idx // capture for closure
			btnDel := widget.NewButton("", "DEL", func() tea.Cmd {
				return i.Parent.OpenDeletePortModal(id)
			})
			btnDel.Centered = true
			btnDel.Role = widget.RoleDanger // Red when focused

			// Add as a 3-column row
			// Weights: 2, 2, 1 (Delete button smaller)
			elements = append(elements, widget.NewRowWithWeights([]widget.Element{btnInfo, btnValue, btnDel}, []int{2, 2, 1}))
		}
	}

	// 4. Action Button
	elements = append(elements, i.spawnBtn)

	// Preserve width if form already exists
	w := 0
	if i.Form != nil {
		w = i.Form.Width
	}

	i.Form = widget.NewForm(elements...)
	i.Form.Width = w
	i.Form.Spacing = 0
}

func (i *Incubator) submitSpawn() tea.Cmd {
	if i.SelectedSource == nil {
		return nil
	}

	// Validate
	if err := i.input.Validator(i.input.Value()); err != nil {
		return nil // Form handles visual error state
	}

	mem, _ := provider.ParseInt(i.memInput.Value())
	cpus, _ := provider.ParseInt(i.cpuInput.Value())

	// Split raw args by space (naive but sufficient for TUI)
	rawArgs := strings.Fields(i.rawArgsInput.Value())

	// Construct Msg
	req := ops.RequestSpawnMsg{
		Name:        i.input.Value(),
		Source:      i.SelectedSource.Title(),
		GUI:         i.toggle.Checked,
		MemoryMB:    mem,
		VCPUs:       cpus,
		UserData:    "",
		Ports:       i.PendingPorts,
		RawQemuArgs: rawArgs,
	}

	// Reset
	i.input.SetValue("")
	i.PendingPorts = nil
	i.rebuildForm()

	// Return command
	return func() tea.Msg { return req }
}

func (i *Incubator) AddPort(p provider.PortForward) {
	i.PendingPorts = append(i.PendingPorts, p)
	i.rebuildForm()
}

func (i *Incubator) RemovePort(index int) {
	if index < 0 || index >= len(i.PendingPorts) {
		return
	}
	i.PendingPorts = append(i.PendingPorts[:index], i.PendingPorts[index+1:]...)
	i.rebuildForm()
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

	// Navigation handled by Form or Parent
	// Capture current form pointer to detect if it changes during Update (e.g. by submitSpawn)
	formBefore := i.Form

	newForm, cmd := i.Form.Update(msg)

	// Only assign the result of Update if the form pointer hasn't been replaced by an action
	if i.Form == formBefore {
		i.Form = newForm
	}

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

	i.input.SetValue("")
	i.PendingPorts = nil
	i.rawArgsInput.SetValue("")
	i.rebuildForm()
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
	ModalAddPort  *widget.FormModal // New Form Modal
	ModalAccel    *widget.ListModal // Accelerator Selection

	prov                     provider.VMProvider
	pendingDeleteForce       bool
	ConfirmDeletePort        *widget.Modal
	PendingPortToDeleteIndex int
	ExistingVMs              map[string]bool // Cache of existing names
	PendingSelection         string          // Name of the source to auto-select on next load
}

// NewHatchery returns a new Hatchery viewlet.
func NewHatchery(prov provider.VMProvider) *Hatchery {
	h := &Hatchery{
		prov:        prov,
		ExistingVMs: make(map[string]bool),
	}

	// 1. Sidebar (Sources)
	t := theme.Current()
	styles := widget.SidebarStyles{
		Normal:   t.Styles.SidebarItem,
		Selected: t.Styles.SidebarItemSelected,
		Dim:      lipgloss.NewStyle().Foreground(t.Palette.TextDim),
		Action:   t.Styles.SidebarItemSelected,
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

	// 5. Modals
	h.ConfirmDelete = widget.NewModal(
		"Delete Template",
		"Are you sure?",
		func() tea.Cmd {
			if item := h.Sidebar.SelectedItem(); item != nil {
				if srcItem, ok := item.(SourceItem); ok && srcItem.Type == "TEMPLATE" {
					return func() tea.Msg {
						return ops.RequestDeleteTemplateMsg{Name: srcItem.Label, Force: h.pendingDeleteForce}
					}
				}
			}
			return nil
		},
		nil,
	)
	h.ConfirmDelete.SetLevel(widget.ModalLevelDanger)

	h.ConfirmDeletePort = widget.NewModal(
		"Remove Port",
		"",
		func() tea.Cmd {
			h.Incubator.RemovePort(h.PendingPortToDeleteIndex)
			return nil
		},
		nil,
	)
	h.ConfirmDeletePort.SetLevel(widget.ModalLevelDanger)

	// Add Port Form Config
	h.ModalAddPort = widget.NewFormModal("Add Port Forwarding", func(res map[string]string) tea.Cmd {
		// Parse result
		lbl := res["label"]
		gp, _ := provider.ParseInt(res["guest"]) // Validated
		hp, _ := provider.ParseInt(res["host"])  // Validated (0 if empty)
		proto := res["proto"]

		pf := provider.PortForward{
			GuestPort: gp,
			HostPort:  hp,
			Protocol:  proto,
			Label:     lbl,
		}

		h.Incubator.AddPort(pf)
		return nil
	}, nil)

	// Row 1: Label
	// Row 1: Label (2/3) + Proto (1/3)
	h.ModalAddPort.AddRow(
		&widget.FormEntry{
			Key:         "label",
			Label:       "Label",
			Placeholder: "e.g. web-admin",
			Width:       20,
			MaxChars:    20,
			Filter:      widget.FilterLabel, // Block spaces and weird chars
			Validator: func(s string) error {
				if len(s) > 20 {
					return fmt.Errorf("too long")
				}
				// Space check redundant due to Filter, but safe to keep
				if strings.Contains(s, " ") {
					return fmt.Errorf("no spaces")
				}
				return nil
			},
		},
		&widget.FormEntry{
			Key: "proto", Label: "Proto", Placeholder: "tcp", Width: 10, MaxChars: 4,
			Filter:    widget.FilterAlphaNumeric, // Only letters/nums
			Validator: nil,                       // Allow free typing (filter handles safety)
		},
	)

	// Row 2: Ports (Guest + Host)
	h.ModalAddPort.AddRow(
		&widget.FormEntry{
			Key: "guest", Label: "Guest*", Placeholder: "8080", MaxChars: 5,
			Filter: widget.FilterNumber, // Digits only
			Validator: func(s string) error {
				if s == "" {
					return fmt.Errorf("required")
				}
				v, err := provider.ParseInt(s)
				if err != nil {
					return fmt.Errorf("number")
				}
				if v < 1 || v > 65535 {
					return fmt.Errorf("range")
				}
				return nil
			},
		},
		&widget.FormEntry{
			Key: "host", Label: "Host", Placeholder: "Auto", MaxChars: 5,
			Filter: widget.FilterNumber, // Digits only
			Validator: func(s string) error {
				if s == "" {
					return nil
				}
				if _, err := provider.ParseInt(s); err != nil {
					return fmt.Errorf("number")
				}
				return nil
			},
		},
	)

	// Accelerator Modal
	h.ModalAccel = widget.NewListModal("Select Accelerator", nil, 60, 20, func(item list.Item) tea.Cmd {
		if ai, ok := item.(AccelItem); ok {
			// Update Incubator
			h.Incubator.PendingAccel = ai.Acc.ID
			h.Incubator.accelSelect.Value = fmt.Sprintf("%s (%s)", ai.Acc.ID, ai.Acc.Class)
			h.Incubator.rebuildForm()
		} else {
			// Handle "None"
			h.Incubator.PendingAccel = ""
			h.Incubator.accelSelect.Value = "None"
			h.Incubator.rebuildForm()
		}
		return nil
	}, nil)

	return h
}

func (h *Hatchery) OpenAcceleratorModal() tea.Cmd {
	return func() tea.Msg {
		// Fetch accelerators dynamically
		devs, err := h.prov.ListAccelerators()
		if err != nil {
			return ops.AcceleratorListMsg{Err: err}
		}

		var items []list.Item
		// Add "None" option (implicitly handled by clearing if selected, or explicit item?)
		// For now simple list.

		for _, d := range devs {
			items = append(items, AccelItem{Acc: d})
		}
		return ops.AcceleratorListMsg{Items: items}
	}
}

func (h *Hatchery) OpenDeletePortModal(index int) tea.Cmd {
	if index < 0 || index >= len(h.Incubator.PendingPorts) {
		return nil // Safety check
	}
	h.PendingPortToDeleteIndex = index
	p := h.Incubator.PendingPorts[index]
	label := p.Label
	if label == "" {
		label = fmt.Sprintf("Port %d", p.GuestPort)
	}
	h.ConfirmDeletePort.Message = fmt.Sprintf("Remove port forwarding for '%s'?", label)
	h.ConfirmDeletePort.Show()
	return nil
}

func (i *Incubator) Reset() {
	i.input.SetValue("")
	i.PendingPorts = nil
	i.rawArgsInput.SetValue("")
	i.toggle.Checked = true // Default to GUI on
	i.rebuildForm()
}

func (h *Hatchery) SetPendingSelection(name string) {
	h.PendingSelection = name
}

func (h *Hatchery) OpenAddPortModal() tea.Cmd {
	return h.ModalAddPort.Show()
}

func (h *Hatchery) Init() tea.Cmd {
	return tea.Batch(
		h.MasterDetail.Init(),
		h.Sidebar.Focus(),
		ops.FetchSources(h.prov, ops.SourceActionSpawn, true, false),
		ops.FetchSources(h.prov, ops.SourceActionSpawn, false, true),
		ops.RefreshFleet(h.prov), // Fetch fleet for validation
	)
}

func (h *Hatchery) Update(msg tea.Msg) (fv.Viewlet, tea.Cmd) {
	// 1. Modals Interception (Blocking)
	if h.ConfirmDelete.IsActive() {
		newModal, cmd := h.ConfirmDelete.Update(msg)
		h.ConfirmDelete = newModal
		return h, cmd
	}

	if h.ConfirmDeletePort.IsActive() {
		newModal, cmd := h.ConfirmDeletePort.Update(msg)
		h.ConfirmDeletePort = newModal
		return h, cmd
	}

	if h.ModalAddPort.IsActive() {
		_, cmd := h.ModalAddPort.Update(msg)
		// No need to reassign ptr as it modifies internal state of struct
		return h, cmd
	}

	if h.ModalAccel.IsActive() {
		_, cmd := h.ModalAccel.Update(msg)
		return h, cmd
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case ops.VMListMsg:
		if msg.Err == nil {
			// Update cache of existing names
			h.ExistingVMs = make(map[string]bool)
			for _, item := range msg.Items {
				h.ExistingVMs[item.Name] = true
			}
		}

	case ops.AcceleratorListMsg:
		if msg.Err != nil {
			// TODO: Error modal?
			return h, nil
		}
		fItems := make([]list.Item, len(msg.Items))
		copy(fItems, msg.Items)
		h.ModalAccel.List.SetItems(fItems)
		h.ModalAccel.Show()
		return h, nil

	case ops.TemplateUsageMsg:
		if msg.Err != nil {
			h.pendingDeleteForce = false
			h.ConfirmDelete.Message = fmt.Sprintf("Delete template '%s'?\n(Usage check failed: %v)", msg.Name, msg.Err)
			h.ConfirmDelete.Show()
			return h, nil
		}

		if msg.InUse {
			h.pendingDeleteForce = true
			h.ConfirmDelete.Title = "CRITICAL WARNING"
			h.ConfirmDelete.Message = fmt.Sprintf("Template '%s' is IN USE by %d VM(s)!\n\nDeleting it will BREAK those VMs.\nAre you absolutely sure?", msg.Name, len(msg.UsedBy))
		} else {
			h.pendingDeleteForce = false
			h.ConfirmDelete.Title = "Delete Template"
			h.ConfirmDelete.Message = fmt.Sprintf("Delete template '%s'?\nThis cannot be undone.", msg.Name)
		}
		h.ConfirmDelete.Show()
		return h, nil

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
				parts := strings.SplitN(s, " ", 2)
				if len(parts) == 2 {
					sType := strings.Trim(parts[0], "[]")
					sName := parts[1]
					items = append(items, SourceItem{Raw: s, Type: sType, Label: sName})
				}
			}
			if len(items) > 0 {
				h.Sidebar.SetItems(items)

				// Auto-select logic
				selectedIndex := 0 // Default to first
				if h.PendingSelection != "" {
					for i, item := range items {
						if s, ok := item.(SourceItem); ok {
							// Check match (Label is typically "name:tag" or "name")
							if s.Label == h.PendingSelection {
								selectedIndex = i
								break
							}
						}
					}
					h.PendingSelection = "" // Clear after use
				}

				h.Sidebar.Select(selectedIndex)
				if selected, ok := items[selectedIndex].(SourceItem); ok {
					h.Incubator.SetSource(&selected)
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

	case fv.SelectionMsg:
		if item, ok := msg.Item.(SourceItem); ok {
			h.Incubator.SetSource(&item)
		}

	case tea.KeyMsg:
		if h.Sidebar.Focused() && msg.String() == "enter" {
			cmds = append(cmds, h.MasterDetail.SetFocus(widget.FocusDetail))
		}

		if h.MasterDetail.ActiveFocus == widget.FocusDetail && msg.String() == "esc" {
			h.Incubator.Reset()
			// MasterDetail handles focus switch on ESC internally, but we intercepted it.
			// Actually MasterDetail.Update handles ESC by switching focus?
			// Let's check MasterDetail logic. It usually does.
			// But if we want to hook, we should do it before or rely on callback?
			// MasterDetail doesn't have "OnBack".
			// So we manually switch and return.
			cmds = append(cmds, h.MasterDetail.SetFocus(widget.FocusSidebar))
			return h, tea.Batch(cmds...)
		}

		if h.Sidebar.Focused() && (msg.String() == "delete" || msg.String() == "backspace") {
			if item := h.Sidebar.SelectedItem(); item != nil {
				srcItem, ok := item.(SourceItem)
				if ok && srcItem.Type == "TEMPLATE" {
					return h, ops.CheckTemplateUsage(h.prov, srcItem.Label)
				}
			}
		}
	}

	newMD, cmd := h.MasterDetail.Update(msg)
	h.MasterDetail = newMD.(*widget.MasterDetail)
	cmds = append(cmds, cmd)

	return h, tea.Batch(cmds...)
}

func (h *Hatchery) View() string {
	if h.ConfirmDelete.IsActive() {
		return h.ConfirmDelete.View(h.Width(), h.Height())
	}
	if h.ConfirmDeletePort.IsActive() {
		return h.ConfirmDeletePort.View(h.Width(), h.Height())
	}
	if h.ModalAddPort.IsActive() {
		// Render content dim/blurred behind?
		// For now just plain overlay logic from kit
		// But MasterDetail should be rendered as "background"
		// h.MasterDetail.View() // Ideally we render this to buffer if needed for overlay effects
		return h.ModalAddPort.View(h.Width(), h.Height())
		// NOTE: Ideally FormModal.View takes a "background" string to overlay properly
		// if utilizing lipgloss.PlaceOverlay correctly, but here we pass dims.
		// If FormModal uses PlaceOverlay, it pads with whitespace.
		// TODO: Advanced Overlay support in Kit.
	}
	if h.ModalAccel.IsActive() {
		return h.ModalAccel.View(h.Width(), h.Height())
	}
	return h.MasterDetail.View()
}

func (h *Hatchery) Resize(r layout.Rect) {
	h.BaseViewlet.Resize(r)
	h.MasterDetail.Resize(r)
}

func (h *Hatchery) HandleMouse(x, y int, msg tea.MouseMsg) (fv.Viewlet, tea.Cmd, bool) {
	if h.ConfirmDelete.IsActive() {
		return h, nil, true
	}
	if h.ModalAddPort.IsActive() {
		return h, nil, true
	}
	if h.ModalAccel.IsActive() {
		return h, nil, true
	}
	return h.MasterDetail.HandleMouse(x, y, msg)
}

func (h *Hatchery) Focus() tea.Cmd {
	return h.MasterDetail.Focus()
}

func (h *Hatchery) Shortcuts() []fv.Shortcut {
	if h.ConfirmDelete.IsActive() || h.ConfirmDeletePort.IsActive() {
		return []fv.Shortcut{
			{Key: "enter", Label: "engage"},
			{Key: "esc", Label: "back"},
		}
	}
	if h.ModalAddPort.IsActive() {
		return []fv.Shortcut{
			{Key: "tab", Label: "next"},
			{Key: "enter", Label: "add"},
			{Key: "esc", Label: "cancel"},
		}
	}
	if h.ModalAccel.IsActive() {
		return []fv.Shortcut{
			{Key: "↑/↓", Label: "select"},
			{Key: "enter", Label: "confirm"},
			{Key: "esc", Label: "cancel"},
		}
	}

	shortcuts := []fv.Shortcut{
		{Key: "↑/↓", Label: "glide"},
	}

	if h.MasterDetail.ActiveFocus == widget.FocusDetail {
		shortcuts = append(shortcuts, h.Incubator.Shortcuts()...)
		shortcuts = append(shortcuts, fv.Shortcut{Key: "esc", Label: "back"})
	} else {
		shortcuts = append(shortcuts, fv.Shortcut{Key: "enter", Label: "engage"})
	}

	if item := h.Sidebar.SelectedItem(); item != nil {
		if srcItem, ok := item.(SourceItem); ok && srcItem.Type == "TEMPLATE" {
			shortcuts = append(shortcuts, fv.Shortcut{Key: "delete", Label: "cull"})
		}
	}

	return shortcuts
}

func (h *Hatchery) IsModalActive() bool {
	return (h.ConfirmDelete != nil && h.ConfirmDelete.IsActive()) ||
		(h.ConfirmDeletePort != nil && h.ConfirmDeletePort.IsActive()) ||
		(h.ModalAddPort != nil && h.ModalAddPort.IsActive()) ||
		(h.ModalAccel != nil && h.ModalAccel.IsActive())
}

func (h *Hatchery) HasActiveTextInput() bool {
	if h.ConfirmDeletePort != nil && h.ConfirmDeletePort.IsActive() {
		return false
	}
	if h.ModalAddPort != nil && h.ModalAddPort.IsActive() {
		return true
	}
	if h.Incubator != nil {
		return h.Incubator.HasActiveTextInput()
	}
	return false
}

func (h *Hatchery) HasActiveFocus() bool {
	if h.ConfirmDeletePort != nil && h.ConfirmDeletePort.IsActive() {
		return true
	}
	if h.ModalAddPort != nil && h.ModalAddPort.IsActive() {
		return true
	}
	if h.Incubator != nil {
		return h.Incubator.HasActiveFocus()
	}
	return false
}

// --- Wrapper for Sidebar with Header ---

// parseTuiPorts handles a comma-separated list of port mappings.
// Implements Section 5.1 of advanced-port-forwarding.md for TUI.

// End of Hatchery Viewlet
