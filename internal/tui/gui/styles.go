package gui

import "github.com/charmbracelet/lipgloss"

type palette struct {
	Accent        lipgloss.Color
	AccentStrong  lipgloss.Color
	Surface       lipgloss.Color
	SurfaceSubtle lipgloss.Color
	Bg            lipgloss.Color
	Text          lipgloss.Color
	TextDim       lipgloss.Color
	Success       lipgloss.Color
	Warning       lipgloss.Color
	Error         lipgloss.Color
}

var colors = palette{
	Accent:        lipgloss.Color("#76C7FF"),
	AccentStrong:  lipgloss.Color("#3BA3E6"),
	Surface:       lipgloss.Color("#151A21"),
	SurfaceSubtle: lipgloss.Color("#1A2028"),
	Bg:            lipgloss.Color("#0F1216"),
	Text:          lipgloss.Color("#E6EBF2"),
	TextDim:       lipgloss.Color("#707D8C"),
	Success:       lipgloss.Color("#3DDC84"),
	Warning:       lipgloss.Color("#F5C26B"),
	Error:         lipgloss.Color("#F06D79"),
}

var (
	// Navigation & Tabs
	tabStyle = lipgloss.NewStyle().
			Foreground(colors.TextDim).
			Padding(0, 2)

	activeTabStyle = lipgloss.NewStyle().
			Foreground(colors.Accent).
			Padding(0, 2).
			Bold(true).
			Underline(true)

	// Sub-Header
	subHeaderStyle = lipgloss.NewStyle().
			Height(1).
			Padding(0, 1).
			PaddingBottom(1)

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
			Width(30).
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(colors.SurfaceSubtle)

	mainContentStyle = lipgloss.NewStyle().
				Padding(0, 2)

	headerStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(colors.SurfaceSubtle)

	footerStyle = lipgloss.NewStyle().
			Height(1).
			Foreground(colors.TextDim).
			Padding(0, 1) // Keep horiz padding

	// Sidebar items
	sidebarItemStyle = lipgloss.NewStyle().
				Foreground(colors.Text).
				Padding(0, 1)

	sidebarItemSelectedStyle = lipgloss.NewStyle().
					Foreground(colors.Accent).
					Padding(0, 1).
					Bold(true)

	// Cards & Sections
	cardStyle = lipgloss.NewStyle().
			Padding(1, 2) // Maintain padding for spacing

	titleStyle = lipgloss.NewStyle().
			Foreground(colors.AccentStrong).
			Bold(true).
			MarginBottom(0) // Remove bottom margin

	// Typography & Badges
	labelStyle = lipgloss.NewStyle().
			Foreground(colors.TextDim).
			Width(12)

	focusedLabelStyle = lipgloss.NewStyle().
				Foreground(colors.Accent).
				Bold(true).
				Width(12)

	valueStyle = lipgloss.NewStyle().
			Foreground(colors.Text)

	badgeStyle = lipgloss.NewStyle().
			Padding(0, 1).
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
			Padding(0, 1).
			MarginRight(1)

	redButtonStyle = buttonStyle.Copy().
			Foreground(colors.Error)

	containerStyle = lipgloss.NewStyle().
			Foreground(colors.Text)

	hatchButtonStyle = lipgloss.NewStyle().
				Foreground(colors.Bg).
				Background(colors.TextDim).
				Padding(0, 3).
				MarginTop(1)

	hatchButtonActiveStyle = hatchButtonStyle.Copy().
				Background(colors.AccentStrong).
				Bold(true)
)
