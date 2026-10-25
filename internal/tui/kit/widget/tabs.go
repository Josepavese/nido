package widget

import (
	"github.com/Josepavese/nido/v5/internal/tui/kit/layout"
	"github.com/Josepavese/nido/v5/internal/tui/kit/theme"
	"github.com/charmbracelet/x/ansi"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// TabsRenderOptions configures the rendering of the tabs.
type TabsRenderOptions struct {
	Routes      []string
	ActiveIndex int
	Width       int
	Height      int

	// Colors
	HighlightColor lipgloss.TerminalColor // For active text
	InactiveColor  lipgloss.TerminalColor // For inactive text
	BorderColor    lipgloss.TerminalColor // For separators/lines (SurfaceHighlight)
}

// RenderTabs renders a stable, contiguous tab bar with rounded "bubble" tabs.
func renderBubbleTabs(opts TabsRenderOptions) string {
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
		Width(tabPrefixWidth).
		Height(2).
		Render("")
	renderedTabs = append(renderedTabs, prefix)

	for i, title := range opts.Routes {
		isActive := i == opts.ActiveIndex
		text := strings.ToUpper(title)

		if isActive {
			renderedTabs = append(renderedTabs, activeStyle.Width(max(theme.Width.TabMin, lipgloss.Width(text)+tabPadding*2)).Render(text))
		} else {
			renderedTabs = append(renderedTabs, inactiveStyle.Width(max(theme.Width.TabMin, lipgloss.Width(text)+tabPadding*2)).Render(text))
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
	return lipgloss.Place(opts.Width, layout.HeaderHeight, lipgloss.Left, lipgloss.Bottom, joined)
}

const (
	tabPrefixWidth = 2
	tabPadding     = 1
	tabBorderWidth = 2
)

// TabBar is one render and its hit geometry. Mouse and pixels share one owner.
type TabBar struct {
	Content string
	Hits    []layout.Rect
}

func RenderTabs(opts TabsRenderOptions) TabBar {
	bar := TabBar{Hits: make([]layout.Rect, len(opts.Routes))}
	if opts.Width <= 0 || opts.Height <= 0 || len(opts.Routes) == 0 {
		return bar
	}
	bubbleW := tabPrefixWidth
	for _, title := range opts.Routes {
		bubbleW += max(theme.Width.TabMin, lipgloss.Width(strings.ToUpper(title))+tabPadding*2) + tabBorderWidth
	}
	if opts.Height >= layout.HeaderHeight && bubbleW <= opts.Width {
		bar.Content = renderBubbleTabs(opts)
		x := tabPrefixWidth
		for i, title := range opts.Routes {
			w := max(theme.Width.TabMin, lipgloss.Width(strings.ToUpper(title))+tabPadding*2) + tabBorderWidth
			bar.Hits[i] = layout.NewRect(x, opts.Height-layout.HeaderHeight+1, w, layout.HeaderHeight-1)
			x += w
		}
		return bar
	}
	// In a small terminal keep every route visible, shortening titles evenly.
	perTab := opts.Width / len(opts.Routes)
	var parts []string
	x := 0
	for i, title := range opts.Routes {
		title = ansi.Truncate(strings.ToUpper(title), max(0, perTab-tabPadding*2), "…")
		style := lipgloss.NewStyle().Foreground(opts.InactiveColor).Padding(0, tabPadding)
		if i == opts.ActiveIndex {
			style = style.Foreground(theme.Current().Palette.Background).Background(opts.HighlightColor).Bold(true)
		}
		part := style.Render(title)
		w := lipgloss.Width(part)
		bar.Hits[i] = layout.NewRect(x, 0, w, 1)
		x += w
		parts = append(parts, part)
	}
	bar.Content = layout.Fit(opts.Width, opts.Height, strings.Join(parts, ""))
	return bar
}
