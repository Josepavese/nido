package main

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

func main() {
	icons := []string{"🪶", "🧬", "🪺", "🥚", "🪵", "🦅", "📦", "a"}

	fmt.Println("Analyzing Icon Widths and Rendering Strategies")
	fmt.Println("==============================================")

	header := fmt.Sprintf("%-5s | %-5s | %-5s | %-20s", "Icon", "RuneC", "GlsW", "Visual Test (Grid)")
	fmt.Println(header)
	fmt.Println(strings.Repeat("-", len(header)+10))

	for _, icon := range icons {
		runeCount := utf8.RuneCountInString(icon)
		glsWidth := lipgloss.Width(icon)

		// Visual test in a grid
		// We want to see if it aligns with the pipe character
		visual := fmt.Sprintf("[%s] |", icon)

		fmt.Printf("%-5s | %-5d | %-5d | %s\n", icon, runeCount, glsWidth, visual)
	}

	fmt.Println("\nTesting Stabilization Strategies (Target Width: 3)")
	fmt.Println("==================================================")

	testStrategy := func(name string, fn func(string) string) {
		fmt.Printf("\nStrategy: %s\n", name)
		for _, icon := range icons {
			processed := fn(icon)
			width := lipgloss.Width(processed)
			// Render with a background to see the box
			style := lipgloss.NewStyle().Background(lipgloss.Color("63")).Foreground(lipgloss.Color("255"))
			rendered := style.Render(processed)

			// Check if "Next Text" starts at the same visual column
			line := fmt.Sprintf("%sNext Text", rendered)
			fmt.Printf("Icon: %s | Width: %d | Output: '%s'\n", icon, width, line)
		}
	}

	// Strategy A: Lipgloss Width(3)
	testStrategy("Lipgloss Width(3)", func(s string) string {
		return lipgloss.NewStyle().Width(3).Render(s)
	})

	// Strategy B: Left Align + Right Padding manually
	testStrategy("Manual Pad Right", func(s string) string {
		w := lipgloss.Width(s)
		gap := 3 - w
		if gap < 1 {
			gap = 1 // Always ensure at least 1 space?
		}
		return s + strings.Repeat(" ", gap)
	})

	// Strategy C: Center in 4 chars?
	testStrategy("Center in 4", func(s string) string {
		return lipgloss.NewStyle().Width(4).Align(lipgloss.Center).Render(s)
	})
}
