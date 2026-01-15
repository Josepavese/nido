package config

import (
	"fmt"

	appconfig "github.com/Josepavese/nido/internal/config"
	"github.com/Josepavese/nido/internal/tui/kit/layout"
	"github.com/Josepavese/nido/internal/tui/kit/theme"
	fv "github.com/Josepavese/nido/internal/tui/kit/view"
	"github.com/Josepavese/nido/internal/tui/kit/widget"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConfigMode defines the active area within the config tab.
type ConfigMode int

const (
	ConfigModeSidebar ConfigMode = iota
	ConfigModeForm
)

// Styles (initialized from theme)
var (
	dimStyle                 = theme.Current().Styles.TextDim
	accentStyle              = theme.Current().Styles.Accent
	successStyle             = theme.Current().Styles.Success
	errorStyle               = theme.Current().Styles.Error
	activeTabStyle           = theme.Current().Styles.AccentStrong
	sidebarItemStyle         = theme.Current().Styles.SidebarItem
	sidebarItemSelectedStyle = theme.Current().Styles.SidebarItemSelected
	cardStyle                = lipgloss.NewStyle().Padding(1) // Simple padding
)

// ConfigItem represents a configurable setting.
type ConfigItem struct {
	Key      string
	Val      string
	Label    string
	Desc     string
	Type     string // "text", "bool", "action", "info"
	ValLabel string
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
func (i ConfigItem) Icon() string        { return "" } // No icon for config settings
func (i ConfigItem) IsAction() bool      { return i.Type == "action" }

// ConfigForm adapts the config form content to the Viewlet interface.
type ConfigForm struct {
	fv.BaseViewlet
	Parent *Config
}

// Config implements the Configuration viewlet.
type Config struct {
	fv.BaseViewlet

	// State
	cfg  *appconfig.Config
	Mode ConfigMode

	// Components
	Sidebar *widget.SidebarList
	Input   textinput.Model
	Spinner spinner.Model

	// Framework
	MasterDetail *widget.MasterDetail
	Pages        *widget.PageManager

	// Data
	items          []ConfigItem
	ActiveKey      string
	CurrentVersion string
	LatestVersion  string
	UpdateChecking bool
	CacheList      []CacheItem
	CacheStats     CacheStats
	Loading        bool
}

type CacheItem struct {
	Name    string
	Version string
	Size    string
}

type CacheStats struct {
	TotalImages int
	TotalSize   string
}

// NewConfig returns a new Config viewlet.
func NewConfig(cfg *appconfig.Config) *Config {
	c := &Config{
		cfg:  cfg,
		Mode: ConfigModeSidebar,
	}

	// Sidebar (initialized strictly)
	c.RefreshItems()

	// Input
	ti := textinput.New()
	ti.CharLimit = 100
	c.Input = ti

	// Spinner
	c.Spinner = spinner.New()
	c.Spinner.Style = accentStyle

	// Framework Components
	c.Pages = widget.NewPageManager()

	// Register Pages
	c.Pages.AddPage("UPDATE", &ConfigPageUpdate{Parent: c})
	c.Pages.AddPage("CACHE", &ConfigPageCache{Parent: c})

	// Generic Pages for other settings
	for _, it := range c.items {
		if it.Key != "UPDATE" && it.Key != "CACHE" {
			c.Pages.AddPage(it.Key, &ConfigPageGeneric{Parent: c, Key: it.Key})
		}
	}

	t := theme.Current()
	border := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(t.Palette.SurfaceSubtle)

	c.MasterDetail = widget.NewMasterDetail(c.Sidebar, c.Pages, border)

	return c
}

// Init initializes the viewlet.
func (c *Config) Init() tea.Cmd {
	return c.Spinner.Tick
}

// RefreshItems reloads the configuration items from the config struct.
func (c *Config) RefreshItems() {
	// Convert to SidebarItems
	c.items = []ConfigItem{
		ConfigItem{Key: "UPDATE", Val: "", Desc: "Check for updates and upgrade Nido."},
		ConfigItem{Key: "CACHE", Val: "", Desc: "Manage cached cloud images."},
		ConfigItem{Key: "BACKUP_DIR", Val: c.cfg.BackupDir, Desc: "Path to store template backups."},
		ConfigItem{Key: "IMAGE_DIR", Val: c.cfg.ImageDir, Desc: "Directory for cached cloud images."},
		ConfigItem{Key: "LINKED_CLONES", Val: fmt.Sprintf("%v", c.cfg.LinkedClones), Desc: "Use Copy-on-Write for disk efficiency."},
		ConfigItem{Key: "SSH_USER", Val: c.cfg.SSHUser, Desc: "Default user for SSH connections."},
		ConfigItem{Key: "TEMPLATE_DEFAULT", Val: c.cfg.TemplateDefault, Desc: "Default source template for new VMs."},
	}

	sidebarItems := make([]widget.SidebarItem, len(c.items))
	for i, it := range c.items {
		sidebarItems[i] = it
	}

	if c.Sidebar == nil {
		// Theme Injection for SidebarList
		t := theme.Current()
		styles := widget.SidebarStyles{
			Normal:   t.Styles.SidebarItem,
			Selected: t.Styles.SidebarItemSelected,
			Dim:      lipgloss.NewStyle().Foreground(t.Palette.TextDim),
			Action:   t.Styles.SidebarItemSelected.Copy().Background(t.Palette.Accent).Foreground(t.Palette.Background),
		}
		c.Sidebar = widget.NewSidebarList(sidebarItems, theme.Width.Sidebar, styles, "")
	} else {
		c.Sidebar.SetItems(sidebarItems)
	}
}

// SetCacheList updates the cache list.
func (c *Config) SetCacheList(items []CacheItem) {
	c.CacheList = items
	c.Loading = false
}

// SetCacheStats updates the cache statistics.
func (c *Config) SetCacheStats(stats CacheStats) {
	c.CacheStats = stats
	c.Loading = false
}

// SetUpdateStatus updates the version info.
func (c *Config) SetUpdateStatus(current, latest string, checking bool) {
	c.CurrentVersion = current
	c.LatestVersion = latest
	c.UpdateChecking = checking
}

// Update handles messages.
func (c *Config) Update(msg tea.Msg) (fv.Viewlet, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case spinner.TickMsg:
		c.Spinner, cmd = c.Spinner.Update(msg)
		return c, cmd

	case fv.SelectionMsg:
		if sel := msg.Item; sel != nil {
			if item, ok := sel.(ConfigItem); ok {
				c.ActiveKey = item.Key
				// Handle special items immediately (Mouse equivalent of Enter)
				if item.Key == "CACHE" {
					cmd = func() tea.Msg { return RequestCacheMsg{} }
				}
				if item.Key == "UPDATE" {
					c.UpdateChecking = true
					cmd = func() tea.Msg { return RequestUpdateMsg{} }
				}
			}
		}

	case tea.KeyMsg:
		if c.Mode == ConfigModeSidebar {
			switch msg.String() {
			case "enter", "right":
				// SidebarList helper
				sel := c.Sidebar.SelectedItem()
				if sel != nil {
					item := sel.(ConfigItem)
					c.ActiveKey = item.Key
					c.Mode = ConfigModeForm

					// Load value into Input
					if item.Key != "UPDATE" && item.Key != "CACHE" {
						c.Input.SetValue(item.Val)
						c.Input.Focus()
						return c, textinput.Blink
					}

					// For special items, return a command request if needed
					if item.Key == "CACHE" {
						// Request Cache Load
						return c, func() tea.Msg { return RequestCacheMsg{} }
					}
				}
			}
			var cmd tea.Cmd
			var v fv.Viewlet
			v, cmd = c.Sidebar.Update(msg)
			// SidebarList is a pointer, simple update is fine but Viewlet returns interface.
			// SidebarList.Update updates its internal model pointer in place mostly.
			_ = v
			return c, cmd
		} else {
			// FocusForm
			switch msg.String() {
			case "esc", "left":
				c.Mode = ConfigModeSidebar
				c.Input.Blur()
				return c, nil
			case "enter":
				// Handle Action
				sel := c.Sidebar.SelectedItem()
				if sel != nil {
					item := sel.(ConfigItem)
					if item.Key == "CACHE" {
						return c, func() tea.Msg { return RequestPruneMsg{} }
					} else if item.Key == "UPDATE" {
						c.UpdateChecking = true
						return c, func() tea.Msg { return RequestUpdateMsg{} }
					} else if item.Key == "LINKED_CLONES" {
						// Toggle
						val := "true"
						if item.Val == "true" {
							val = "false"
						}
						return c, func() tea.Msg { return SaveConfigMsg{Key: item.Key, Value: val} }
					} else {
						// Save Text Input
						return c, func() tea.Msg { return SaveConfigMsg{Key: item.Key, Value: c.Input.Value()} }
					}
				}
			}
			c.Input, cmd = c.Input.Update(msg)
			return c, cmd
		}
	}
	newV, cmdV := c.MasterDetail.Update(msg)
	c.MasterDetail = newV.(*widget.MasterDetail)
	return c, tea.Batch(cmd, cmdV)
}

// Render logic (View)
func (c *Config) View() string {
	return c.MasterDetail.View()
}

func (c *Config) Resize(r layout.Rect) {
	c.BaseViewlet.Resize(r)
	c.MasterDetail.Resize(r)
}

func (c *Config) Shortcuts() []fv.Shortcut {
	return c.MasterDetail.Shortcuts()
}

func (c *Config) HandleMouse(x, y int, msg tea.MouseMsg) (fv.Viewlet, tea.Cmd, bool) {
	return c.MasterDetail.HandleMouse(x, y, msg)
}

// custom messages
type RequestCacheMsg struct{}
type RequestPruneMsg struct{}
type RequestUpdateMsg struct{}
type SaveConfigMsg struct{ Key, Value string }
