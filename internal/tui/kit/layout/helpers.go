package layout

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// HStack joins items horizontally with the specified gap between them.
// Items are aligned at the top by default.
func HStack(gap int, items ...string) string {
	if len(items) == 0 {
		return ""
	}
	if len(items) == 1 {
		return items[0]
	}

	spacer := strings.Repeat(" ", gap)
	result := items[0]
	for i := 1; i < len(items); i++ {
		result = lipgloss.JoinHorizontal(lipgloss.Top, result, spacer, items[i])
	}
	return result
}

// VStack joins items vertically with the specified gap (blank lines) between them.
func VStack(gap int, items ...string) string {
	if len(items) == 0 {
		return ""
	}
	if len(items) == 1 {
		return items[0]
	}

	var spacer string
	if gap > 0 {
		spacer = strings.Repeat("\n", gap)
	}

	result := items[0]
	for i := 1; i < len(items); i++ {
		if spacer != "" {
			result = lipgloss.JoinVertical(lipgloss.Left, result, spacer, items[i])
		} else {
			result = lipgloss.JoinVertical(lipgloss.Left, result, items[i])
		}
	}
	return result
}

// Fill expands content to fill the specified width with spaces.
func Fill(w int, content string) string {
	contentWidth := lipgloss.Width(content)
	if contentWidth >= w {
		return content
	}
	padding := w - contentWidth
	return content + strings.Repeat(" ", padding)
}

// Center centers content within the specified width.
func Center(w int, content string) string {
	return lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Render(content)
}

// Right aligns content to the right within the specified width.
func Right(w int, content string) string {
	return lipgloss.NewStyle().Width(w).Align(lipgloss.Right).Render(content)
}
