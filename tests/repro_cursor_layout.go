package main

import (
	"fmt"
	"strings"

	"github.com/Josepavese/nido/internal/tui/kit/widget"
	"github.com/charmbracelet/lipgloss"
)

func main() {
	label := "Name"
	width := 30
	// Inner width available = 30 - 2 (Border) - 2 (Padding) = 26.
	// Label "Name" len 4 -> 10 (Compact fixed width)
	// Content Avail = 26 - 10 = 16.

	fmt.Println("----- LAYOUT COMP STABILITY TEST -----")

	// Case A: Normal content (Blink Off)
	// Pure text.
	contentA := "bird-name"

	// Case B: Content with Cursor (Blink On)
	// Simulate what bubbles/textinput usually does:
	// It often styles the character under cursor with ReverseVideo (\x1b[7m).
	// If cursor is at end, it appends a distinct cursor character or whitespace with style.
	// Let's test a few common scenarios.

	// Scenario B1: Cursor at end (Space with reverse video)
	// "bird-name" + reversed space
	contentB1 := "bird-name" + "\x1b[7m \x1b[0m"

	// Scenario B2: Cursor block char
	contentB2 := "bird-name" + "█"

	scenarios := map[string]string{
		"Blink_OFF":      contentA,
		"Blink_ON_Rev":   contentB1,
		"Blink_ON_Block": contentB2,
	}

	for name, content := range scenarios {
		render := widget.RenderBoxedField(label, content, "", true, width)

		msg := fmt.Sprintf("[%s] TotalWidth: %d", name, lipgloss.Width(render))

		// Split lines and check the content line (middle line usually)
		lines := strings.Split(render, "\n")
		// The box usually has 3 lines: Top border, Content, Bottom border.
		if len(lines) >= 3 {
			// Check middle line length explicitly
			visibleLen := lipgloss.Width(lines[1])
			msg += fmt.Sprintf(" | MiddleLineVis: %d", visibleLen)
			// Print the raw line to see alignment
			fmt.Printf("%-40s | Content: %q\n", msg, lines[1])
		} else {
			fmt.Printf("%s | <Error: Too few lines>\n", msg)
		}
	}
}
