package viewlet

import (
	"fmt"
	"io"

	"github.com/Josepavese/nido/internal/config"
	"github.com/Josepavese/nido/internal/tui/layout"
	"github.com/Josepavese/nido/internal/tui/theme"
	"github.com/charmbracelet/bubbles/list"
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
	dimStyle                 = lipgloss.NewStyle().Foreground(theme.Current().Palette.TextDim)
	accentStyle              = lipgloss.NewStyle().Foreground(theme.Current().Palette.Accent)
	successStyle             = lipgloss.NewStyle().Foreground(theme.Current().Palette.Success)
	errorStyle               = lipgloss.NewStyle().Foreground(theme.Current().Palette.Error)
	activeTabStyle           = lipgloss.NewStyle().Foreground(theme.Current().Palette.Accent).Bold(true)
	sidebarItemStyle         = lipgloss.NewStyle().Foreground(theme.Current().Palette.TextDim)
	sidebarItemSelectedStyle = lipgloss.NewStyle().Foreground(theme.Current().Palette.Accent).Bold(true)
	cardStyle                = lipgloss.NewStyle().Padding(1) // Simple padding
)

// ConfigItem represents a setting in the sidebar.
type ConfigItem struct {
	Key  string
	Val  string
	Desc string
}

func (i ConfigItem) String() string      { return fmt.Sprintf("%-18s", i.Key) }
func (i ConfigItem) Title() string       { return fmt.Sprintf("%-18s", i.Key) }
func (i ConfigItem) Description() string { return "" }
func (i ConfigItem) FilterValue() string { return i.Key }

// ConfigDelegate helps render items without extra padding.
type ConfigDelegate struct{}

func (d ConfigDelegate) Height() int                             { return 1 }
func (d ConfigDelegate) Spacing() int                            { return 0 }
func (d ConfigDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d ConfigDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	str, ok := listItem.(fmt.Stringer)
	if !ok {
		return
	}
	// Check if this item is selected
	if index == m.Index() {
		fmt.Fprint(w, sidebarItemSelectedStyle.Render(str.String()))
	} else {
		fmt.Fprint(w, sidebarItemStyle.Render(str.String()))
	}
}

// Config implements the Configuration viewlet.
type Config struct {
	BaseViewlet

	// State
	cfg  *config.Config
	Mode ConfigMode

	// Components
	Sidebar list.Model
	Input   textinput.Model
	Spinner spinner.Model

	// Data
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
func NewConfig(cfg *config.Config) *Config {
	c := &Config{
		cfg:  cfg,
		Mode: ConfigModeSidebar,
	}

	// Sidebar
	c.RefreshItems()

	// Input
	ti := textinput.New()
	ti.CharLimit = 100
	c.Input = ti

	// Spinner
	c.Spinner = spinner.New()
	c.Spinner.Style = accentStyle

	return c
}

// Init initializes the viewlet.
func (c *Config) Init() tea.Cmd {
	return c.Spinner.Tick
}

// RefreshItems reloads the configuration items from the config struct.
func (c *Config) RefreshItems() {
	items := []list.Item{
		ConfigItem{Key: "UPDATE", Val: "", Desc: "Check for updates and upgrade Nido."},
		ConfigItem{Key: "CACHE", Val: "", Desc: "Manage cached cloud images."},
		ConfigItem{Key: "BACKUP_DIR", Val: c.cfg.BackupDir, Desc: "Path to store template backups."},
		ConfigItem{Key: "IMAGE_DIR", Val: c.cfg.ImageDir, Desc: "Directory for cached cloud images."},
		ConfigItem{Key: "LINKED_CLONES", Val: fmt.Sprintf("%v", c.cfg.LinkedClones), Desc: "Use Copy-on-Write for disk efficiency."},
		ConfigItem{Key: "SSH_USER", Val: c.cfg.SSHUser, Desc: "Default user for SSH connections."},
		ConfigItem{Key: "TEMPLATE_DEFAULT", Val: c.cfg.TemplateDefault, Desc: "Default source template for new VMs."},
	}

	// Re-use existing list if possible to preserve state, or create new
	if c.Sidebar.Items() == nil {
		d := ConfigDelegate{}
		l := list.New(items, d, 28, 10)
		l.SetShowPagination(true)
		l.SetShowTitle(false)
		l.SetShowHelp(false)
		l.SetShowStatusBar(false)
		c.Sidebar = l
	} else {
		c.Sidebar.SetItems(items)
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
func (c *Config) Update(msg tea.Msg) (Viewlet, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case spinner.TickMsg:
		c.Spinner, cmd = c.Spinner.Update(msg)
		return c, cmd
	case tea.KeyMsg:
		if c.Mode == ConfigModeSidebar {
			switch msg.String() {
			case "enter", "right":
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
			c.Sidebar, cmd = c.Sidebar.Update(msg)
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
	return c, nil
}

// Render logic (View)
func (c *Config) View() string {
	// 2 Columns: Sidebar | Form
	sidebarWidth := 28 // Should match item width
	sidebar := cardStyle.Render(c.Sidebar.View())

	// Calculate form width: Total - Sidebar - Gap
	// Using layout.HStack so gap is standardized (theme.Space.SM? or MD?)
	gap := theme.Space.MD
	formWidth := c.Width - sidebarWidth - gap
	if formWidth < 20 {
		formWidth = 20
	}

	formStyle := cardStyle.Copy().Width(formWidth).Height(c.Height)
	var formContent string

	sel := c.Sidebar.SelectedItem()
	if sel != nil {
		item := sel.(ConfigItem)
		formContent = c.renderForm(item)
	} else {
		formContent = dimStyle.Render("Select a setting to edit.")
	}

	// Use layout helper for consistent gap
	return layout.HStack(gap,
		sidebar,
		formStyle.Render(formContent),
	)
}

func (c *Config) renderForm(item ConfigItem) string {
	var s string

	if item.Key == "UPDATE" {
		if c.UpdateChecking {
			s += fmt.Sprintf("%s Checking for updates...\n", c.Spinner.View())
		} else {
			ver := c.CurrentVersion
			if ver == "" {
				ver = "unknown"
			}
			s += fmt.Sprintf("%-18s %s\n", dimStyle.Render("Current Version:"), accentStyle.Render(ver))
			s += fmt.Sprintf("%-18s %s\n\n", dimStyle.Render("GitHub:"), "https://github.com/Josepavese/nido")

			btn := "[ CHECK FOR UPDATES ]"
			if c.Mode == ConfigModeForm {
				s += activeTabStyle.Render(btn)
			} else {
				s += dimStyle.Render(btn)
			}
		}
		s += "\n\n" + dimStyle.Italic(true).Render("Press Enter to check for updates.")
	} else if item.Key == "CACHE" {
		if c.Loading {
			s += fmt.Sprintf("%s Loading cache info...\n", c.Spinner.View())
		} else {
			s += fmt.Sprintf("%-15s %d images (%s)\n\n", dimStyle.Render("Total:"), c.CacheStats.TotalImages, c.CacheStats.TotalSize)

			for i, img := range c.CacheList {
				if i >= 6 {
					s += dimStyle.Render(fmt.Sprintf("  ... and %d more\n", len(c.CacheList)-6))
					break
				}
				s += fmt.Sprintf("  %-15s %-20s %s\n", img.Name, img.Version, dimStyle.Render(img.Size))
			}
			s += "\n"
			btn := "[ PRUNE UNUSED ]"
			if c.Mode == ConfigModeForm {
				s += activeTabStyle.Render(btn)
			} else {
				s += dimStyle.Render(btn)
			}
		}
		s += "\n\n" + dimStyle.Italic(true).Render("Press Enter to prune unused cached images.")
	} else if item.Key == "LINKED_CLONES" {
		s += fmt.Sprintf("%-15s %s\n", dimStyle.Render("Key:"), accentStyle.Render(item.Key))
		state := "DISABLED"
		color := dimStyle
		if item.Val == "true" {
			state = "ENABLED"
			color = successStyle
		} else {
			color = errorStyle
		}
		s += fmt.Sprintf("%-15s %s\n\n", dimStyle.Render("Value:"), fmt.Sprintf("[ %s ]", color.Render(state)))
		s += dimStyle.Render("Press Enter to toggle.")
		s += "\n" + dimStyle.Italic(true).Render(item.Desc)
	} else {
		// Standard Input
		s += fmt.Sprintf("%-15s %s\n", dimStyle.Render("Key:"), accentStyle.Render(item.Key))
		s += fmt.Sprintf("%-15s %s\n\n", dimStyle.Render("Current:"), item.Val)

		if c.Mode == ConfigModeForm {
			s += fmt.Sprintf("%s\n", c.Input.View())
			s += dimStyle.Render("Press Enter to save, Esc to cancel.")
		} else {
			s += dimStyle.Render("Press Enter/Right to edit.")
		}
		s += "\n\n" + dimStyle.Italic(true).Render(item.Desc)
	}

	return s
}

func (c *Config) Resize(width, height int) {
	c.Width = width
	c.Height = height
	c.Sidebar.SetSize(28, height)
}

func (c *Config) Shortcuts() []Shortcut {
	return DefaultShortcuts()
}

// custom messages
type RequestCacheMsg struct{}
type RequestPruneMsg struct{}
type RequestUpdateMsg struct{}
type SaveConfigMsg struct{ Key, Value string }
