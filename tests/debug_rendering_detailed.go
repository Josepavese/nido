package main

import (
	"fmt"

	"github.com/Josepavese/nido/internal/tui/kit/layout"
	"github.com/Josepavese/nido/internal/tui/kit/shell"
	"github.com/Josepavese/nido/internal/tui/kit/theme"
	"github.com/Josepavese/nido/internal/tui/kit/widget"
	"github.com/charmbracelet/lipgloss"
)

func main() {
	// 1. Setup Theme & Styles (Partial from wiring.go)
	t := theme.Theme{
		Palette: theme.Palette{
			SurfaceSubtle: lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383B42"},
			AccentStrong:  lipgloss.AdaptiveColor{Light: "#F25D94", Dark: "#F25D94"},
			TextDim:       lipgloss.AdaptiveColor{Light: "#A7AA9F", Dark: "#868991"},
			Accent:        lipgloss.AdaptiveColor{Light: "#F25D94", Dark: "#F25D94"},
		},
	}

	headerStyle := lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, false, true, false).BorderForeground(t.Palette.SurfaceSubtle)

	sh := shell.NewShell()
	sh.Width = 80
	sh.Height = 24
	sh.Styles = shell.ShellStyles{
		Header: headerStyle,
		StatusBar: widget.StatusBarStyles{
			Label: lipgloss.NewStyle().Foreground(t.Palette.TextDim),
		},
	}

	// 2. Add Routes
	sh.AddRoute(shell.Route{Key: "FLEET", Title: "FLEET"})
	sh.AddRoute(shell.Route{Key: "HATCHERY", Title: "HATCHERY"})
	sh.SwitchTo("FLEET")

	// 3. Resize (Calc Grid)
	// Manual Grid Calc
	grid := layout.CalculateGrid(80, 24, 0)

	fmt.Printf("Grid Header: %+v\n", grid.Header)

	// 4. Manual Render Header
	// We allow access to private renderHeader via reflection if needed, but easier to replicate logic here
	// Logic from shell.go:
	var tabs []string
	activeKey := "FLEET"
	for _, r := range []string{"FLEET", "HATCHERY"} { // Simplified iteration
		style := lipgloss.NewStyle().Padding(0, 1).Foreground(t.Palette.TextDim)
		if r == activeKey {
			style = style.Bold(true).
				Foreground(t.Palette.AccentStrong).
				Reverse(true)
		}
		tabs = append(tabs, style.Render(r))
	}
	startTabs := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	renderedHeader := headerStyle.Render(startTabs)

	fmt.Printf("--- RAW HEADER CONTENT ---\n%s\n--------------------------\n", renderedHeader)
	fmt.Printf("Dimensions: w=%d, h=%d\n", lipgloss.Width(renderedHeader), lipgloss.Height(renderedHeader))

	// 5. Place
	placed := place(grid.Header, renderedHeader)
	fmt.Printf("--- PLACED HEADER ---\n%s\n---------------------\n", placed)

}

func place(r layout.Rect, content string) string {
	if r.Height == 0 || r.Width == 0 {
		return ""
	}

	// Use Lipgloss to size it exactly.
	style := lipgloss.NewStyle().
		Width(r.Width).
		Height(r.Height).
		MaxHeight(r.Height).
		MaxWidth(r.Width)

	return style.Render(content)
}
