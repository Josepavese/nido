package main

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func main() {
	emojis := []string{"🪺", "🥚", "🪶", "🦅", "📦", "💾", "💿"}

	fmt.Println("Emoji Width Debugging")
	fmt.Println("--------------------")

	for _, e := range emojis {
		w := lipgloss.Width(e)
		rendered := lipgloss.NewStyle().Width(2).Render(e)
		renderedW := lipgloss.Width(rendered)

		fmt.Printf("Emoji: %s | Raw Width: %d | Style(Width 2) Width: %d | Rendered: [%s]\n", e, w, renderedW, rendered)
	}

	fmt.Println("\nSidebar Mock Rendering (Icon + Space + Name)")
	for _, e := range emojis {
		iconStyle := lipgloss.NewStyle().Width(2).Render(e)
		str := fmt.Sprintf("%s %s", iconStyle, "ubuntu")
		fmt.Printf("[%s] (Width: %d)\n", str, lipgloss.Width(str))
	}
}
