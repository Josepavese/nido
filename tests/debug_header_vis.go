package main

import (
	"fmt"

	"github.com/Josepavese/nido/internal/tui/kit/layout"
	"github.com/charmbracelet/lipgloss"
)

func main() {
	width := 80
	height := 24

	fmt.Printf("Calculating Grid for W=%d H=%d, Stack=0\n", width, height)

	// Calc with 0 stack
	g := layout.CalculateGrid(width, height, 0)

	fmt.Printf("Header: %+v\n", g.Header)
	fmt.Printf("Stack:  %+v\n", g.ActionStack)
	fmt.Printf("Body:   %+v\n", g.Body)

	// Mock Header Render
	startTabs := "FLEET  HATCHERY"
	headerStyle := lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, false, true, false)
	headerContent := headerStyle.Render(startTabs)

	fmt.Printf("Header Content Height: %d\n", lipgloss.Height(headerContent))

	// Place
	placed := place(g.Header, headerContent)
	fmt.Printf("Placed Header len: %d\n", len(placed))
	fmt.Printf("Placed Header Height: %d\n", lipgloss.Height(placed))

	// Mock Join
	joined := lipgloss.JoinVertical(lipgloss.Left, placed, "BODY")
	fmt.Printf("Joined Height: %d\n", lipgloss.Height(joined))

	fmt.Println("--- Visual Check ---")
	fmt.Println(joined)
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
