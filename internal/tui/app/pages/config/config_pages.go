package config

import (
	"fmt"

	"github.com/Josepavese/nido/internal/tui/app/ops"
	"github.com/Josepavese/nido/internal/tui/kit/layout"
	"github.com/Josepavese/nido/internal/tui/kit/theme"
	fv "github.com/Josepavese/nido/internal/tui/kit/view"
	"github.com/Josepavese/nido/internal/tui/kit/widget"
	tea "github.com/charmbracelet/bubbletea"
)

// --- Global Config Form ---

type ConfigPageGlobalForm struct {
	fv.BaseViewlet
	Parent *Config
	Form   *widget.Form
	Header *widget.Card
	Modal  *widget.Modal

	// Inputs
	InputSSHUser   *widget.Input
	InputBackupDir *widget.Input
	InputImageDir  *widget.Input

	// Toggles
	ToggleLinkedClones *widget.Toggle

	SubmitButton *widget.Button
}

func NewConfigPageGlobalForm(parent *Config) *ConfigPageGlobalForm {
	p := &ConfigPageGlobalForm{Parent: parent}

	p.Header = widget.NewCard(theme.IconSystem, "Core", "Edit technical preferences.")

	// Config Fields
	p.InputSSHUser = widget.NewInput("SSH User", "vmuser", nil)
	p.InputSSHUser.SetValue(parent.cfg.SSHUser)

	p.InputBackupDir = widget.NewInput("Backup Dir", "/tmp/libvirt-pool/backups", nil)
	p.InputBackupDir.SetValue(parent.cfg.BackupDir)

	p.InputImageDir = widget.NewInput("Image Dir", "~/.nido/images", nil)
	p.InputImageDir.SetValue(parent.cfg.ImageDir)

	p.ToggleLinkedClones = widget.NewToggle("Linked Clones", parent.cfg.LinkedClones)

	// Modal
	p.Modal = widget.NewModal(
		"Confirm Configuration",
		"Are you sure you want to apply these changes?",
		func() tea.Cmd {
			updates := map[string]string{
				"SSH_USER":      p.InputSSHUser.Value(),
				"BACKUP_DIR":    p.InputBackupDir.Value(),
				"IMAGE_DIR":     p.InputImageDir.Value(),
				"LINKED_CLONES": fmt.Sprint(p.ToggleLinkedClones.Checked),
			}
			return tea.Batch(
				ops.SaveConfigMany(parent.cfg, updates),
				func() tea.Msg { return FocusSidebarMsg{} },
			)
		},
		nil,
	)

	p.SubmitButton = widget.NewSubmitButton("SAVE", "APPLY CHANGES", func() tea.Cmd {
		p.Modal.Show()
		return nil
	})

	p.Form = widget.NewForm(
		p.InputSSHUser,
		p.ToggleLinkedClones,
		p.InputBackupDir,
		p.InputImageDir,
		p.SubmitButton,
	)
	p.Form.Spacing = 0

	return p
}

func (p *ConfigPageGlobalForm) Init() tea.Cmd { return nil }
func (p *ConfigPageGlobalForm) Update(msg tea.Msg) (fv.Viewlet, tea.Cmd) {
	if !p.Focused() {
		return p, nil
	}

	if msg, ok := msg.(tea.KeyMsg); ok && !p.Modal.IsActive() {
		switch msg.String() {
		case "tab":
			return p, p.Form.NextField()
		case "shift+tab":
			return p, p.Form.PrevField()
		}
	}

	if p.Modal.IsActive() {
		newModal, cmd := p.Modal.Update(msg)
		p.Modal = newModal
		return p, cmd
	}

	newForm, cmd := p.Form.Update(msg)
	p.Form = newForm
	return p, cmd
}

func (p *ConfigPageGlobalForm) View() string {
	w := p.Width()
	safeWidth := w - 4
	if safeWidth > 60 {
		safeWidth = 60
	}
	p.Form.Width = safeWidth

	// Parent handles modal rendering globally
	if p.Modal.IsActive() {
		return ""
	}

	// Custom Layout: Header + Horizontal Row + Rest
	rowUser := p.InputSSHUser.View(safeWidth/2 - 1)
	rowClone := p.ToggleLinkedClones.View(safeWidth/2 - 1)
	row := layout.HStack(2, rowUser, rowClone)

	return layout.VStack(0,
		p.Header.View(safeWidth),
		row,
		p.InputBackupDir.View(safeWidth),
		p.InputImageDir.View(safeWidth),
		layout.VStack(1, "", p.SubmitButton.View(safeWidth)), // Add some spacing before button
	)
}
func (p *ConfigPageGlobalForm) Focus() tea.Cmd {
	p.BaseViewlet.Focus()
	return p.Form.Focus()
}
func (p *ConfigPageGlobalForm) Blur() {
	p.BaseViewlet.Blur()
	p.Form.Blur()
}
func (p *ConfigPageGlobalForm) HandleMouse(x, y int, msg tea.MouseMsg) (fv.Viewlet, tea.Cmd, bool) {
	return p, nil, false // Standard Form doesn't need custom mouse in detail yet
}

func (p *ConfigPageGlobalForm) IsModalActive() bool {
	return p.Modal != nil && p.Modal.IsActive()
}

func (p *ConfigPageGlobalForm) HasActiveInput() bool {
	return p.Form != nil && p.Form.HasActiveInput()
}

// --- Update Page ---

type ConfigPageUpdate struct {
	fv.BaseViewlet
	Parent *Config

	Form        *widget.Form
	Header      *widget.Card
	Current     *widget.Input
	CheckButton *widget.Button
}

func NewConfigPageUpdate(parent *Config) *ConfigPageUpdate {
	p := &ConfigPageUpdate{Parent: parent}
	p.Header = widget.NewCard(theme.IconTemplate, "Evolution", "Check for newer versions")

	p.Current = widget.NewInput("Current Version", "Loading...", nil)
	p.Current.Disabled = true

	p.CheckButton = widget.NewSubmitButton("Action", "CHECK UPDATE", func() tea.Cmd {
		parent.UpdateChecking = true
		return ops.CheckUpdate()
	})

	p.Form = widget.NewForm(p.Header, p.Current, p.CheckButton)
	p.Form.Spacing = 1
	return p
}

func (p *ConfigPageUpdate) Init() tea.Cmd { return nil }

func (p *ConfigPageUpdate) Update(msg tea.Msg) (fv.Viewlet, tea.Cmd) {
	// Updates values dynamically
	if p.Parent.CurrentVersion != "" {
		p.Current.SetValue(p.Parent.CurrentVersion)
	}

	if !p.Focused() {
		return p, nil
	}
	newForm, cmd := p.Form.Update(msg)
	p.Form = newForm
	return p, cmd
}

func (p *ConfigPageUpdate) View() string {
	w := p.Width()
	safeWidth := w - 4
	if safeWidth > 60 {
		safeWidth = 60
	}
	p.Form.Width = safeWidth
	return p.Form.View(safeWidth)
}

func (p *ConfigPageUpdate) Focus() tea.Cmd { p.BaseViewlet.Focus(); return p.Form.Focus() }
func (p *ConfigPageUpdate) Blur()          { p.BaseViewlet.Blur(); p.Form.Blur() }

// --- Cache Page (Maintenance) ---

type ConfigPageCache struct {
	fv.BaseViewlet
	Parent *Config

	Form        *widget.Form
	Header      *widget.Card
	Stats       *widget.Input
	PruneButton *widget.Button
}

func NewConfigPageCache(parent *Config) *ConfigPageCache {
	p := &ConfigPageCache{Parent: parent}
	p.Header = widget.NewCard(theme.IconCache, "Artifacts", "Manage local storage")

	p.Stats = widget.NewInput("Cache Stats", "Loading...", nil)
	p.Stats.Disabled = true

	p.PruneButton = widget.NewSubmitButton("Action", "PRUNE UNUSED", func() tea.Cmd {
		return func() tea.Msg { return ops.RequestPruneMsg{} }
	})

	p.Form = widget.NewForm(p.Header, p.Stats, p.PruneButton)
	return p
}

func (p *ConfigPageCache) Init() tea.Cmd { return nil }

func (p *ConfigPageCache) Update(msg tea.Msg) (fv.Viewlet, tea.Cmd) {
	// Sync Stats
	stats := fmt.Sprintf("%d images • %s", p.Parent.CacheStats.TotalImages, p.Parent.CacheStats.TotalSize)
	p.Stats.SetValue(stats)

	if !p.Focused() {
		return p, nil
	}
	newForm, cmd := p.Form.Update(msg)
	p.Form = newForm
	return p, cmd
}

func (p *ConfigPageCache) View() string {
	w := p.Width()
	safeWidth := w - 4
	if safeWidth > 60 {
		safeWidth = 60
	}
	p.Form.Width = safeWidth
	return p.Form.View(safeWidth)
}
func (p *ConfigPageCache) Focus() tea.Cmd { p.BaseViewlet.Focus(); return p.Form.Focus() }
func (p *ConfigPageCache) Blur()          { p.BaseViewlet.Blur(); p.Form.Blur() }

// --- Doctor Page ---

type ConfigPageDoctor struct {
	fv.BaseViewlet
	Parent *Config

	Form      *widget.Form
	Header    *widget.Card
	RunButton *widget.Button
	Output    string // Simple text output for now, maybe scrollable later
}

func NewConfigPageDoctor(parent *Config) *ConfigPageDoctor {
	p := &ConfigPageDoctor{Parent: parent}
	p.Header = widget.NewCard(theme.IconDoctor, "Health", "Run system health check")

	p.RunButton = widget.NewSubmitButton("Diagnostic", "RUN CHECKS", func() tea.Cmd {
		return ops.RunDoctor()
	})

	p.Form = widget.NewForm(p.Header, p.RunButton)
	return p
}

func (p *ConfigPageDoctor) Init() tea.Cmd { return nil }
func (p *ConfigPageDoctor) Update(msg tea.Msg) (fv.Viewlet, tea.Cmd) {
	if !p.Focused() {
		return p, nil
	}
	newForm, cmd := p.Form.Update(msg)
	p.Form = newForm
	return p, cmd
}
func (p *ConfigPageDoctor) View() string {
	w := p.Width()
	safeWidth := w - 4
	if safeWidth > 60 {
		safeWidth = 60
	}
	p.Form.Width = safeWidth

	base := p.Form.View(safeWidth)

	// Append output if present
	if p.Output != "" {
		t := theme.Current()
		outStyle := t.Styles.TextDim.Copy().Width(safeWidth).PaddingTop(1)
		base = layout.VStack(0, base, outStyle.Render("Creating diagnostic report...\n"+p.Output))
	}
	return base
}
func (p *ConfigPageDoctor) Focus() tea.Cmd { p.BaseViewlet.Focus(); return p.Form.Focus() }
func (p *ConfigPageDoctor) Blur()          { p.BaseViewlet.Blur(); p.Form.Blur() }

// --- Appearance Page ---

type ConfigPageAppearance struct {
	fv.BaseViewlet
	Parent *Config
	Form   *widget.Form
	Header *widget.Card
	Modal  *widget.Modal

	// Inputs
	InputSidebarWidth     *widget.Input
	InputSidebarWideWidth *widget.Input
	InputTabMinWidth      *widget.Input
	InputGapScale         *widget.Input

	SubmitButton *widget.Button
}

func NewConfigPageAppearance(parent *Config) *ConfigPageAppearance {
	p := &ConfigPageAppearance{Parent: parent}
	p.Header = widget.NewCard(theme.IconLayout, "Rice", "UI and aesthetic tweaks.")

	tui := parent.cfg.TUI

	p.InputSidebarWidth = widget.NewInput("Sidebar W", "30", nil)
	p.InputSidebarWidth.SetValue(fmt.Sprint(tui.SidebarWidth))

	p.InputSidebarWideWidth = widget.NewInput("Wide Width", "38", nil)
	p.InputSidebarWideWidth.SetValue(fmt.Sprint(tui.SidebarWideWidth))

	p.InputTabMinWidth = widget.NewInput("Tab Min W", "6", nil)
	p.InputTabMinWidth.SetValue(fmt.Sprint(tui.TabMinWidth))

	p.InputGapScale = widget.NewInput("Gap Scale", "1", nil)
	p.InputGapScale.SetValue(fmt.Sprint(tui.GapScale))

	p.Modal = widget.NewModal(
		"Confirm Theme",
		"Apply these visual changes?",
		func() tea.Cmd {
			updates := map[string]string{
				"TUI_SIDEBAR_WIDTH":      p.InputSidebarWidth.Value(),
				"TUI_SIDEBAR_WIDE_WIDTH": p.InputSidebarWideWidth.Value(),
				"TUI_TAB_MIN_WIDTH":      p.InputTabMinWidth.Value(),
				"TUI_GAP_SCALE":          p.InputGapScale.Value(),
			}
			return tea.Batch(
				ops.SaveConfigMany(parent.cfg, updates),
				func() tea.Msg { return FocusSidebarMsg{} },
			)
		},
		nil,
	)

	p.SubmitButton = widget.NewSubmitButton("SAVE", "APPLY THEME", func() tea.Cmd {
		p.Modal.Show()
		return nil
	})

	p.Form = widget.NewForm(
		p.Header,
		p.InputSidebarWidth,
		p.InputSidebarWideWidth,
		p.InputTabMinWidth,
		p.InputGapScale,
		p.SubmitButton,
	)
	p.Form.Spacing = 0

	return p
}

func (p *ConfigPageAppearance) Init() tea.Cmd { return nil }

func (p *ConfigPageAppearance) Update(msg tea.Msg) (fv.Viewlet, tea.Cmd) {
	if !p.Focused() {
		return p, nil
	}

	if msg, ok := msg.(tea.KeyMsg); ok && !p.Modal.IsActive() {
		switch msg.String() {
		case "tab":
			return p, p.Form.NextField()
		case "shift+tab":
			return p, p.Form.PrevField()
		}
	}

	if p.Modal.IsActive() {
		newModal, cmd := p.Modal.Update(msg)
		p.Modal = newModal
		return p, cmd
	}

	newForm, cmd := p.Form.Update(msg)
	p.Form = newForm
	return p, cmd
}

func (p *ConfigPageAppearance) View() string {
	w := p.Width()
	safeWidth := w - 4
	if safeWidth > 60 {
		safeWidth = 60
	}

	// Parent handles modal rendering globally
	if p.Modal.IsActive() {
		return ""
	}

	p.Form.Width = safeWidth

	// Custom Layout for Appearance
	// Row 1: Sidebar W | Wide W
	rowSidebar := layout.HStack(1,
		p.InputSidebarWidth.View((safeWidth/2)-1),
		p.InputSidebarWideWidth.View((safeWidth/2)-1),
	)
	// Row 2: Tab Min W | Gap Scale
	rowOther := layout.HStack(1,
		p.InputTabMinWidth.View((safeWidth/2)-1),
		p.InputGapScale.View((safeWidth/2)-1),
	)

	return layout.VStack(0,
		p.Header.View(safeWidth),
		rowSidebar,
		rowOther,
		layout.VStack(1, "", p.SubmitButton.View(safeWidth)),
	)
}

func (p *ConfigPageAppearance) Focus() tea.Cmd { p.BaseViewlet.Focus(); return p.Form.Focus() }
func (p *ConfigPageAppearance) Blur()          { p.BaseViewlet.Blur(); p.Form.Blur() }

func (p *ConfigPageAppearance) IsModalActive() bool {
	return p.Modal != nil && p.Modal.IsActive()
}

func (p *ConfigPageAppearance) HasActiveInput() bool {
	return p.Form != nil && p.Form.HasActiveInput()
}
