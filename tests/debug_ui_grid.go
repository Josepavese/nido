package main

import (
	"fmt"
	"strings"

	"github.com/Josepavese/nido/internal/tui/kit/layout"
	"github.com/Josepavese/nido/internal/tui/kit/theme"
	"github.com/Josepavese/nido/internal/tui/kit/widget"
	"github.com/charmbracelet/lipgloss"
)

func main() {
	// Enable Theme
	_ = theme.Current()

	fmt.Println("=== UI Grid & Alignment Verification ===")

	// 1. Test RenderIcon Math
	testIcon := func(name, icon string) {
		rendered := theme.RenderIcon(icon)
		width := lipgloss.Width(rendered)
		fmt.Printf("Icon [%s] (%s): Rendered Width = %d, Raw String: '%s'\n", name, icon, width, rendered)
		// Check leading space
		if !strings.HasPrefix(rendered, " ") {
			fmt.Printf("WARNING: [%s] Missing leading space!\n", name)
		}
	}

	testIcon("Egg", theme.IconPackage)
	testIcon("Feather", theme.IconCache)
	testIcon("Nest", theme.IconRegistry)
	testIcon("Bird", "🐦")
	testIcon("Sleep", "💤")

	// 2. Real Calculation
	// Assuming some dummy width and height for testing purposes
	width := 100
	height := 50
	realGrid := layout.CalculateGrid(width, height, 0)
	_ = realGrid // Use the variable to avoid unused error for now

	// 2. Test Card Render inside a fixed box
	// Simulating Sidebar Header
	card := widget.NewCard(theme.IconRegistry, "Registry", "Manager")
	sidebarWidth := 30
	cardView := card.View(sidebarWidth)
	cardLines := strings.Split(cardView, "\n")

	fmt.Printf("\n--- Card Render (Width %d) ---\n", sidebarWidth)
	for i, line := range cardLines {
		w := lipgloss.Width(line)
		msg := "OK"
		if w != sidebarWidth {
			msg = fmt.Sprintf("FAIL (Exp %d, Got %d)", sidebarWidth, w)
		}
		fmt.Printf("L%d: [%s] -> %s\n", i, line, msg)
	}

	// 3. Test Sidebar Item Render
	// Simulating Sidebar Item
	// We manually reconstruct the string building logic from SidebarDelegate
	itemTitle := "ubuntu-24.04"
	itemIcon := theme.IconCache // Feather

	// Delegate Logic Re-enactment
	iconRendered := theme.RenderIcon(itemIcon)
	itemStr := fmt.Sprintf("%s%s", iconRendered, itemTitle)

	// Sidebar Width logic (Boxed Sidebar subtracts borders -> 28?)
	// Let's say list width is 28.
	listWidth := 28
	truncated := itemStr
	if lipgloss.Width(itemStr) > listWidth {
		truncated = lipgloss.NewStyle().MaxWidth(listWidth).Render(itemStr)
	}

	lineW := lipgloss.Width(truncated)
	fmt.Printf("\n--- Sidebar Item Line ---\n")
	fmt.Printf("Item: '%s'\n", truncated)
	fmt.Printf("Width: %d (Max %d)\n", lineW, listWidth)

	// 4. Test Header Item (No Icon)
	headerTitle := "── LOCAL CACHE ──"
	// Registry uses "" padding
	headerStr := headerTitle
	headerW := lipgloss.Width(headerStr)
	fmt.Printf("\n--- Header Item Line ---\n")
	fmt.Printf("Item: '%s'\n", headerStr)
	fmt.Printf("Width: %d (Max %d)\n", headerW, listWidth)
}
