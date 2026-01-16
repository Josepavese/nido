package config

import (
	"fmt"

	appconfig "github.com/Josepavese/nido/internal/config"
	"github.com/Josepavese/nido/internal/tui/app/ops"
	"github.com/Josepavese/nido/internal/tui/kit/layout"
	"github.com/Josepavese/nido/internal/tui/kit/theme"
	fv "github.com/Josepavese/nido/internal/tui/kit/view"
	"github.com/Josepavese/nido/internal/tui/kit/widget"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ... (top of file)

// ... (bottom of file)
func (c *Config) HandleMouse(x, y int, msg tea.MouseMsg) (fv.Viewlet, tea.Cmd, bool) {
	return c.MasterDetail.HandleMouse(x, y, msg)
}

// ConfigItem represents a configurable setting or menu item.
type ConfigItem struct {
	Key   string // Unique ID
	Val   string
	Label string
	Desc  string
	Type  string // "setting", "page"
}

func (i ConfigItem) String() string { return i.Key }
func (i ConfigItem) Title() string {
	if i.Label != "" {
		return i.Label
	}
	return i.Key
}
func (i ConfigItem) Description() string { return i.Desc }
func (i ConfigItem) FilterValue() string { return i.Key }
func (i ConfigItem) Icon() string {
	switch i.Key {
	case "SETTINGS":
		return theme.IconSystem
	case "APPEARANCE":
		return theme.IconLayout
	case "UPDATE":
		return theme.IconTemplate // Evolution
	case "CACHE":
		return theme.IconCache // Artifacts
	case "DOCTOR":
		return theme.IconDoctor // Health
	}
	return theme.IconSystem
}
func (i ConfigItem) IsAction() bool { return false }

// Config implements the System/Settings viewlet.
type Config struct {
	fv.BaseViewlet

	// State
	cfg *appconfig.Config

	// Components
	Sidebar *widget.SidebarList
	Spinner spinner.Model

	// Framework
	MasterDetail *widget.MasterDetail
	Pages        *widget.PageManager

	// Data
	items          []ConfigItem
	CurrentVersion string
	UpdateChecking bool
	CacheStats     CacheStats
	DoctorOutput   string // Store last doctor output

	// Detail Pages
	// Detail Pages
	PageGlobal     *ConfigPageGlobalForm
	PageAppearance *ConfigPageAppearance
	PageUpdate     *ConfigPageUpdate
	PageCache      *ConfigPageCache
	PageDoctor     *ConfigPageDoctor
}

type CacheStats struct {
	TotalImages int
	TotalSize   string
}

// NewConfig returns a new System viewlet.
func NewConfig(cfg *appconfig.Config) *Config {
	c := &Config{
		cfg: cfg,
	}

	// Initialize Spinner
	c.Spinner = spinner.New()
	c.Spinner.Spinner = spinner.MiniDot
	c.Spinner.Style = theme.Current().Styles.Accent

	// Initialize Pages Manager
	c.Pages = widget.NewPageManager()

	// 1. Sidebar (initialized in RefreshItems)
	c.RefreshItems()

	// 2. MasterDetail
	t := theme.Current()
	border := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(t.Palette.SurfaceSubtle)

	c.MasterDetail = widget.NewMasterDetail(
		widget.NewBoxedSidebar(
			widget.NewCard(theme.IconSystem, "System", "Engine"),
			c.Sidebar,
		),
		c.Pages,
		border,
	)
	c.MasterDetail.AutoSwitch = false

	return c
}

// Init initializes the viewlet.
func (c *Config) Init() tea.Cmd {
	return tea.Batch(
		c.MasterDetail.Init(),
		c.Spinner.Tick,
	)
}

// RefreshItems reloads the configuration items.
func (c *Config) RefreshItems() {
	// Define the menu structure
	// Define the menu structure
	c.items = []ConfigItem{
		// Main Configuration Page
		{Key: "SETTINGS", Label: "Core", Desc: "Core system settings", Type: "page"},
		{Key: "APPEARANCE", Label: "Rice", Desc: "UI and aesthetic tweaks", Type: "page"},

		// Special Pages
		{Key: "UPDATE", Label: "Evolution", Desc: "Update Nido versions", Type: "page"},
		{Key: "DOCTOR", Label: "Health", Desc: "System health check", Type: "page"},
		{Key: "CACHE", Label: "Artifacts", Desc: "Manage image storage", Type: "page"},
	}

	// Update Sidebar
	sidebarItems := make([]widget.SidebarItem, len(c.items))
	for i, it := range c.items {
		sidebarItems[i] = it
	}

	if c.Sidebar == nil {
		t := theme.Current()
		styles := widget.SidebarStyles{
			Normal:   t.Styles.SidebarItem,
			Selected: t.Styles.SidebarItemSelected,
			Dim:      lipgloss.NewStyle().Foreground(t.Palette.TextDim),
			Action:   t.Styles.SidebarItemSelected.Copy(),
		}
		c.Sidebar = widget.NewSidebarList(sidebarItems, theme.Width.Sidebar, styles, "SETTINGS")
		c.Pages.SwitchTo("SETTINGS")
	} else {
		c.Sidebar.SetItems(sidebarItems)
	}

	// registering/updating generic edit pages
	// Register Global Form
	if c.PageGlobal == nil {
		c.PageGlobal = NewConfigPageGlobalForm(c)
		c.Pages.AddPage("SETTINGS", c.PageGlobal)
	}
	if c.PageAppearance == nil {
		c.PageAppearance = NewConfigPageAppearance(c)
		c.Pages.AddPage("APPEARANCE", c.PageAppearance)
	}

	// Register Special Pages (Singletons)
	if c.PageUpdate == nil {
		c.PageUpdate = NewConfigPageUpdate(c)
		c.Pages.AddPage("UPDATE", c.PageUpdate)
	}
	if c.PageDoctor == nil {
		c.PageDoctor = NewConfigPageDoctor(c)
		c.Pages.AddPage("DOCTOR", c.PageDoctor)
	}
	if c.PageCache == nil {
		c.PageCache = NewConfigPageCache(c)
		c.Pages.AddPage("CACHE", c.PageCache)
	}

	// Restore selection logic?
	// Usually auto-handled by sidebar index, but Pages need manual switch from Sidebar Selection event.
}

func (c *Config) SetCacheStats(stats CacheStats) {
	c.CacheStats = stats
}

func (c *Config) SetUpdateStatus(current, latest string, checking bool) {
	c.CurrentVersion = current
	c.UpdateChecking = checking
}

// Update handles messages.
func (c *Config) Update(msg tea.Msg) (fv.Viewlet, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		c.Spinner, cmd = c.Spinner.Update(msg)
		return c, cmd

	// Sidebar Selection -> Switch Page
	case fv.SelectionMsg:
		if item, ok := msg.Item.(ConfigItem); ok {
			c.Pages.SwitchTo(item.Key)

			// Trigger loads for special pages
			if item.Key == "CACHE" {
				// Request Cache Stats? We might need an ops message for simple stats.
				// ops.ListCache(prov) gives list, here we want explicit stats.
				// For now, let's assume global cache list is updated elsewhere or we request it.
				cmds = append(cmds, func() tea.Msg { return RequestCacheMsg{} })
			}
			if item.Key == "UPDATE" {
				cmds = append(cmds, func() tea.Msg { return ops.RequestUpdateMsg{} })
			}
		}

	case tea.KeyMsg:
		// Centralized Focus Management (Kit Philosophy)

		// 1. Sidebar to Detail (Enter/Right)
		if c.Sidebar.Focused() {
			if msg.String() == "enter" || msg.String() == "right" {
				// Only switch to detail if it has focusable elements
				// We check this via HasActiveInput or similar?
				// The strict standard is: Does the active page have potential input?
				// Unfortunately Page interface is weak. We rely on MasterDetail to try focusing.
				// However, we should check if it's "safe".
				// For now, consistent behavior: allow switch, if nothing there, it's a dead end but escapes via Esc.
				cmds = append(cmds, c.MasterDetail.SetFocus(widget.FocusDetail))
				return c, tea.Batch(cmds...)
			}
		}

		// 2. Detail to Sidebar (Esc)
		// This captures Esc from ANY detail page (Global, Appearance, etc.)
		if !c.Sidebar.Focused() {
			if msg.String() == "esc" {
				cmds = append(cmds, c.MasterDetail.SetFocus(widget.FocusSidebar))
				return c, tea.Batch(cmds...)
			}
		}

		// 3. Fallback: Detail to Sidebar (Left) if at start of form?
		// (Optional, user didn't ask, sticking to strict Esc request)

	case ops.DoctorResultMsg:
		c.DoctorOutput = msg.Output
		if msg.Err != nil {
			c.DoctorOutput += fmt.Sprintf("\nError: %v", msg.Err)
		}
		if c.PageDoctor != nil {
			c.PageDoctor.Output = c.DoctorOutput
		}

	case ops.RequestUpdateMsg: // Loopback from internal page
		c.UpdateChecking = true
		cmds = append(cmds, ops.CheckUpdate())

	case ops.UpdateCheckMsg:
		c.UpdateChecking = false
		c.CurrentVersion = msg.Current

	case FocusSidebarMsg:
		cmds = append(cmds, c.MasterDetail.SetFocus(widget.FocusSidebar))

	case ops.CacheListMsg:
		// We use this to compute stats
		count := len(msg.Items)
		// Size not summed in msg? msg.Items has Size string.
		// Simulating stat update:
		c.CacheStats = CacheStats{TotalImages: count, TotalSize: "Calculated"}
		// (Real size sum needs parsing)
	}

	newMD, mdCmd := c.MasterDetail.Update(msg)
	c.MasterDetail = newMD.(*widget.MasterDetail)
	cmds = append(cmds, mdCmd)

	return c, tea.Batch(cmds...)
}

func (c *Config) View() string {
	base := c.MasterDetail.View()

	// If a modal is active in any page, overlay it on EVERYTHING
	if c.IsModalActive() {
		if p := c.Pages.ActivePage(); p != nil {
			// This is slightly tricky: we need a way to get the modal from the page
			// For now, we know only these two have it.
			var modal *widget.Modal
			if pf, ok := p.(*ConfigPageGlobalForm); ok {
				modal = pf.Modal
			} else if pa, ok := p.(*ConfigPageAppearance); ok {
				modal = pa.Modal
			}

			if modal != nil {
				return modal.View(c.Width(), c.Height())
			}
		}
	}

	return base
}

func (c *Config) Resize(r layout.Rect) {
	c.BaseViewlet.Resize(r)
	c.MasterDetail.Resize(r)
}

func (c *Config) Focus() tea.Cmd {
	return c.MasterDetail.Focus()
}

func (c *Config) Shortcuts() []fv.Shortcut {
	return c.MasterDetail.Shortcuts()
}

func (c *Config) IsModalActive() bool {
	if c.MasterDetail == nil {
		return false
	}
	return c.MasterDetail.IsModalActive()
}

func (c *Config) HasActiveInput() bool {
	res := false
	if c.MasterDetail != nil {
		res = c.MasterDetail.HasActiveInput()
	}
	return res
}

// Internal Messages (kept for compatibility)
type RequestCacheMsg struct{}
type RequestPruneMsg struct{}
type FocusSidebarMsg struct{}
