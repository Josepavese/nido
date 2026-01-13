package theme

import "github.com/charmbracelet/lipgloss"

// Typography provides factory functions for text styles.
// Each function takes a Palette and returns a base Style
// that can be further customized.
//
// Usage:
//
//	t := theme.Current()
//	title := theme.Typography.Title(t.Palette).Render("Hello")
var Typography = struct {
	// Title returns a style for primary headings.
	Title func(p Palette) lipgloss.Style

	// Subtitle returns a style for secondary headings.
	Subtitle func(p Palette) lipgloss.Style

	// Body returns a style for normal text.
	Body func(p Palette) lipgloss.Style

	// Label returns a style for form labels and captions.
	Label func(p Palette) lipgloss.Style

	// Mono returns a style for monospace/code text.
	Mono func(p Palette) lipgloss.Style

	// Dim returns a style for de-emphasized text.
	Dim func(p Palette) lipgloss.Style

	// Accent returns a style for highlighted/accent text.
	Accent func(p Palette) lipgloss.Style

	// Success returns a style for success messages.
	Success func(p Palette) lipgloss.Style

	// Warning returns a style for warning messages.
	Warning func(p Palette) lipgloss.Style

	// Error returns a style for error messages.
	Error func(p Palette) lipgloss.Style
}{
	Title: func(p Palette) lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(p.AccentStrong).
			Bold(true)
	},

	Subtitle: func(p Palette) lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(p.Accent).
			Bold(true)
	},

	Body: func(p Palette) lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(p.Text)
	},

	Label: func(p Palette) lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(p.TextDim).
			Width(Width.Label)
	},

	Mono: func(p Palette) lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(p.Text)
		// Note: Terminal fonts are already monospace
	},

	Dim: func(p Palette) lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(p.TextDim)
	},

	Accent: func(p Palette) lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(p.AccentStrong).
			Bold(true)
	},

	Success: func(p Palette) lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(p.Success)
	},

	Warning: func(p Palette) lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(p.Warning)
	},

	Error: func(p Palette) lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(p.Error).
			Bold(true)
	},
}
