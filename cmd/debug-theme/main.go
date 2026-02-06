package main

import (
	"fmt"

	"github.com/Josepavese/nido/internal/tui/kit/theme"
	"github.com/Josepavese/nido/internal/tui/kit/widget"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

func main() {
	// 1. Force theme load (just in case)
	_ = theme.Current()

	// 2. Mock Items
	items := []list.Item{
		widget.SimpleListItem("Theme A"),
		widget.SimpleListItem("Theme B"),
	}

	// 3. Create Modal
	modal := widget.NewListModal(
		"Debug Theme Selector",
		items,
		40, 20,
		nil, nil,
	)
	modal.Show()

	// 4. Render
	view := modal.View(80, 24)
	fmt.Printf("View Length: %d\n", len(view))
	fmt.Printf("View Content:\n%q\n", view)

	// 5. Test Lipgloss Place
	placed := lipgloss.Place(80, 24, lipgloss.Center, lipgloss.Center, view)
	fmt.Printf("Placed Length: %d\n", len(placed))
}
