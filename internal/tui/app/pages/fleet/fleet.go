package fleet

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/Josepavese/nido/internal/provider"
	"github.com/Josepavese/nido/internal/tui/app/ops"
	"github.com/Josepavese/nido/internal/tui/kit/layout"
	"github.com/Josepavese/nido/internal/tui/kit/theme"
	view "github.com/Josepavese/nido/internal/tui/kit/view"
	widget "github.com/Josepavese/nido/internal/tui/kit/widget"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FleetItem adapter for standard SidebarList
type FleetItem struct {
	Name  string
	State string
}

func (i FleetItem) Title() string       { return i.Name }
func (i FleetItem) Description() string { return i.State }
func (i FleetItem) FilterValue() string { return i.Name }
func (i FleetItem) String() string      { return i.Name }
func (i FleetItem) Icon() string {
	if i.State == "running" {
		return theme.IconBird
	}
	return theme.IconSleep
}
func (i FleetItem) IsAction() bool { return false }

// FleetDetail holds the state for the detailed view logic
type FleetDetail struct {
	Name           string
	State          string
	PID            int
	IP             string
	SSHPort        int
	VNCPort        int
	SSHUser        string
	DiskPath       string
	DiskMissing    bool
	BackingPath    string
	BackingMissing bool
}

// Fleet implements the Viewlet interface using MasterDetail
type Fleet struct {
	view.BaseViewlet

	// Components
	Sidebar      *widget.SidebarList
	DetailView   *ComponentsDetail
	Pages        *widget.PageManager
	MasterDetail *widget.MasterDetail

	// Data
	items  []FleetItem
	detail FleetDetail
}

// NewFleet creates the viewlet
func NewFleet(prov provider.VMProvider) *Fleet {
	f := &Fleet{
		detail: FleetDetail{},
	}

	// 1. Sidebar (Empty initially)
	t := theme.Current()
	styles := widget.SidebarStyles{
		Normal:   t.Styles.SidebarItem,
		Selected: t.Styles.SidebarItemSelected,
		Dim:      lipgloss.NewStyle().Foreground(t.Palette.TextDim),
		Action:   t.Styles.SidebarItemSelected.Copy(),
	}
	f.Sidebar = widget.NewSidebarList(nil, theme.Width.Sidebar, styles, theme.RenderIcon(theme.IconFleet))

	// 2. Detail View (The "Main" content)
	f.DetailView = NewComponentsDetail(f)
	f.Pages = widget.NewPageManager()
	f.Pages.AddPage("DETAIL", f.DetailView)
	f.Pages.SwitchTo("DETAIL")

	// 3. MasterDetail Container
	// We use a simple border for the content
	border := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(t.Palette.SurfaceSubtle)

	f.MasterDetail = widget.NewMasterDetail(
		widget.NewBoxedSidebar(
			widget.NewCard(theme.IconFleet, "Fleet", "Manager"),
			f.Sidebar,
		),
		f.Pages,
		border,
	)
	f.MasterDetail.AutoSwitch = false // We handle data updates manually

	return f
}

func (f *Fleet) Init() tea.Cmd {
	return f.MasterDetail.Init()
}

func (f *Fleet) Update(msg tea.Msg) (view.Viewlet, tea.Cmd) {
	var cmds []tea.Cmd

	// Modal Interception: If detail has an active modal (Confirm or Error), we must block MasterDetail
	if f.DetailView.ConfirmDelete.IsActive() {
		_, cmd := f.DetailView.Update(msg)
		return f, cmd
	}
	if f.DetailView.ErrorModal.IsActive() {
		_, cmd := f.DetailView.Update(msg)
		return f, cmd
	}

	switch msg := msg.(type) {
	// Sidebar Selection
	case view.SelectionMsg:
		if item, ok := msg.Item.(FleetItem); ok {
			// User selected a VM in the sidebar
			// Request details for it
			cmds = append(cmds, func() tea.Msg {
				return ops.VMDetailRequestMsg{Name: item.Name}
			})
		}

	// Data Updates
	case ops.VMListMsg:
		if msg.Err == nil {
			// 1. Capture current selection to restore it
			var targetName string
			if sel := f.Sidebar.SelectedItem(); sel != nil {
				targetName = sel.Title()
			}

			// 2. Update list
			newItems := make([]widget.SidebarItem, len(msg.Items))
			f.items = make([]FleetItem, len(msg.Items))
			targetIndex := 0 // Default to first

			for i, v := range msg.Items {
				fi := FleetItem{Name: v.Name, State: v.State}
				newItems[i] = fi
				f.items[i] = fi

				// Check if this is our previously selected one
				if v.Name == targetName {
					targetIndex = i
				}
			}
			f.Sidebar.SetItems(newItems)

			// 3. Restore Selection & Refresh Detail
			if len(newItems) > 0 {
				f.Sidebar.Select(targetIndex)

				// Always fetch fresh detail for the active item (status might have changed)
				// This fixes:
				// 1. Initial Load (targetIndex=0, fetches detail)
				// 2. Selection preservation (targetIndex=saved, fetches detail)
				if selected, ok := newItems[targetIndex].(FleetItem); ok {
					cmds = append(cmds, func() tea.Msg {
						return ops.VMDetailRequestMsg{Name: selected.Name}
					})
				}
			}
		}

	case ops.VMDetailMsg:
		if msg.Err == nil {
			f.detail = FleetDetail{
				Name:           msg.Detail.Name,
				State:          msg.Detail.State,
				PID:            msg.Detail.PID,
				IP:             msg.Detail.IP,
				SSHPort:        msg.Detail.SSHPort,
				VNCPort:        msg.Detail.VNCPort,
				SSHUser:        msg.Detail.SSHUser,
				DiskPath:       msg.Detail.DiskPath,
				DiskMissing:    msg.Detail.DiskMissing,
				BackingPath:    msg.Detail.BackingPath,
				BackingMissing: msg.Detail.BackingMissing,
			}
			// Update the detail view
			f.DetailView.UpdateDetail(f.detail)
		}

	// Forward Actions from Detail View
	case FleetActionMsg:
		switch msg.Action {
		case "start":
			return f, func() tea.Msg { return ops.RequestOpMsg{Op: ops.OpStart, Name: msg.Name} }
		case "stop":
			return f, func() tea.Msg { return ops.RequestOpMsg{Op: ops.OpStop, Name: msg.Name} }
		case "delete":
			return f, func() tea.Msg { return ops.RequestOpMsg{Op: ops.OpDelete, Name: msg.Name} }
		case "toggle":
			if f.detail.State == "running" {
				return f, func() tea.Msg { return ops.RequestOpMsg{Op: ops.OpStop, Name: msg.Name} }
			}
			return f, func() tea.Msg { return ops.RequestOpMsg{Op: ops.OpStart, Name: msg.Name} }
		}

	// Operation Results (Error Handling)
	case ops.OpResultMsg:
		if msg.Err != nil {
			title, details := f.mapError(msg.Err)
			f.DetailView.ErrorModal.Title = title
			f.DetailView.ErrorModal.Message = details
			f.DetailView.ErrorModal.Show()
			return f, nil
		}
		// Success cases might trigger refresh automatically via other means or we can force it
		// Usually App handles refresh, but if we need immediate feedback we can do it here.
		// For now, errors are the priority.

	case tea.KeyMsg:
		switch msg.String() {
		case "enter", " ":
			// Toggle VM power (start/stop)
			if selectedItem := f.Sidebar.SelectedItem(); selectedItem != nil {
				if item, ok := selectedItem.(FleetItem); ok {
					cmds = append(cmds, func() tea.Msg { return FleetActionMsg{Action: "toggle", Name: item.Name} })
				}
			}
		case "s":
			// SSH Shortcut
			cmds = append(cmds, f.DetailView.openSSH())
		case "v":
			// VNC Shortcut
			cmds = append(cmds, f.DetailView.openVNC())
		case "backspace", "delete":
			// Delete Shortcut - show confirmation modal
			f.DetailView.ConfirmDelete.Show()
		}
	}

	// Delegate to MasterDetail
	newMD, mdCmd := f.MasterDetail.Update(msg)
	f.MasterDetail = newMD.(*widget.MasterDetail)
	cmds = append(cmds, mdCmd)

	return f, tea.Batch(cmds...)
}

func (f *Fleet) View() string {
	return f.MasterDetail.View()
}

func (f *Fleet) Resize(r layout.Rect) {
	f.BaseViewlet.Resize(r)
	f.MasterDetail.Resize(r)
}

func (f *Fleet) Focus() tea.Cmd {
	return f.MasterDetail.Focus()
}

func (f *Fleet) Shortcuts() []view.Shortcut {
	if f.DetailView.ConfirmDelete.IsActive() {
		return []view.Shortcut{
			{Key: "enter", Label: "confirm"},
			{Key: "esc", Label: "cancel"},
		}
	}
	if f.DetailView.ErrorModal.IsActive() {
		return []view.Shortcut{
			{Key: "enter", Label: "close"},
			{Key: "esc", Label: "close"},
		}
	}

	shortcuts := []view.Shortcut{
		{Key: "↑/↓", Label: "navigate"},
	}

	if selectedItem := f.Sidebar.SelectedItem(); selectedItem != nil {
		if item, ok := selectedItem.(FleetItem); ok {
			// Power Hint
			if item.State == "running" {
				shortcuts = append(shortcuts, view.Shortcut{Key: "space", Label: "stop"})
				// Actions only available when running
				shortcuts = append(shortcuts, view.Shortcut{Key: "s", Label: "ssh"})
				shortcuts = append(shortcuts, view.Shortcut{Key: "v", Label: "vnc"})
			} else {
				shortcuts = append(shortcuts, view.Shortcut{Key: "space", Label: "start"})
			}
			// Delete Hint (always available)
			shortcuts = append(shortcuts, view.Shortcut{Key: "canc", Label: "delete"})
		}
	}

	return shortcuts
}

func (f *Fleet) HandleMouse(x, y int, msg tea.MouseMsg) (view.Viewlet, tea.Cmd, bool) {
	return f.MasterDetail.HandleMouse(x, y, msg)
}

// IsModalActive allows the App to block global navigation (tabs) when the modal is open.
func (f *Fleet) IsModalActive() bool {
	return f.DetailView.ConfirmDelete.IsActive() || f.DetailView.ErrorModal.IsActive()
}

// --- Detail Component ---

type ComponentsDetail struct {
	view.BaseViewlet
	Parent *Fleet

	// Form components
	Form     *widget.Form
	header   *widget.Card
	pidInput *widget.Input
	ipInput  *widget.Input

	// Debug
	// Ports
	sshInput *widget.Input
	vncInput *widget.Input

	diskInput *widget.Input

	// Modal
	ConfirmDelete *widget.Modal
	ErrorModal    *widget.Modal
}

func NewComponentsDetail(parent *Fleet) *ComponentsDetail {
	c := &ComponentsDetail{
		Parent: parent,
	}

	// Header
	c.header = widget.NewCard("🦅", "Select VM", "")

	// Row 1: PID - IP - Name (all disabled, boxed)
	c.pidInput = widget.NewInput("PID", "", nil)
	c.pidInput.Disabled = true

	c.ipInput = widget.NewInput("IP", "", nil)
	c.ipInput.Disabled = true

	// Debug / Test Inputs
	// Row 2: SSH - VNC (all disabled, boxed)
	c.sshInput = widget.NewInput("SSH Port", "", nil)
	c.sshInput.Disabled = true

	c.vncInput = widget.NewInput("VNC Port", "", nil)
	c.vncInput.Disabled = true

	c.diskInput = widget.NewInput("Disk", "", nil)
	c.diskInput.Disabled = true

	// Modal for delete confirmation
	c.ConfirmDelete = widget.NewModal(
		"Delete VM",
		"Are you sure?",
		func() tea.Cmd {
			if c.Parent.detail.Name != "" {
				return func() tea.Msg {
					return ops.RequestOpMsg{Op: ops.OpDelete, Name: c.Parent.detail.Name}
				}
			}
			return nil
		},
		nil,
	)

	// Error Modal (Single button)
	c.ErrorModal = widget.NewAlertModal(
		"Error",
		"An unexpected error occurred.",
		nil, // Dismiss just closes
	)

	// Build form with rows
	c.rebuildForm()

	return c
}

func (c *ComponentsDetail) rebuildForm() {
	// 1. Standard Row (2 cols)
	row1 := widget.NewRow(c.pidInput, c.ipInput)

	// 2. Ports Row (2 cols)
	row2 := widget.NewRow(c.sshInput, c.vncInput)

	c.Form = widget.NewForm(
		c.header,
		row1,
		row2,
		c.diskInput,
	)
	c.Form.Spacing = 0
}

func (c *ComponentsDetail) UpdateDetail(d FleetDetail) {
	if d.Name == "" {
		// No VM selected
		c.header.Icon = "🦅"
		c.header.Title = "Select VM"
		c.header.Subtitle = ""

		c.pidInput.SetValue("")
		c.ipInput.SetValue("")
		c.diskInput.SetValue("")

		return
	}

	// Update header
	icon := theme.IconSleep
	if d.State == "running" {
		icon = theme.IconBird
	}
	status := strings.ToUpper(d.State)
	c.header.Icon = icon
	c.header.Title = d.Name
	c.header.Subtitle = status

	// Update fields

	c.pidInput.SetValue(fmt.Sprintf("%d", d.PID))
	c.ipInput.SetValue(d.IP)
	c.sshInput.SetValue(fmt.Sprintf("%d", d.SSHPort))
	c.vncInput.SetValue(fmt.Sprintf("%d", d.VNCPort))

	// Disk with error highlighting
	c.diskInput.SetValue(d.DiskPath) // Always show the path cleanly
	c.diskInput.Error = ""           // Reset error state

	if d.DiskMissing {
		c.diskInput.Error = "Disk image not found"
	} else if d.BackingMissing {
		c.diskInput.Error = "Backing file (template) missing"
	}

	// Update button states and labels

	// No buttons to update state for anymore as they are now keyboard shortcuts
}

func (c *ComponentsDetail) openSSH() tea.Cmd {
	d := c.Parent.detail
	if d.State != "running" {
		return nil
	}
	sshCmd := fmt.Sprintf("ssh -p %d %s@localhost", d.SSHPort, d.SSHUser)
	// TODO: Cross-platform terminal opening
	return tea.ExecProcess(exec.Command("x-terminal-emulator", "-e", sshCmd), nil)
}

func (c *ComponentsDetail) openVNC() tea.Cmd {
	d := c.Parent.detail
	if d.State != "running" {
		return nil
	}
	vncAddr := fmt.Sprintf("localhost:%d", d.VNCPort)
	// TODO: Cross-platform VNC viewer opening
	return tea.ExecProcess(exec.Command("vncviewer", vncAddr), nil)
}

func (c *ComponentsDetail) togglePower() tea.Cmd {
	d := c.Parent.detail
	if d.Name == "" {
		return nil
	}

	if d.State == "running" {
		return func() tea.Msg {
			return ops.RequestOpMsg{Op: ops.OpStop, Name: d.Name}
		}
	}
	return func() tea.Msg {
		return ops.RequestOpMsg{Op: ops.OpStart, Name: d.Name}
	}
}

func (c *ComponentsDetail) Init() tea.Cmd { return nil }

func (c *ComponentsDetail) Update(msg tea.Msg) (view.Viewlet, tea.Cmd) {
	// Modal interception
	if c.ConfirmDelete.IsActive() {
		newModal, cmd := c.ConfirmDelete.Update(msg)
		c.ConfirmDelete = newModal
		return c, cmd
	}
	if c.ErrorModal.IsActive() {
		newModal, cmd := c.ErrorModal.Update(msg)
		c.ErrorModal = newModal
		return c, cmd
	}

	// Delegate to Form
	if !c.Focused() {
		return c, nil
	}

	newForm, cmd := c.Form.Update(msg)
	c.Form = newForm
	return c, cmd
}

func (c *ComponentsDetail) Resize(r layout.Rect) {
	c.BaseViewlet.Resize(r)
}

func (c *ComponentsDetail) Shortcuts() []view.Shortcut {
	if c.ConfirmDelete.IsActive() {
		return []view.Shortcut{
			{Key: "enter", Label: "confirm"},
			{Key: "esc", Label: "cancel"},
		}
	}
	if c.ErrorModal.IsActive() {
		return []view.Shortcut{
			{Key: "enter", Label: "close"},
		}
	}

	shortcuts := []view.Shortcut{
		{Key: "tab", Label: "next"},
		{Key: "enter", Label: "action"},
	}

	if c.Parent.detail.Name != "" {
		shortcuts = append(shortcuts, view.Shortcut{Key: "delete", Label: "delete"})
	}

	return shortcuts
}

func (c *ComponentsDetail) HandleMouse(x, y int, msg tea.MouseMsg) (view.Viewlet, tea.Cmd, bool) {
	if c.ConfirmDelete.IsActive() || c.ErrorModal.IsActive() {
		return c, nil, true
	}
	return c, nil, false
}

func (c *ComponentsDetail) View() string {
	// Modal overlay
	if c.ConfirmDelete.IsActive() {
		return c.ConfirmDelete.View(c.Width(), c.Height())
	}
	if c.ErrorModal.IsActive() {
		return c.ErrorModal.View(c.Width(), c.Height())
	}

	if c.Parent.detail.Name == "" {
		return layout.Center(c.Width(), "Select a VM from the fleet...")
	}

	w := c.Width()
	padding := theme.Current().Layout.ContainerPadding
	safeWidth := w - (2 * padding)

	if safeWidth > 60 {
		safeWidth = 60
	}
	if safeWidth < 40 {
		safeWidth = 40
	}

	// Pass width to Declarative Form and ActionStack
	c.Form.Width = safeWidth

	// Standardized Left Alignment (matches Registry/Hatchery)
	return c.Form.View(safeWidth)
}

func (c *ComponentsDetail) Focus() tea.Cmd {
	c.BaseViewlet.Focus()
	return c.Form.Focus()
}

func (c *ComponentsDetail) Blur() {
	c.BaseViewlet.Blur()
}

// mapError analyzes a raw error and returns a human-friendly title and message.
// It mimics the logic used for JSON error responses (RFC 7807 inspired).
func (f *Fleet) mapError(err error) (string, string) {
	raw := err.Error()

	// 1. Missing Backing File / Template
	if strings.Contains(raw, "Could not open backing file") || strings.Contains(raw, "No such file or directory") {
		return "Missing Resource (ERR_NOT_FOUND)",
			" The VM's backing template or disk image is missing.\n\n" +
				" • Hint: Check if the template was deleted.\n" +
				" • Hint: Try deleting and re-creating this VM."
	}

	// 2. Port Conflict
	if strings.Contains(raw, "bind: address already in use") {
		return "Port Conflict (ERR_NET)",
			" The VM could not start because its SSH or VNC port is already in use.\n\n" +
				" • Hint: Check if another VM is using these ports.\n" +
				" • Hint: Wait a few seconds if you just stopped it."
	}

	// 3. Generic QEMU Exit Code
	if strings.Contains(raw, "exit status") {
		// Try to extract stderr if available in the string
		// Format usually: "exit status 1 (stderr: ...)"
		if start := strings.Index(raw, "(stderr:"); start != -1 {
			msg := raw[start+9:] // Skip "(stderr: "
			if end := strings.LastIndex(msg, ")"); end != -1 {
				msg = msg[:end]
			}
			return "Hypervisor Error (ERR_QEMU)",
				fmt.Sprintf("QEMU failed to start.\n\nDetails: %s", strings.TrimSpace(msg))
		}
		return "Hypervisor Error (ERR_QEMU)", "The virtualization process exited unexpectedly."
	}

	// 4. Default / Internal
	return "System Error (ERR_INTERNAL)",
		fmt.Sprintf("An unexpected error occurred:\n\n%s", raw)
}

type FleetActionMsg struct {
	Action string // "start", "stop", "toggle", "delete"
	Name   string
}
