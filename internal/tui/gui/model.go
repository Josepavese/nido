package gui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Josepavese/nido/internal/config"
	"github.com/Josepavese/nido/internal/image"
	"github.com/Josepavese/nido/internal/provider"
	"github.com/Josepavese/nido/internal/tui/layout"
	"github.com/Josepavese/nido/internal/tui/services"
	"github.com/Josepavese/nido/internal/tui/theme"
	"github.com/Josepavese/nido/internal/tui/viewlet"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// isInputFocused checks if any text input is currently focused.
func (m model) isInputFocused() bool {
	if m.activeTab == tabHatchery && m.hatcheryFocus == focusHatchForm && m.hatcheryView.IsTyping() {
		return true
	}
	if m.activeTab == tabConfig && m.configView.Mode == viewlet.ConfigModeForm {
		return true
	}
	return false
}

type model struct {
	prov    provider.VMProvider
	cfg     *config.Config
	strings UIStrings
	keymap  Keymap

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
	// Config Viewlet reused

	logs     []string
	logsView *viewlet.Logs

	// New fields from the instruction
	quitting         bool
	err              error
	downloading      bool
	downloadProgress float64
	downloadChan     chan float64

	// Quick Actions UI state
	highlightSSH bool
	highlightVNC bool

	// Viewlet instances (for incremental migration)
	configView   *viewlet.Config
	helpView     *viewlet.Help
	fleetView    *viewlet.Fleet
	hatcheryView *viewlet.Hatchery // Ready for future wiring
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

	return hatcheryState{
		Sidebar: sb,
	}
}

func initialModel(prov provider.VMProvider, cfg *config.Config) model {
	// Apply environment overrides for TUI-related settings
	cfg.ApplyEnvOverrides()

	// Sync theme/layout tokens with config overrides
	theme.ApplyOverrides(theme.Overrides{
		SidebarWidth:     cfg.TUI.SidebarWidth,
		SidebarWideWidth: cfg.TUI.SidebarWideWidth,
		InsetContent:     cfg.TUI.InsetContent,
		TabMinWidth:      cfg.TUI.TabMinWidth,
		ExitZoneWidth:    cfg.TUI.ExitZoneWidth,
		GapScale:         cfg.TUI.GapScale,
	})

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

	prog := progress.New(progress.WithScaledGradient(colors.AccentStrong.Dark, colors.Accent.Dark))
	prog.ShowPercentage = true

	// Initialize Logs Viewlet
	lView := viewlet.NewLogs()
	lView.SetContent(strings.Join([]string{fmt.Sprintf("[%s] Nido GUI ready. Systems nominal.", time.Now().Format("15:04:05"))}, "\n"))

	// Strings (allow future overrides)
	uiStrings := DefaultStrings()
	if len(cfg.TUI.TabLabels) >= 5 {
		uiStrings.TabLabels = cfg.TUI.TabLabels
	}
	if cfg.TUI.FooterLink != "" {
		uiStrings.FooterLink = cfg.TUI.FooterLink
	}
	km := DefaultKeymap()

	return model{
		prov:      prov,
		cfg:       cfg,
		activeTab: tabFleet,
		list:      l,
		page:      pg,
		spinner:   spin,
		progress:  prog,
		loading:   bool(false),
		logs:      []string{fmt.Sprintf("[%s] Nido GUI ready. Systems nominal.", time.Now().Format("15:04:05"))},
		logsView:  lView,
		hatchery:  newHatcheryState(cfg),
		// Config Viewlet (replaces state)
		configView:   viewlet.NewConfig(cfg),
		helpView:     viewlet.NewHelp(),
		fleetView:    viewlet.NewFleet(),
		hatcheryView: viewlet.NewHatchery(),
		strings:      uiStrings,
		keymap:       km,
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
		services.RefreshFleet(m.prov),
	)
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
			m.logsView.SetContent(strings.Join(m.logs, "\n"))
			return m, nil
		}
		m.logs = append(m.logs, fmt.Sprintf("[%s] Download complete for %s.", time.Now().Format("15:04:05"), msg.name))
		m.logsView.SetContent(strings.Join(m.logs, "\n"))

		// Resume Spawn
		name, _, gui := m.hatcheryView.GetValues()
		m.activeTab = tabFleet
		m.op = opSpawn
		m.loading = true
		return m, services.SpawnVM(m.prov, name, msg.path, "", gui)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Calculate available height for content using layout overhead
		bodyHeight := m.height - layout.SpacingOverhead
		if bodyHeight < 1 {
			bodyHeight = 1
		}

		// Apply dynamic height to all components
		m.list.SetSize(18, bodyHeight)             // Fleet
		m.hatchery.Sidebar.SetSize(18, bodyHeight) // Hatchery
		m.configView.Resize(m.width, bodyHeight)   // Config Viewlet

		// Logs Viewport
		m.logsView.Resize(m.width-8, bodyHeight)

	case tickMsg:
		m.spinner, _ = m.spinner.Update(msg)
		cmds = append(cmds, tea.Tick(time.Millisecond*80, func(time.Time) tea.Msg { return tickMsg{} }))

		// Update inputs for blink
		// Update inputs for blink
		// Update inputs for blink (delegate to viewlet)
		if m.activeTab == tabHatchery {
			var cmd tea.Cmd
			_, cmd = m.hatcheryView.Update(msg)
			cmds = append(cmds, cmd)
		}
	// --- Services Messages ---
	case services.VMListMsg:
		newM, newCmds := m.handleVMListMsg(msg)
		return newM, tea.Batch(newCmds...)
	case services.VMDetailMsg:
		return m.handleDetailMsg(msg), nil
	case services.SourcesLoadedMsg:
		return m.handleSourcesLoadedMsg(msg), nil
	case services.OpResultMsg:
		return m.handleOpResultMsg(msg)
	case services.LogMsg:
		return m.handleLogMsg(msg), nil
	case services.ConfigSavedMsg:
		return m.handleConfigSavedMsg(msg), nil
	case services.UpdateCheckMsg:
		return m.handleUpdateCheckMsg(msg), nil
	case services.CacheListMsg:
		return m.handleCacheListMsg(msg), nil
	case services.CacheStatsMsg:
		return m.handleCacheStatsMsg(msg), nil
	case services.CachePruneMsg:
		newM, newCmds := m.handleCachePruneMsg(msg)
		return newM, tea.Batch(newCmds...)

	case resetHighlightMsg:
		if msg.action == "ssh" {
			m.highlightSSH = false
		} else {
			m.highlightVNC = false
		}
	case logMsg:
		m.logs = append(m.logs, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg.text))
		// Update Viewport Content
		m.logsView.SetContent(strings.Join(m.logs, "\n"))
	case opResultMsg:
		m.loading = false
		if msg.err != nil {
			m.logs = append(m.logs, fmt.Sprintf("[%s] Operation %s failed: %v", time.Now().Format("15:04:05"), msg.op, msg.err))
		} else {
			m.logs = append(m.logs, fmt.Sprintf("[%s] Operation %s complete.", time.Now().Format("15:04:05"), msg.op))
		}
		m.logsView.SetContent(strings.Join(m.logs, "\n"))
		m.op = opNone
		cmds = append(cmds, services.RefreshFleet(m.prov))
	case configSavedMsg:
		m.loading = false
		m.logs = append(m.logs, fmt.Sprintf("[%s] Config %s updated to %s", time.Now().Format("15:04:05"), msg.key, msg.value))
		m.logsView.SetContent(strings.Join(m.logs, "\n"))
		// Refresh viewlet
		m.configView.RefreshItems()
	case updateCheckMsg:
		if msg.err != nil {
			m.logs = append(m.logs, fmt.Sprintf("[%s] Update check failed: %v", time.Now().Format("15:04:05"), msg.err))
			m.configView.SetUpdateStatus("", "", false)
		} else {
			m.configView.SetUpdateStatus(msg.current, msg.latest, false)
			m.logs = append(m.logs, fmt.Sprintf("[%s] Version check complete: %s", time.Now().Format("15:04:05"), msg.current))
		}
		m.logsView.SetContent(strings.Join(m.logs, "\n"))
	case cacheListMsg:
		m.loading = false
		if msg.err != nil {
			m.logs = append(m.logs, fmt.Sprintf("[%s] Cache list failed: %v", time.Now().Format("15:04:05"), msg.err))
		} else {
			m.configView.SetCacheList(msg.items)
		}
		m.logsView.SetContent(strings.Join(m.logs, "\n"))
	case cacheStatsMsg:
		m.loading = false
		if msg.err != nil {
			m.logs = append(m.logs, fmt.Sprintf("[%s] Cache info failed: %v", time.Now().Format("15:04:05"), msg.err))
		} else {
			m.configView.SetCacheStats(msg.stats)
		}
		m.logsView.SetContent(strings.Join(m.logs, "\n"))
	case cachePruneMsg:
		m.loading = false
		if msg.err != nil {
			m.logs = append(m.logs, fmt.Sprintf("[%s] Cache prune failed: %v", time.Now().Format("15:04:05"), msg.err))
		} else {
			m.logs = append(m.logs, fmt.Sprintf("[%s] Cache pruned successfully", time.Now().Format("15:04:05")))
			cmds = append(cmds, services.ListCache(m.prov), services.FetchCacheStats(m.prov)) // Refresh cache view
		}
		m.logsView.SetContent(strings.Join(m.logs, "\n"))
	case viewlet.RequestCacheMsg:
		m.loading = true
		cmds = append(cmds, services.ListCache(m.prov), services.FetchCacheStats(m.prov))

	case viewlet.RequestPruneMsg:
		m.loading = true
		cmds = append(cmds, services.PruneCache(m.prov))

	case viewlet.RequestUpdateMsg:
		cmds = append(cmds, services.CheckUpdate())

	case viewlet.SaveConfigMsg:
		m.loading = true
		cmds = append(cmds, services.SaveConfig(m.cfg, msg.Key, msg.Value))

	case tea.KeyMsg:
		// Modal Logic Removed (Handled by Viewlet)

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
			// Allow header interactions (tabs/exit) to take precedence
			if msg.Y == 0 {
				return m.handleMouse(msg)
			}
			var cmd tea.Cmd
			var v viewlet.Viewlet
			v, cmd = m.logsView.Update(msg)
			m.logsView = v.(*viewlet.Logs)
			return m, cmd
		}
		newModel, cmd := m.handleMouse(msg)
		return newModel, cmd
	}

	// Handle Hatchery Input Updates
	if m.activeTab == tabHatchery && m.hatcheryFocus == focusHatchForm {
		var cmd tea.Cmd
		// Delegate entirely to viewlet
		updatedView, cmd := m.hatcheryView.Update(msg)
		if hView, ok := updatedView.(*viewlet.Hatchery); ok {
			m.hatcheryView = hView
		}
		cmds = append(cmds, cmd)
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
					cmds = append(cmds, services.FetchVMInfo(m.prov, m.detailName))
				}
			}
		}
	}
	return m, tea.Batch(cmds...)
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	// 1. Global Shortcuts (quit, tab switching, refresh)
	if newM, cmd, handled := m.handleGlobalKeys(msg); handled {
		return newM, cmd, true
	}

	// 2. Navigation (Left/Right arrows for tab cycling)
	if newM, cmd, handled := m.handleNavigationKeys(msg); handled {
		return newM, cmd, true
	}

	// 3. Tab-Specific Logic
	switch m.activeTab {
	case tabFleet:
		return m.handleFleetKeys(msg)
	case tabHatchery:
		return m.handleHatcheryKeys(msg)
	case tabConfig:
		return m.handleConfigKeys(msg)
	case tabLogs:
		return m.handleLogsKeys(msg)
	}

	return m, nil, false
}

func (m model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Action != tea.MouseActionPress {
		return m, nil
	}

	// 1. Header Logic (tabs, exit button)
	if newM, cmd, handled := m.handleHeaderMouse(msg); handled {
		return newM, cmd
	}

	// 2. Tab-Specific Logic
	switch m.activeTab {
	case tabFleet:
		return m.handleFleetMouse(msg)
	case tabHatchery:
		return m.handleHatcheryMouse(msg)
	case tabConfig:
		// Mouse support for config viewlet not fully implemented
		return m, nil
	}

	return m, nil
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing Mission Control..."
	}

	// Use layout package for responsive dimensions
	dim := layout.Calculate(m.width, m.height)

	// Check if terminal is viable
	if !dim.IsViable() {
		return layout.TooSmallMessage(m.width, m.height)
	}

	// Header, Sub-Header, Footer State
	shellCfg := ShellConfig{
		Width:     m.width,
		Height:    m.height,
		ActiveTab: m.activeTab,
		Strings:   m.strings,
	}

	footerState := FooterState{
		Width:            m.width,
		Loading:          m.loading,
		Downloading:      m.downloading,
		DownloadProgress: m.downloadProgress,
		Operation:        string(m.op),
		SpinnerView:      m.spinner.View(),
		FooterLink:       m.strings.FooterLink,
	}
	if m.downloading {
		m.progress.Width = m.width - 20
		footerState.ProgressView = m.progress.ViewAs(m.downloadProgress)
	}

	// Render shell and derive body height (shellHeight includes 1-line gaps)
	header, subHeader, footer, shellHeight := RenderShell(shellCfg, footerState)

	bodyHeight := m.height - shellHeight
	if bodyHeight < 0 {
		bodyHeight = 0
	}

	var body string

	if m.activeTab == tabLogs || m.activeTab == tabHelp || m.activeTab == tabConfig {
		// Full Width Views
		var content string
		if m.activeTab == tabLogs {
			// Logs
			m.logsView.Resize(m.width-theme.Inset.TotalContent, bodyHeight)
			content = m.logsView.View()
		} else if m.activeTab == tabConfig {
			// Config
			// NOTE: Config viewlet has its own internal sidebar.
			// Ideally we should expose sidebar control or pass full bodyHeight.
			m.configView.Resize(m.width-theme.Inset.TotalContent, bodyHeight)
			content = m.configView.View()
		} else {
			// Help
			m.helpView.Resize(m.width-theme.Inset.TotalContent, bodyHeight)
			content = m.helpView.View()
		}
		body = mainContentStyle.
			Width(m.width - theme.Inset.TotalContent).
			Height(bodyHeight).
			Render(content)
	} else {
		// Split View (Sidebar + Main)
		sidebarW := dim.Breakpoint.SidebarWidth()
		if sidebarW == 0 {
			sidebarW = theme.Width.Sidebar
		}

		var sidebarView string
		switch m.activeTab {
		case tabHatchery:
			m.hatchery.Sidebar.SetSize(sidebarW, bodyHeight) // Resize to fit exact body height
			sidebarView = m.hatchery.Sidebar.View()

		default:
			// Fleet Sidebar
			m.list.SetSize(sidebarW, bodyHeight) // Resize to fit exact body height
			sidebarView = m.list.View()
		}

		sidebar := sidebarStyle.Width(sidebarW).Height(bodyHeight).Render(sidebarView)

		// Main Content
		mainWidth := m.width - lipgloss.Width(sidebar)
		if mainWidth < 0 {
			mainWidth = 0
		}

		var content string
		switch m.activeTab {
		case tabFleet:
			m.fleetView.Resize(mainWidth-theme.Inset.TotalContent, bodyHeight)
			content = m.fleetView.View()
		case tabHatchery:
			m.hatcheryView.Resize(mainWidth-theme.Inset.TotalContent, bodyHeight)
			content = m.hatcheryView.View()
		case tabConfig:
			m.configView.Resize(mainWidth-theme.Inset.TotalContent, bodyHeight)
			content = m.configView.View()
		}

		mainArea := mainContentStyle.
			Width(mainWidth - theme.Inset.TotalContent).
			Height(bodyHeight).
			Render(content)

		body = lipgloss.JoinHorizontal(lipgloss.Top, sidebar, mainArea)
	}

	// Assemble complete shell with consistent gaps
	return layout.VStack(theme.Gap(1), header, subHeader, body, footer)
}

// Run starts the TUI with given provider/config.

func Run(ctx context.Context, prov provider.VMProvider, cfg *config.Config) error {
	p := tea.NewProgram(initialModel(prov, cfg), tea.WithContext(ctx), tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
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
