package widget

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// TabsRenderOptions configures the rendering of the tabs.
type TabsRenderOptions struct {
	Routes      []string
	ActiveIndex int
	Width       int

	// Colors
	HighlightColor lipgloss.TerminalColor // For active text
	InactiveColor  lipgloss.TerminalColor // For inactive text
	BorderColor    lipgloss.TerminalColor // For separators/lines (SurfaceHighlight)
}

// RenderTabs renders a stable, contiguous tab bar with rounded "bubble" tabs.
func RenderTabs(opts TabsRenderOptions) string {
	if len(opts.Routes) == 0 {
		return ""
	}

	// Active tab style: Open bottom (3 lines high)
	activeStyle := lipgloss.NewStyle().
		Border(lipgloss.Border{
			Top:         "─",
			Bottom:      " ",
			Left:        "│",
			Right:       "│",
			TopLeft:     "╭",
			TopRight:    "╮",
			BottomLeft:  "┘",
			BottomRight: "└",
		}, true).
		BorderForeground(opts.BorderColor).
		Foreground(opts.HighlightColor).
		Padding(0, 1).
		Bold(true)

	// Inactive tab style: Closed bottom (3 lines high to maintain baseline)
	inactiveStyle := lipgloss.NewStyle().
		Border(lipgloss.Border{
			Top:         "─",
			Bottom:      "─",
			Left:        "│",
			Right:       "│",
			TopLeft:     "╭",
			TopRight:    "╮",
			BottomLeft:  "┴", // Connects to the baseline
			BottomRight: "┴",
		}, true).
		BorderForeground(opts.BorderColor).
		Foreground(opts.InactiveColor).
		Padding(0, 1)

	var renderedTabs []string

	// Stable Prefix: Always 2 lines of baseline to prevent shifting
	prefix := lipgloss.NewStyle().
		Border(lipgloss.Border{Bottom: "─"}, false, false, true, false).
		BorderForeground(opts.BorderColor).
		Width(2).
		Height(2).
		Render("")
	renderedTabs = append(renderedTabs, prefix)

	for i, title := range opts.Routes {
		isActive := i == opts.ActiveIndex
		text := strings.ToUpper(title)

		if isActive {
			renderedTabs = append(renderedTabs, activeStyle.Render(text))
		} else {
			renderedTabs = append(renderedTabs, inactiveStyle.Render(text))
		}
	}

	// Join pieces horizontally, aligned to the bottom
	tabsRow := lipgloss.JoinHorizontal(lipgloss.Bottom, renderedTabs...)

	// Remaining width gap line
	tabsWidth := lipgloss.Width(tabsRow)
	gapWidth := opts.Width - tabsWidth
	if gapWidth < 0 {
		gapWidth = 0
	}

	gap := ""
	if gapWidth > 0 {
		gap = lipgloss.NewStyle().
			Border(lipgloss.Border{Bottom: "─"}, false, false, true, false).
			BorderForeground(opts.BorderColor).
			Width(gapWidth).
			Height(2).
			Render("")
	}

	// Force bottom alignment to prevent top clipping
	// This pushes the 3-line tabs down to lines 1-3, leaving line 0 empty
	joined := lipgloss.JoinHorizontal(lipgloss.Bottom, tabsRow, gap)
	return lipgloss.Place(opts.Width, 4, lipgloss.Left, lipgloss.Bottom, joined)
}
