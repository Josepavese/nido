package config

import (
	"fmt"
	"strings"

	"github.com/Josepavese/nido/v4/internal/build"
	"github.com/Josepavese/nido/v4/internal/image"
	"github.com/Josepavese/nido/v4/internal/tui/app/ops"
	"github.com/Josepavese/nido/v4/internal/tui/kit/layout"
	"github.com/Josepavese/nido/v4/internal/tui/kit/theme"
	fv "github.com/Josepavese/nido/v4/internal/tui/kit/view"
	"github.com/Josepavese/nido/v4/internal/tui/kit/widget"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

	p.InputBackupDir = widget.NewInput("Backup Dir", parent.cfg.BackupDir, nil)
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
				ops.SaveConfigMany(parent.cfg, updates, parent.publish),
				func() tea.Msg { return FocusSidebarMsg{} },
			)
		},
		nil,
	)

	p.SubmitButton = widget.NewSubmitButton("SAVE", "SAVE", func() tea.Cmd {
		p.Modal.Show()
		return nil
	})

	p.Form = widget.NewForm(
		p.Header,
		widget.NewRow(p.InputSSHUser, p.ToggleLinkedClones),
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

	// Tab handling delegated to Form widget

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
	safeWidth := layout.FormWidth(w, theme.Inset.TotalContent)
	p.Form.Width = safeWidth

	// Parent handles modal rendering globally
	if p.Modal.IsActive() {
		return ""
	}

	return p.Form.Viewport(safeWidth, p.Height())
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
	if p.Modal != nil && p.Modal.IsActive() {
		cmd, handled := p.Modal.HandleMouse(x, y, msg)
		return p, cmd, handled
	}
	cmd, handled := p.Form.HandleMouse(x, y, msg)
	return p, cmd, handled
}

func (p *ConfigPageGlobalForm) IsModalActive() bool {
	return p.Modal != nil && p.Modal.IsActive()
}

func (p *ConfigPageGlobalForm) HasActiveTextInput() bool {
	return p.Form != nil && p.Form.HasActiveTextInput()
}

func (p *ConfigPageGlobalForm) HasActiveFocus() bool {
	return p.Form != nil && p.Form.HasActiveFocus()
}

func (p *ConfigPageGlobalForm) Focusable() bool {
	if p.Form == nil {
		return false
	}
	return p.Form.Focusable()
}

func (p *ConfigPageGlobalForm) Shortcuts() []fv.Shortcut {
	if overlay := p.Parent.activeOverlay(); overlay != nil {
		return overlay.Shortcuts()
	}
	return []fv.Shortcut{
		{Key: "tab", Label: "glide"},
		{Key: "enter", Label: "engage"},
		{Key: "esc", Label: "back"},
	}
}

// --- Update Page ---

type ConfigPageUpdate struct {
	fv.BaseViewlet
	Parent *Config

	Form         *widget.Form
	Header       *widget.Card
	Current      *widget.Input
	Latest       *widget.Input
	CheckButton  *widget.Button
	UpdateButton *widget.Button
	ConfirmModal *widget.Modal
	ResultModal  *widget.Modal
}

func NewConfigPageUpdate(parent *Config) *ConfigPageUpdate {
	p := &ConfigPageUpdate{Parent: parent}
	p.Header = widget.NewCard(theme.IconTemplate, "Evolution", "Check for newer versions")

	p.Current = widget.NewInput("Current", build.Version, nil)
	p.Current.Disabled = true

	p.Latest = widget.NewInput("Latest", "Loading...", nil)
	p.Latest.Disabled = true

	p.CheckButton = widget.NewSubmitButton("Action", "CHECK", func() tea.Cmd {
		parent.UpdateChecking = true
		return func() tea.Msg { return ops.RequestUpdateMsg{Manual: true} }
	})

	p.ConfirmModal = widget.NewModal(
		"Evolutionary Ascent",
		"Confirm migration to the next evolutionary state?",
		func() tea.Cmd {
			return func() tea.Msg { return ops.RequestApplyUpdateMsg{} }
		},
		nil,
	)

	p.UpdateButton = widget.NewSubmitButton("Action", "UPDATE", func() tea.Cmd {
		p.ConfirmModal.Show()
		return nil
	})
	p.UpdateButton.SetColor(theme.Current().Palette.Success)

	p.Form = widget.NewForm(p.Header, p.Current, p.Latest, p.CheckButton)
	p.Form.Spacing = 0
	return p
}

func (p *ConfigPageUpdate) Init() tea.Cmd { return nil }

func (p *ConfigPageUpdate) Update(msg tea.Msg) (fv.Viewlet, tea.Cmd) {
	if p.ResultModal != nil && p.ResultModal.IsActive() {
		newModal, cmd := p.ResultModal.Update(msg)
		p.ResultModal = newModal
		return p, cmd
	}

	if p.ConfirmModal.IsActive() {
		newModal, cmd := p.ConfirmModal.Update(msg)
		p.ConfirmModal = newModal
		return p, cmd
	}

	switch msg := msg.(type) {
	case ops.UpdateCheckMsg:
		if msg.Manual && msg.Latest == msg.Current && msg.Err == nil {
			p.ResultModal = widget.NewAlertModal(
				"Evolutionary Peak",
				"Your Nest is already at the latest evolutionary state.\nNo ascension required.",
				nil,
			)
			p.ResultModal.SetLevel(widget.ModalLevelSuccess)
			p.ResultModal.Show()
		} else if msg.Manual && msg.Err != nil {
			p.ResultModal = widget.NewAlertModal(
				"Check Failed",
				fmt.Sprintf("Failed to communicate with the mother nest:\n%v", msg.Err),
				nil,
			)
			p.ResultModal.Show()
		}
		return p, nil

	case ops.ApplyUpdateMsg:
		if msg.Err == nil {
			p.ResultModal = widget.NewAlertModal(
				"Evolution Complete",
				"Nido has been successfully upgraded.\nPlease restart the application to apply changes.",
				nil,
			)
			p.ResultModal.SetLevel(widget.ModalLevelSuccess)
		} else {
			p.ResultModal = widget.NewAlertModal(
				"Evolution Failed",
				fmt.Sprintf("Failed to upgrade Nido:\n%v", msg.Err),
				nil,
			)
			p.ResultModal.SetLevel(widget.ModalLevelDanger)
		}
		p.ResultModal.Show()
		return p, nil
	}

	// Updates values dynamically
	if p.Parent.CurrentVersion != "" {
		p.Current.SetValue(p.Parent.CurrentVersion)
	}
	if p.Parent.LatestVersion != "" {
		p.Latest.SetValue(p.Parent.LatestVersion)
	} else if p.Parent.UpdateChecking {
		p.Latest.SetValue("Checking...")
	} else {
		p.Latest.SetValue("Unknown (Offline?)")
	}

	// Dynamic Action Button
	isNewer := p.Parent.LatestVersion != "" && p.Parent.LatestVersion != p.Parent.CurrentVersion
	if isNewer {
		p.Form.Elements[3] = p.UpdateButton
	} else {
		p.Form.Elements[3] = p.CheckButton
	}

	if !p.Focused() {
		return p, nil
	}
	newForm, cmd := p.Form.Update(msg)
	p.Form = newForm
	return p, cmd
}

func (p *ConfigPageUpdate) View() string {
	if p.IsModalActive() {
		return "" // Parent handles rendering
	}
	w := p.Width()
	safeWidth := layout.FormWidth(w, theme.Inset.TotalContent)
	p.Form.Width = safeWidth
	return p.Form.Viewport(safeWidth, p.Height())
}
func (p *ConfigPageUpdate) IsModalActive() bool {
	return (p.ConfirmModal != nil && p.ConfirmModal.IsActive()) || (p.ResultModal != nil && p.ResultModal.IsActive())
}
func (p *ConfigPageUpdate) HandleMouse(x, y int, msg tea.MouseMsg) (fv.Viewlet, tea.Cmd, bool) {
	if p.ResultModal != nil && p.ResultModal.IsActive() {
		cmd, handled := p.ResultModal.HandleMouse(x, y, msg)
		return p, cmd, handled
	}
	if p.ConfirmModal != nil && p.ConfirmModal.IsActive() {
		cmd, handled := p.ConfirmModal.HandleMouse(x, y, msg)
		return p, cmd, handled
	}
	cmd, handled := p.Form.HandleMouse(x, y, msg)
	return p, cmd, handled
}

func (p *ConfigPageUpdate) Focus() tea.Cmd { p.BaseViewlet.Focus(); return p.Form.Focus() }
func (p *ConfigPageUpdate) Blur()          { p.BaseViewlet.Blur(); p.Form.Blur() }
func (p *ConfigPageUpdate) HasActiveTextInput() bool {
	return p.Form != nil && p.Form.HasActiveTextInput()
}

func (p *ConfigPageUpdate) HasActiveFocus() bool {
	return p.Form != nil && p.Form.HasActiveFocus()
}

func (p *ConfigPageUpdate) Focusable() bool {
	if p.Form == nil {
		return false
	}
	return p.Form.Focusable()
}

func (p *ConfigPageUpdate) Shortcuts() []fv.Shortcut {
	if overlay := p.Parent.activeOverlay(); overlay != nil {
		return overlay.Shortcuts()
	}
	return []fv.Shortcut{
		{Key: "enter", Label: "engage"},
		{Key: "esc", Label: "back"},
	}
}

// --- Cache Page (Maintenance) ---

type ConfigPageCache struct {
	fv.BaseViewlet
	Parent *Config

	Form        *widget.Form
	Header      *widget.Card
	Stats       *widget.Input
	PruneButton *widget.Button
	ResultModal *widget.Modal
}

func NewConfigPageCache(parent *Config) *ConfigPageCache {
	p := &ConfigPageCache{Parent: parent}
	p.Header = widget.NewCard(theme.IconCache, "Artifacts", "Manage local storage")

	p.Stats = widget.NewInput("Cache Stats", "Loading...", nil)
	p.Stats.Disabled = true

	p.PruneButton = widget.NewSubmitButton("Action", "PRUNE", func() tea.Cmd {
		return func() tea.Msg { return ops.RequestPruneMsg{} }
	})

	p.Form = widget.NewForm(p.Header, p.Stats, p.PruneButton)
	return p
}

func (p *ConfigPageCache) Init() tea.Cmd { return nil }

func (p *ConfigPageCache) Update(msg tea.Msg) (fv.Viewlet, tea.Cmd) {
	if p.ResultModal != nil && p.ResultModal.IsActive() {
		newModal, cmd := p.ResultModal.Update(msg)
		p.ResultModal = newModal
		return p, cmd
	}

	// Handle Prune Result locally to show Modal
	if res, ok := msg.(ops.OpResultMsg); ok && res.Op == "prune" && res.Err == nil {
		title := "Prune Complete"
		body := "Cache is clean."

		if stats, ok := res.Data.(ops.PruneStats); ok {
			if stats.Count == 0 {
				body = "Nothing to prune.\nYour cache is already efficient."
			} else {
				body = fmt.Sprintf("Removed %d images.\nReclaimed %s.", stats.Count, image.FormatBytes(stats.Reclaimed))
			}
		}

		p.ResultModal = widget.NewAlertModal(title, body, nil)
		p.ResultModal.SetLevel(widget.ModalLevelSuccess)
		p.ResultModal.Show()
		return p, nil
	}

	// Sync Stats
	stats := "Loading..."
	// We use p.Parent.CacheStats.TotalSize != "" as a signal that data has arrived at least once
	if p.Parent.CacheStats.TotalSize != "" {
		stats = fmt.Sprintf("%d images • %s", p.Parent.CacheStats.TotalImages, p.Parent.CacheStats.TotalSize)
	}
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
	safeWidth := layout.FormWidth(w, theme.Inset.TotalContent)
	p.Form.Width = safeWidth

	if p.ResultModal != nil && p.ResultModal.IsActive() {
		// If using full screen replace:
		// return p.ResultModal.View()
		// But usually we hide form? ConfigPageUpdate returns empty string to let parent handle?
		// Note lines 118-121 in ConfigPageGlobalForm: "if p.Modal.IsActive() { return "" }"
		// ConfigPageUpdate also does "if p.IsModalActive() { return "" }"
		// So I should do the same.
		return ""
	}

	return p.Form.Viewport(safeWidth, p.Height())
}
func (p *ConfigPageCache) Focus() tea.Cmd { p.BaseViewlet.Focus(); return p.Form.Focus() }
func (p *ConfigPageCache) Blur()          { p.BaseViewlet.Blur(); p.Form.Blur() }
func (p *ConfigPageCache) HasActiveTextInput() bool {
	return p.Form != nil && p.Form.HasActiveTextInput()
}

func (p *ConfigPageCache) HasActiveFocus() bool {
	return p.Form != nil && p.Form.HasActiveFocus()
}

func (p *ConfigPageCache) Focusable() bool {
	if p.Form == nil {
		return false
	}
	return p.Form.Focusable()
}

func (p *ConfigPageCache) Shortcuts() []fv.Shortcut {
	if overlay := p.Parent.activeOverlay(); overlay != nil {
		return overlay.Shortcuts()
	}
	return []fv.Shortcut{
		{Key: "enter", Label: "purge"},
		{Key: "esc", Label: "back"},
	}
}

// Added missing interface method for Modal support if parent checks it
func (p *ConfigPageCache) IsModalActive() bool {
	return p.ResultModal != nil && p.ResultModal.IsActive()
}

func (p *ConfigPageCache) HandleMouse(x, y int, msg tea.MouseMsg) (fv.Viewlet, tea.Cmd, bool) {
	if p.ResultModal != nil && p.ResultModal.IsActive() {
		cmd, handled := p.ResultModal.HandleMouse(x, y, msg)
		return p, cmd, handled
	}
	cmd, handled := p.Form.HandleMouse(x, y, msg)
	return p, cmd, handled
}

// --- Doctor Page ---

type ConfigPageDoctor struct {
	fv.BaseViewlet
	Parent *Config

	Form      *widget.Form
	RunButton *widget.Button
	Reports   []ops.DoctorReport
	Err       error
	Modal     *widget.Modal
}

func NewConfigPageDoctor(parent *Config) *ConfigPageDoctor {
	p := &ConfigPageDoctor{Parent: parent}

	p.RunButton = widget.NewSubmitButton("", "RUN DIAGNOSTICS", func() tea.Cmd {
		return ops.RunDoctor()
	})

	p.Form = widget.NewForm(widget.NewCard(theme.IconDoctor, "Health", "Keep the nest in good shape."), p.RunButton)
	p.Form.Spacing = 0
	return p
}

func (p *ConfigPageDoctor) Init() tea.Cmd { return nil }
func (p *ConfigPageDoctor) Update(msg tea.Msg) (fv.Viewlet, tea.Cmd) {
	if p.Modal != nil && p.Modal.IsActive() {
		newModal, cmd := p.Modal.Update(msg)
		p.Modal = newModal
		return p, cmd
	}

	if !p.Focused() {
		return p, nil
	}

	switch msg := msg.(type) {
	case ops.DoctorResultMsg:
		p.Reports = msg.Reports
		p.Err = msg.Err
		if p.Err != nil {
			p.Modal = widget.NewAlertModal("Diagnostics failed", p.Err.Error(), func() tea.Cmd { return func() tea.Msg { return FocusSidebarMsg{} } })
			p.Modal.Show()
			return p, nil
		}

		if p.Err == nil && len(p.Reports) > 0 {
			// Create Result Modal
			passed := 0
			for _, r := range p.Reports {
				if r.Passed {
					passed++
				}
			}
			total := len(p.Reports)
			integrity := (float64(passed) / float64(total)) * 100

			var reportLines []string
			for _, r := range p.Reports {
				icon := theme.IconCheck
				if !r.Passed {
					icon = theme.IconError
				}
				line := fmt.Sprintf("%-2s %-12s %s", icon, r.Label, r.Details)
				reportLines = append(reportLines, line)
			}

			summary := fmt.Sprintf("NEST INTEGRITY: %.0f%%\n\n%s", integrity, strings.Join(reportLines, "\n"))
			p.Modal = widget.NewAlertModal("System Health Report", summary, func() tea.Cmd {
				return func() tea.Msg { return FocusSidebarMsg{} }
			})
			p.Modal.SetLevel(widget.ModalLevelSuccess)
			if integrity < 100 {
				p.Modal.SetLevel(widget.ModalLevelDanger)
			}
			p.Modal.MessageAlign = lipgloss.Left
			p.Modal.Show()
			return p, nil
		}

		// After results are in (if no modal), return focus to sidebar
		return p, func() tea.Msg { return FocusSidebarMsg{} }
	}

	newForm, cmd := p.Form.Update(msg)
	p.Form = newForm
	return p, cmd
}
func (p *ConfigPageDoctor) View() string {
	w := p.Width()
	safeWidth := layout.FormWidth(w, theme.Inset.TotalContent)
	p.Form.Width = safeWidth

	return p.Form.Viewport(safeWidth, p.Height())
}
func (p *ConfigPageDoctor) Focus() tea.Cmd { p.BaseViewlet.Focus(); return p.Form.Focus() }
func (p *ConfigPageDoctor) Blur()          { p.BaseViewlet.Blur(); p.Form.Blur() }
func (p *ConfigPageDoctor) HasActiveTextInput() bool {
	return p.Form != nil && p.Form.HasActiveTextInput()
}

func (p *ConfigPageDoctor) HasActiveFocus() bool {
	return p.Form != nil && p.Form.HasActiveFocus()
}

func (p *ConfigPageDoctor) Focusable() bool {
	if p.Form == nil {
		return false
	}
	return p.Form.Focusable()
}

func (p *ConfigPageDoctor) Shortcuts() []fv.Shortcut {
	if overlay := p.Parent.activeOverlay(); overlay != nil {
		return overlay.Shortcuts()
	}
	return []fv.Shortcut{
		{Key: "enter", Label: "scan"},
		{Key: "esc", Label: "back"},
	}
}
func (p *ConfigPageDoctor) IsModalActive() bool {
	return p.Modal != nil && p.Modal.IsActive()
}
func (p *ConfigPageDoctor) HandleMouse(x, y int, msg tea.MouseMsg) (fv.Viewlet, tea.Cmd, bool) {
	if p.Modal != nil && p.Modal.IsActive() {
		cmd, handled := p.Modal.HandleMouse(x, y, msg)
		return p, cmd, handled
	}
	cmd, handled := p.Form.HandleMouse(x, y, msg)
	return p, cmd, handled
}

// --- Appearance Page ---

type ConfigPageAppearance struct {
	fv.BaseViewlet
	Parent *Config
	Form   *widget.Form
	Header *widget.Card
	Modal  *widget.Modal

	// Theme Selection
	SelectTheme *widget.Select
	ThemeModal  *widget.ListModal

	// Inputs
	InputSidebarWidth     *widget.Input
	InputSidebarWideWidth *widget.Input
	InputTabMinWidth      *widget.Input
	InputGapScale         *widget.Input

	SubmitButton *widget.Button
}

func NewConfigPageAppearance(parent *Config) *ConfigPageAppearance {
	// Ensure user themes are loaded
	_ = theme.LoadUserThemes()

	p := &ConfigPageAppearance{Parent: parent}
	p.Header = widget.NewCard(theme.IconLayout, "Rice", "UI and aesthetic tweaks.")

	tui := parent.cfg.TUI

	// Theme Select
	currentThemeName := parent.cfg.Theme
	if currentThemeName == "" {
		currentThemeName = "Auto"
	}

	// Prepare items for modal
	avail := theme.AvailableThemes()
	var items []list.Item
	for _, n := range avail {
		p := theme.GetPalette(n)
		items = append(items, widget.ThemeItem{
			Name:         n,
			PrimaryColor: p.Accent, // Use Accent as the primary identifying color
		})
	}

	p.ThemeModal = widget.NewListModal(
		"Select Theme",
		items,
		44, 12, // User Requested: "More compact width"
		func(selected list.Item) tea.Cmd {
			// On Select
			// We now cast to ThemeItem
			ti, ok := selected.(widget.ThemeItem)
			if !ok {
				return nil
			}
			name := ti.Name
			p.SelectTheme.SetValue(name)

			// Update Config & Apply
			// We update config in memory, but maybe we should save immediately?
			// The original "SAVE" button saves everything.
			// Ideally live preview would be nice, but per plan, we just set the field value.
			// However, to see it, we might want to apply it.
			theme.SetTheme(name)

			return func() tea.Msg { return FocusSidebarMsg{} } // Return valid command or msg
		},
		func() tea.Cmd {
			theme.SetTheme(p.SelectTheme.Value)
			return nil
		},
	)

	// Live Preview
	p.ThemeModal.OnHighlight = func(item list.Item) {
		if ti, ok := item.(widget.ThemeItem); ok {
			theme.SetTheme(ti.Name)
		}
	}

	p.SelectTheme = widget.NewSelect("Theme", currentThemeName, func() tea.Cmd {
		// Pre-select the current theme
		target := strings.ToLower(p.SelectTheme.Value)
		items := p.ThemeModal.List.Items()
		for i, item := range items {
			if ti, ok := item.(widget.ThemeItem); ok {
				if strings.ToLower(ti.Name) == target {
					p.ThemeModal.List.Select(i)
					break
				}
			}
		}
		p.ThemeModal.Show()
		return nil
	})

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
				"THEME":                  p.SelectTheme.Value,
				"TUI_SIDEBAR_WIDTH":      p.InputSidebarWidth.Value(),
				"TUI_SIDEBAR_WIDE_WIDTH": p.InputSidebarWideWidth.Value(),
				"TUI_TAB_MIN_WIDTH":      p.InputTabMinWidth.Value(),
				"TUI_GAP_SCALE":          p.InputGapScale.Value(),
			}
			return tea.Batch(
				ops.SaveConfigMany(parent.cfg, updates, parent.publish),
				func() tea.Msg { return FocusSidebarMsg{} },
			)
		},
		nil,
	)

	p.SubmitButton = widget.NewSubmitButton("SAVE", "SAVE", func() tea.Cmd {
		p.Modal.Show()
		return nil
	})

	p.Form = widget.NewForm(
		p.Header,
		p.SelectTheme, // Added first
		widget.NewRow(p.InputSidebarWidth, p.InputSidebarWideWidth),
		widget.NewRow(p.InputTabMinWidth, p.InputGapScale),
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

	// Tab handling delegated to Form widget

	if p.ThemeModal.IsActive() {
		_, cmd := p.ThemeModal.Update(msg)
		return p, cmd
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
	safeWidth := layout.FormWidth(w, theme.Inset.TotalContent)

	// Render Modals on top
	if p.ThemeModal.IsActive() {
		// Overlay logic: render the modal centered
		// Since View() returns string, proper overlay relies on the parent Shell
		// or we return just the modal if it consumes full screen.
		// ListModal returns a styled box.
		// We'll trust standard bubbletea render loop or parent overlay logic.
		// If Parent Config handles Modal overlays via IsModalActive check returning "",
		// then we should do the same here if the parent draws it?
		// No, ConfigPageAppearance.View IS the content.
		// If we return the modal string, it replaces the content.
		// To do proper overlay (transparent background), we need complex composites.
		// For now, replacing the view is acceptable for this TUI style (modal focus).
		return p.ThemeModal.View(w, p.Height())
	}

	if p.Modal.IsActive() {
		return ""
	}

	p.Form.Width = safeWidth

	return p.Form.Viewport(safeWidth, p.Height())
}

func (p *ConfigPageAppearance) Focus() tea.Cmd { p.BaseViewlet.Focus(); return p.Form.Focus() }
func (p *ConfigPageAppearance) Blur()          { p.BaseViewlet.Blur(); p.Form.Blur() }

func (p *ConfigPageAppearance) IsModalActive() bool {
	return (p.Modal != nil && p.Modal.IsActive()) || (p.ThemeModal != nil && p.ThemeModal.IsActive())
}

func (p *ConfigPageAppearance) HasActiveTextInput() bool {
	return p.Form != nil && p.Form.HasActiveTextInput()
}

func (p *ConfigPageAppearance) HasActiveFocus() bool {
	return p.Form != nil && p.Form.HasActiveFocus()
}

func (p *ConfigPageAppearance) Focusable() bool {
	if p.Form == nil {
		return false
	}
	return p.Form.Focusable()
}

func (p *ConfigPageAppearance) Shortcuts() []fv.Shortcut {
	if overlay := p.Parent.activeOverlay(); overlay != nil {
		return overlay.Shortcuts()
	}
	return []fv.Shortcut{
		{Key: "tab", Label: "glide"},
		{Key: "enter", Label: "engage"},
		{Key: "esc", Label: "back"},
	}
}

func (p *ConfigPageAppearance) HandleMouse(x int, y int, msg tea.MouseMsg) (fv.Viewlet, tea.Cmd, bool) {
	if p.IsModalActive() {
		return p, nil, true
	}

	cmd, handled := p.Form.HandleMouse(x, y, msg)
	return p, cmd, handled
}

// --- Uninstall Page (Danger Zone) ---

type ConfigPageUninstall struct {
	fv.BaseViewlet
	Parent *Config

	Form            *widget.Form
	Header          *widget.Card
	WarningText     *widget.Card
	UninstallButton *widget.Button
	Modal           *widget.Modal
}

func NewConfigPageUninstall(parent *Config) *ConfigPageUninstall {
	p := &ConfigPageUninstall{Parent: parent}
	p.Header = widget.NewCard(theme.IconSelfDestruct, "Self Destruct", "Uninstall Nido and delete all templates.")

	p.Modal = widget.NewModal(
		"SELF DESTRUCT",
		"Are you absolutely sure?\nThis will permanently delete all data, templates, images, and the application itself.",
		func() tea.Cmd {
			return func() tea.Msg { return ops.RequestUninstallMsg{} }
		},
		nil,
	)
	p.Modal.SetWidth(60)
	// Make the modal scary
	p.Modal.SetLevel(widget.ModalLevelDanger)

	p.UninstallButton = widget.NewSubmitButton("", "UNINSTALL NIDO", func() tea.Cmd {
		p.Modal.Show()
		return nil
	})
	p.UninstallButton.SetColor(theme.Current().Palette.Error)
	p.UninstallButton.BorderColor = theme.Current().Palette.Error

	p.Form = widget.NewForm(p.Header, p.UninstallButton)
	p.Form.Spacing = 0
	return p
}

func (p *ConfigPageUninstall) Init() tea.Cmd { return nil }

func (p *ConfigPageUninstall) Update(msg tea.Msg) (fv.Viewlet, tea.Cmd) {
	if p.Modal != nil && p.Modal.IsActive() {
		newModal, cmd := p.Modal.Update(msg)
		p.Modal = newModal
		return p, cmd
	}

	if !p.Focused() {
		return p, nil
	}

	newForm, cmd := p.Form.Update(msg)
	p.Form = newForm
	return p, cmd
}

func (p *ConfigPageUninstall) View() string {
	w := p.Width()
	safeWidth := layout.FormWidth(w, theme.Inset.TotalContent)
	p.Form.Width = safeWidth

	if p.Modal.IsActive() {
		return ""
	}

	return p.Form.Viewport(safeWidth, p.Height())
}

func (p *ConfigPageUninstall) Focus() tea.Cmd { p.BaseViewlet.Focus(); return p.Form.Focus() }
func (p *ConfigPageUninstall) Blur()          { p.BaseViewlet.Blur(); p.Form.Blur() }
func (p *ConfigPageUninstall) HasActiveTextInput() bool {
	return p.Form != nil && p.Form.HasActiveTextInput()
}

func (p *ConfigPageUninstall) HasActiveFocus() bool {
	return p.Form != nil && p.Form.HasActiveFocus()
}

func (p *ConfigPageUninstall) Focusable() bool {
	if p.Form == nil {
		return false
	}
	return p.Form.Focusable()
}

func (p *ConfigPageUninstall) IsModalActive() bool {
	return p.Modal != nil && p.Modal.IsActive()
}

func (p *ConfigPageUninstall) HandleMouse(x, y int, msg tea.MouseMsg) (fv.Viewlet, tea.Cmd, bool) {
	if p.Modal != nil && p.Modal.IsActive() {
		cmd, handled := p.Modal.HandleMouse(x, y, msg)
		return p, cmd, handled
	}
	cmd, handled := p.Form.HandleMouse(x, y, msg)
	return p, cmd, handled
}

func (p *ConfigPageUninstall) Shortcuts() []fv.Shortcut {
	if overlay := p.Parent.activeOverlay(); overlay != nil {
		return overlay.Shortcuts()
	}
	return []fv.Shortcut{
		{Key: "enter", Label: "initiate"},
		{Key: "esc", Label: "back"},
	}
}
