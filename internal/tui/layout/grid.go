// Package layout provides declarative layout helpers for the Nido TUI.
// It eliminates manual lipgloss.JoinHorizontal/Vertical calls and
// replaces hardcoded widths with calculated, responsive layouts.
//
// Usage:
//
//	content := layout.HStack(theme.Space.SM,
//	    sidebar.View(),
//	    mainContent.View(),
//	)
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

	// Create gap spacer
	spacer := strings.Repeat(" ", gap)

	// Join with spacer
	result := items[0]
	for i := 1; i < len(items); i++ {
		result = lipgloss.JoinHorizontal(lipgloss.Top, result, spacer, items[i])
	}
	return result
}

// HStackCenter joins items horizontally with center vertical alignment.
func HStackCenter(gap int, items ...string) string {
	if len(items) == 0 {
		return ""
	}
	if len(items) == 1 {
		return items[0]
	}

	spacer := strings.Repeat(" ", gap)
	result := items[0]
	for i := 1; i < len(items); i++ {
		result = lipgloss.JoinHorizontal(lipgloss.Center, result, spacer, items[i])
	}
	return result
}

// HStackBottom joins items horizontally with bottom vertical alignment.
func HStackBottom(gap int, items ...string) string {
	if len(items) == 0 {
		return ""
	}
	if len(items) == 1 {
		return items[0]
	}

	spacer := strings.Repeat(" ", gap)
	result := items[0]
	for i := 1; i < len(items); i++ {
		result = lipgloss.JoinHorizontal(lipgloss.Bottom, result, spacer, items[i])
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

	// Create gap as blank lines
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

// VStackCenter joins items vertically with center horizontal alignment.
func VStackCenter(gap int, items ...string) string {
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
			result = lipgloss.JoinVertical(lipgloss.Center, result, spacer, items[i])
		} else {
			result = lipgloss.JoinVertical(lipgloss.Center, result, items[i])
		}
	}
	return result
}

// Grid arranges items in a grid with the specified number of columns.
// Items flow left-to-right, top-to-bottom.
func Grid(cols int, gap int, items []string) string {
	if len(items) == 0 || cols < 1 {
		return ""
	}

	var rows []string
	for i := 0; i < len(items); i += cols {
		end := i + cols
		if end > len(items) {
			end = len(items)
		}
		row := HStack(gap, items[i:end]...)
		rows = append(rows, row)
	}

	return VStack(gap, rows...)
}

// Pad adds padding around content.
func Pad(top, right, bottom, left int, content string) string {
	style := lipgloss.NewStyle().
		PaddingTop(top).
		PaddingRight(right).
		PaddingBottom(bottom).
		PaddingLeft(left)
	return style.Render(content)
}

// PadH adds horizontal padding (left and right).
func PadH(amount int, content string) string {
	return Pad(0, amount, 0, amount, content)
}

// PadV adds vertical padding (top and bottom).
func PadV(amount int, content string) string {
	return Pad(amount, 0, amount, 0, content)
}

// Width sets a fixed width for content.
func Width(w int, content string) string {
	return lipgloss.NewStyle().Width(w).Render(content)
}

// Height sets a fixed height for content.
func Height(h int, content string) string {
	return lipgloss.NewStyle().Height(h).Render(content)
}

// MaxWidth constrains content to a maximum width.
func MaxWidth(w int, content string) string {
	return lipgloss.NewStyle().MaxWidth(w).Render(content)
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
