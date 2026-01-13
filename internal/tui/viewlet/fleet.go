package viewlet

import (
	"fmt"
	"strings"

	"github.com/Josepavese/nido/internal/tui/components"
	"github.com/Josepavese/nido/internal/tui/layout"
	"github.com/Josepavese/nido/internal/tui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FleetItem represents a VM in the fleet list.
type FleetItem struct {
	Name    string
	State   string
	PID     int
	SSHPort int
	VNCPort int
	SSHUser string
}

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
	BaseViewlet

	items        []FleetItem
	selectedIdx  int
	table        components.Table
	detail       FleetDetail
	highlightSSH bool
	highlightVNC bool
}

// NewFleet creates a new Fleet viewlet.
func NewFleet() *Fleet {
	columns := []components.TableColumn{
		{Title: "⬤", Width: 3}, // Status indicator
		{Title: "Name", Width: 15},
		{Title: "PID", Width: 8},
		{Title: "SSH", Width: 12},
		{Title: "VNC", Width: 10},
	}

	table := components.NewTable(columns, nil, 10)

	return &Fleet{
		table:  table,
		detail: FleetDetail{},
	}
}

// Init initializes the Fleet viewlet.
func (f *Fleet) Init() tea.Cmd {
	return nil // Refresh command would be returned here
}

// Update handles messages for the Fleet viewlet.
func (f *Fleet) Update(msg tea.Msg) (Viewlet, tea.Cmd) {
	var cmd tea.Cmd

	// Handle mouse events if delegated
	if mouseMsg, ok := msg.(tea.MouseMsg); ok && mouseMsg.Action == tea.MouseActionPress {
		// Detail Panel Interaction (Right Pane)
		// We need relative coordinates for hit testing buttons.
		y := mouseMsg.Y

		// Reset highlights
		f.highlightSSH = false
		f.highlightVNC = false

		if y >= 14 && y <= 22 {
			// Button Area logic (approximate X checks based on rendering)
			// Start/Stop: X 24-38 (local 0-14)
			// Kill: X 38-50 (local 14-26)
			// Delete: X 50-68 (local 26-44)

			// We subtract sidebar width (24) to get local X relative to viewlet start
			localX := mouseMsg.X - 24

			if localX >= 0 {
				if localX < 14 { // START/STOP
					action := "start"
					if f.detail.State == "running" {
						action = "stop"
					}
					return f, func() tea.Msg { return FleetActionMsg{Action: action, Name: f.detail.Name} }
				} else if localX >= 14 && localX < 26 { // KILL
					return f, func() tea.Msg { return FleetActionMsg{Action: "stop", Name: f.detail.Name} }
				} else if localX >= 26 && localX < 44 { // DELETE
					return f, func() tea.Msg { return FleetActionMsg{Action: "delete", Name: f.detail.Name} }
				}
			}
		} else if y == 7 && f.detail.State == "running" && f.detail.SSHPort > 0 {
			// SSH Click
			if mouseMsg.X > 30 {
				f.highlightSSH = true
				return f, func() tea.Msg { return FleetActionMsg{Action: "ssh", Name: f.detail.Name} }
			}
		} else if y == 8 && f.detail.VNCPort > 0 {
			// VNC Click
			if mouseMsg.X > 30 {
				f.highlightVNC = true
				return f, func() tea.Msg { return FleetActionMsg{Action: "vnc", Name: f.detail.Name} }
			}
		}
	}

	// Pass to table (updates selection if it supports mouse)
	f.table, cmd = f.table.Update(msg)

	return f, cmd
}

// FleetActionMsg is sent when a user clicks an action button
type FleetActionMsg struct {
	Action string
	Name   string
}

// View renders the Fleet viewlet.
func (f *Fleet) View() string {
	t := theme.Current()

	if len(f.items) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(t.Palette.TextDim).
			Padding(theme.Space.MD)

		return emptyStyle.Render("🪺 The nest is quiet. No VMs are currently making noise.")
	}
	// Render Table (Always visible if items exist)
	tableView := f.table.View()

	// If no detail is selected, show Table + Empty Placeholder
	if f.detail.Name == "" {
		// Calculate available width for detail panel
		// Table is approx 55 chars + padding.
		// Sidebar/Table separation handled by layout.HStack
		tableWidth := 55 // Fixed sum of columns + padding
		gap := theme.Space.SM
		availWidth := f.Width - tableWidth - gap
		if availWidth < 20 {
			availWidth = 20
		}

		emptyDetail := lipgloss.NewStyle().
			Foreground(t.Palette.TextDim).
			Padding(theme.Space.MD).
			Align(lipgloss.Center).
			Width(availWidth).
			Render(fmt.Sprintf("%s\n%s",
				lipgloss.NewStyle().Foreground(t.Palette.AccentStrong).Bold(true).Render("🦅 THE NEST"),
				"Select a bird from the nest to inspect its flight data."),
			)

		return layout.HStack(gap, tableView, emptyDetail)
	}

	// If Detail is selected, show Table | Detail
	return layout.HStack(theme.Space.SM, tableView, f.renderDetail())
}

// renderDetail renders the detail panel for a VM.
func (f *Fleet) renderDetail() string {
	t := theme.Current()

	// Colors
	dimStyle := lipgloss.NewStyle().Foreground(t.Palette.TextDim)
	errorStyle := lipgloss.NewStyle().Foreground(t.Palette.Error)
	successStyle := lipgloss.NewStyle().Foreground(t.Palette.Success)
	accentStyle := lipgloss.NewStyle().Foreground(t.Palette.Accent)
	cardStyle := theme.Current().Styles.SidebarItem.Copy().Padding(1).Border(lipgloss.RoundedBorder()).BorderForeground(t.Palette.SurfaceSubtle)
	buttonStyle := lipgloss.NewStyle().Foreground(t.Palette.Text).Border(lipgloss.RoundedBorder()).Padding(0, 1).MarginRight(1)
	redButtonStyle := buttonStyle.Copy().BorderForeground(t.Palette.Error).Foreground(t.Palette.Error)

	titleStyle := lipgloss.NewStyle().Foreground(t.Palette.AccentStrong).Bold(true)

	// Status Header
	statusEmoji := "💤"
	statusColor := dimStyle
	if f.detail.State == "running" {
		statusEmoji = "🐦"
		statusColor = successStyle
	}

	title := titleStyle.Render(fmt.Sprintf("%s %s", statusEmoji, strings.ToUpper(f.detail.Name)))

	// Connection Info
	vncLine := "—"
	if f.detail.VNCPort > 0 {
		vncLine = fmt.Sprintf("127.0.0.1:%d", f.detail.VNCPort)
	}

	sshVal := fmt.Sprintf("ssh -p %d %s@%s", f.detail.SSHPort, f.detail.SSHUser, f.detail.IP)
	if f.highlightSSH {
		sshVal = accentStyle.Render(sshVal)
	}

	vncVal := vncLine
	if f.highlightVNC {
		vncVal = accentStyle.Render(vncVal)
	}

	// Disk Info Helpers
	renderDisk := func() string {
		path := f.detail.DiskPath
		if f.detail.DiskMissing {
			path = errorStyle.Render(fmt.Sprintf("MISSING (%s)", f.detail.DiskPath))
		}
		// Approx available width logic: Viewlet width - padding
		// Table (55) + Gap (~2) + Detail Padding (~4) = approx 62
		avail := f.Width - 65
		if avail < 10 {
			avail = 10
		}
		return f.truncatePath(path, avail)
	}

	renderBacking := func() string {
		path := f.detail.BackingPath
		switch {
		case f.detail.BackingPath == "":
			return "—"
		case f.detail.BackingMissing:
			path = errorStyle.Render(fmt.Sprintf("MISSING (%s)", f.detail.BackingPath))
		}
		avail := f.Width - 65
		if avail < 10 {
			avail = 10
		}
		return f.truncatePath(path, avail)

	}

	infoCard := cardStyle.Render(layout.VStack(0,
		f.renderDetailLine("Status", statusColor.Render(f.detail.State)),
		f.renderDetailLine("SSH", sshVal),
		f.renderDetailLine("VNC", vncVal),
		f.renderDetailLine("Disk", renderDisk()),
		f.renderDetailLine("Backing", renderBacking()),
		f.renderDetailLine("PID", fmt.Sprintf("%d", f.detail.PID)),
	))

	// Dynamic Button 1
	btnStartStop := buttonStyle.Render("[↵] START")
	if f.detail.State == "running" {
		btnStartStop = buttonStyle.BorderForeground(t.Palette.Error).Foreground(t.Palette.Error).Render("[↵] STOP")
	} else {
		btnStartStop = buttonStyle.BorderForeground(t.Palette.Success).Foreground(t.Palette.Success).Render("[↵] START")
	}

	actions := layout.HStack(0,
		btnStartStop,
		redButtonStyle.Render("[X] KILL"),
		redButtonStyle.Render("[DEL] DELETE"),
	)

	// Responsive wrap for buttons if narrow
	// Assuming detail panel width is half of total viewlet width
	detailWidth := f.Width / 2
	if detailWidth < 46 {
		// Vertical Layout for buttons
		vButtonStyle := buttonStyle.Copy().MarginRight(0).MarginBottom(1)
		vRedStyle := redButtonStyle.Copy().MarginRight(0).MarginBottom(1)

		btnStartStopV := vButtonStyle.Render("[↵] START")
		if f.detail.State == "running" {
			btnStartStopV = vButtonStyle.BorderForeground(t.Palette.Error).Foreground(t.Palette.Error).Render("[↵] STOP")
		} else {
			btnStartStopV = vButtonStyle.BorderForeground(t.Palette.Success).Foreground(t.Palette.Success).Render("[↵] START")
		}

		actions = layout.VStack(0,
			btnStartStopV,
			vRedStyle.Render("[X] KILL"),
			vRedStyle.Render("[DEL] DELETE"),
		)
	}

	return layout.VStack(theme.Space.SM,
		title,
		infoCard,
		actions,
	)
}

func (f *Fleet) renderDetailLine(label, value string) string {
	t := theme.Current()
	labelStyle := lipgloss.NewStyle().Foreground(t.Palette.TextDim).Width(theme.Width.Label)
	valueStyle := lipgloss.NewStyle().Foreground(t.Palette.Text)
	return layout.HStack(0, labelStyle.Render(label), valueStyle.Render(value))
}

func (f *Fleet) truncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	// Head truncation: .../path/file.img
	return "..." + path[len(path)-(maxLen-3):]
}

// Resize updates the viewlet dimensions.
func (f *Fleet) Resize(width, height int) {
	f.BaseViewlet.Resize(width, height)
	f.table.SetHeight(height - 2)
}

// Shortcuts returns Fleet-specific shortcuts.
func (f *Fleet) Shortcuts() []Shortcut {
	return []Shortcut{
		{Key: "↵", Label: "start/stop"},
		{Key: "del", Label: "delete"},
		{Key: "i", Label: "info"},
		{Key: "s", Label: "ssh"},
	}
}

// SetItems updates the VM list.
func (f *Fleet) SetItems(items []FleetItem) {
	f.items = items

	// Convert to table rows
	rows := make([]components.TableRow, len(items))
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

		rows[i] = components.TableRow{
			status,
			item.Name,
			fmt.Sprintf("%d", item.PID),
			sshPort,
			vncPort,
		}
	}

	f.table.SetRows(rows)
}

// SetDetail updates the detail view for the selected VM.
func (f *Fleet) SetDetail(detail FleetDetail) {
	f.detail = detail
}

// ClearDetail clears the detail view.
func (f *Fleet) ClearDetail() {
	f.detail = FleetDetail{}
}

// SelectedItem returns the currently selected VM.
func (f *Fleet) SelectedItem() *FleetItem {
	if f.selectedIdx < 0 || f.selectedIdx >= len(f.items) {
		return nil
	}
	return &f.items[f.selectedIdx]
}
