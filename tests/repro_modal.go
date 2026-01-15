package main

import (
	"fmt"
	"strings"

	"github.com/Josepavese/nido/internal/tui/kit/theme"
	"github.com/Josepavese/nido/internal/tui/kit/widget"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func main() {
	// Initialize Theme
	_ = theme.Current()

	// Create a dummy alert modal
	m := widget.NewAlertModal(
		"Test Error",
		"This is a simulation of the error modal.\nWe are checking for artifacts.",
		func() tea.Cmd { return nil },
	)
	m.Show()

	// Dimensions to simulate the Detail Pane
	width := 60
	height := 20

	// Render
	view := m.View(width, height)

	// Analyze the output
	fmt.Println("----- RENDERED VIEW START -----")
	fmt.Println(view)
	fmt.Println("----- RENDERED VIEW END -----")

	// Mathematical Analysis
	lines := strings.Split(view, "\n")
	fmt.Printf("Total Lines: %d (Expected: %d)\n", len(lines), height)

	for i, line := range lines {
		// Check visible width
		w := lipgloss.Width(line)
		fmt.Printf("Line %02d VisLen: %d (Target: %d) | Content: %q\n", i, w, width, line)

		if w != width && w > 0 { // Ignore empty lines if PlaceOverlay uses them for vertical padding
			fmt.Printf(" [!] Width Mismatch on line %d\n", i)
		}
	}

	fmt.Println("\n----- RAW ANSI CHECK -----")
	// Check for background color leakage or resets
	if strings.Contains(view, "\x1b[0m") {
		fmt.Println("Found ANSI Reset codes - Check if they clear background prematurely.")
	}
}
