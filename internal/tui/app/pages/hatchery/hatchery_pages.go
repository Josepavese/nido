package hatchery

import (
	"strings"

	"github.com/Josepavese/nido/internal/tui/app/ops"
	"github.com/Josepavese/nido/internal/tui/kit/theme"
	fv "github.com/Josepavese/nido/internal/tui/kit/view"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HatcheryPageSpawn handles the VM spawning form.
type HatcheryPageSpawn struct {
	fv.BaseViewlet
	Parent *Hatchery
}

func (p *HatcheryPageSpawn) Init() tea.Cmd { return nil }
func (p *HatcheryPageSpawn) Update(msg tea.Msg) (fv.Viewlet, tea.Cmd) {
	h := p.Parent
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down", "enter":
			if msg.String() == "enter" && h.focusIndex == 3 {
				// SUBMIT
				name := h.inputs[0].Value()
				if name == "" {
					return p, func() tea.Msg { return fv.LogMsg{Text: "Name is required!"} }
				}
				if h.SpawnSource == "" {
					return p, func() tea.Msg { return fv.LogMsg{Text: "Source is required!"} }
				}
				// Reset form
				h.inputs[0].SetValue("")
				h.focusIndex = 0

				return p, tea.Batch(
					func() tea.Msg {
						return ops.RequestSpawnMsg{
							Name:     name,
							Source:   h.SpawnSource,
							UserData: "",
							GUI:      h.guiEnabled,
						}
					},
					func() tea.Msg { return fv.StatusMsg{Loading: true, Operation: "spawn"} },
				)
			}

			// Handle Enter on Source/Toggle
			if msg.String() == "enter" {
				if h.focusIndex == 1 { // Source
					h.isSelecting = true
					return p, nil
				}
				if h.focusIndex == 2 { // GUI Toggle
					h.guiEnabled = !h.guiEnabled
					return p, nil
				}
			}

			h.focusIndex++
			if h.focusIndex > 3 {
				h.focusIndex = 0
			}
			// Skip over "Source" if it's just a label? No it's interactive.

		case "shift+tab", "up":
			h.focusIndex--
			if h.focusIndex < 0 {
				h.focusIndex = 3
			}
		}
	}

	// Handle text input only if focused
	if h.focusIndex == 0 {
		h.inputs[0], cmd = h.inputs[0].Update(msg)
	}

	return p, cmd
}
func (p *HatcheryPageSpawn) View() string {
	h := p.Parent
	t := theme.Current()

	titleStyle := lipgloss.NewStyle().Foreground(t.Palette.AccentStrong).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(t.Palette.TextDim).Width(10)
	activeLabelStyle := lipgloss.NewStyle().Foreground(t.Palette.Accent).Width(10).Bold(true)

	header := titleStyle.Render("🐣 SPAWN VM")

	var form strings.Builder
	// Name
	lStyle := labelStyle
	if h.focusIndex == 0 {
		lStyle = activeLabelStyle
	}
	form.WriteString(lipgloss.JoinHorizontal(lipgloss.Left, lStyle.Render("Name"), h.inputs[0].View()) + "\n\n")

	// Source
	lStyle = labelStyle
	valStyle := lipgloss.NewStyle().Foreground(t.Palette.TextDim)
	src := h.SpawnSource
	if src == "" {
		src = "Select..."
	}
	if h.focusIndex == 1 {
		lStyle = activeLabelStyle
		valStyle = valStyle.Foreground(t.Palette.Accent)
	}
	form.WriteString(lipgloss.JoinHorizontal(lipgloss.Left, lStyle.Render("Source"), valStyle.Render(src)) + "\n\n")

	// Options
	lStyle = labelStyle
	if h.focusIndex == 2 {
		lStyle = activeLabelStyle
	}
	toggle := h.renderToggle("GUI", h.guiEnabled)
	form.WriteString(lipgloss.JoinHorizontal(lipgloss.Left, lStyle.Render("Options"), toggle) + "\n\n")

	// Submit
	btnText := "[ START INCUBATION ]"
	btnStyle := lipgloss.NewStyle().Foreground(t.Palette.TextDim)
	if h.focusIndex == 3 {
		btnStyle = lipgloss.NewStyle().Foreground(t.Palette.AccentStrong).Bold(true)
	}
	form.WriteString(btnStyle.Render(btnText))

	return lipgloss.NewStyle().Width(p.Width()).Height(p.Height()).Render(
		lipgloss.JoinVertical(lipgloss.Left, header, "\n", form.String()),
	)
}

// HatcheryPageTemplate handles the template creation form.
type HatcheryPageTemplate struct {
	fv.BaseViewlet
	Parent *Hatchery
}

func (p *HatcheryPageTemplate) Init() tea.Cmd { return nil }
func (p *HatcheryPageTemplate) Update(msg tea.Msg) (fv.Viewlet, tea.Cmd) {
	h := p.Parent
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down", "enter":
			if msg.String() == "enter" && h.focusIndex == 2 { // Submit at index 2
				// SUBMIT
				name := h.inputs[0].Value()
				if name == "" {
					return p, func() tea.Msg { return fv.LogMsg{Text: "Name is required!"} }
				}
				if h.TemplateSource == "" {
					return p, func() tea.Msg { return fv.LogMsg{Text: "Source VM is required!"} }
				}
				// Reset
				h.inputs[0].SetValue("")
				h.focusIndex = 0

				return p, tea.Batch(
					func() tea.Msg {
						return ops.RequestCreateTemplateMsg{
							Name: name,
							Source: h.TemplateSource,
						}
					},
					func() tea.Msg { return fv.StatusMsg{Loading: true, Operation: "create-template"} },
				)
			}

			if msg.String() == "enter" {
				if h.focusIndex == 1 { // Source
					h.isSelecting = true
					return p, nil
				}
			}

			h.focusIndex++
			if h.focusIndex > 2 {
				h.focusIndex = 0
			}

		case "shift+tab", "up":
			h.focusIndex--
			if h.focusIndex < 0 {
				h.focusIndex = 2
			}
		}
	}

	if h.focusIndex == 0 {
		h.inputs[0], cmd = h.inputs[0].Update(msg)
	}
	return p, cmd
}
func (p *HatcheryPageTemplate) View() string {
	h := p.Parent
	t := theme.Current()

	titleStyle := lipgloss.NewStyle().Foreground(t.Palette.AccentStrong).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(t.Palette.TextDim).Width(10)
	activeLabelStyle := lipgloss.NewStyle().Foreground(t.Palette.Accent).Width(10).Bold(true)

	header := titleStyle.Render("📦 CREATE TEMPLATE")

	var form strings.Builder
	// Name
	lStyle := labelStyle
	if h.focusIndex == 0 {
		lStyle = activeLabelStyle
	}
	form.WriteString(lipgloss.JoinHorizontal(lipgloss.Left, lStyle.Render("Name"), h.inputs[0].View()) + "\n\n")

	// Source
	lStyle = labelStyle
	valStyle := lipgloss.NewStyle().Foreground(t.Palette.TextDim)
	src := h.TemplateSource
	if src == "" {
		src = "Select..."
	}
	if h.focusIndex == 1 {
		lStyle = activeLabelStyle
		valStyle = valStyle.Foreground(t.Palette.Accent)
	}
	form.WriteString(lipgloss.JoinHorizontal(lipgloss.Left, lStyle.Render("Source"), valStyle.Render(src)) + "\n\n")

	// Submit
	btnText := "[ FREEZE TEMPLATE ]"
	btnStyle := lipgloss.NewStyle().Foreground(t.Palette.TextDim)
	if h.focusIndex == 2 {
		btnStyle = lipgloss.NewStyle().Foreground(t.Palette.AccentStrong).Bold(true)
	}
	form.WriteString(btnStyle.Render(btnText))

	return lipgloss.NewStyle().Width(p.Width()).Height(p.Height()).Render(
		lipgloss.JoinVertical(lipgloss.Left, header, "\n", form.String()),
	)
}
