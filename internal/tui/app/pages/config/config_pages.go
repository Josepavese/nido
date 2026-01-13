package config

import (
	"fmt"

	fv "github.com/Josepavese/nido/internal/tui/kit/view"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConfigPageUpdate handles the Update settings view.
type ConfigPageUpdate struct {
	fv.BaseViewlet
	Parent *Config
}

func (p *ConfigPageUpdate) Init() tea.Cmd                            { return nil }
func (p *ConfigPageUpdate) Update(msg tea.Msg) (fv.Viewlet, tea.Cmd) { return p, nil }
func (p *ConfigPageUpdate) View() string {
	c := p.Parent
	var s string
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
		if c.Mode == ConfigModeForm && c.ActiveKey == "UPDATE" {
			s += activeTabStyle.Render(btn)
		} else {
			s += dimStyle.Render(btn)
		}
	}
	s += "\n\n" + dimStyle.Italic(true).Render("Press Enter to check for updates.")
	return lipgloss.NewStyle().Width(p.Width()).Height(p.Height()).Render(s)
}

// ConfigPageCache handles the Cache settings view.
type ConfigPageCache struct {
	fv.BaseViewlet
	Parent *Config
}

func (p *ConfigPageCache) Init() tea.Cmd                            { return nil }
func (p *ConfigPageCache) Update(msg tea.Msg) (fv.Viewlet, tea.Cmd) { return p, nil }
func (p *ConfigPageCache) View() string {
	c := p.Parent
	var s string
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
		if c.Mode == ConfigModeForm && c.ActiveKey == "CACHE" {
			s += activeTabStyle.Render(btn)
		} else {
			s += dimStyle.Render(btn)
		}
	}
	s += "\n\n" + dimStyle.Italic(true).Render("Press Enter to prune unused cached images.")
	return lipgloss.NewStyle().Width(p.Width()).Height(p.Height()).Render(s)
}

// ConfigPageGeneric handles standard text/toggle settings.
type ConfigPageGeneric struct {
	fv.BaseViewlet
	Parent *Config
	Key    string
}

func (p *ConfigPageGeneric) Init() tea.Cmd                            { return nil }
func (p *ConfigPageGeneric) Update(msg tea.Msg) (fv.Viewlet, tea.Cmd) { return p, nil }
func (p *ConfigPageGeneric) View() string {
	c := p.Parent
	// We need to find the item to render it
	var item ConfigItem
	found := false
	for _, it := range c.items {
		if it.Key == p.Key {
			item = it
			found = true
			break
		}
	}

	if !found {
		return dimStyle.Render("Setting not found.")
	}

	var s string
	if item.Key == "LINKED_CLONES" {
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
		s += fmt.Sprintf("%-15s %s\n", dimStyle.Render("Key:"), accentStyle.Render(item.Key))
		s += fmt.Sprintf("%-15s %s\n\n", dimStyle.Render("Current:"), item.Val)

		if c.Mode == ConfigModeForm && c.ActiveKey == p.Key {
			s += fmt.Sprintf("%s\n", c.Input.View())
			s += dimStyle.Render("Press Enter to save, Esc to cancel.")
		} else {
			s += dimStyle.Render("Press Enter/Right to edit.")
		}
		s += "\n\n" + dimStyle.Italic(true).Render(item.Desc)
	}
	return lipgloss.NewStyle().Width(p.Width()).Height(p.Height()).Render(s)
}
