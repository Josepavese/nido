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
	Strings   UIStrings
}

const (
	headerReserve = 6 // space reserved for exit and spacer
)

// RenderTabs renders the main tab bar with the active tab highlighted.
func RenderTabs(width int, activeTab tab, labels []string) string {
	t := theme.Current()

	tabStyle := lipgloss.NewStyle().
		Foreground(t.Palette.TextDim).
		MaxHeight(1)
		// Padding added dynamically based on width

	activeTabStyle := lipgloss.NewStyle().
		Foreground(t.Palette.Accent).
		Padding(0, theme.Gap(theme.Space.SM)).
		Bold(true).
		Underline(true).
		MaxHeight(1)

	// Calculate tab width dynamically
	availableWidth := width - headerReserve
	tabCount := len(labels)
	if tabCount == 0 {
		return ""
	}
	tabWidth := availableWidth / tabCount

	// Only enforce TabMin if we have plenty of space,
	// otherwise shrink to fit (prevent wrapping).
	// Logic: If shrinking below min is better than wrapping.
	if tabWidth < theme.Width.TabMin {
		// Checks if we really need to clamp?
		// Actually, standard behavior should be to FIT.
		// If tabWidth < 4, it's unusable, but wrapping is worse.
		// Let's allow shrinking down to 4 chars.
		if tabWidth < 4 {
			tabWidth = 4
		}
	}

	var tabs []string
	for i, label := range labels {
		style := tabStyle.Width(tabWidth).Align(lipgloss.Center)

		// Only add padding if we have enough room (e.g. width > 12)
		// Otherwise text needs every cell.
		if tabWidth > 12 {
			style = style.Padding(0, theme.Gap(theme.Space.SM))
		}

		if i == int(activeTab) {
			style = activeTabStyle.Width(tabWidth).Align(lipgloss.Center)
			if tabWidth > 12 {
				style = style.Padding(0, theme.Gap(theme.Space.SM))
			}
		}
		// Prevent wrapping on spaces by using NBSP
		safeLabel := strings.ReplaceAll(label, " ", "\u00A0")
		tabs = append(tabs, style.Render(safeLabel))
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
		context = "THE NEST"
		nav = "Select a bird to inspect. " + arrows
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
	FooterLink       string
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
			Padding(0, theme.Gap(theme.Space.XS)).
			Width(state.Width).
			Render(status)
	}

	if state.Loading {
		status := fmt.Sprintf("%s EXECUTING %s... ", state.SpinnerView, strings.ToUpper(state.Operation))
		return lipgloss.NewStyle().
			Height(1).
			Foreground(t.Palette.TextDim).
			Padding(0, theme.Gap(theme.Space.XS)).
			Width(state.Width).
			Render(status)
	}

	// Normal state with status bar
	sb := components.NewStatusBar(state.Width)
	sb.SetItems([]components.StatusBarItem{
		{Key: "🟢", Label: "NOMINAL"},
	})

	link := fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", state.FooterLink, state.FooterLink)
	sb.SetStatus(fmt.Sprintf("🏠 There is no place like 127.0.0.1 | %s", link))

	return sb.View()
}

// RenderShell renders header, subheader, and footer and returns their string
// forms along with the total shell height, accounting for 1-line gaps between
// sections (header/subheader/body/footer).
func RenderShell(cfg ShellConfig, footerState FooterState) (header, subHeader, footer string, totalHeight int) {
	t := theme.Current()

	headerStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(t.Palette.SurfaceSubtle)

	subHeaderStyle := lipgloss.NewStyle().
		Padding(0, theme.Gap(theme.Space.XS)).
		PaddingBottom(theme.Gap(theme.Space.XS))

	header = headerStyle.Width(cfg.Width).Render(RenderTabs(cfg.Width, cfg.ActiveTab, cfg.Strings.TabLabels))
	subHeader = subHeaderStyle.Width(cfg.Width - 2).Render(RenderSubHeader(cfg.ActiveTab))
	footer = RenderFooter(footerState)

	headerH := lipgloss.Height(header)
	subHeaderH := lipgloss.Height(subHeader)
	footerH := lipgloss.Height(footer)

	// When stacked with a single blank line gap between sections
	totalHeight = headerH + subHeaderH + footerH + 3

	return header, subHeader, footer, totalHeight
}
