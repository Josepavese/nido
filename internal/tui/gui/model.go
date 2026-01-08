package gui

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath" // Added sort package
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/Josepavese/nido/internal/config"
	"github.com/Josepavese/nido/internal/image"
	"github.com/Josepavese/nido/internal/provider"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tab int

const (
	tabFleet tab = iota
	tabHatchery
	tabLogs
	tabConfig
	tabHelp
)

type fleetFocus int

const (
	focusList fleetFocus = iota
)

type hatcheryFocus int

const (
	focusHatchSidebar hatcheryFocus = iota
	focusHatchForm
)

type configFocus int

const (
	focusConfigSidebar configFocus = iota
	focusConfigForm
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
	name := i.name
	// Truncate name if too long for 18-char sidebar:
	// Sidebar Width (18) - Indicator (2) - Space (1) - Padding (2) = 13 chars safe
	if len(name) > 13 {
		name = name[:12] + "..."
	}
	return fmt.Sprintf("%s %s", indicator, name)
}
func (i vmItem) Description() string { return i.state }
func (i vmItem) FilterValue() string { return i.name }
func (i vmItem) String() string      { return i.Title() }

type spawnItem struct{}

func (i spawnItem) Title() string       { return "+ Spawn new bird (VM)" }
func (i spawnItem) Description() string { return "" }
func (i spawnItem) FilterValue() string { return "" }
func (i spawnItem) String() string      { return i.Title() }

type operation string

const (
	opNone           operation = ""
	opSpawn          operation = "spawn"
	opStart          operation = "start"
	opStop           operation = "stop"
	opDelete         operation = "delete"
	opRefresh        operation = "refresh"
	opInfo           operation = "info"
	opCreateTemplate operation = "create-template"
)

type tickMsg struct{}
type vmListMsg struct{ items []list.Item }
type logMsg struct {
	level string
	text  string
}
type opResultMsg struct {
	op   operation
	err  error
	path string // Optional: for templates
}
type detailMsg struct {
	name   string
	detail provider.VMDetail
	err    error
}

type updateCheckMsg struct {
	current string
	latest  string
	err     error
}

type cacheListMsg struct {
	items []CacheItem
	err   error
}

type cacheStatsMsg struct {
	stats CacheStats
	err   error
}

type cachePruneMsg struct {
	err error
}

func (m model) isInputFocused() bool {
	if m.activeTab == tabHatchery && m.hatcheryFocus == focusHatchForm && m.hatchery.FocusIndex == 0 {
		return true
	}
	if m.activeTab == tabConfig && m.configFocus == focusConfigForm {
		return true
	}
	return false
}

type hatcheryState struct {
	// Sidebar
	Sidebar list.Model

	// Unified state for form navigation
	FocusIndex int

	// Mode 0: SPAWN
	SpawnInputs    []textinput.Model
	SpawnSource    string
	SpawnSelecting bool

	// Mode 1: CREATE TEMPLATE
	TemplateInputs    []textinput.Model
	TemplateSource    string
	TemplateSelecting bool

	// Shared Source Modal state
	IsSelecting bool       // Global modal open flag
	SourceList  list.Model // The list for the modal

	// Options
	GUI bool
}

type configState struct {
	Sidebar   list.Model
	Input     textinput.Model
	ActiveKey string
	ErrorMsg  string

	// Update state
	CurrentVersion string
	LatestVersion  string
	UpdateChecking bool
	UpdateRunning  bool

	// Cache state
	CacheList  []CacheItem
	CacheStats CacheStats
}

type CacheItem struct {
	Name     string
	Version  string
	Size     string
	Modified string
}

type CacheStats struct {
	TotalImages int
	TotalSize   string
	Oldest      string
	Newest      string
}

type hatchTypeItem struct {
	title string
	desc  string
}

// Custom Stringer methods for items
func (i hatchTypeItem) String() string      { return i.title }
func (i hatchTypeItem) Title() string       { return i.title }
func (i hatchTypeItem) Description() string { return i.desc }
func (i hatchTypeItem) FilterValue() string { return i.title }

type configItem struct {
	key  string
	val  string
	desc string
}

func (i configItem) String() string      { return fmt.Sprintf("%-18s", i.key) }
func (i configItem) Title() string       { return fmt.Sprintf("%-18s", i.key) }
func (i configItem) Description() string { return "" }
func (i configItem) FilterValue() string { return i.key }

// customDelegate for Sidebar items to prevent padding shifts
type customDelegate struct{}

func (d customDelegate) Height() int                             { return 1 }
func (d customDelegate) Spacing() int                            { return 0 }
func (d customDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d customDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	// Check for Spawn Item first
	if _, ok := listItem.(spawnItem); ok {

		str := "+ Spawn new bird (VM)"
		if index == m.Index() {
			fmt.Fprint(w, hatchButtonActiveStyle.Render(str))
		} else {
			fmt.Fprint(w, hatchButtonStyle.Render(str))
		}
		return
	}

	str, ok := listItem.(fmt.Stringer)
	if !ok {
		return
	}

	// Check if this item is selected
	if index == m.Index() {
		// Just render the string with the selected style, NO extra padding/margins
		fmt.Fprint(w, sidebarItemSelectedStyle.Render(str.String()))
	} else {
		// Render normal
		fmt.Fprint(w, sidebarItemStyle.Render(str.String()))
	}
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

	hatchery      hatcheryState
	hatcheryFocus hatcheryFocus
	config        configState
	configFocus   configFocus
	logs          []string
	logViewport   viewport.Model

	// New fields from the instruction
	quitting         bool
	err              error
	downloading      bool
	downloadProgress float64
	downloadChan     chan float64

	// Quick Actions UI state
	highlightSSH bool
	highlightVNC bool
}

type resetHighlightMsg struct {
	action string
}

func newHatcheryState(cfg *config.Config) hatcheryState {
	// Custom Delegate to prevent "jumping" (remove default padding/borders)
	d := customDelegate{}

	items := []list.Item{
		hatchTypeItem{title: "SPAWN VM"},
		hatchTypeItem{title: "CREATE TEMPLATE"},
	}

	sb := list.New(items, d, 28, 5) // Matches active view width
	sb.SetShowTitle(false)
	sb.SetShowHelp(false)
	sb.SetShowStatusBar(false)
	sb.SetShowPagination(false) // Disable aggressive pagination for small lists

	// Spawn: Inputs
	nameSpawn := textinput.New()
	nameSpawn.Placeholder = "vm-name"
	nameSpawn.Prompt = ""
	nameSpawn.CharLimit = 50
	nameSpawn.Focus()

	// Template: Inputs
	nameTemplate := textinput.New()
	nameTemplate.Placeholder = "template-name"
	nameTemplate.Prompt = ""
	nameTemplate.CharLimit = 50

	// Source List for Modal (Shared)
	sl := list.New([]list.Item{}, d, 28, 10)
	sl.SetShowTitle(false)
	sl.SetShowHelp(false)
	sl.SetShowStatusBar(false)

	return hatcheryState{
		Sidebar:        sb,
		FocusIndex:     0,
		SpawnInputs:    []textinput.Model{nameSpawn},
		TemplateInputs: []textinput.Model{nameTemplate},
		SourceList:     sl,
		GUI:            true, // Default to GUI enabled
	}
}

func newConfigState(cfg *config.Config) configState {
	// Custom Delegate to prevent "jumping"
	d := customDelegate{}

	items := getConfigItems(cfg)

	// User requested pagination to 4 elements.
	// Re-enable visual pagination (dots) as requested.
	sb := list.New(items, d, 28, 10)
	sb.SetShowPagination(true)
	sb.SetShowTitle(false)
	sb.SetShowHelp(false)
	sb.SetShowStatusBar(false)

	ti := textinput.New()
	ti.CharLimit = 100

	return configState{
		Sidebar: sb,
		Input:   ti,
	}
}

func initialModel(prov provider.VMProvider, cfg *config.Config) model {
	items := []list.Item{}
	// Use customDelegate to match Config/Hatchery styling (no extra padding)
	d := customDelegate{}

	// Extra reduced sidebar width to 18
	l := list.New(items, d, 18, 10)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.DisableQuitKeybindings()
	l.SetFilteringEnabled(false)
	l.SetShowPagination(true)

	pg := paginator.New()
	pg.Type = paginator.Dots
	pg.PerPage = 10
	pg.InactiveDot = dimStyle.Render("•")
	pg.ActiveDot = accentStyle.Render("◉")

	spin := spinner.New()
	spin.Style = accentStyle

	prog := progress.New(progress.WithScaledGradient(string(colors.AccentStrong), string(colors.Accent)))
	prog.ShowPercentage = true

	// Initialize Viewport for Logs
	vp := viewport.New(0, 9)
	vp.SetContent(strings.Join([]string{fmt.Sprintf("[%s] Nido GUI ready. Systems nominal.", time.Now().Format("15:04:05"))}, "\n"))

	return model{
		prov:        prov,
		cfg:         cfg,
		activeTab:   tabFleet,
		list:        l,
		page:        pg,
		spinner:     spin,
		progress:    prog,
		loading:     bool(false),
		logs:        []string{fmt.Sprintf("[%s] Nido GUI ready. Systems nominal.", time.Now().Format("15:04:05"))},
		logViewport: vp,
		hatchery:    newHatcheryState(cfg),
		config:      newConfigState(cfg),
	}
}

// Helper to get inactive delegate (visual deselect)
func getInactiveDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	d.ShowDescription = false
	d.Styles.SelectedTitle = sidebarItemStyle // Render selected as normal
	d.Styles.NormalTitle = sidebarItemStyle
	return d
}

// Helper to get active delegate
func getActiveDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	d.ShowDescription = false
	d.Styles.SelectedTitle = sidebarItemSelectedStyle
	d.Styles.NormalTitle = sidebarItemStyle
	return d
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

		// Sort VMs alphabetically by Name
		sort.Slice(vms, func(i, j int) bool {
			return strings.ToLower(vms[i].Name) < strings.ToLower(vms[j].Name)
		})

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
		// Append Spawn Item at the end
		items = append(items, spawnItem{})
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

func (m model) createTemplateCmd(vmName, templateName string) tea.Cmd {
	return func() tea.Msg {
		path, err := m.prov.CreateTemplate(vmName, templateName)
		return opResultMsg{op: "create-template", err: err, path: path}
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

func (m model) checkUpdateCmd() tea.Cmd {
	return func() tea.Msg {
		// Get current version
		out, err := exec.Command("nido", "version").Output()
		if err != nil {
			return updateCheckMsg{err: err}
		}
		current := strings.TrimSpace(string(out))
		// Extract version number (e.g., "Nido v4.3.6 (State: Evolved)" -> "v4.3.6")
		parts := strings.Fields(current)
		if len(parts) >= 2 {
			current = parts[1]
		}
		return updateCheckMsg{current: current, latest: current} // TODO: Check GitHub for latest
	}
}

func (m model) cacheListCmd() tea.Cmd {
	return func() tea.Msg {
		items, err := m.prov.ListCachedImages()
		if err != nil {
			return cacheListMsg{err: err}
		}
		var cacheItems []CacheItem
		for _, img := range items {
			cacheItems = append(cacheItems, CacheItem{
				Name:    img.Name,
				Version: img.Version,
				Size:    img.Size,
			})
		}
		return cacheListMsg{items: cacheItems}
	}
}

func (m model) cacheStatsCmd() tea.Cmd {
	return func() tea.Msg {
		info, err := m.prov.CacheInfo()
		if err != nil {
			return cacheStatsMsg{err: err}
		}
		return cacheStatsMsg{stats: CacheStats{
			TotalImages: info.Count,
			TotalSize:   info.TotalSize,
		}}
	}
}

func (m model) cachePruneCmd() tea.Cmd {
	return func() tea.Msg {
		err := m.prov.CachePrune(true) // unused only
		return cachePruneMsg{err: err}
	}
}

func (m model) saveConfigCmd(key, value string) tea.Cmd {
	return func() tea.Msg {
		// Find config file path
		home, _ := os.UserHomeDir()
		path := filepath.Join(home, ".nido", "config.env")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			cwd, _ := os.Getwd()
			path = filepath.Join(cwd, "config", "config.env")
		}

		err := config.UpdateConfig(path, key, value)
		if err != nil {
			return logMsg{level: "error", text: fmt.Sprintf("Save failed: %v", err)}
		}

		// Reload config into memory
		newCfg, _ := config.LoadConfig(path)
		*m.cfg = *newCfg

		return configSavedMsg{key: key, value: value}
	}
}

// Custom message for loading sources (Images/Templates)
type sourcesLoadedMsg struct {
	items []list.Item
	err   error
}

// Custom message for config saved
type configSavedMsg struct{ key, value string }

// Simple string item for list
type listItem string

func getConfigItems(cfg *config.Config) []list.Item {
	items := []list.Item{
		// System actions (at top)
		configItem{
			key:  "UPDATE",
			val:  "",
			desc: "Check for updates and upgrade Nido.",
		},
		configItem{
			key:  "CACHE",
			val:  "",
			desc: "Manage cached cloud images.",
		},
		// Config values
		configItem{
			key:  "BACKUP_DIR",
			val:  cfg.BackupDir,
			desc: "Path to store template backups.",
		},
		configItem{
			key:  "IMAGE_DIR",
			val:  cfg.ImageDir,
			desc: "Directory for cached cloud images.",
		},
		configItem{
			key:  "LINKED_CLONES",
			val:  fmt.Sprintf("%v", cfg.LinkedClones),
			desc: "Use Copy-on-Write for disk efficiency.",
		},
		configItem{
			key:  "SSH_USER",
			val:  cfg.SSHUser,
			desc: "Default user for SSH connections.",
		},
		configItem{
			key:  "TEMPLATE_DEFAULT",
			val:  cfg.TemplateDefault,
			desc: "Default source template for new VMs.",
		},
	}

	return items
}

func (i listItem) String() string      { return string(i) }
func (i listItem) FilterValue() string { return string(i) }
func (i listItem) Title() string       { return string(i) }
func (i listItem) Description() string { return "Source Image / Template" }

func (m model) fetchSources(action int) tea.Cmd {
	return func() tea.Msg {
		var srcList []string

		// Use the passed action explicitly
		if action == 0 { // Spawn VM -> List Images AND Templates
			images, err := m.prov.ListImages()
			if err != nil {
				return sourcesLoadedMsg{err: err}
			}
			// Sort Images
			sort.Strings(images)

			templates, err := m.prov.ListTemplates()
			if err != nil {
				return sourcesLoadedMsg{err: err}
			}
			// Sort Templates
			sort.Strings(templates)

			// Append Templates FIRST, then Images
			for _, tpl := range templates {
				srcList = append(srcList, fmt.Sprintf("[TEMPLATE] %s", tpl))
			}
			for _, img := range images {
				srcList = append(srcList, fmt.Sprintf("[IMAGE] %s", img))
			}
		} else { // Create Template -> List VMs
			vms, err := m.prov.List()
			if err != nil {
				return sourcesLoadedMsg{err: err}
			}
			// Sort VMs
			sort.Slice(vms, func(i, j int) bool {
				return vms[i].Name < vms[j].Name
			})
			for _, vm := range vms {
				srcList = append(srcList, fmt.Sprintf("[VM] %s", vm.Name))
			}
		}

		if len(srcList) == 0 {
			// Add a dummy entry if nothing found to avoid blank modal
			return sourcesLoadedMsg{err: fmt.Errorf("no images or templates found")}
		}

		items := make([]list.Item, len(srcList))
		for i, s := range srcList {
			items[i] = listItem(s)
		}
		return sourcesLoadedMsg{items: items}
	}
}
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case downloadProgressMsg:
		if m.downloading {
			m.downloadProgress = float64(msg)
			return m, waitForDownloadProgress(m.downloadChan)
		}

	case downloadFinishedMsg:
		m.downloading = false
		m.loading = false // Reset loading state
		if msg.err != nil {
			m.logs = append(m.logs, fmt.Sprintf("[%s] Download failed: %v", time.Now().Format("15:04:05"), msg.err))
			m.logViewport.SetContent(strings.Join(m.logs, "\n"))
			m.logViewport.GotoBottom()
			return m, nil
		}
		m.logs = append(m.logs, fmt.Sprintf("[%s] Download complete for %s.", time.Now().Format("15:04:05"), msg.name))
		m.logViewport.SetContent(strings.Join(m.logs, "\n"))

		// Resume Spawn
		// Resume Spawn
		name := m.hatchery.SpawnInputs[0].Value()
		m.activeTab = tabFleet
		m.op = opSpawn
		m.loading = true
		return m, m.spawnCmd(name, msg.path, "", m.hatchery.GUI)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Calculate available height for content
		// Header(2) + SubHeader(2) + Footer(1) + Spacing(3) = 8
		bodyHeight := m.height - 8
		if bodyHeight < 1 {
			bodyHeight = 1
		}

		// Apply dynamic height to all components
		m.list.SetSize(18, bodyHeight)             // Fleet
		m.hatchery.Sidebar.SetSize(18, bodyHeight) // Hatchery
		m.config.Sidebar.SetSize(18, bodyHeight)   // Config

		// Logs Viewport
		m.logViewport.Width = m.width - 8
		m.logViewport.Height = bodyHeight

	case tickMsg:
		m.spinner, _ = m.spinner.Update(msg)
		cmds = append(cmds, tea.Tick(time.Millisecond*80, func(time.Time) tea.Msg { return tickMsg{} }))

		// Update inputs for blink
		// Update inputs for blink
		for i := range m.hatchery.SpawnInputs {
			var cmd tea.Cmd
			m.hatchery.SpawnInputs[i], cmd = m.hatchery.SpawnInputs[i].Update(msg)
			cmds = append(cmds, cmd)
		}
		for i := range m.hatchery.TemplateInputs {
			var cmd tea.Cmd
			m.hatchery.TemplateInputs[i], cmd = m.hatchery.TemplateInputs[i].Update(msg)
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
				if v, ok := sel.(vmItem); ok {
					m.detailName = v.name
					cmds = append(cmds, m.infoCmd(m.detailName))
				}
			}
		}
	case detailMsg:
		if msg.err != nil {
			if msg.name == m.detailName {
				m.detailName = ""
				m.detail = provider.VMDetail{}
			}
			m.logs = append(m.logs, fmt.Sprintf("[%s] Info failed: %v", time.Now().Format("15:04:05"), msg.err))
			m.logViewport.SetContent(strings.Join(m.logs, "\n"))
			m.logViewport.GotoBottom()
		} else if msg.name == m.detailName {
			m.detail = msg.detail
		}
	case sourcesLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.logs = append(m.logs, fmt.Sprintf("[%s] Failed to load sources: %v", time.Now().Format("15:04:05"), msg.err))
			m.logViewport.SetContent(strings.Join(m.logs, "\n"))
			m.logViewport.GotoBottom()
		} else {
			m.hatchery.SourceList.SetItems(msg.items)
			m.hatchery.IsSelecting = true
		}
	case resetHighlightMsg:
		if msg.action == "ssh" {
			m.highlightSSH = false
		} else {
			m.highlightVNC = false
		}
	case logMsg:
		m.logs = append(m.logs, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg.text))
		// Update Viewport Content
		m.logViewport.SetContent(strings.Join(m.logs, "\n"))
		m.logViewport.GotoBottom()
	case opResultMsg:
		m.loading = false
		if msg.err != nil {
			m.logs = append(m.logs, fmt.Sprintf("[%s] Operation %s failed: %v", time.Now().Format("15:04:05"), msg.op, msg.err))
		} else {
			m.logs = append(m.logs, fmt.Sprintf("[%s] Operation %s complete.", time.Now().Format("15:04:05"), msg.op))
		}
		m.logViewport.SetContent(strings.Join(m.logs, "\n"))
		m.logViewport.GotoBottom()
		m.op = opNone
		cmds = append(cmds, m.refreshCmd())
	case configSavedMsg:
		m.loading = false
		m.logs = append(m.logs, fmt.Sprintf("[%s] Config %s updated to %s", time.Now().Format("15:04:05"), msg.key, msg.value))
		m.logViewport.SetContent(strings.Join(m.logs, "\n"))
		m.logViewport.GotoBottom()
		// Refresh sidebar items to reflect new state (e.g. toggles)
		idx := m.config.Sidebar.Index()
		m.config.Sidebar.SetItems(getConfigItems(m.cfg))
		m.config.Sidebar.Select(idx)
	case updateCheckMsg:
		m.config.UpdateChecking = false
		if msg.err != nil {
			m.logs = append(m.logs, fmt.Sprintf("[%s] Update check failed: %v", time.Now().Format("15:04:05"), msg.err))
		} else {
			m.config.CurrentVersion = msg.current
			m.config.LatestVersion = msg.latest
			m.logs = append(m.logs, fmt.Sprintf("[%s] Version check complete: %s", time.Now().Format("15:04:05"), msg.current))
		}
		m.logViewport.SetContent(strings.Join(m.logs, "\n"))
		m.logViewport.GotoBottom()
	case cacheListMsg:
		m.loading = false
		if msg.err != nil {
			m.logs = append(m.logs, fmt.Sprintf("[%s] Cache list failed: %v", time.Now().Format("15:04:05"), msg.err))
		} else {
			m.config.CacheList = msg.items
		}
		m.logViewport.SetContent(strings.Join(m.logs, "\n"))
		m.logViewport.GotoBottom()
	case cacheStatsMsg:
		m.loading = false
		if msg.err != nil {
			m.logs = append(m.logs, fmt.Sprintf("[%s] Cache info failed: %v", time.Now().Format("15:04:05"), msg.err))
		} else {
			m.config.CacheStats = msg.stats
		}
		m.logViewport.SetContent(strings.Join(m.logs, "\n"))
		m.logViewport.GotoBottom()
	case cachePruneMsg:
		m.loading = false
		if msg.err != nil {
			m.logs = append(m.logs, fmt.Sprintf("[%s] Cache prune failed: %v", time.Now().Format("15:04:05"), msg.err))
		} else {
			m.logs = append(m.logs, fmt.Sprintf("[%s] Cache pruned successfully", time.Now().Format("15:04:05")))
			cmds = append(cmds, m.cacheListCmd(), m.cacheStatsCmd()) // Refresh cache view
		}
		m.logViewport.SetContent(strings.Join(m.logs, "\n"))
		m.logViewport.GotoBottom()
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
					item := sel.(listItem)
					if m.hatchery.Sidebar.Index() == 0 { // Spawn VM
						m.hatchery.SpawnSource = string(item)
					} else { // Create Template
						m.hatchery.TemplateSource = string(item)
					}
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
		if m.activeTab == tabLogs {
			var cmd tea.Cmd
			m.logViewport, cmd = m.logViewport.Update(msg)
			return m, cmd
		}
		newModel, cmd := m.handleMouse(msg)
		return newModel, cmd
	}

	// Handle Hatchery Input Updates
	if m.activeTab == tabHatchery && m.hatcheryFocus == focusHatchForm {
		var inputCmds []tea.Cmd
		if m.hatchery.Sidebar.Index() == 0 { // Spawn VM
			if m.hatchery.FocusIndex < len(m.hatchery.SpawnInputs) {
				var cmd tea.Cmd
				m.hatchery.SpawnInputs[m.hatchery.FocusIndex], cmd = m.hatchery.SpawnInputs[m.hatchery.FocusIndex].Update(msg)
				inputCmds = append(inputCmds, cmd)
			}
		} else { // Create Template
			if m.hatchery.FocusIndex < len(m.hatchery.TemplateInputs) {
				var cmd tea.Cmd
				m.hatchery.TemplateInputs[m.hatchery.FocusIndex], cmd = m.hatchery.TemplateInputs[m.hatchery.FocusIndex].Update(msg)
				inputCmds = append(inputCmds, cmd)
			}
		}
		// Append these commands to the main cmds slice
		cmds = append(cmds, inputCmds...)
	} else if m.activeTab == tabFleet {
		prev := m.detailName
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)

		if sel := m.list.SelectedItem(); sel != nil {
			if _, ok := sel.(spawnItem); ok {
				// Special Case: Spawn Item Selected
				m.detailName = ""
				m.detail = provider.VMDetail{} // Clear detail view
			} else {
				// VM Item Selected
				m.detailName = sel.(vmItem).name
				if m.detailName != prev {
					cmds = append(cmds, m.infoCmd(m.detailName))
				}
			}
		}
	}
	return m, tea.Batch(cmds...)
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	// 1. Global Shortcuts
	switch msg.String() {
	case "q":
		if m.isInputFocused() {
			return m, nil, false
		}
		return m, tea.Quit, true
	case "ctrl+c":
		return m, tea.Quit, true
	case "1":
		if m.isInputFocused() {
			return m, nil, false
		}
		m.activeTab = tabFleet
		return m, nil, true
	case "2":
		if m.isInputFocused() {
			return m, nil, false
		}
		m.activeTab = tabHatchery
		return m, nil, true
	case "3":
		if m.isInputFocused() {
			return m, nil, false
		}
		m.activeTab = tabLogs
		return m, nil, true
	case "4":
		if m.isInputFocused() {
			return m, nil, false
		}
		m.activeTab = tabConfig
		return m, nil, true
	case "5", "h":
		if m.isInputFocused() {
			return m, nil, false
		}
		m.activeTab = tabHelp
		return m, nil, true
	case "r":
		if m.isInputFocused() {
			return m, nil, false
		}
		m.loading = true
		m.op = opRefresh
		return m, m.refreshCmd(), true
	}

	// 2. Navigation (Arrows)
	if msg.String() == "left" || msg.String() == "right" {
		// Exception: In Hatchery AND focused on Form AND in input field -> let form handle arrows
		if m.activeTab == tabHatchery && m.hatcheryFocus == focusHatchForm && m.hatchery.FocusIndex == 0 {
			return m, nil, false
		}
		// Exception: In Config AND focused on Form -> let form handle arrows
		if m.activeTab == tabConfig && m.configFocus == focusConfigForm {
			return m, nil, false
		}

		// Perform Switch
		if msg.String() == "left" {
			m.activeTab = (m.activeTab - 1 + 5) % 5
		} else {
			m.activeTab = (m.activeTab + 1) % 5
		}

		// Reset focus when entering tabs
		if m.activeTab == tabHatchery {
			m.hatcheryFocus = focusHatchSidebar
		} else if m.activeTab == tabConfig {
			m.configFocus = focusConfigSidebar
		}
		return m, nil, true
	}

	// 3. Tab Specific Logic
	if m.activeTab == tabHatchery {
		if m.hatcheryFocus == focusHatchSidebar {
			switch msg.String() {
			case "right", "tab":
				m.hatcheryFocus = focusHatchForm
				m.hatchery.FocusIndex = 0
				m.updateHatcheryFocus()
				return m, nil, true
			case "enter":
				// If on sidebar, clicking enter also enters form (common flow)
				m.hatcheryFocus = focusHatchForm
				m.hatchery.FocusIndex = 0
				m.updateHatcheryFocus()
				return m, nil, true
			}
			var cmd tea.Cmd
			m.hatchery.Sidebar, cmd = m.hatchery.Sidebar.Update(msg)
			return m, cmd, true
		} else {
			// Form Interaction
			maxIndex := 3 // Spawn: Name, Source, GUI, Button
			if m.hatchery.Sidebar.Index() == 1 {
				maxIndex = 2 // Template: Name, Source, Button
			}

			switch msg.String() {
			case "tab", "shift+tab":
				if msg.String() == "tab" {
					m.hatchery.FocusIndex++
				} else {
					m.hatchery.FocusIndex--
				}
				if m.hatchery.FocusIndex > maxIndex {
					m.hatchery.FocusIndex = 0
				} else if m.hatchery.FocusIndex < 0 {
					m.hatchery.FocusIndex = maxIndex
				}
				m.updateHatcheryFocus()
				return m, nil, true

			case "up":
				m.hatchery.FocusIndex--
				if m.hatchery.FocusIndex < 0 {
					m.hatchery.FocusIndex = maxIndex
				}
				m.updateHatcheryFocus()
				return m, nil, true
			case "down":
				m.hatchery.FocusIndex++
				if m.hatchery.FocusIndex > maxIndex {
					m.hatchery.FocusIndex = 0
				}
				m.updateHatcheryFocus()
				return m, nil, true
			case "left", "esc":
				m.hatcheryFocus = focusHatchSidebar
				return m, nil, true
			case "enter", " ":
				if m.hatchery.FocusIndex == 1 {
					// Source Trigger
					m.loading = true
					return m, m.fetchSources(m.hatchery.Sidebar.Index()), true
				}
				// GUI Toggle Trigger
				if m.hatchery.FocusIndex == 2 && m.hatchery.Sidebar.Index() == 0 {
					m.hatchery.GUI = !m.hatchery.GUI
					return m, nil, true
				}
				// Submit Button Trigger
				if m.hatchery.FocusIndex == maxIndex {
					newM, cmd := m.submitHatchery()
					return newM, cmd, true
				}

				// Next field
				m.hatchery.FocusIndex++
				m.updateHatcheryFocus()
				return m, nil, true
			}
			// Input Handling
			if m.hatchery.FocusIndex == 0 {
				var cmd tea.Cmd
				if m.hatchery.Sidebar.Index() == 0 {
					m.hatchery.SpawnInputs[0], cmd = m.hatchery.SpawnInputs[0].Update(msg)
				} else {
					m.hatchery.TemplateInputs[0], cmd = m.hatchery.TemplateInputs[0].Update(msg)
				}
				return m, cmd, true
			}
		}
	} else if m.activeTab == tabConfig {
		if m.configFocus == focusConfigSidebar {
			switch msg.String() {
			case "right", "tab", "enter":
				sel := m.config.Sidebar.SelectedItem()
				if sel != nil {
					item := sel.(configItem)

					// UPDATE action
					if item.key == "UPDATE" {
						m.config.UpdateChecking = true
						return m, m.checkUpdateCmd(), true
					}

					// CACHE action
					if item.key == "CACHE" {
						m.configFocus = focusConfigForm
						return m, nil, true // Prune on second Enter in form
					}

					// Boolean Toggle Logic
					if item.key == "LINKED_CLONES" {
						// Toggle immediately
						current := item.val == "true"
						newVal := "false"
						if !current {
							newVal = "true"
						}
						// Save immediately
						m.loading = true
						return m, m.saveConfigCmd(item.key, newVal), true
					}

					m.config.ActiveKey = item.key
					m.config.Input.SetValue(item.val)
					m.config.Input.Focus()
					m.configFocus = focusConfigForm
				}
				return m, nil, true
			}
			var cmd tea.Cmd
			prevIdx := m.config.Sidebar.Index()
			m.config.Sidebar, cmd = m.config.Sidebar.Update(msg)

			// Dynamic Input Update on Scroll
			if m.config.Sidebar.Index() != prevIdx {
				sel := m.config.Sidebar.SelectedItem()
				if sel != nil {
					item := sel.(configItem)
					m.config.ActiveKey = item.key
					m.config.Input.SetValue(item.val)

					// Load cache data when CACHE is selected
					if item.key == "CACHE" {
						m.loading = true
						return m, tea.Batch(cmd, m.cacheListCmd(), m.cacheStatsCmd()), true
					}
				}
			}
			return m, cmd, true
		} else {
			// Key Editor / Form interaction
			sel := m.config.Sidebar.SelectedItem()
			if sel != nil {
				item := sel.(configItem)

				// CACHE form: Enter triggers prune
				if item.key == "CACHE" {
					switch msg.String() {
					case "esc", "shift+tab", "left":
						m.configFocus = focusConfigSidebar
						return m, nil, true
					case "enter":
						m.loading = true
						return m, m.cachePruneCmd(), true
					}
					return m, nil, true
				}

				// UPDATE form: Enter triggers check
				if item.key == "UPDATE" {
					switch msg.String() {
					case "esc", "shift+tab", "left":
						m.configFocus = focusConfigSidebar
						return m, nil, true
					case "enter":
						m.config.UpdateChecking = true
						return m, m.checkUpdateCmd(), true
					}
					return m, nil, true
				}
			}

			// Standard key editor
			switch msg.String() {
			case "esc", "shift+tab": // Removed "left" to allow cursor navigation
				m.configFocus = focusConfigSidebar
				m.config.Input.Blur()
				return m, nil, true
			case "enter":
				// Auto-save logic was requested for toggles, implemented via direct toggle in sidebar
				val := m.config.Input.Value()
				key := m.config.ActiveKey
				m.loading = true
				m.configFocus = focusConfigSidebar
				m.config.Input.Blur()
				return m, m.saveConfigCmd(key, val), true
			}
			var cmd tea.Cmd
			m.config.Input, cmd = m.config.Input.Update(msg)
			return m, cmd, true
		}
	} else if m.activeTab == tabLogs {
		// Forward keys to viewport
		var cmd tea.Cmd
		m.logViewport, cmd = m.logViewport.Update(msg)
		return m, cmd, true
	}

	if m.activeTab == tabFleet {
		switch msg.String() {
		case "enter":
			if sel := m.list.SelectedItem(); sel != nil {
				if _, ok := sel.(spawnItem); ok {
					m.activeTab = tabHatchery
					return m, nil, true
				}
				if item, ok := sel.(vmItem); ok {
					if item.state == "running" {
						m.loading = true
						m.op = opStop
						return m, m.stopCmd(item.name), true
					}
					m.loading = true
					m.op = opStart
					return m, m.startCmd(item.name), true
				}
			}
		case "x":
			if sel := m.list.SelectedItem(); sel != nil {
				if item, ok := sel.(vmItem); ok {
					m.loading = true
					m.op = opStop
					return m, m.stopCmd(item.name), true
				}
			}
		case "delete":
			if sel := m.list.SelectedItem(); sel != nil {
				if item, ok := sel.(vmItem); ok {
					m.loading = true
					m.op = opDelete
					return m, m.deleteCmd(item.name), true
				}
			}
		}
		// Fallback to list navigation
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd, true
	}

	return m, nil, false
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

		// Tab Switching (5 tabs)
		availableWidth := m.width - 6
		tabWidth := availableWidth / 5
		if tabWidth > 0 {
			clickIndex := msg.X / tabWidth
			if clickIndex >= 0 && clickIndex <= 4 {
				m.activeTab = tab(clickIndex)
				return m, nil
			}
		}
	}

	// 2. Sidebar Logic (Fleet View)
	if m.activeTab == tabFleet {
		// sidebar width (18) + border (1) = 19. Let's use 20 as the barrier.
		if msg.X < 20 {
			row := msg.Y - 4 // Offset 4 (Header 2 + SubHeader 2)
			if row >= 0 {
				pageStart := m.list.Paginator.Page * m.list.Paginator.PerPage
				index := pageStart + row
				if index >= 0 && index < len(m.list.Items()) {
					m.list.Select(index)
					if sel := m.list.SelectedItem(); sel != nil {
						if v, ok := sel.(vmItem); ok {
							m.detailName = v.name
							return m, m.infoCmd(m.detailName)
						} else if _, ok := sel.(spawnItem); ok {
							m.activeTab = tabHatchery
							return m, nil
						}
					}
				} else if index >= len(m.list.Items()) && index <= len(m.list.Items())+3 {
					// Check if the previous item (the last one) is a spawnItem that wrapped
					lastIdx := len(m.list.Items()) - 1
					if lastIdx >= 0 {
						if _, ok := m.list.Items()[lastIdx].(spawnItem); ok {
							m.list.Select(lastIdx)
							m.activeTab = tabHatchery
							return m, nil
						}
					}
				}
			}
		} else {
			// Main Area Interactions (Buttons)
			// Y Calculation: Header(2) + SubHeader(2) + Title(1) + CardPaddingTop(1) + CardContent(6) + CardPaddingBottom(1) = 13
			// Buttons start after line 13.
			localX := msg.X - 20
			if msg.Y >= 14 && msg.Y <= 22 {
				if sel := m.list.SelectedItem(); sel != nil {
					if item, ok := sel.(vmItem); ok {
						if localX >= 0 && localX < 14 { // [ENTER] START/STOP
							if item.state == "running" {
								m.loading = true
								m.op = opStop
								return m, m.stopCmd(item.name)
							}
							m.loading = true
							m.op = opStart
							return m, m.startCmd(item.name)
						} else if localX >= 14 && localX < 26 { // [X] KILL
							m.loading = true
							m.op = opStop
							return m, m.stopCmd(item.name)
						} else if localX >= 26 && localX < 44 { // [DEL] DELETE
							m.loading = true
							m.op = opDelete
							return m, m.deleteCmd(item.name)
						}
					}
				}
			} else if msg.Y == 7 && localX >= 2 && m.detail.State == "running" && m.detail.SSHPort > 0 {
				// CLICK ON SSH LINE
				m.highlightSSH = true
				sshCmd := fmt.Sprintf("ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -p %d %s@%s", m.detail.SSHPort, m.detail.SSHUser, m.detail.IP)
				if m.detail.IP == "" || m.detail.IP == "—" {
					sshCmd = fmt.Sprintf("ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -p %d %s@127.0.0.1", m.detail.SSHPort, m.detail.SSHUser)
				}
				return m, tea.Batch(
					m.openTerminalCmd(sshCmd),
					tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg {
						return resetHighlightMsg{action: "ssh"}
					}),
				)
			} else if msg.Y == 8 && localX >= 2 && m.detail.VNCPort > 0 {
				// CLICK ON VNC LINE
				m.highlightVNC = true
				vncAddr := fmt.Sprintf("%s:%d", m.detail.IP, m.detail.VNCPort)
				return m, tea.Batch(
					m.openVNCCmd(vncAddr),
					tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg {
						return resetHighlightMsg{action: "vnc"}
					}),
				)
			}
		}
	}

	// 3. Hatchery Click logic
	if m.activeTab == tabHatchery {
		if msg.X < 28 {
			// Sidebar Click
			row := msg.Y - 4
			if row >= 0 && row <= 1 {
				m.hatchery.Sidebar.Select(row)
				m.hatcheryFocus = focusHatchSidebar
				return m, nil
			}
		} else {
			// Form Click
			row := msg.Y - 4
			if row == 0 { // Name Input
				m.hatcheryFocus = focusHatchForm
				m.hatchery.FocusIndex = 0
				m.updateHatcheryFocus()
				return m, nil
			} else if row == 2 { // Source Select
				m.hatcheryFocus = focusHatchForm
				m.hatchery.FocusIndex = 1
				m.updateHatcheryFocus()
				m.loading = true
				return m, m.fetchSources(m.hatchery.Sidebar.Index())
			} else if row == 4 { // GUI Toggle (Spawn) or Button (Template)
				m.hatcheryFocus = focusHatchForm
				if m.hatchery.Sidebar.Index() == 0 {
					m.hatchery.FocusIndex = 2
					m.hatchery.GUI = !m.hatchery.GUI
					m.updateHatcheryFocus()
					return m, nil
				} else {
					m.hatchery.FocusIndex = 2
					m.updateHatcheryFocus()
					return m.submitHatchery()
				}
			} else if row == 6 && m.hatchery.Sidebar.Index() == 0 { // Button (Spawn)
				m.hatcheryFocus = focusHatchForm
				m.hatchery.FocusIndex = 3
				m.updateHatcheryFocus()
				return m.submitHatchery()
			}
		}
	}

	// 4. Config Sidebar Logic
	if m.activeTab == tabConfig && msg.X < 28 {
		row := msg.Y - 5
		if row >= 0 && row < len(m.config.Sidebar.Items()) {
			m.config.Sidebar.Select(row)
			m.configFocus = focusConfigSidebar
			return m, nil
		}
	}

	return m, nil
}

func (m model) submitHatchery() (tea.Model, tea.Cmd) {
	isSpawn := m.hatchery.Sidebar.Index() == 0
	var name, source string

	if isSpawn {
		name = m.hatchery.SpawnInputs[0].Value()
		source = m.hatchery.SpawnSource
	} else {
		name = m.hatchery.TemplateInputs[0].Value()
		source = m.hatchery.TemplateSource
	}

	// Input Validation
	if name == "" {
		m.logs = append(m.logs, fmt.Sprintf("[%s] Hatchery: Name is required!", time.Now().Format("15:04:05")))
		m.logViewport.SetContent(strings.Join(m.logs, "\n"))
		m.logViewport.GotoBottom()
		return m, nil
	}
	if source == "" {
		m.logs = append(m.logs, fmt.Sprintf("[%s] Hatchery: Source is required!", time.Now().Format("15:04:05")))
		m.logViewport.SetContent(strings.Join(m.logs, "\n"))
		m.logViewport.GotoBottom()
		return m, nil
	}

	m.loading = true
	m.activeTab = tabFleet // Switch back to view progress

	if isSpawn {
		// SPAWN
		m.op = opSpawn

		// Resolve Source Path
		realSource := source
		if strings.Contains(source, "[IMAGE]") {
			// Extract Name:Version
			tag := strings.TrimPrefix(source, "[IMAGE] ")
			tag = strings.TrimSpace(tag)

			// Resolve image directory
			imgDir := m.cfg.ImageDir
			if imgDir == "" {
				home, _ := os.UserHomeDir()
				imgDir = filepath.Join(home, ".nido", "images")
			}

			// Parse tag
			parts := strings.Split(tag, ":")
			if len(parts) == 2 {
				name, ver := parts[0], parts[1]
				imgPath := filepath.Join(imgDir, fmt.Sprintf("%s-%s.qcow2", name, ver))

				// Check if exists
				if _, err := os.Stat(imgPath); os.IsNotExist(err) {
					// Need download!
					catalog, err := image.LoadCatalog(imgDir, image.DefaultCacheTTL)
					if err == nil {
						_, verEntry, err := catalog.FindImage(name, ver)
						if err == nil {
							// START ASYNC DOWNLOAD
							m.downloading = true
							m.downloadProgress = 0
							m.downloadChan = make(chan float64)
							m.logs = append(m.logs, fmt.Sprintf("[%s] Starting download for %s:%s...", time.Now().Format("15:04:05"), name, ver))
							m.logViewport.SetContent(strings.Join(m.logs, "\n"))
							m.logViewport.GotoBottom()

							// Return batch: start download routine AND start listener routine
							return m, tea.Batch(
								m.downloadImageCmd(verEntry.URL, imgPath, name, verEntry.SizeBytes, m.downloadChan),
								waitForDownloadProgress(m.downloadChan),
							)
						}
					}
					// If catalog/image not found, proceed and let spawn fail naturally or use fallback
				}

				realSource = imgPath
			} else {
				// Fallback for simple names if any (legacy flat files?)
				realSource = filepath.Join(imgDir, tag)
			}
		} else if strings.Contains(source, "[TEMPLATE]") {
			realSource = strings.TrimPrefix(source, "[TEMPLATE] ")
			realSource = strings.TrimSpace(realSource)
		}

		return m, m.spawnCmd(name, realSource, "", m.hatchery.GUI)
	} else {
		// CREATE TEMPLATE
		m.op = "create-template"
		vmName := strings.TrimPrefix(source, "[VM] ")
		vmName = strings.TrimSpace(vmName)
		return m, m.createTemplateCmd(vmName, name)
	}
}

func (m *model) updateHatcheryFocus() {
	isSpawn := m.hatchery.Sidebar.Index() == 0
	if m.hatchery.FocusIndex == 0 {
		if isSpawn {
			m.hatchery.SpawnInputs[0].Focus()
		} else {
			m.hatchery.TemplateInputs[0].Focus()
		}
	} else {
		if isSpawn {
			m.hatchery.SpawnInputs[0].Blur()
		} else {
			m.hatchery.TemplateInputs[0].Blur()
		}
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

	if m.activeTab == tabHatchery && m.hatchery.IsSelecting {
		// Modal overlay for source selection
		body = m.renderSourceModal(m.width, bodyHeight)
	} else if m.activeTab == tabLogs || m.activeTab == tabHelp {
		// Full Width Views (No Sidebar)
		var content string
		if m.activeTab == tabLogs {
			content = m.renderLogs(bodyHeight)
		} else {
			content = m.renderHelp()
		}
		body = mainContentStyle.
			Width(m.width - 4).
			Height(bodyHeight).
			Render(content)
	} else {
		// Split View (Sidebar + Main)
		var sidebarView string
		switch m.activeTab {
		case tabHatchery:
			sidebarView = m.hatchery.Sidebar.View()
		case tabConfig:
			sidebarView = m.config.Sidebar.View()
		default:
			sidebarView = m.list.View()
		}

		sidebarContent := sidebarView
		// No manual button adjustment needed anymore!

		sidebar := sidebarStyle.Height(bodyHeight).Render(sidebarContent)
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
		case tabHatchery:
			content = m.renderHatchery(bodyHeight)
		case tabConfig:
			content = m.renderConfig(bodyHeight)
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

func (m model) renderSourceModal(w, h int) string {
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

	// Ensure list items are set
	if len(m.hatchery.SourceList.Items()) == 0 {
		// If empty, fetch immediately (failsafe)
		// But usually fetchSources sets it.
		// If purely visual, show placeholder
	}

	modal := cardStyle.BorderForeground(lipgloss.Color("39")).Render(m.hatchery.SourceList.View())

	return lipgloss.Place(w, h,
		lipgloss.Center, lipgloss.Center,
		modal,
	)
}

func (m model) renderHatchery(h int) string {
	// Heading
	titleStr := "🦅 SPAWN NEW BIRD"
	descStr := "Choose an image or template to incubator a new instance."
	if m.hatchery.Sidebar.Index() == 1 {
		titleStr = "❄️  CREATE TEMPLATE"
		descStr = "Archive a running bird into a reusable template."
	}
	title := titleStyle.Render(titleStr)
	desc := dimStyle.Render(descStr)

	form := strings.Builder{}

	// Input: Name
	isSpawn := m.hatchery.Sidebar.Index() == 0
	labelColor := dimStyle
	if m.hatcheryFocus == focusHatchForm && m.hatchery.FocusIndex == 0 {
		labelColor = accentStyle
	}
	inputView := ""
	if isSpawn {
		inputView = m.hatchery.SpawnInputs[0].View()
	} else {
		inputView = m.hatchery.TemplateInputs[0].View()
	}
	form.WriteString(fmt.Sprintf("%-15s %s\n\n", labelColor.Render("Name:"), inputView))

	// Input: Source
	labelColor = dimStyle
	sourceVal := ""
	if isSpawn {
		sourceVal = m.hatchery.SpawnSource
	} else {
		sourceVal = m.hatchery.TemplateSource
	}

	if sourceVal == "" {
		sourceVal = "( Select Source... )"
	}
	if m.hatcheryFocus == focusHatchForm && m.hatchery.FocusIndex == 1 {
		labelColor = accentStyle
		sourceVal = accentStyle.Render(sourceVal)
	} else {
		sourceVal = dimStyle.Render(sourceVal)
	}
	form.WriteString(fmt.Sprintf("%-15s %s\n\n", labelColor.Render("Source:"), sourceVal))

	// Options
	maxIndex := 3
	if m.hatchery.Sidebar.Index() == 0 {
		guiCheck := "[ ]"
		if m.hatchery.GUI {
			guiCheck = "[x]"
		}
		labelColor = dimStyle
		if m.hatcheryFocus == focusHatchForm && m.hatchery.FocusIndex == 2 {
			labelColor = accentStyle
			guiCheck = accentStyle.Render(guiCheck)
		}
		form.WriteString(fmt.Sprintf("%-15s %s Enable GUI (VNC)\n\n", labelColor.Render("Options:"), guiCheck))
	} else {
		maxIndex = 2
	}

	// Submit Button
	btnText := "[ START INCUBATION ]"
	if m.hatchery.Sidebar.Index() == 1 {
		btnText = "[ FREEZE TEMPLATE ]"
	}
	btn := dimStyle.Render(btnText)
	if m.hatcheryFocus == focusHatchForm && m.hatchery.FocusIndex == maxIndex {
		btn = activeTabStyle.Render(btnText)
	}
	form.WriteString(btn)

	mainContent := lipgloss.JoinVertical(lipgloss.Left,
		title,
		desc,
		"",
		cardStyle.Render(form.String()),
	)

	return mainContent
}

func (m model) renderConfig(h int) string {
	// Removed Title/Description
	form := strings.Builder{}

	sel := m.config.Sidebar.SelectedItem()
	if sel != nil {
		item := sel.(configItem)

		// Special views for UPDATE and CACHE
		if item.key == "UPDATE" {
			if m.config.UpdateChecking {
				form.WriteString(fmt.Sprintf("%s Checking for updates...\n", m.spinner.View()))
			} else {
				currentVer := m.config.CurrentVersion
				if currentVer == "" {
					currentVer = "unknown"
				}
				form.WriteString(fmt.Sprintf("%-18s %s\n", dimStyle.Render("Current Version:"), accentStyle.Render(currentVer)))
				form.WriteString(fmt.Sprintf("%-18s %s\n\n", dimStyle.Render("GitHub:"), "https://github.com/Josepavese/nido"))

				if m.configFocus == focusConfigForm {
					form.WriteString(activeTabStyle.Render("[ CHECK FOR UPDATES ]"))
				} else {
					form.WriteString(dimStyle.Render("[ CHECK FOR UPDATES ]"))
				}
			}
			form.WriteString("\n\n" + dimStyle.Italic(true).Render("Press Enter to check for updates."))

		} else if item.key == "CACHE" {
			if m.loading {
				form.WriteString(fmt.Sprintf("%s Loading cache info...\n", m.spinner.View()))
			} else {
				stats := m.config.CacheStats
				form.WriteString(fmt.Sprintf("%-15s %d images (%s)\n\n", dimStyle.Render("Total:"), stats.TotalImages, stats.TotalSize))

				// List cached images
				for i, img := range m.config.CacheList {
					if i >= 5 {
						form.WriteString(dimStyle.Render(fmt.Sprintf("  ... and %d more\n", len(m.config.CacheList)-5)))
						break
					}
					form.WriteString(fmt.Sprintf("  %-15s %-20s %s\n", img.Name, img.Version, dimStyle.Render(img.Size)))
				}
				form.WriteString("\n")

				if m.configFocus == focusConfigForm {
					form.WriteString(activeTabStyle.Render("[ PRUNE UNUSED ]"))
				} else {
					form.WriteString(dimStyle.Render("[ PRUNE UNUSED ]"))
				}
			}
			form.WriteString("\n\n" + dimStyle.Italic(true).Render("Press Enter to prune unused cached images."))

		} else if item.key == "LINKED_CLONES" {
			form.WriteString(fmt.Sprintf("%-15s %s\n", dimStyle.Render("Key:"), accentStyle.Render(item.key)))
			// Boolean Toggle View
			state := "DISABLED"
			color := dimStyle
			if item.val == "true" {
				state = "ENABLED"
				color = successStyle
			} else {
				color = errorStyle
			}

			// Visual Toggle
			toggle := fmt.Sprintf("[ %s ]", color.Render(state))
			form.WriteString(fmt.Sprintf("%-15s %s\n\n", dimStyle.Render("Value:"), toggle))
			form.WriteString(dimStyle.Render("Press Enter/Tab on sidebar to toggle immediately."))
			form.WriteString("\n" + dimStyle.Italic(true).Render(item.desc))

		} else {
			form.WriteString(fmt.Sprintf("%-15s %s\n", dimStyle.Render("Key:"), accentStyle.Render(item.key)))
			// Status Bar / Footer
			// If Downloading, show progress bar
			if m.downloading {
				// Calculate available width for bar
				w := m.width - 20
				m.progress.Width = w
				bar := m.progress.ViewAs(m.downloadProgress)
				form.WriteString(fmt.Sprintf("\n %s Downloading... %s\n", m.spinner.View(), bar))
			} else if m.loading {
				form.WriteString(fmt.Sprintf("\n %s Working...\n", m.spinner.View()))
			} else if m.err != nil {
				form.WriteString(fmt.Sprintf("\n ❌ Error: %v\n", m.err))
			} else {
				// Standard Text Input
				form.WriteString(fmt.Sprintf("%-15s %s\n\n", dimStyle.Render("Value:"), m.config.Input.View()))

				if m.configFocus == focusConfigForm {
					form.WriteString(activeTabStyle.Render("[↵] SAVE SEQUENCE"))
				} else {
					form.WriteString(dimStyle.Render("[↵] EDIT SEQUENCE"))
				}
			}
			// Helper text at the bottom
			form.WriteString("\n" + dimStyle.Italic(true).Render(item.desc))
		}
	} else {
		form.WriteString(dimStyle.Render("Select a key from the sidebar to edit."))
	}

	if m.config.ErrorMsg != "" {
		form.WriteString("\n\n" + errorStyle.Render(m.config.ErrorMsg))
	}

	mainContent := lipgloss.JoinVertical(lipgloss.Left,
		cardStyle.Render(form.String()),
	)

	return mainContent
}

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
	case tabConfig:
		context = "GENETIC CONFIG"
		nav = "Modify Nido's core DNA. " + arrows
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

		return lipgloss.JoinVertical(lipgloss.Left, title, content)
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

	sshVal := fmt.Sprintf("ssh -p %d %s@%s", m.detail.SSHPort, m.detail.SSHUser, m.detail.IP)
	if m.highlightSSH {
		sshVal = accentStyle.Render(sshVal)
	}

	vncVal := vncLine
	if m.highlightVNC {
		vncVal = accentStyle.Render(vncVal)
	}

	infoCard := cardStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
		m.renderDetailLine("Status", statusColor.Render(m.detail.State)),
		m.renderDetailLine("SSH", sshVal),
		m.renderDetailLine("VNC", vncVal),
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

	// Sidebar(18) + Border(1) + Padding(1) = 20
	availWidth := m.width - 20
	// Buttons Width approx: 14 + 12 + 18 + margins = ~45
	if availWidth < 46 {
		// Switch to Vertical Layout
		// Remove right margins for vertical stack alignment? Button style has MarginRight(1).
		// Render items individually to control margins precisely if needed, or just stack.
		// If we stack, MarginRight doesn't hurt, but MarginTop might be needed for spacing.
		// Current buttonStyle has no MarginTop.
		// We can add a spacer or use a style with margin.
		vButtonStyle := buttonStyle.Copy().MarginRight(0).MarginBottom(1)
		vRedStyle := redButtonStyle.Copy().MarginRight(0).MarginBottom(1)

		btnStartStopV := vButtonStyle.Render("[↵] START")
		if m.detail.State == "running" {
			btnStartStopV = vButtonStyle.BorderForeground(colors.Error).Foreground(colors.Error).Render("[↵] STOP")
		} else {
			btnStartStopV = vButtonStyle.BorderForeground(colors.Success).Foreground(colors.Success).Render("[↵] START")
		}

		actions = lipgloss.JoinVertical(lipgloss.Left,
			btnStartStopV,
			vRedStyle.Render("[X] KILL"),
			vRedStyle.Render("[DEL] DELETE"),
		)
	}

	mainContent := lipgloss.JoinVertical(lipgloss.Left,
		title,
		infoCard,
		actions,
	)

	return mainContent
}

func (m model) renderDetailLine(label, value string) string {
	return lipgloss.JoinHorizontal(lipgloss.Top, labelStyle.Render(label), valueStyle.Render(value))
}

func (m model) renderDiskLine() string {
	path := m.detail.DiskPath
	if m.detail.DiskMissing {
		path = errorStyle.Render(fmt.Sprintf("MISSING (%s)", m.detail.DiskPath))
	}
	// Sidebar(18) + Padding(6) + Label(12) = 36 -> Safety 42
	avail := m.width - 42
	if avail < 10 {
		avail = 10
	}
	return m.truncatePath(path, avail)
}

func (m model) renderBackingLine() string {
	path := m.detail.BackingPath
	switch {
	case m.detail.BackingPath == "":
		return "—"
	case m.detail.BackingMissing:
		path = errorStyle.Render(fmt.Sprintf("MISSING (%s)", m.detail.BackingPath))
	}
	avail := m.width - 55
	if avail < 10 {
		avail = 10
	}
	return m.truncatePath(path, avail)
}

func (m model) truncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	// Head truncation: .../path/file.img
	return "..." + path[len(path)-(maxLen-3):]
}

func (m model) renderTabs() string {
	// Debug check: ensure active tab is read
	_ = m.activeTab
	var tabs []string
	labels := []string{"1 FLEET", "2 HATCHERY", "3 LOGS", "4 CONFIG", "5 HELP"}

	availableWidth := m.width - 6 // Extra safety for [X]
	tabWidth := availableWidth / 5

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

	gap := m.width - rowWidth - exitWidth - 1
	if gap < 0 {
		gap = 0
	}
	spacer := strings.Repeat(" ", gap)

	return lipgloss.JoinHorizontal(lipgloss.Top, row, spacer, exitBtn)
}

func (m model) renderLogs(height int) string {
	// Restore card style but match viewport height strictly
	// Viewport height is set to 9 in Update.
	// We wrap it in a card that allows it to show.
	return lipgloss.JoinVertical(lipgloss.Left,
		cardStyle.Width(m.width-4).Height(height).Render(m.logViewport.View()),
	)
}

func (m model) renderHelp() string {
	// Removed Title

	text := `NAVIGATION
  1-5 Keys   Directly switch between main views
  ←/→ Arrows cycle through tabs sequentially

VIEW CONTROLS
  [1] FLEET     ↑/↓ Select  ·  ↵ Start/Stop  ·  [X] Kill  ·  [DEL] Delete
  [2] HATCHERY  Tab Cycle Fields  ·  Space/↵ Select  ·  ←/→ Cycle Options
  [4] CONFIG    ↑/↓ Select Key  ·  ↵ Edit/Toggle  ·  Esc Cancel

GLOBAL
  Mouse supported on all meaningful elements.
  Press 'q' or Ctrl+C to exit Nido.`

	// Ensure transparency
	return lipgloss.JoinVertical(lipgloss.Left,
		cardStyle.Width(m.width-4).Render(dimStyle.Render(text)),
	)
}

func (m model) renderFooter() string {
	// Footer Alignment:
	// Sidebar Width: 18 (Content) + 1 (Border) = 19 Visual Chars.
	// We want the divider `│` to align with the border (Index 18).
	// Left Block: 1 Padding + 17 Chars = 18.
	// "🟢SYSTEMS NOMINAL" is 2 (Runes) + 15 (Chars) = 17 Chars. Perfect.

	// "🟢 NOMINAL"
	leftText := "🟢 NOMINAL"

	if m.downloading {
		// Calculate available width for bar
		// Footer full width minus label "Downloading..." (approx 15)
		w := m.width - 20
		if w < 10 {
			w = 10
		}
		m.progress.Width = w
		bar := m.progress.ViewAs(m.downloadProgress)
		status := fmt.Sprintf(" %s Downloading... %s", m.spinner.View(), bar)
		return footerStyle.Width(m.width).Render(status)
	}

	if m.loading {
		// Just render simplified loading state if loading, or keep alignment?
		// Loading spinner varies. Let's just keep the old full-width style for loading to avoid flicker.
		status := fmt.Sprintf("%s EXECUTING %s... ", m.spinner.View(), strings.ToUpper(string(m.op)))
		return footerStyle.Width(m.width).Render(status)
	}

	link := fmt.Sprintf("\x1b]8;;https://github.com/Josepavese\x1b\\%s\x1b]8;;\x1b\\", "github.com/Josepavese")
	rightText := fmt.Sprintf("🏠 There is no place like 127.0.0.1 | %s", link)

	// Left Block: Width 18. Padding Left 1.
	// We use Width(18) to ensure the separator stays aligned even with shorter text.
	// IMPORTANT: Width includes padding if set? No, usually Width is content width.
	// But let's try setting Width(18) and PaddingLeft(1).
	// If it sums up, we might overshoot.
	// Ideally: Padding(1) + Content(17).
	// Let's use Width(17) + PaddingLeft(1) logic manually or relying on lipgloss.
	// Safest: Width(18) with PaddingLeft(1). Lipgloss usually handles "Width is total" vs "Width is content" vaguely.
	// Let's test Width(18).
	leftBlock := lipgloss.NewStyle().
		Foreground(colors.TextDim).
		Width(18).           // Force total width
		Padding(0, 0, 0, 1). // Left 1
		Render(leftText)

	// Separator
	sep := lipgloss.NewStyle().
		Foreground(colors.SurfaceSubtle). // Match sidebar border color
		Render("│")

	// Right Block
	rightBlock := lipgloss.NewStyle().
		Foreground(colors.TextDim).
		Padding(0, 0, 0, 1). // Space after separator
		Render(rightText)

	// Join Horizontal
	return lipgloss.JoinHorizontal(lipgloss.Top, leftBlock, sep, rightBlock)
}

// Run starts the TUI with given provider/config.
func (m model) openTerminalCmd(sshCmd string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "linux":
			// Try common terminals
			terms := []string{"x-terminal-emulator", "gnome-terminal", "konsole", "xfce4-terminal", "xterm"}
			var termFound string
			for _, t := range terms {
				if _, err := exec.LookPath(t); err == nil {
					termFound = t
					break
				}
			}

			if termFound != "" {
				// Wrap command to keep terminal open on error
				// Use a more robust bash wrapper
				wrappedCmd := fmt.Sprintf("%s || (echo ''; echo '----------------------------------------'; echo '⚠️  SSH SESSION FAILED'; echo '----------------------------------------'; echo 'Press Enter to close this terminal...'; read)", sshCmd)

				switch termFound {
				case "gnome-terminal", "xfce4-terminal":
					cmd = exec.Command(termFound, "--", "bash", "-c", wrappedCmd)
				case "konsole":
					cmd = exec.Command(termFound, "-e", "bash", "-c", wrappedCmd)
				default:
					cmd = exec.Command(termFound, "-e", "bash", "-c", wrappedCmd)
				}
			} else {
				return logMsg{text: "No terminal emulator found"}
			}
		case "darwin":
			// macOS: Use osascript to open Terminal
			appleScript := fmt.Sprintf(`tell application "Terminal" to do script "%s"`, sshCmd)
			cmd = exec.Command("osascript", "-e", appleScript)
		case "windows":
			// Windows: Open cmd and run ssh
			cmd = exec.Command("cmd", "/c", "start", "cmd", "/k", sshCmd)
		default:
			return logMsg{text: fmt.Sprintf("Quick SSH not supported on %s", runtime.GOOS)}
		}

		if cmd != nil {
			err := cmd.Start()
			if err != nil {
				return logMsg{text: fmt.Sprintf("Failed to open terminal: %v", err)}
			}
		}
		return nil
	}
}

func (m model) openVNCCmd(vncAddr string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "linux":
			// Use xdg-open for vnc:// or try common viewers
			if _, err := exec.LookPath("xdg-open"); err == nil {
				cmd = exec.Command("xdg-open", "vnc://"+vncAddr)
			} else if _, err := exec.LookPath("gvncviewer"); err == nil {
				cmd = exec.Command("gvncviewer", vncAddr)
			} else if _, err := exec.LookPath("vncviewer"); err == nil {
				cmd = exec.Command("vncviewer", vncAddr)
			}
		case "darwin":
			cmd = exec.Command("open", "vnc://"+vncAddr)
		case "windows":
			// Windows: vnc:// might not be registered by default, but we can try
			cmd = exec.Command("cmd", "/c", "start", "vnc://"+vncAddr)
		}

		if cmd != nil {
			err := cmd.Start()
			if err != nil {
				return logMsg{text: fmt.Sprintf("Failed to open VNC: %v", err)}
			}
		} else {
			return logMsg{text: "No VNC viewer or xdg-open found"}
		}
		return nil
	}
}

func Run(ctx context.Context, prov provider.VMProvider, cfg *config.Config) error {
	p := tea.NewProgram(initialModel(prov, cfg), tea.WithContext(ctx), tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

// Messages for download progress
type downloadProgressMsg float64
type downloadFinishedMsg struct {
	err  error
	path string
	name string
}

func waitForDownloadProgress(sub chan float64) tea.Cmd {
	return func() tea.Msg {
		if sub == nil {
			return nil
		}
		p, ok := <-sub
		if !ok {
			return nil
		}
		return downloadProgressMsg(p)
	}
}

func (m model) downloadImageCmd(url, dest, name string, size int64, sub chan float64) tea.Cmd {
	return func() tea.Msg {
		dl := image.Downloader{
			Quiet: true,
			OnProgress: func(current, total int64) {
				if total > 0 {
					// Non-blocking send
					select {
					case sub <- float64(current) / float64(total):
					default:
					}
				}
			},
		}

		err := dl.Download(url, dest, size)
		close(sub) // Close channel when done
		return downloadFinishedMsg{err: err, path: dest, name: name}
	}
}

// Helper to check if file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
