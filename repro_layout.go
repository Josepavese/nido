package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Simplified RenderBoxedField from form.go to debug logic
func RenderBoxedField(label, content, errorMsg string, focused bool, width int) string {
	// 1. Determine Label Width
	labelWidth := 14 // Standard from theme
	if width > 0 && width < 30 {
		labelWidth = 6 // Compact
	}

	labelStyle := lipgloss.NewStyle().
		Width(labelWidth).
		MaxWidth(labelWidth).
		Align(lipgloss.Left)

	contentStyle := lipgloss.NewStyle()
	validation := ""
	if errorMsg != "" {
		validation = " 🔺"
	}

	// 2. Calculate Available Space
	// Box Overhead: 2 (Borders) + 2 (Padding) = 4
	innerWidth := width - 4
	if innerWidth < 0 {
		innerWidth = 0
	}

	// Render Label (Simulated)
	if len(label) > labelWidth {
		label = label[:labelWidth] // Simplified truncation
	}
	renderedLabel := labelStyle.Render(label)

	valWidth := len(validation) // Simplified width check

	// THE ISSUE MIGHT BE HERE:
	// If lipgloss adds extra padding or if my width math is off by 1-2 chars
	contentAvail := innerWidth - labelWidth - valWidth
	if contentAvail < 0 {
		contentAvail = 0
	}

	// Content Truncation Logic
	effectiveContent := content
	if len(content) > contentAvail {
		effectiveContent = content[:contentAvail] // Strict truncation
	}

	// Compose
	middleBlock := lipgloss.PlaceHorizontal(contentAvail, lipgloss.Left, contentStyle.Render(effectiveContent))

	inner := lipgloss.JoinHorizontal(lipgloss.Top,
		renderedLabel,
		middleBlock,
		validation,
	)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1)

	if width > 0 {
		boxStyle = boxStyle.Width(width - 2) // Match main logic
	}

	return boxStyle.Render(inner)
}

func main() {
	// Scenario from User Screenshot: IP Truncation
	// IP: "127.0.0.1" -> ".0.0.1" (Shows it's truncated from START? No, lipgloss might align right if overflowing?)
	// Or maybe the input control itself is scrolling?
	// Let's test basic width calc.

	testWidth := 40 // Half of an 80-col terminal approx
	label := "IP"
	content := "127.0.0.1"

	fmt.Printf("Testing Width: %d\n", testWidth)
	out := RenderBoxedField(label, content, "", false, testWidth)
	fmt.Println(out)

	// Measure actual width
	lines := strings.Split(out, "\n")
	if len(lines) > 0 {
		fmt.Printf("Actual Rendered Width: %d\n", lipgloss.Width(lines[0]))
	}

	// Diagnostics
	labelWidth := 14
	if testWidth < 30 {
		labelWidth = 6
	}
	fmt.Printf("Inner Width Calc: %d - 4 = %d\n", testWidth, testWidth-4)
	fmt.Printf("Label Width: %d\n", labelWidth)
	fmt.Printf("Content Avail: %d - %d = %d\n", testWidth-4, labelWidth, (testWidth-4)-labelWidth)
	fmt.Printf("Content Len: %d\n", len(content))
}
