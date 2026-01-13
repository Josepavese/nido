// Package gui provides shell rendering utilities for the Nido TUI.
// This file contains the header, tabbar, subheader, and footer rendering
// which are independent of specific viewlet content.
package gui

import (
	"fmt"
	"strings"

	"github.com/Josepavese/nido/internal/tui/components"
	"github.com/Josepavese/nido/internal/tui/layout"
	"github.com/Josepavese/nido/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// ShellConfig holds configuration for the shell rendering.
type ShellConfig struct {
	Width     int
	Height    int
	ActiveTab tab
}

// TabLabels defines the tab names and their shortcut keys.
var TabLabels = []string{"1 FLEET", "2 HATCHERY", "3 LOGS", "4 CONFIG", "5 HELP"}

// RenderTabs renders the main tab bar with the active tab highlighted.
func RenderTabs(width int, activeTab tab) string {
	t := theme.Current()

	tabStyle := lipgloss.NewStyle().
		Foreground(t.Palette.TextDim).
		Padding(0, theme.Space.SM)

	activeTabStyle := lipgloss.NewStyle().
		Foreground(t.Palette.Accent).
		Padding(0, theme.Space.SM).
		Bold(true).
		Underline(true)

	availableWidth := width - 6 // Space for [X] button
	tabWidth := availableWidth / len(TabLabels)

	var tabs []string
	for i, label := range TabLabels {
		style := tabStyle.Width(tabWidth).Align(lipgloss.Center)
		if i == int(activeTab) {
			style = activeTabStyle.Width(tabWidth).Align(lipgloss.Center)
		}
		tabs = append(tabs, style.Render(label))
	}

	row := layout.HStack(0, tabs...)
	rowWidth := lipgloss.Width(row)

	// Exit button
	exitBtn := lipgloss.NewStyle().
		Foreground(t.Palette.Error).
		Bold(true).
		Render("[X]")
	exitWidth := lipgloss.Width(exitBtn)

	// Spacer to push exit button to the right
	gap := width - rowWidth - exitWidth - 1
	if gap < 0 {
		gap = 0
	}
	spacer := strings.Repeat(" ", gap)

	return layout.HStack(0, row, spacer, exitBtn)
}

// SubHeaderContent returns the context title and navigation hint for a tab.
func SubHeaderContent(activeTab tab) (context, nav string) {
	arrows := "Use ←/→ arrows to navigate tabs."

	switch activeTab {
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

	return context, nav
}

// RenderSubHeader renders the context bar below the tabs.
func RenderSubHeader(activeTab tab) string {
	t := theme.Current()

	contextStyle := lipgloss.NewStyle().
		Foreground(t.Palette.AccentStrong).
		Bold(true)

	navStyle := lipgloss.NewStyle().
		Foreground(t.Palette.TextDim).
		Italic(true)

	context, nav := SubHeaderContent(activeTab)

	return layout.HStack(theme.Space.SM,
		contextStyle.Render(context),
		navStyle.Render(nav),
	)
}

// FooterState holds the current footer state for rendering.
type FooterState struct {
	Width            int
	Loading          bool
	Downloading      bool
	DownloadProgress float64
	Operation        string
	SpinnerView      string
	ProgressView     string
}

// RenderFooter renders the status bar footer.
func RenderFooter(state FooterState) string {
	t := theme.Current()

	// Loading state
	if state.Downloading {
		status := fmt.Sprintf(" %s Downloading... %s", state.SpinnerView, state.ProgressView)
		return lipgloss.NewStyle().
			Height(1).
			Foreground(t.Palette.TextDim).
			Padding(0, theme.Space.XS).
			Width(state.Width).
			Render(status)
	}

	if state.Loading {
		status := fmt.Sprintf("%s EXECUTING %s... ", state.SpinnerView, strings.ToUpper(state.Operation))
		return lipgloss.NewStyle().
			Height(1).
			Foreground(t.Palette.TextDim).
			Padding(0, theme.Space.XS).
			Width(state.Width).
			Render(status)
	}

	// Normal state with status bar
	sb := components.NewStatusBar(state.Width)
	sb.SetItems([]components.StatusBarItem{
		{Key: "🟢", Label: "NOMINAL"},
	})

	link := fmt.Sprintf("\x1b]8;;https://github.com/Josepavese\x1b\\%s\x1b]8;;\x1b\\", "github.com/Josepavese")
	sb.SetStatus(fmt.Sprintf("🏠 There is no place like 127.0.0.1 | %s", link))

	return sb.View()
}

// RenderShell renders the complete shell structure including the body.
func RenderShell(cfg ShellConfig, footerState FooterState, body string) string {
	t := theme.Current()

	headerStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(t.Palette.SurfaceSubtle)

	subHeaderStyle := lipgloss.NewStyle().
		Height(1).
		Padding(0, theme.Space.XS).
		PaddingBottom(theme.Space.XS)

	header := headerStyle.Width(cfg.Width).Render(RenderTabs(cfg.Width, cfg.ActiveTab))
	subHeader := subHeaderStyle.Width(cfg.Width - 2).Render(RenderSubHeader(cfg.ActiveTab))
	footer := RenderFooter(footerState)

	// Ensure consistent vertical layout
	return lipgloss.JoinVertical(lipgloss.Left, header, subHeader, body, footer)
}
