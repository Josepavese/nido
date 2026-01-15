package main

import (
	"fmt"
	"os"

	"github.com/Josepavese/nido/internal/tui/kit/theme"
	"github.com/charmbracelet/lipgloss"
)

func main() {
	themes := []string{"dark", "light", "pink", "high-contrast", "matrix"}

	fmt.Println("🦅 Nido Theme Preview")
	fmt.Println("======================")

	for _, tName := range themes {
		os.Setenv("NIDO_THEME", tName)
		t := theme.Current()

		titleStyle := t.Styles.Title.Copy().
			Padding(0, 1).
			Width(20).
			Align(lipgloss.Center)

		fmt.Printf("\n--- Theme: %s ---\n", tName)
		fmt.Println(titleStyle.Render(tName))

		// Preview some semantic colors
		previewColor("Accent  ", t.Palette.Accent)
		previewColor("Success ", t.Palette.Success)
		previewColor("Error   ", t.Palette.Error)
		previewColor("Text    ", t.Palette.Text)
		previewColor("TextDim ", t.Palette.TextDim)
		fmt.Println()
	}
}

func previewColor(name string, color lipgloss.AdaptiveColor) {
	style := lipgloss.NewStyle().
		Foreground(color).
		Bold(true)

	block := lipgloss.NewStyle().
		Background(color).
		Foreground(color). // Hidden text
		Render("  ")

	fmt.Printf("%s %s %s\n", name, block, style.Render("THE QUICK BROWN FOX"))
}
