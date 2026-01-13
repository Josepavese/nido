// Package gui implements the Nido interactive TUI using Bubble Tea.
// This file defines the visual styles used throughout the interface.
//
// All colors and spacing values are derived from the theme package
// to ensure consistency and support for light/dark modes.
package gui

import (
	"github.com/Josepavese/nido/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// activeTheme is the current theme, initialized once at startup.
// This caches the theme detection result for performance.
var activeTheme = theme.Current()

// colors provides convenient access to the active palette.
// Legacy alias maintained for compatibility during migration.
var colors = activeTheme.Palette

var (
	// Navigation & Tabs
	tabStyle = lipgloss.NewStyle().
			Foreground(colors.TextDim).
			Padding(0, theme.Space.SM)

	activeTabStyle = lipgloss.NewStyle().
			Foreground(colors.Accent).
			Padding(0, theme.Space.SM).
			Bold(true).
			Underline(true)

	// Sub-Header
	subHeaderStyle = lipgloss.NewStyle().
			Height(1).
			Padding(0, theme.Space.XS).
			PaddingBottom(theme.Space.XS)

	subHeaderContextStyle = lipgloss.NewStyle().
				Foreground(colors.AccentStrong).
				Bold(true)

	subHeaderNavStyle = lipgloss.NewStyle().
				Foreground(colors.TextDim).
				Italic(true)

	// Layout Containers
	baseStyle = lipgloss.NewStyle().
			Foreground(colors.Text)

	sidebarStyle = lipgloss.NewStyle().
			Width(theme.Width.Sidebar).
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(colors.SurfaceSubtle)

	mainContentStyle = lipgloss.NewStyle().
				Padding(0, theme.Space.XS)

	headerStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(colors.SurfaceSubtle)

	footerStyle = lipgloss.NewStyle().
			Height(1).
			Foreground(colors.TextDim).
			Padding(0, theme.Space.XS)

	// Sidebar items
	sidebarItemStyle = lipgloss.NewStyle().
				Foreground(colors.Text).
				Padding(0, theme.Space.XS)

	sidebarItemSelectedStyle = lipgloss.NewStyle().
					Foreground(colors.Accent).
					Padding(0, theme.Space.XS).
					Bold(true)

	// Cards & Sections
	cardStyle = lipgloss.NewStyle().
			Padding(theme.Space.XS, theme.Space.SM)

	titleStyle = lipgloss.NewStyle().
			Foreground(colors.AccentStrong).
			Bold(true).
			MarginBottom(0)

	// Typography & Badges
	labelStyle = lipgloss.NewStyle().
			Foreground(colors.TextDim).
			Width(theme.Width.Label)

	focusedLabelStyle = lipgloss.NewStyle().
				Foreground(colors.Accent).
				Bold(true).
				Width(theme.Width.Label)

	valueStyle = lipgloss.NewStyle().
			Foreground(colors.Text)

	badgeStyle = lipgloss.NewStyle().
			Padding(0, theme.Space.XS).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(colors.TextDim)

	errorStyle = lipgloss.NewStyle().
			Foreground(colors.Error).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(colors.Success)

	warningStyle = lipgloss.NewStyle().
			Foreground(colors.Warning)

	accentStyle = lipgloss.NewStyle().
			Foreground(colors.AccentStrong).
			Bold(true)

	buttonStyle = lipgloss.NewStyle().
			Foreground(colors.Accent).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colors.SurfaceSubtle).
			Padding(0, theme.Space.XS).
			MarginRight(theme.Space.XS)

	redButtonStyle = buttonStyle.
			Foreground(colors.Error)

	containerStyle = lipgloss.NewStyle().
			Foreground(colors.Text)

	hatchButtonStyle = lipgloss.NewStyle().
				Foreground(colors.Background).
				Background(colors.TextDim).
				Width(theme.Width.Sidebar).
				Align(lipgloss.Center).
				MarginTop(theme.Space.XS)

	hatchButtonActiveStyle = hatchButtonStyle.
				Background(colors.AccentStrong).
				Bold(true)
)
