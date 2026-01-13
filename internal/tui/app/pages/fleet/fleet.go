package fleet

import (
	"fmt"
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

// FleetItem represents a VM in the fleet list (implements SidebarItem).
type FleetItem struct {
	Name    string
	State   string
	PID     int
	SSHPort int
	VNCPort int
	SSHUser string
}

func (i FleetItem) Title() string       { return i.Name }
func (i FleetItem) Description() string { return i.State }
func (i FleetItem) FilterValue() string { return i.Name }
func (i FleetItem) String() string      { return i.Name }
func (i FleetItem) Icon() string {
	if i.State == "running" {
		return "🟢"
	}
	return "🔴"
}
func (i FleetItem) IsAction() bool { return false }

// SpawnItem represents the action to create a new VM.
type SpawnItem struct{}

func (i SpawnItem) Title() string       { return "Spawn new bird (VM)" }
func (i SpawnItem) Description() string { return "" }
func (i SpawnItem) FilterValue() string { return "" }
func (i SpawnItem) String() string      { return i.Title() }
func (i SpawnItem) Icon() string        { return "+" }
func (i SpawnItem) IsAction() bool      { return true }

// FleetFilterItem represents a sidebar filter.
type FleetFilterItem struct {
	Label  string
	Filter string // "all", "running", "stopped"
}

func (i FleetFilterItem) Title() string       { return i.Label }
func (i FleetFilterItem) Description() string { return "" }
func (i FleetFilterItem) FilterValue() string { return i.Filter }
func (i FleetFilterItem) String() string      { return i.Label }
func (i FleetFilterItem) Icon() string        { return "" }
func (i FleetFilterItem) IsAction() bool      { return false }

// FleetDetail represents the detailed state of a VM.
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

// Fleet implements the Fleet viewlet showing VM list and details.
type Fleet struct {
	view.BaseViewlet

	items        []FleetItem
	selectedIdx  int
	table        widget.Table
	detail       FleetDetail
	highlightSSH bool
	highlightVNC bool

	// Pages
	// Components
	Sidebar      *widget.SidebarList
	MasterDetail *widget.MasterDetail
	Pages        *widget.PageManager
	activeFilter string
}

// NewFleet creates a new Fleet viewlet.
func NewFleet(prov provider.VMProvider) *Fleet {
	columns := []widget.TableColumn{
		{Title: "⬤", Width: 3}, // Status indicator
		{Title: "Name", Width: 15},
		{Title: "PID", Width: 8},
		{Title: "SSH", Width: 12},
		{Title: "VNC", Width: 10},
	}

	table := widget.NewTable(columns, nil, 10)

	f := &Fleet{
		table:  table,
		detail: FleetDetail{},
	}

	// Sidebar Items
	sidebarItems := []widget.SidebarItem{
		FleetFilterItem{Label: "All Birds", Filter: "all"},
		FleetFilterItem{Label: "Running", Filter: "running"},
		FleetFilterItem{Label: "Stopped", Filter: "stopped"},
	}

	// Sidebar
	t := theme.Current()
	styles := widget.SidebarStyles{
		Normal:   t.Styles.SidebarItem,
		Selected: t.Styles.SidebarItemSelected,
		Dim:      lipgloss.NewStyle().Foreground(t.Palette.TextDim),
		Action:   t.Styles.SidebarItemSelected.Copy(), // Not used for filters
	}
	f.Sidebar = widget.NewSidebarList(sidebarItems, theme.Width.Sidebar, styles, "🦅 ")

	// Pages
	f.Pages = widget.NewPageManager()
	f.Pages.AddPage("LIST", &FleetPageList{Parent: f})
	// We might eventually add a dedicated DETAIL page, but for now MasterDetail routes all selection
	// here. We want filters to just update the list view.
	// MasterDetail typically switches pages. Here we want to filter the same page.
	// So we might need to intercept SelectionMsg in Update or set AutoSwitch=false.

	// MasterDetail Wiring
	border := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(t.Palette.SurfaceSubtle)

	f.MasterDetail = widget.NewMasterDetail(f.Sidebar, f.Pages, border)
	f.MasterDetail.AutoSwitch = false // We handle selection manually for filtering

	f.activeFilter = "all"
	f.Pages.SwitchTo("LIST")

	return f
}

// Init initializes the Fleet viewlet.
func (f *Fleet) Init() tea.Cmd {
	return f.MasterDetail.Init()
}

// Update handles messages.
func (f *Fleet) Update(msg tea.Msg) (view.Viewlet, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case view.SelectionMsg:
		if sel := msg.Item; sel != nil {
			if _, ok := sel.(SpawnItem); ok {
				return f, func() tea.Msg { return view.SwitchTabMsg{TabIndex: 1} }
			}
			// Sidebar Selection (Filter)
			if filter, ok := sel.(FleetFilterItem); ok {
				f.activeFilter = filter.Filter
				// Trigger refilter logic (visual only for now as we don't have local filtering logic implemented fully yet)
				// Actually we should implement filtering in View() or re-set items.
				// Since we don't store full list separately from shown list, filtering is tricky without fetching again.
				// Let's defer full filtering logic implementation to next step if complex.
				// But user wants SIDEBAR.
				// So sticking to just showing it is step 1.
			}
		}
	case ops.VMListMsg:
		if msg.Err != nil {
			return f, func() tea.Msg { return view.LogMsg{Text: fmt.Sprintf("List failed: %v", msg.Err)} }
		}
		items := make([]FleetItem, len(msg.Items))
		for i, v := range msg.Items {
			items[i] = FleetItem{
				Name:    v.Name,
				State:   v.State,
				PID:     v.PID,
				SSHPort: v.SSHPort,
				VNCPort: v.VNCPort,
				SSHUser: v.SSHUser,
			}
		}
		f.SetItems(items)
		return f, func() tea.Msg { return view.StatusMsg{Loading: false, Operation: "refresh"} }

	case ops.VMDetailMsg:
		if msg.Err == nil {
			f.SetDetail(FleetDetail{
				Name:        msg.Detail.Name,
				State:       msg.Detail.State,
				PID:         msg.Detail.PID,
				IP:          msg.Detail.IP,
				SSHPort:     msg.Detail.SSHPort,
				VNCPort:     msg.Detail.VNCPort,
				SSHUser:     msg.Detail.SSHUser,
				DiskPath:    msg.Detail.DiskPath,
				DiskMissing: msg.Detail.DiskMissing,
			})
		}
	}

	// Update delegates to MasterDetail
	newMD, cmd := f.MasterDetail.Update(msg)
	f.MasterDetail = newMD.(*widget.MasterDetail)

	return f, cmd
}

// View renders the active page via MasterDetail.
func (f *Fleet) View() string {
	return f.MasterDetail.View()
}

func (f *Fleet) Resize(r layout.Rect) {
	f.BaseViewlet.Resize(r)
	f.MasterDetail.Resize(r)
}

func (f *Fleet) Shortcuts() []view.Shortcut {
	return f.MasterDetail.Shortcuts()
}

func (f *Fleet) HandleMouse(x, y int, msg tea.MouseMsg) (view.Viewlet, tea.Cmd, bool) {
	return f.MasterDetail.HandleMouse(x, y, msg)
}

// FleetPageList hosts the table + detail view.
type FleetPageList struct {
	view.BaseViewlet
	Parent *Fleet
}

func (p *FleetPageList) Init() tea.Cmd { return nil }
func (p *FleetPageList) Update(msg tea.Msg) (view.Viewlet, tea.Cmd) {
	f := p.Parent
	var cmd tea.Cmd

	// Detail Panel Interaction (Mouse)
	if mouseMsg, ok := msg.(tea.MouseMsg); ok && mouseMsg.Action == tea.MouseActionPress {
		y := mouseMsg.Y
		f.highlightSSH = false
		f.highlightVNC = false

		if y >= 14 && y <= 22 {
			localX := mouseMsg.X - 24
			if localX >= 0 {
				if localX < 14 { // START/STOP
					action := "start"
					if f.detail.State == "running" {
						action = "stop"
					}
					return p, func() tea.Msg { return FleetActionMsg{Action: action, Name: f.detail.Name} }
				} else if localX >= 14 && localX < 26 { // KILL
					return p, func() tea.Msg { return FleetActionMsg{Action: "stop", Name: f.detail.Name} }
				} else if localX >= 26 && localX < 44 { // DELETE
					return p, func() tea.Msg { return FleetActionMsg{Action: "delete", Name: f.detail.Name} }
				}
			}
		} else if y == 7 && f.detail.State == "running" && f.detail.SSHPort > 0 {
			if mouseMsg.X > 30 {
				f.highlightSSH = true
				return p, func() tea.Msg { return FleetActionMsg{Action: "ssh", Name: f.detail.Name} }
			}
		} else if y == 8 && f.detail.VNCPort > 0 {
			if mouseMsg.X > 30 {
				f.highlightVNC = true
				return p, func() tea.Msg { return FleetActionMsg{Action: "vnc", Name: f.detail.Name} }
			}
		}
	}

	f.table, cmd = f.table.Update(msg)
	return p, cmd
}

func (p *FleetPageList) View() string {
	f := p.Parent
	t := theme.Current()

	if len(f.items) == 0 {
		return lipgloss.NewStyle().Foreground(t.Palette.TextDim).Padding(2).Render("🪺 The nest is quiet.")
	}

	tableView := f.table.View()
	if f.detail.Name == "" {
		return lipgloss.NewStyle().MaxWidth(p.Width()).Render(tableView)
	}

	return lipgloss.NewStyle().MaxWidth(p.Width()).Render(
		layout.VStack(theme.Space.SM, tableView, f.renderDetail()),
	)
}

func (p *FleetPageList) Resize(r layout.Rect) {
	p.BaseViewlet.Resize(r)
	f := p.Parent
	if f.detail.Name != "" {
		f.table.SetHeight(r.Height - 13)
	} else {
		f.table.SetHeight(r.Height)
	}
}

func (p *FleetPageList) Shortcuts() []view.Shortcut {
	return []view.Shortcut{
		{Key: "↵", Label: "start/stop"},
		{Key: "x", Label: "stop"},
		{Key: "del", Label: "delete"},
		{Key: "s", Label: "ssh"},
	}
}

// FleetActionMsg
type FleetActionMsg struct {
	Action string
	Name   string
}

// Helpers moved from Fleet
func (f *Fleet) renderDetail() string {
	t := theme.Current()
	dimStyle := lipgloss.NewStyle().Foreground(t.Palette.TextDim)
	successStyle := lipgloss.NewStyle().Foreground(t.Palette.Success)
	accentStyle := lipgloss.NewStyle().Foreground(t.Palette.Accent)
	cardStyle := theme.Current().Styles.SidebarItem.Copy().Padding(1).Border(lipgloss.RoundedBorder()).BorderForeground(t.Palette.SurfaceSubtle)
	buttonStyle := lipgloss.NewStyle().Foreground(t.Palette.Text).Border(lipgloss.RoundedBorder()).Padding(0, 1).MarginRight(1)
	redButtonStyle := buttonStyle.Copy().BorderForeground(t.Palette.Error).Foreground(t.Palette.Error)
	titleStyle := lipgloss.NewStyle().Foreground(t.Palette.AccentStrong).Bold(true)

	statusEmoji := "💤"
	statusColor := dimStyle
	if f.detail.State == "running" {
		statusEmoji = "🐦"
		statusColor = successStyle
	}
	title := titleStyle.Render(fmt.Sprintf("%s %s", statusEmoji, strings.ToUpper(f.detail.Name)))

	sshVal := fmt.Sprintf("ssh -p %d %s@%s", f.detail.SSHPort, f.detail.SSHUser, f.detail.IP)
	if f.highlightSSH {
		sshVal = accentStyle.Render(sshVal)
	}

	vncLine := "—"
	if f.detail.VNCPort > 0 {
		vncLine = fmt.Sprintf(":%d", f.detail.VNCPort)
	}
	vncVal := vncLine
	if f.highlightVNC {
		vncVal = accentStyle.Render(vncVal)
	}

	infoCard := cardStyle.Render(layout.VStack(0,
		f.renderDetailLine("Status", statusColor.Render(f.detail.State)),
		f.renderDetailLine("SSH", sshVal),
		f.renderDetailLine("VNC", vncVal),
		f.renderDetailLine("PID", fmt.Sprintf("%d", f.detail.PID)),
	))

	btnStartStop := buttonStyle.Render("[↵] START")
	if f.detail.State == "running" {
		btnStartStop = redButtonStyle.Render("[↵] STOP")
	} else {
		btnStartStop = buttonStyle.BorderForeground(t.Palette.Success).Foreground(t.Palette.Success).Render("[↵] START")
	}

	return layout.VStack(theme.Space.SM, title, infoCard, layout.HStack(0, btnStartStop, redButtonStyle.Render("[X] KILL"), redButtonStyle.Render("[DEL] DELETE")))
}

func (f *Fleet) renderDetailLine(label, value string) string {
	t := theme.Current()
	labelStyle := lipgloss.NewStyle().Foreground(t.Palette.TextDim).Width(theme.Width.Label)
	valueStyle := lipgloss.NewStyle().Foreground(t.Palette.Text)
	return layout.HStack(0, labelStyle.Render(label), valueStyle.Render(value))
}

func (f *Fleet) SetItems(items []FleetItem) {
	f.items = items
	rows := make([]widget.TableRow, len(items))
	for i, item := range items {
		status := "○"
		if item.State == "running" {
			status = "●"
		}
		sshPort := "—"
		if item.SSHPort > 0 {
			sshPort = fmt.Sprintf(":%d", item.SSHPort)
		}
		vncPort := "—"
		if item.VNCPort > 0 {
			vncPort = fmt.Sprintf(":%d", item.VNCPort)
		}
		rows[i] = widget.TableRow{status, item.Name, fmt.Sprintf("%d", item.PID), sshPort, vncPort}
	}
	f.table.SetRows(rows)
}

func (f *Fleet) SetDetail(detail FleetDetail) { f.detail = detail }
func (f *Fleet) ClearDetail()                 { f.detail = FleetDetail{} }
