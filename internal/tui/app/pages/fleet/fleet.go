package fleet

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/Josepavese/nido/internal/pkg/sysutil"
	"github.com/Josepavese/nido/internal/provider"
	"github.com/Josepavese/nido/internal/tui/app/ops"
	"github.com/Josepavese/nido/internal/tui/kit/layout"
	"github.com/Josepavese/nido/internal/tui/kit/theme"
	view "github.com/Josepavese/nido/internal/tui/kit/view"
	widget "github.com/Josepavese/nido/internal/tui/kit/widget"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	// Import Hatchery for AccelItem reuse? Or ideally move AccelItem to shared?
	// Moving AccelItem to shared widget or ops is cleaner.
	// BUT I can't move it easily without refactoring Hatchery imports.
	// For now, I will DUPLICATE AccelItem adapter in fleet.go to avoid import cycle if hatchery imports fleet (unlikely but possible).
	// Actually, hatchery imports widget, theme, ops.
	// Fleet imports widget, theme, ops.
	// I'll define AccelItem in fleet.go locally.
	"github.com/charmbracelet/bubbles/list"
)

// FleetItem adapter for standard SidebarList
type FleetItem struct {
	Name          string
	State         string
	Transitioning bool   // New: tracked locally for fast feedback
	SpinnerFrame  string // New: frame from fleet's spinner
}

func (i FleetItem) Title() string       { return i.Name }
func (i FleetItem) Description() string { return i.State }
func (i FleetItem) FilterValue() string { return i.Name }
func (i FleetItem) String() string      { return i.Name }
func (i FleetItem) Icon() string {
	if i.Transitioning {
		return i.SpinnerFrame
	}
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
	MemoryMB       int
	VCPUs          int
	SSHUser        string
	DiskPath       string
	DiskMissing    bool
	BackingPath    string
	BackingMissing bool
	Forwarding     []provider.PortForward
	Accelerators   []string // New: Accelerators for PASSTHROUGH
}

// Fleet implements the Viewlet interface using MasterDetail
type Fleet struct {
	view.BaseViewlet
	provider provider.VMProvider // SSOT: Store provider reference

	// Components
	Sidebar      *widget.SidebarList
	DetailView   *ComponentsDetail
	Pages        *widget.PageManager
	MasterDetail *widget.MasterDetail

	// Data
	items         []FleetItem
	detail        FleetDetail
	transitioning map[string]bool // New: track active fast operations
	spinner       spinner.Model   // New: local spinner for sidebar

	// Local State
	existingTemplates []string
	pendingTemplateVM string
	TemplateModal     *CreateTemplateModal
	ConfirmDelete     *widget.Modal
	ErrorModal        *widget.Modal
	ModalAccel        *widget.ListModal // Accelerator Selection
	pendingAccelVM    string            // Track which VM we are editing
}

// NewFleet creates the viewlet
func NewFleet(prov provider.VMProvider) *Fleet {
	t := theme.Current()

	// Initialize Spinner
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(t.Palette.Accent)

	f := &Fleet{
		provider:      prov,
		detail:        FleetDetail{},
		transitioning: make(map[string]bool),
		spinner:       s,
		TemplateModal: NewCreateTemplateModal(),
	}

	// Modal for delete confirmation (Full-screen handled by Fleet.View)
	f.ConfirmDelete = widget.NewModal(
		"Delete VM",
		"Are you sure?",
		func() tea.Cmd {
			if f.detail.Name != "" {
				return func() tea.Msg {
					return ops.RequestOpMsg{Op: ops.OpDelete, Name: f.detail.Name}
				}
			}
			return nil
		},
		nil,
	)
	f.ConfirmDelete.SetLevel(widget.ModalLevelDanger)

	// Error Modal (Single button)
	f.ErrorModal = widget.NewAlertModal(
		"Error",
		"An unexpected error occurred.",
		nil, // Dismiss just closes
	)

	// Accelerator Modal
	f.ModalAccel = widget.NewListModal("Select Accelerator", nil, 60, 20, func(item list.Item) tea.Cmd {
		// Callback handled in Update via active check?
		// No, NewListModal takes onSelect.
		if f.pendingAccelVM == "" {
			return nil
		}

		var selectedID string
		if ai, ok := item.(AccelItem); ok {
			selectedID = ai.Acc.ID
		} else {
			// None?
			selectedID = "" // Clear
		}

		// Update Config
		// We use REPLACEMENT logic for single-device mode
		newAccelList := []string{}
		if selectedID != "" {
			newAccelList = []string{selectedID}
		}

		updates := provider.VMConfigUpdates{
			Accelerators: &newAccelList,
		}

		return ops.UpdateVMConfig(f.provider, f.pendingAccelVM, updates)
	}, nil)

	// 1. Sidebar (Empty initially)
	styles := widget.SidebarStyles{
		Normal:   t.Styles.SidebarItem,
		Selected: t.Styles.SidebarItemSelected,
		Dim:      lipgloss.NewStyle().Foreground(t.Palette.TextDim),
		Action:   t.Styles.SidebarItemSelected,
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

func (f *Fleet) OpenAcceleratorModal(vmName string) tea.Cmd {
	return func() tea.Msg {
		// Set context
		f.pendingAccelVM = vmName

		// Fetch accelerators dynamically
		devs, err := f.provider.ListAccelerators()
		if err != nil {
			// Show error modal?
			return ops.AcceleratorListMsg{Err: err}
		}

		var items []list.Item
		// Add "None" option?
		// items = append(items, AccelItem{Acc: provider.Accelerator{ID: "", Class: "None"}})
		// Actually treating empty selection or explicit "None" item is design choice.
		// For now simple list of available devices.

		for _, d := range devs {
			items = append(items, AccelItem{Acc: d})
		}

		f.ModalAccel.List.SetItems(items)
		f.ModalAccel.Show()
		return nil
	}
}
func (f *Fleet) Update(msg tea.Msg) (view.Viewlet, tea.Cmd) {
	var cmds []tea.Cmd

	// Modal Interception: Page-level modals block everything
	if f.ConfirmDelete.IsActive() {
		newModal, cmd := f.ConfirmDelete.Update(msg)
		f.ConfirmDelete = newModal
		return f, cmd
	}
	if f.ErrorModal.IsActive() {
		newModal, cmd := f.ErrorModal.Update(msg)
		f.ErrorModal = newModal
		return f, cmd
	}
	if f.TemplateModal.IsActive() {
		return f, f.TemplateModal.Update(msg)
	}
	if f.ModalAccel.IsActive() {
		_, cmd := f.ModalAccel.Update(msg)
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
				// Spinner Logic:
				// If this item is transitioning, use the current spinner frame
				isTransitioning := f.transitioning[v.Name]
				frame := ""
				if isTransitioning {
					frame = f.spinner.View()
				}

				fi := FleetItem{
					Name:          v.Name,
					State:         v.State,
					Transitioning: isTransitioning,
					SpinnerFrame:  frame,
				}
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
				MemoryMB:       msg.Detail.MemoryMB,
				VCPUs:          msg.Detail.VCPUs,
				SSHUser:        msg.Detail.SSHUser,
				DiskPath:       msg.Detail.DiskPath,
				DiskMissing:    msg.Detail.DiskMissing,
				BackingPath:    msg.Detail.BackingPath,
				BackingMissing: msg.Detail.BackingMissing,
				Forwarding:     msg.Detail.Forwarding,
				Accelerators:   msg.Detail.Accelerators,
			}
			f.DetailView.UpdateDetail(f.detail)

			// Safely clear transitioning if state matches expected?
			// Actually, ops logic clears it on OpResult.
			// But if we get a detail update and it's what we want, maybe we should clear?
			// Let's stick to OpResult for determinism.
		}

	case ops.TemplateListMsg:
		if msg.Err == nil {
			f.existingTemplates = msg.Templates
			// If we were waiting to open the modal for a VM, do it now
			if f.pendingTemplateVM != "" {
				cmds = append(cmds, f.TemplateModal.Show(f.pendingTemplateVM, f.existingTemplates))
				f.pendingTemplateVM = ""
			}
		} else {
			// Failed to list templates? Show error
			f.pendingTemplateVM = ""
			f.ErrorModal.Title = "Error"
			f.ErrorModal.Message = fmt.Sprintf("Failed to list templates:\n%v", msg.Err)
			f.ErrorModal.Show()
		}

	// Forward Actions from Detail View
	case FleetActionMsg:
		// Mark as transitioning locally
		f.transitioning[msg.Name] = true
		cmds = append(cmds, f.spinner.Tick) // Start ticking!

		switch msg.Action {
		case "start":
			return f, func() tea.Msg { return ops.RequestOpMsg{Op: ops.OpStart, Name: msg.Name} }
		case "stop":
			return f, func() tea.Msg { return ops.RequestOpMsg{Op: ops.OpStop, Name: msg.Name} }
		case "delete":
			// Delete is still global, but we can track it too if we want.
			// User said fast ops. Delete might be slow.
			// Let's NOT track delete locally to keep distinction.
			delete(f.transitioning, msg.Name)
			return f, func() tea.Msg { return ops.RequestOpMsg{Op: ops.OpDelete, Name: msg.Name} }
		case "toggle":
			f.transitioning[msg.Name] = true
			if f.detail.State == "running" {
				return f, func() tea.Msg { return ops.RequestOpMsg{Op: ops.OpStop, Name: msg.Name} }
			}
			return f, func() tea.Msg { return ops.RequestOpMsg{Op: ops.OpStart, Name: msg.Name} }
		}

	// Operation Results (Error Handling)
	case ops.OpResultMsg:
		// Clear transitioning state
		if msg.Path == "" { // Use Path or Name? OpResultMsg currently lacks Name field...
			// Wait, OpResultMsg in commands.go:
			// type OpResultMsg struct { Op string; Err error; Path string }
			// It doesn't have Name! We need to fix commands.go or infer it?
			// Ideally commands.go should return Name.
			// Current fix: Adding Name to OpResultMsg in commands.go is best.
			// BUT, for now in the viewlet, we might not know essentialy WHICH vm failed if we have multiple.
			// However, Nido TUI is single-threaded mostly for users.
			// Let's check commands.go first.
		}
		// If we can't identify the VM, we might clear ALL?
		// Or we can rely on VMListMsg refesh which usually follows OpResult.

		if msg.Err != nil {
			title, details := f.mapError(msg.Err)
			f.ErrorModal.Title = title
			f.ErrorModal.Message = details
			f.ErrorModal.Show()
			// Clear all transitioning on error to be safe
			f.transitioning = make(map[string]bool)
		} else {
			// Success! wiring.go will trigger RefreshFleet.
			// We can clear active transitions here IF we knew the name.
			// Since we don't, let's clear all? Or wait for List update?
			// If we wait for List update, the icons might still spin if we don't clear map.
			// We MUST clear the map.
			f.transitioning = make(map[string]bool)
		}
		return f, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		f.spinner, cmd = f.spinner.Update(msg)
		cmds = append(cmds, cmd)

		// Force re-render of sidebar items with new frame
		// We re-use current items but update spinner frame
		currentItems := f.Sidebar.Items() // Generic items
		newItems := make([]widget.SidebarItem, len(currentItems))

		anySpinning := false
		for i, item := range currentItems {
			if fi, ok := item.(FleetItem); ok {
				if f.transitioning[fi.Name] {
					fi.SpinnerFrame = f.spinner.View()
					anySpinning = true
				}
				newItems[i] = fi
			}
		}

		if anySpinning {
			f.Sidebar.SetItems(newItems)
		}
		return f, tea.Batch(cmds...)

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
			// SSH Shortcut (In-place)
			cmds = append(cmds, f.DetailView.openSSH(false))
		case "S":
			// SSH Shortcut (Windowed)
			cmds = append(cmds, f.DetailView.openSSH(true))
		case "v":
			// VNC Shortcut
			cmds = append(cmds, f.DetailView.openVNC())
		case "t":
			// Template Creation (Context: Selected VM)
			if selectedItem := f.Sidebar.SelectedItem(); selectedItem != nil {
				if item, ok := selectedItem.(FleetItem); ok {
					if item.State != "shutoff" && item.State != "stopped" {
						f.ErrorModal.Title = "Cannot Create Template"
						f.ErrorModal.Message = fmt.Sprintf("VM '%s' must be stopped before creating a template.", item.Name)
						f.ErrorModal.Show()
					} else {
						// 1. Set pending
						f.pendingTemplateVM = item.Name
						// 2. Fetch templates to validate uniqueness
						// 2. Fetch templates to validate uniqueness
						cmds = append(cmds, func() tea.Msg { return ops.RequestTemplateListMsg{} })
					}
				}
			}
		case "backspace", "delete":
			// Delete Shortcut - show confirmation modal
			f.ConfirmDelete.Show()
		}
	}

	// Delegate to MasterDetail
	newMD, mdCmd := f.MasterDetail.Update(msg)
	f.MasterDetail = newMD.(*widget.MasterDetail)
	cmds = append(cmds, mdCmd)

	return f, tea.Batch(cmds...)
}

func (f *Fleet) View() string {
	// Overlay Modals (Full-screen)
	if f.ConfirmDelete.IsActive() {
		return f.ConfirmDelete.View(f.Width(), f.Height())
	}
	if f.ErrorModal.IsActive() {
		return f.ErrorModal.View(f.Width(), f.Height())
	}
	if f.TemplateModal.IsActive() {
		return f.TemplateModal.View(f.Width(), f.Height())
	}
	if f.ModalAccel.IsActive() {
		return f.ModalAccel.View(f.Width(), f.Height())
	}
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
	if f.IsModalActive() {
		return []view.Shortcut{
			{Key: "enter", Label: "engage"},
			{Key: "esc", Label: "back"},
		}
	}

	shortcuts := []view.Shortcut{
		{Key: "↑/↓", Label: "glide"},
	}

	if selectedItem := f.Sidebar.SelectedItem(); selectedItem != nil {
		if item, ok := selectedItem.(FleetItem); ok {
			// Engage Hint (Toggle)
			toggleLabel := "engage"
			if item.State == "running" {
				toggleLabel = "exhale" // stop
			}
			shortcuts = append(shortcuts, view.Shortcut{Key: "enter/space", Label: toggleLabel})

			// Running specific
			if item.State == "running" {
				shortcuts = append(shortcuts, view.Shortcut{Key: "s", Label: "ssh"})
				shortcuts = append(shortcuts, view.Shortcut{Key: "S", Label: "ssh (win)"})
				shortcuts = append(shortcuts, view.Shortcut{Key: "v", Label: "vnc"})
			} else {
				// Stopped specific
				shortcuts = append(shortcuts, view.Shortcut{Key: "t", Label: "template"})
			}

			// Delete Hint (always available)
			shortcuts = append(shortcuts, view.Shortcut{Key: "backspace", Label: "evict"})
		}
	}

	if f.MasterDetail.ActiveFocus == widget.FocusDetail {
		shortcuts = append(shortcuts, view.Shortcut{Key: "esc", Label: "back"})
	}

	return shortcuts
}

func (f *Fleet) HandleMouse(x, y int, msg tea.MouseMsg) (view.Viewlet, tea.Cmd, bool) {
	return f.MasterDetail.HandleMouse(x, y, msg)
}

// IsModalActive allows the App to block global navigation (tabs) when the modal is open.
func (f *Fleet) IsModalActive() bool {
	return f.ConfirmDelete.IsActive() || f.ErrorModal.IsActive() || f.TemplateModal.IsActive() || f.ModalAccel.IsActive()
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
	memInput *widget.Input
	cpuInput *widget.Input

	diskInput *widget.Input
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

	c.memInput = widget.NewInput("Memory", "", nil)
	c.memInput.Disabled = true

	c.cpuInput = widget.NewInput("vCPUs", "", nil)
	c.cpuInput.Disabled = true

	c.diskInput = widget.NewInput("Disk", "", nil)
	c.diskInput.Disabled = true

	// Build form with rows
	c.rebuildForm()

	return c
}

func (c *ComponentsDetail) rebuildForm() {
	var elements []widget.Element

	// 1. Header
	elements = append(elements, c.header)

	// 2. Standard Row (2 cols)
	elements = append(elements, widget.NewRow(c.pidInput, c.ipInput))

	// 3. Ports Row (2 cols)
	elements = append(elements, widget.NewRow(c.sshInput, c.vncInput))

	// 4. Resources Row (2 cols)
	elements = append(elements, widget.NewRow(c.memInput, c.cpuInput))

	// 5. Disk
	elements = append(elements, c.diskInput)

	// 5. Dynamic Ports
	forwarding := c.Parent.detail.Forwarding
	if len(forwarding) > 0 {
		// Header for ports section? Or just rows
		// User requested: label + proto | guest | host

		for _, pf := range forwarding {
			// Col 1: Label (+Proto)
			lbl := pf.Label
			if lbl == "" {
				lbl = "-"
			}
			if pf.Protocol != "" {
				lbl = fmt.Sprintf("%s (%s)", lbl, pf.Protocol)
			}
			btnLabel := widget.NewButton("Port", lbl, nil)
			btnLabel.Disabled = true
			btnLabel.Centered = true

			// Col 2: Guest
			btnGuest := widget.NewButton("Guest", fmt.Sprintf("%d", pf.GuestPort), nil)
			btnGuest.Disabled = true
			btnGuest.Centered = true

			// Col 3: Host
			btnHost := widget.NewButton("Host", fmt.Sprintf("%d", pf.HostPort), nil)
			btnHost.Disabled = true
			btnHost.Centered = true

			// Weight 2:1:1 or 1:1:1? User said "all 3 on same row"
			// Label can be long, so maybe 2:1:1
			elements = append(elements, widget.NewRowWithWeights(
				[]widget.Element{btnLabel, btnGuest, btnHost},
				[]int{2, 1, 1},
			))
		}
	} else {
		// Show "No Ports" placeholder?
		// User didn't ask for a placeholder, but maybe useful.
		// For now, if empty, we show nothing extra or maybe a "Ports: None" disabled input?
		// Existing code used c.portsInput.SetValue("None").
		// If we want to keep that behavior:
		noPorts := widget.NewInput("Ports", "None", nil)
		noPorts.Disabled = true
		elements = append(elements, noPorts)
	}

	// 6. Accelerators
	accelerators := c.Parent.detail.Accelerators

	// Create a button to Manage Accelerators
	accLabel := "None"
	if len(accelerators) > 0 {
		accLabel = strings.Join(accelerators, ", ")
	}

	// If stopped, allow editing
	if c.Parent.detail.State == "stopped" || c.Parent.detail.State == "shutoff" {
		btnAccel := widget.NewButton("Accelerators", accLabel+" (Edit)", func() tea.Cmd {
			return c.Parent.OpenAcceleratorModal(c.Parent.detail.Name)
		})
		btnAccel.Centered = true // Make it look like an active element
		elements = append(elements, btnAccel)
	} else {
		// Read only input if running
		accInput := widget.NewInput("Accelerators", accLabel, nil)
		accInput.Disabled = true
		elements = append(elements, accInput)
	}

	c.Form = widget.NewForm(elements...)
	c.Form.Spacing = 0
}

func (c *ComponentsDetail) UpdateDetail(d FleetDetail) {
	if d.Name == "" {
		// No VM selected
		c.header.Icon = "🦅"
		c.header.Title = "Select VM"
		c.header.Subtitle = "Choose a pilot from the fleet roster..."

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
	c.memInput.SetValue(fmt.Sprintf("%d MB", d.MemoryMB))
	c.cpuInput.SetValue(fmt.Sprintf("%d", d.VCPUs))

	// Disk with error highlighting
	c.diskInput.SetValue(d.DiskPath) // Always show the path cleanly
	c.diskInput.Error = ""           // Reset error state

	if d.DiskMissing {
		c.diskInput.Error = "Disk image not found"
	} else if d.BackingMissing {
		c.diskInput.Error = "Backing file (template) missing"
	}

	// Update form structure with new ports
	c.rebuildForm()
}

func (c *ComponentsDetail) openSSH(windowed bool) tea.Cmd {
	d := c.Parent.detail
	if d.State != "running" {
		return nil
	}

	sshCmd, err := c.Parent.provider.SSHCommand(d.Name)
	if err != nil {
		return func() tea.Msg {
			return ops.OpResultMsg{Err: fmt.Errorf("failed to generate SSH command: %w", err), Path: d.Name}
		}
	}

	if windowed {
		// Windowed: Spawn a separate terminal emulator window
		termBin, termArgs := sysutil.TerminalCommand(sshCmd)
		return tea.ExecProcess(exec.Command(termBin, termArgs...), nil)
	}

	// In-Place: Directly execute SSH in current terminal.
	// We parse the sshCmd which looks like: ssh -o ... -p [port] [user]@[ip]
	parts := strings.Split(sshCmd, " ")
	if len(parts) < 1 {
		return nil
	}

	// Reconstruct args avoiding the 'ssh' binary itself in args[0]
	// Also add some quality of life options for TUI users
	extraOpts := []string{"-o", "LogLevel=ERROR", "-o", "ConnectTimeout=5"}
	allArgs := append(extraOpts, parts[1:]...)

	return tea.ExecProcess(exec.Command("ssh", allArgs...), nil)
}

func (c *ComponentsDetail) openVNC() tea.Cmd {
	d := c.Parent.detail
	if d.State != "running" {
		return nil
	}
	vncAddr := fmt.Sprintf("localhost:%d", d.VNCPort)

	// Use SSOT VNC opener from sysutil
	vncBin, vncArgs := sysutil.VNCCommand(vncAddr)
	return tea.ExecProcess(exec.Command(vncBin, vncArgs...), nil)
}

func (c *ComponentsDetail) Init() tea.Cmd { return nil }

func (c *ComponentsDetail) Update(msg tea.Msg) (view.Viewlet, tea.Cmd) {
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
	return c, nil, false
}

func (c *ComponentsDetail) View() string {
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
	// We always render the form, even if empty, to maintain alignment.
	return c.Form.View(safeWidth)
}

func (c *ComponentsDetail) Focus() tea.Cmd {
	c.BaseViewlet.Focus()
	return c.Form.Focus()
}

func (c *ComponentsDetail) Blur() {
	c.BaseViewlet.Blur()
}

func (c *ComponentsDetail) Focusable() bool {
	// If the form has no active elements, we shouldn't accept focus.
	// But Form doesn't have a "Focusable()" check for whole form easily accessible?
	// Currently all inputs are Disabled=true.
	// Let's iterate inputs or just return false since we know they are disabled.
	// However, user might want to select text? TUI doesn't support text selection yet.
	// So for now, return false.
	return false
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

	// 2. KVM Permission Denied
	if strings.Contains(raw, "Permission denied") && (strings.Contains(raw, "KVM") || strings.Contains(raw, "kvm")) {
		return "Hypervisor Error (ERR_QEMU) 🔐",
			" Nido could not access the KVM acceleration module.\n\n" +
				" • Problem: Your user does not have permission to use /dev/kvm.\n" +
				" • Fix: Run 'sudo usermod -aG kvm $USER && newgrp kvm'\n" +
				" • Note: If you just ran the fixer, you MUST restart your terminal session."
	}

	// 3. Out of Memory
	if strings.Contains(raw, "Cannot allocate memory") {
		return "Hypervisor Error (ERR_MEM) 🧠",
			" The host system does not have enough free RAM to hatch this VM.\n\n" +
				" • Hint: Close other applications or VMs.\n" +
				" • Hint: If running nested, increase the host VM's memory."
	}

	// 4. Port Conflict
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
