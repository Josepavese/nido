package gui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Josepavese/nido/internal/config"
	"github.com/Josepavese/nido/internal/provider"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tab int

const (
	tabFleet tab = iota
	tabHatchery
	tabLogs
	tabHelp
)

type fleetFocus int

const (
	focusList fleetFocus = iota
	focusHatch
)

type vmItem struct {
	name    string
	state   string
	pid     int
	sshPort int
	vncPort int
	sshUser string
}

func (i vmItem) Title() string {
	indicator := "🔴"
	if i.state == "running" {
		indicator = "🟢"
	}
	return fmt.Sprintf("%s %s", indicator, i.name)
}
func (i vmItem) Description() string { return i.state }
func (i vmItem) FilterValue() string { return i.name }

type operation string

const (
	opNone    operation = ""
	opSpawn   operation = "spawn"
	opStart   operation = "start"
	opStop    operation = "stop"
	opDelete  operation = "delete"
	opRefresh operation = "refresh"
	opInfo    operation = "info"
)

type tickMsg struct{}
type vmListMsg struct{ items []list.Item }
type logMsg struct {
	level string
	text  string
}
type opResultMsg struct {
	op  operation
	err error
}
type detailMsg struct {
	name   string
	detail provider.VMDetail
	err    error
}

type spawnState struct {
	gui        bool
	inputs     []textinput.Model
	focusIndex int
	errorMsg   string
}

type hatcheryState struct {
	// 0: Spawn VM, 1: Create Template
	Action int

	// Inputs
	Inputs     []textinput.Model
	FocusIndex int

	// Source Selection
	SelectedSource string     // The chosen value (e.g. "ubuntu:24.04" or "my-template")
	IsSelecting    bool       // Modal is open
	SourceList     list.Model // The list for the modal
}

type model struct {
	prov provider.VMProvider
	cfg  *config.Config

	width  int
	height int

	activeTab tab
	list      list.Model
	page      paginator.Model
	spinner   spinner.Model
	progress  progress.Model
	loading   bool
	op        operation

	detailName string
	detail     provider.VMDetail
	fleetFocus fleetFocus

	spawn    spawnState
	hatchery hatcheryState // New Full-screen Hatchery State
	logs     []string
}

func newHatcheryState(cfg *config.Config) hatcheryState {
	// Inputs
	name := textinput.New()
	name.Placeholder = ""
	name.CharLimit = 50
	name.Focus()

	// Initial List for Modal
	// Delegate (minimal)
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.Styles.SelectedTitle = sidebarItemSelectedStyle.Foreground(lipgloss.Color("#00FFFF")) // Cyan selection

	l := list.New([]list.Item{}, delegate, 40, 10)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)

	return hatcheryState{
		Action:         0, // 0=Spawn
		Inputs:         []textinput.Model{name},
		FocusIndex:     0,
		SelectedSource: "", // Empty initially
		SourceList:     l,
		IsSelecting:    false,
	}
}

// Deprecated: newSpawnState kept for now until full migration
func newSpawnState(cfg *config.Config) spawnState {
	name := textinput.New()
	name.Placeholder = "vm-name"
	name.Prompt = "" // Handled purely by gui layout
	name.CharLimit = 50

	template := textinput.New()
	template.Placeholder = cfg.TemplateDefault
	template.Prompt = ""
	template.CharLimit = 120

	userData := textinput.New()
	userData.Placeholder = "(optional path)"
	userData.Prompt = ""
	userData.CharLimit = 200

	inputs := []textinput.Model{name, template, userData}
	inputs[0].Focus()

	return spawnState{
		gui:        true,
		inputs:     inputs,
		focusIndex: 0,
	}
}

func initialModel(prov provider.VMProvider, cfg *config.Config) model {
	items := []list.Item{}
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.Styles.SelectedTitle = sidebarItemSelectedStyle
	delegate.Styles.NormalTitle = sidebarItemStyle

	l := list.New(items, delegate, 28, 10)
	l.SetShowTitle(false) // Disable title
	l.SetShowHelp(false)
	l.SetShowStatusBar(false) // Hide status bar to prevent duplicate "No items"
	l.DisableQuitKeybindings()
	l.SetFilteringEnabled(false)
	l.SetShowPagination(false)

	pg := paginator.New()
	pg.Type = paginator.Dots
	pg.PerPage = 10
	pg.InactiveDot = dimStyle.Render("•")
	pg.ActiveDot = accentStyle.Render("◉")

	spin := spinner.New()
	spin.Spinner = spinner.Dot
	spin.Style = accentStyle

	prog := progress.New(progress.WithScaledGradient(string(colors.AccentStrong), string(colors.Accent)))
	prog.ShowPercentage = false

	return model{
		prov:      prov,
		cfg:       cfg,
		activeTab: tabFleet,
		list:      l,
		page:      pg,
		spinner:   spin,
		progress:  prog,
		logs:      []string{"Nido GUI ready. Systems nominal."},
		spawn:     newSpawnState(cfg),
		hatchery:  newHatcheryState(cfg),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		tea.EnableMouseCellMotion,
		tea.Tick(time.Millisecond*80, func(time.Time) tea.Msg { return tickMsg{} }),
		m.refreshCmd(),
	)
}

func (m model) refreshCmd() tea.Cmd {
	return func() tea.Msg {
		vms, err := m.prov.List()
		if err != nil {
			return logMsg{level: "error", text: fmt.Sprintf("List failed: %v", err)}
		}
		items := make([]list.Item, 0, len(vms))
		for _, v := range vms {
			items = append(items, vmItem{
				name:    v.Name,
				state:   v.State,
				pid:     v.PID,
				sshPort: v.SSHPort,
				vncPort: v.VNCPort,
				sshUser: v.SSHUser,
			})
		}
		return vmListMsg{items: items}
	}
}

func (m model) infoCmd(name string) tea.Cmd {
	return func() tea.Msg {
		detail, err := m.prov.Info(name)
		return detailMsg{name: name, detail: detail, err: err}
	}
}

func (m model) spawnCmd(name, template, userData string, guiFlag bool) tea.Cmd {
	return func() tea.Msg {
		opts := provider.VMOptions{
			DiskPath:     template,
			UserDataPath: userData,
			Gui:          guiFlag,
			SSHUser:      "",
		}
		err := m.prov.Spawn(name, opts)
		return opResultMsg{op: opSpawn, err: err}
	}
}

func (m model) startCmd(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.prov.Start(name, provider.VMOptions{Gui: true})
		return opResultMsg{op: opStart, err: err}
	}
}

func (m model) stopCmd(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.prov.Stop(name, true)
		return opResultMsg{op: opStop, err: err}
	}
}

func (m model) deleteCmd(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.prov.Delete(name)
		return opResultMsg{op: opDelete, err: err}
	}
}

// Custom message for loading sources (Images/Templates)
type sourcesLoadedMsg struct {
	items []list.Item
	err   error
}

// Simple string item for list
type listItem string

func (i listItem) FilterValue() string { return string(i) }
func (i listItem) Title() string       { return string(i) }
func (i listItem) Description() string { return "Source Image / Template" }

func (m model) fetchSources(action int) tea.Cmd {
	return func() tea.Msg {
		var srcList []string

		if action == 0 { // Spawn VM -> List Images AND Templates
			images, err := m.prov.ListImages()
			if err != nil {
				return sourcesLoadedMsg{err: err}
			}
			for _, img := range images {
				srcList = append(srcList, fmt.Sprintf("[IMAGE] %s", img))
			}
			templates, err := m.prov.ListTemplates()
			if err != nil {
				return sourcesLoadedMsg{err: err}
			}
			for _, tpl := range templates {
				srcList = append(srcList, fmt.Sprintf("[TEMPLATE] %s", tpl))
			}
		} else { // Create Template
			// For creating a template, source usually isn't relevant in this wizard flow yet
			return sourcesLoadedMsg{items: []list.Item{}}
		}

		items := make([]list.Item, len(srcList))
		for i, s := range srcList {
			items[i] = listItem(s) // Use simple string item wrapper
		}
		return sourcesLoadedMsg{items: items}
	}
}
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(28, m.height-6)
	case tickMsg:
		m.spinner, _ = m.spinner.Update(msg)
		cmds = append(cmds, tea.Tick(time.Millisecond*80, func(time.Time) tea.Msg { return tickMsg{} }))

		// Update inputs for blink
		for i := range m.hatchery.Inputs {
			var cmd tea.Cmd
			m.hatchery.Inputs[i], cmd = m.hatchery.Inputs[i].Update(msg)
			cmds = append(cmds, cmd)
		}
	case vmListMsg:
		m.list.SetItems(msg.items)
		m.page.SetTotalPages((len(msg.items) + m.page.PerPage - 1) / m.page.PerPage)
		m.loading = false
		m.op = opNone
		// Fix: Always refresh detail if we have one, to sync text status with sidebar
		if m.detailName != "" {
			cmds = append(cmds, m.infoCmd(m.detailName))
		} else if len(msg.items) > 0 {
			// Initial selection
			if sel := m.list.SelectedItem(); sel != nil {
				m.detailName = sel.(vmItem).name
				cmds = append(cmds, m.infoCmd(m.detailName))
			}
		}
	case detailMsg:
		if msg.err != nil {
			if msg.name == m.detailName {
				m.detailName = ""
				m.detail = provider.VMDetail{}
			}
			m.logs = append(m.logs, fmt.Sprintf("Info failed: %v", msg.err))
		} else if msg.name == m.detailName {
			m.detail = msg.detail
		}
	case sourcesLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.logs = append(m.logs, fmt.Sprintf("Failed to load sources: %v", msg.err))
		} else {
			m.hatchery.SourceList.SetItems(msg.items)
			m.hatchery.IsSelecting = true
		}
	case logMsg:
		m.logs = append(m.logs, msg.text)
	case opResultMsg:
		m.loading = false
		if msg.err != nil {
			m.logs = append(m.logs, fmt.Sprintf("Operation %s failed: %v", msg.op, msg.err))
		} else {
			m.logs = append(m.logs, fmt.Sprintf("Operation %s complete.", msg.op))
		}
		m.op = opNone
		cmds = append(cmds, m.refreshCmd())
	case tea.KeyMsg:
		// Hatchery Modal Interaction
		if m.activeTab == tabHatchery && m.hatchery.IsSelecting {
			switch msg.String() {
			case "esc":
				m.hatchery.IsSelecting = false
				return m, nil
			case "enter":
				sel := m.hatchery.SourceList.SelectedItem()
				if sel != nil {
					m.hatchery.SelectedSource = sel.FilterValue()
					// Move to next field after selection
					m.hatchery.IsSelecting = false
					m.hatchery.FocusIndex++
					m.updateHatcheryFocus()
				}
				return m, nil
			}
			var cmd tea.Cmd
			m.hatchery.SourceList, cmd = m.hatchery.SourceList.Update(msg)
			return m, cmd
		}

		var handled bool
		var cmd tea.Cmd
		var newModel tea.Model
		newModel, cmd, handled = m.handleKey(msg)
		if handled {
			return newModel, cmd
		}
		// If not handled, fall through to component updates
		m = newModel.(model)
	case tea.MouseMsg:
		newModel, cmd := m.handleMouse(msg)
		return newModel, cmd
	}

	if m.activeTab == tabHatchery {
		for i := range m.spawn.inputs {
			var cmd tea.Cmd
			m.spawn.inputs[i], cmd = m.spawn.inputs[i].Update(msg)
			cmds = append(cmds, cmd)
		}
	} else if m.activeTab == tabFleet {
		if m.fleetFocus == focusList {
			prev := m.detailName
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			cmds = append(cmds, cmd)
			if sel := m.list.SelectedItem(); sel != nil {
				m.detailName = sel.(vmItem).name
				if m.detailName != prev {
					cmds = append(cmds, m.infoCmd(m.detailName))
				}
			}

			// Capture key for focus switching
			if keyMsg, ok := msg.(tea.KeyMsg); ok {
				if keyMsg.String() == "down" {
					if m.list.Index() == len(m.list.Items())-1 {
						m.fleetFocus = focusHatch
					}
				}
			}
		} else {
			// Button is focused
			if keyMsg, ok := msg.(tea.KeyMsg); ok {
				switch keyMsg.String() {
				case "up":
					m.fleetFocus = focusList
				case "enter":
					m.activeTab = tabHatchery
					m.fleetFocus = focusList // reset for next time
				}
			}
		}
	}
	return m, tea.Batch(cmds...)
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	// 1. Global Shortcuts
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit, true
	case "1":
		m.activeTab = tabFleet
		return m, nil, true
	case "2":
		m.activeTab = tabHatchery
		return m, nil, true
	case "3":
		m.activeTab = tabLogs
		return m, nil, true
	case "4", "h":
		m.activeTab = tabHelp
		return m, nil, true
	case "r":
		m.loading = true
		m.op = opRefresh
		return m, m.refreshCmd(), true
	}

	// 2. Navigation (Arrows)
	// We want to handle Left/Right for Tab switching, BUT NOT if we are in Hatchery inputs or Fleet list?
	// Actually, user wants Left/Right to switch tabs generally, unless focused on input?
	// Existing logic checked for Hatchery input focus.

	if msg.String() == "left" || msg.String() == "right" {
		// Exception: In Hatchery AND focused on Name (0) or Source (1) -> let component handle it
		if m.activeTab == tabHatchery && m.hatchery.FocusIndex <= 1 {
			return m, nil, false // Let component handle it
		}

		// Perform Switch
		prevTab := m.activeTab
		if msg.String() == "left" {
			m.activeTab = (m.activeTab - 1 + 4) % 4
		} else {
			m.activeTab = (m.activeTab + 1) % 4
		}

		// Trap Fix: If we just entered Hatchery via arrows, focus the button (neutral)
		if m.activeTab == tabHatchery && prevTab != tabHatchery {
			m.spawn.focusIndex = len(m.spawn.inputs) // focus on button
			m.updateFocus()
		}
		return m, nil, true
	}

	// 3. Tab Specific Logic
	if m.activeTab == tabHatchery {
		// Shortcuts for Action Switching
		if msg.String() == "1" {
			m.hatchery.Action = 0
			return m, nil, true
		} else if msg.String() == "2" {
			m.hatchery.Action = 1
			return m, nil, true
		}

		maxIndex := 3
		if m.hatchery.Action == 1 {
			maxIndex = 2 // No options for template
		}

		switch msg.String() {
		case "tab", "down":
			m.hatchery.FocusIndex++
			if m.hatchery.FocusIndex > maxIndex {
				m.hatchery.FocusIndex = 0
			}
			m.updateHatcheryFocus()
			return m, nil, true
		case "shift+tab", "up":
			m.hatchery.FocusIndex--
			if m.hatchery.FocusIndex < 0 {
				m.hatchery.FocusIndex = maxIndex
			}
			m.updateHatcheryFocus()
			return m, nil, true
		case " ":
			// Toggle Options (Focus Index 2 is Options in Action 0)
			if m.hatchery.Action == 0 && m.hatchery.FocusIndex == 2 {
				m.spawn.gui = !m.spawn.gui // Reuse spawn flag for now
				return m, nil, true
			}
			return m, nil, false // Space in Name input
		case "enter":
			// Button Trigger
			if m.hatchery.FocusIndex == maxIndex {
				newM, cmd := m.submitHatchery()
				return newM, cmd, true
			}
			// Source Trigger
			if m.hatchery.FocusIndex == 1 {
				m.loading = true // Show spinner while fetching?
				return m, m.fetchSources(m.hatchery.Action), true
			}

			// Next field
			m.hatchery.FocusIndex++
			m.updateHatcheryFocus()
			return m, nil, true
		}

		// Input Handling (Name is at Index 0)
		if m.hatchery.FocusIndex == 0 {
			var cmd tea.Cmd
			m.hatchery.Inputs[0], cmd = m.hatchery.Inputs[0].Update(msg)
			return m, cmd, true
		}

		// Source Cycling (Index 1)
		if m.hatchery.FocusIndex == 1 {
			items := m.hatchery.SourceList.Items()
			if len(items) > 0 {
				currIdx := -1
				for i, item := range items {
					if item.FilterValue() == m.hatchery.SelectedSource {
						currIdx = i
						break
					}
				}
				if msg.String() == "left" {
					currIdx = (currIdx - 1 + len(items)) % len(items)
					m.hatchery.SelectedSource = items[currIdx].FilterValue()
					return m, nil, true
				} else if msg.String() == "right" {
					currIdx = (currIdx + 1) % len(items)
					m.hatchery.SelectedSource = items[currIdx].FilterValue()
					return m, nil, true
				}
			}
		}

		return m, nil, false
	}

	if m.activeTab == tabFleet {
		switch msg.String() {
		case "enter":
			if sel := m.list.SelectedItem(); sel != nil {
				item := sel.(vmItem)
				if item.state == "running" {
					m.loading = true
					m.op = opStop
					return m, m.stopCmd(item.name), true
				}
				m.loading = true
				m.op = opStart
				return m, m.startCmd(item.name), true
			}
		case "x":
			if sel := m.list.SelectedItem(); sel != nil {
				m.loading = true
				m.op = opStop
				return m, m.stopCmd(sel.(vmItem).name), true
			}
		case "delete":
			if sel := m.list.SelectedItem(); sel != nil {
				m.loading = true
				m.op = opDelete
				return m, m.deleteCmd(sel.(vmItem).name), true
			}
		}
		// Allow up/down to fall through to list
		return m, nil, false
	}

	return m, nil, false
}

func (m *model) updateFocus() {
	for i := range m.spawn.inputs {
		if i == m.spawn.focusIndex {
			m.spawn.inputs[i].Focus()
		} else {
			m.spawn.inputs[i].Blur()
		}
	}
}

func (m model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Action != tea.MouseActionPress {
		return m, nil
	}

	// 1. Header Logic
	if msg.Y == 0 {
		// Exit Button Click (Rightmost 4 chars)
		if msg.X >= m.width-4 {
			return m, tea.Quit
		}

		// Tab Switching
		availableWidth := m.width - 4
		tabWidth := availableWidth / 4
		if tabWidth > 0 {
			clickIndex := msg.X / tabWidth
			if clickIndex >= 0 && clickIndex <= 3 {
				m.activeTab = tab(clickIndex)
				return m, nil
			}
		}
	}

	// 2. Sidebar Logic (Fleet View)
	if m.activeTab == tabFleet {
		// Sidebar interactions
		if msg.X < 34 {
			row := msg.Y - 5 // Offset 5 (Header 2 + SubHeader 2 + 1 padding)
			if row >= 0 {
				pageStart := m.list.Paginator.Page * m.list.Paginator.PerPage
				index := pageStart + row
				if index >= 0 && index < len(m.list.Items()) {
					m.list.Select(index)
					if sel := m.list.SelectedItem(); sel != nil {
						m.detailName = sel.(vmItem).name
						return m, m.infoCmd(m.detailName)
					}
				}
			}
		} else {
			// Main Area Interactions (Buttons)
			// Y Offset calculation:
			// Header (2) + SubHeader (2) = 4
			// Title (2ish) + Card (6ish) = ~8
			// Buttons start approx at Y=12 or 13.
			// Let's broaden the hit area for usability.
			if msg.Y >= 11 && msg.Y <= 15 {
				// X Offsets (Rough estimates based on text length + padding)
				// Sidebar (34)
				// Buttons: "[ENTER] POWER" (~15), "[X] KILL" (~10), "[DEL] RM" (~10)
				// Spacing is handled by styles (MarginRight 1).
				// If JoinHorizontal stacks them:
				// Btn1: 0 .. ~15
				// Btn2: 16 .. ~26
				// Btn3: 27 .. ~37
				// + Offset 34 => 34..49, 50..60, 61..71

				localX := msg.X - 34 // relative to main content

				if sel := m.list.SelectedItem(); sel != nil {
					item := sel.(vmItem)
					if localX >= 0 && localX < 16 { // [ENTER] START/STOP
						if item.state == "running" {
							m.loading = true
							m.op = opStop
							return m, m.stopCmd(item.name)
						}
						m.loading = true
						m.op = opStart
						return m, m.startCmd(item.name)
					} else if localX >= 16 && localX < 26 { // [X] KILL
						m.loading = true
						m.op = opStop
						return m, m.stopCmd(item.name)
					} else if localX >= 26 && localX < 45 { // [DEL] DELETE
						m.loading = true
						m.op = opDelete
						return m, m.deleteCmd(item.name)
					}
				}
			}

			// 3. Speed Hatch Button (Bottom of Fleet Main Area)
			if msg.Y >= m.height-lipgloss.Height(m.renderFooter())-2 {
				m.activeTab = tabHatchery
				m.fleetFocus = focusList // reset
				return m, nil
			}
		}
	}
	return m, nil
}

func (m model) submitHatchery() (tea.Model, tea.Cmd) {
	name := m.hatchery.Inputs[0].Value()
	source := m.hatchery.SelectedSource

	if name == "" {
		m.logs = append(m.logs, "Hatchery: Name is required to spawn!")
		return m, nil
	}
	if source == "" {
		m.logs = append(m.logs, "Hatchery: Source (Image/Template) is required!")
		return m, nil
	}

	m.loading = true
	m.activeTab = tabFleet // Switch back to view progress

	if m.hatchery.Action == 0 {
		// SPAWN
		m.op = opSpawn
		// Default template to source if not specified?
		// Actually Source IS the template/image.
		return m, m.spawnCmd(name, source, "", m.spawn.gui) // Empty UserData for now
	} else {
		// CREATE TEMPLATE
		// Stub
		return m, func() tea.Msg {
			return opResultMsg{op: "create-template", err: nil} // TODO: Implement
		}
	}
}

func (m *model) updateHatcheryFocus() {
	// Only Input[0] (Name) can be focused text input
	if m.hatchery.FocusIndex == 0 {
		m.hatchery.Inputs[0].Focus()
	} else {
		m.hatchery.Inputs[0].Blur()
	}
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing Mission Control..."
	}

	// Header
	header := headerStyle.Width(m.width).Render(m.renderTabs())
	headerHeight := 2

	// Sub-Header
	subHeader := subHeaderStyle.Width(m.width - 2).Render(m.renderSubHeader())
	subHeaderHeight := 2

	// Footer
	footer := m.renderFooter()
	footerHeight := lipgloss.Height(footer)

	// Body height
	bodyHeight := m.height - headerHeight - subHeaderHeight - footerHeight
	if bodyHeight < 0 {
		bodyHeight = 0
	}

	var body string

	if m.activeTab == tabHatchery {
		// Full Screen Hatchery
		body = m.renderHatcheryFullScreen(m.width, bodyHeight)
	} else {
		// Standard Split View (Sidebar + Main)

		// Sidebar
		sidebar := sidebarStyle.Height(bodyHeight).Render(m.list.View())
		sidebarWidth := lipgloss.Width(sidebar)

		// Main Content
		mainWidth := m.width - sidebarWidth
		if mainWidth < 0 {
			mainWidth = 0
		}

		var content string
		switch m.activeTab {
		case tabFleet:
			content = m.renderFleet(bodyHeight)
		case tabLogs:
			content = m.renderLogs(bodyHeight)
		case tabHelp:
			content = m.renderHelp()
		}

		mainArea := mainContentStyle.
			Width(mainWidth - 4).
			Height(bodyHeight).
			Render(content)

		body = lipgloss.JoinHorizontal(lipgloss.Top, sidebar, mainArea)
	}

	// Use standard rendering to allow transparency
	return lipgloss.JoinVertical(lipgloss.Left, header, subHeader, body, footer)
}

func (m model) renderHatcheryFullScreen(w, h int) string {
	if m.hatchery.IsSelecting {
		// Set list dimensions slightly smaller than screen
		lw, lh := 60, 20
		if w < 60 {
			lw = w - 4
		}
		if h < 20 {
			lh = h - 4
		}
		m.hatchery.SourceList.SetSize(lw, lh)
		m.hatchery.SourceList.Title = "Select Source"

		modal := cardStyle.BorderForeground(lipgloss.Color("39")).Render(m.hatchery.SourceList.View())

		return lipgloss.Place(w, h,
			lipgloss.Center, lipgloss.Center,
			modal,
		)
	}

	// 1. Action Selector (Tabs)
	actionStyle := dimStyle
	selectedActionStyle := activeTabStyle

	spawnBtn := actionStyle.Render(" [1] SPAWN VM ")
	templateBtn := actionStyle.Render(" [2] CREATE TEMPLATE ")

	if m.hatchery.Action == 0 {
		spawnBtn = selectedActionStyle.Render(" [1] SPAWN VM ")
	} else {
		templateBtn = selectedActionStyle.Render(" [2] CREATE TEMPLATE ")
	}

	actionBar := lipgloss.JoinHorizontal(lipgloss.Top, spawnBtn, "   ", templateBtn)

	// 2. Form Content
	var form strings.Builder

	// Input: Name
	labelColor := dimStyle
	if m.hatchery.FocusIndex == 0 {
		labelColor = accentStyle
	}
	form.WriteString(fmt.Sprintf("%-15s %s\n\n", labelColor.Render("Name:"), m.hatchery.Inputs[0].View()))

	// Input: Source (Image or Template)
	labelColor = dimStyle
	sourceVal := m.hatchery.SelectedSource
	if sourceVal == "" {
		sourceVal = "( Select Source... )"
	}
	if m.hatchery.FocusIndex == 1 {
		labelColor = accentStyle
		sourceVal = accentStyle.Render(sourceVal)
	} else {
		sourceVal = dimStyle.Render(sourceVal)
	}
	form.WriteString(fmt.Sprintf("%-15s %s\n\n", labelColor.Render("Source:"), sourceVal))

	// Options (Only for Spawn)
	if m.hatchery.Action == 0 {
		guiCheck := "[ ]"
		if m.spawn.gui { // Reuse spawn flag for now
			guiCheck = "[x]"
		}
		labelColor = dimStyle
		if m.hatchery.FocusIndex == 2 {
			labelColor = accentStyle
			guiCheck = accentStyle.Render(guiCheck)
		}
		form.WriteString(fmt.Sprintf("%-15s %s Enable GUI (VNC)\n\n", labelColor.Render("Options:"), guiCheck))
	}

	// Submit Button
	btnText := "[ START INCUBATION ]"
	if m.hatchery.Action == 1 {
		btnText = "[ FREEZE TEMPLATE ]"
	}

	btn := dimStyle.Render(btnText)
	// Focus index mapping: 0=Name, 1=Source, 2=Options, 3=Submit
	targetIndex := 3
	if m.hatchery.Action == 1 {
		targetIndex = 2 // No options for template creation
	}

	if m.hatchery.FocusIndex == targetIndex {
		btn = activeTabStyle.Render(btnText)
	}

	formContent := lipgloss.JoinVertical(lipgloss.Left,
		actionBar,
		"\n\n",
		form.String(),
		"\n",
		btn,
	)

	// Left-aligned view with standard padding
	return containerStyle.Padding(2, 4).Render(formContent)
}

// Deprecated: renderHatchery logic moved to renderHatcheryFullScreen
func (m model) renderHatchery() string { return "" }

func (m model) renderSubHeader() string {
	var context, nav string
	arrows := "Use ←/→ arrows to navigate tabs."

	switch m.activeTab {
	case tabFleet:
		context = "FLEET VIEW"
		nav = "Monitor and manage active instances. " + arrows
	case tabHatchery:
		context = "HATCHERY"
		nav = "Spawn new birds. Tab to cycle fields. " + arrows
	case tabLogs:
		context = "FLIGHT LOGS"
		nav = "System activity log. " + arrows
	case tabHelp:
		context = "HELP CENTER"
		nav = "Command reference. " + arrows
	}

	return lipgloss.JoinHorizontal(lipgloss.Top,
		subHeaderContextStyle.Render(context),
		"  ", // Explicit spacer with background color inheritance
		subHeaderNavStyle.Render(nav),
	)
}

func (m model) renderFleet(height int) string {
	if m.detailName == "" {
		// Align empty state with standard layout
		title := titleStyle.Render("🦅 THE NEST")
		content := cardStyle.Render(dimStyle.Render("Select a bird from the nest to inspect its flight data."))

		// Speed Hatch Button even in empty state
		hStyle := hatchButtonStyle
		if m.fleetFocus == focusHatch {
			hStyle = hatchButtonActiveStyle
		}
		btnHatch := hStyle.Width(40).Render("[⊕] SPEED HATCH (Spawn new bird)")

		spacer := strings.Repeat("\n", height-lipgloss.Height(title)-lipgloss.Height(content)-lipgloss.Height(btnHatch)-2)
		return lipgloss.JoinVertical(lipgloss.Left, title, content, spacer, btnHatch)
	}

	statusEmoji := "💤"
	statusColor := dimStyle
	if m.detail.State == "running" {
		statusEmoji = "🐦"
		statusColor = successStyle
	}

	title := titleStyle.Render(fmt.Sprintf("%s %s", statusEmoji, strings.ToUpper(m.detailName)))

	vncLine := "—"
	if m.detail.VNCPort > 0 {
		vncLine = fmt.Sprintf("127.0.0.1:%d", m.detail.VNCPort)
	}

	infoCard := cardStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
		m.renderDetailLine("Status", statusColor.Render(m.detail.State)),
		m.renderDetailLine("SSH", fmt.Sprintf("ssh -p %d %s@%s", m.detail.SSHPort, m.detail.SSHUser, m.detail.IP)),
		m.renderDetailLine("VNC", vncLine),
		m.renderDetailLine("Disk", m.renderDiskLine()),
		m.renderDetailLine("Backing", m.renderBackingLine()),
		m.renderDetailLine("PID", fmt.Sprintf("%d", m.detail.PID)),
	))

	// Dynamic Button 1
	btnStartStop := buttonStyle.Render("[↵] START")
	if m.detail.State == "running" {
		btnStartStop = buttonStyle.BorderForeground(colors.Error).Foreground(colors.Error).Render("[↵] STOP")
	} else {
		btnStartStop = buttonStyle.BorderForeground(colors.Success).Foreground(colors.Success).Render("[↵] START")
	}

	actions := lipgloss.JoinHorizontal(lipgloss.Top,
		btnStartStop,
		redButtonStyle.Render("[X] KILL"),
		redButtonStyle.Render("[DEL] DELETE"),
	)

	// Speed Hatch Button
	hStyle := hatchButtonStyle
	if m.fleetFocus == focusHatch {
		hStyle = hatchButtonActiveStyle
	}
	btnHatch := hStyle.Width(40).Render("[⊕] SPEED HATCH (Spawn new bird)")

	mainContent := lipgloss.JoinVertical(lipgloss.Left,
		title,
		infoCard,
		actions,
	)

	// Calculate vertical gap to push button to the bottom
	contentHeight := lipgloss.Height(mainContent)
	gapHeight := height - contentHeight - lipgloss.Height(btnHatch) - 2
	if gapHeight < 0 {
		gapHeight = 0
	}
	spacer := strings.Repeat("\n", gapHeight)

	return lipgloss.JoinVertical(lipgloss.Left, mainContent, spacer, btnHatch)
}

func (m model) renderDetailLine(label, value string) string {
	return lipgloss.JoinHorizontal(lipgloss.Top, labelStyle.Render(label), valueStyle.Render(value))
}

func (m model) renderDiskLine() string {
	if m.detail.DiskMissing {
		return errorStyle.Render(fmt.Sprintf("MISSING (%s)", m.detail.DiskPath))
	}
	return m.detail.DiskPath
}

func (m model) renderBackingLine() string {
	switch {
	case m.detail.BackingPath == "":
		return "—"
	case m.detail.BackingMissing:
		return errorStyle.Render(fmt.Sprintf("MISSING (%s)", m.detail.BackingPath))
	default:
		return m.detail.BackingPath
	}
}

func (m model) renderTabs() string {
	var tabs []string
	labels := []string{"1 FLEET", "2 HATCHERY", "3 LOGS", "4 HELP"}

	// Calculate exact width to ensure button is pinned right
	// Reserve 6 chars for exit (4 button + 2 padding/safety)
	availableWidth := m.width - 6
	tabWidth := availableWidth / 4

	for i, label := range labels {
		style := tabStyle.Width(tabWidth).Align(lipgloss.Center)
		if i == int(m.activeTab) {
			style = activeTabStyle.Width(tabWidth).Align(lipgloss.Center)
		}
		tabs = append(tabs, style.Render(label))
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	rowWidth := lipgloss.Width(row)

	exitBtn := errorStyle.Render("[X]")
	exitWidth := lipgloss.Width(exitBtn)

	// Add spacer to push X to the edge
	gap := m.width - rowWidth - exitWidth
	if gap < 0 {
		gap = 0
	}
	spacer := strings.Repeat(" ", gap)

	return lipgloss.JoinHorizontal(lipgloss.Top, row, spacer, exitBtn)
}

func (m model) renderLogs(height int) string {
	title := titleStyle.Render("📜 FLIGHT LOGS")

	// Constrain logs to body height - title
	// Use the passed height (bodyHeight) for accurate clipping
	limit := height - 2
	if limit < 1 {
		limit = 1
	}
	start := len(m.logs) - limit
	if start < 0 {
		start = 0
	}

	logLines := strings.Join(m.logs[start:], "\n")
	// Ensure transparency
	return lipgloss.NewStyle().Render(
		lipgloss.JoinVertical(lipgloss.Left, title,
			cardStyle.Width(m.width-36).Height(limit).Render(dimStyle.Render(logLines))),
	)
}

func (m model) renderHelp() string {
	title := titleStyle.Render("❓ COMMAND CENTER HELP")

	text := `Tabs: 1 Fleet · 2 Hatchery · 3 Logs · 4 Help
Fleet: click or ↑/↓ select · ↵ start/stop · x kill · del delete
Hatchery: tab through fields · ←/→ cycle · ↵ hatch
Mouse: click tabs, list rows, and action buttons.`

	// Ensure transparency
	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		cardStyle.Width(m.width-4).Render(dimStyle.Render(text)),
	)
}

func (m model) renderFooter() string {
	status := " SYSTEMS NOMINAL 🟢 | 🪺 Happy Hatching! | v1.0"
	if m.loading {
		status = fmt.Sprintf(" %s EXECUTING %s... ", m.spinner.View(), strings.ToUpper(string(m.op)))
	}

	return footerStyle.Width(m.width).Render(status)
}

// Run starts the TUI with given provider/config.
func Run(ctx context.Context, prov provider.VMProvider, cfg *config.Config) error {
	p := tea.NewProgram(initialModel(prov, cfg), tea.WithContext(ctx), tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}
