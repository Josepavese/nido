package theme

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Theme holds the active palette and terminal capability flags.
type Theme struct {
	// Palette contains all color tokens
	Palette Palette

	// Is256 indicates if we're using 256-color fallback
	Is256 bool

	// IsDark indicates if the terminal has a dark background
	IsDark bool

	// Styles contains pre-defined lipgloss styles
	Styles Styles

	// Layout contains spacing/sizing metrics
	Layout Layout
}

// Styles holds common reusable styles
type Styles struct {
	SidebarItem         lipgloss.Style
	SidebarItemSelected lipgloss.Style
	Text                lipgloss.Style
	TextDim             lipgloss.Style
	TextMuted           lipgloss.Style
	Accent              lipgloss.Style
	AccentStrong        lipgloss.Style
	Success             lipgloss.Style
	Error               lipgloss.Style
	Warning             lipgloss.Style
	ButtonActive        lipgloss.Style

	// Semantic components
	Label  lipgloss.Style
	Value  lipgloss.Style
	Title  lipgloss.Style
	Border lipgloss.Style
}

// themeMode represents the user's theme preference
type themeMode int

const (
	themeModeAuto themeMode = iota
	themeModeDark
	themeModeLight
	themeModePink
	themeModeHighContrast
	themeModeMatrix
)

// Current returns the active theme based on:
// 1. NIDO_THEME environment variable (light|dark|auto)
// 2. Terminal background detection (if auto)
// 3. 256-color fallback if true color is not available
//
// Note: The Palette uses AdaptiveColor which automatically selects
// Light or Dark values based on lipgloss.HasDarkBackground().
// The IsDark flag is for reference only; the actual color selection
// happens in lipgloss when rendering.
func Current() Theme {
	mode := parseThemeMode(os.Getenv("NIDO_THEME"))
	is256 := shouldUse256Colors()

	// Determine if terminal has dark background
	isDark := true // default to dark
	if mode == themeModeLight {
		isDark = false
	} else if mode == themeModeAuto {
		isDark = lipgloss.HasDarkBackground()
	}

	// Select appropriate palette
	// Note: AdaptiveColor automatically handles light/dark rendering
	// The palette contains both Light and Dark values; lipgloss picks the right one
	var palette Palette
	switch mode {
	case themeModePink:
		palette = Pink
	case themeModeHighContrast:
		palette = HighContrast
	case themeModeMatrix:
		palette = Matrix
	case themeModeDark:
		palette = Dark
	case themeModeLight:
		palette = Light
	default:
		// themeModeAuto or fallback
		if is256 {
			palette = Palette256
		} else if isDark {
			palette = Dark
		} else {
			palette = Light
		}
	}

	styles := Styles{
		SidebarItem:         lipgloss.NewStyle().Foreground(palette.TextDim),
		SidebarItemSelected: lipgloss.NewStyle().Foreground(palette.Accent).Bold(true),
		Text:                lipgloss.NewStyle().Foreground(palette.Text),
		TextDim:             lipgloss.NewStyle().Foreground(palette.TextDim),
		TextMuted:           lipgloss.NewStyle().Foreground(palette.TextMuted),
		Accent:              lipgloss.NewStyle().Foreground(palette.Accent),
		AccentStrong:        lipgloss.NewStyle().Foreground(palette.AccentStrong).Bold(true),
		Success:             lipgloss.NewStyle().Foreground(palette.Success),
		Error:               lipgloss.NewStyle().Foreground(palette.Error),
		Warning:             lipgloss.NewStyle().Foreground(palette.Warning),
		ButtonActive:        lipgloss.NewStyle().Foreground(palette.Background).Background(palette.Accent).Bold(true).Padding(0, 2),

		// Standardized Component Styles
		Label: lipgloss.NewStyle().Foreground(palette.TextDim),
		Value: lipgloss.NewStyle().Foreground(palette.Text),
		Title: lipgloss.NewStyle().Foreground(palette.Accent).Bold(true),
		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(palette.SurfaceHighlight),
	}

	layout := Layout{
		ContainerPadding: 2, // Standard symmetry (was 2 implied)
	}

	return Theme{
		Palette: palette,
		Is256:   is256,
		IsDark:  isDark,
		Styles:  styles,
		Layout:  layout,
	}
}

// Layout holds metrics for UI spacing and sizing.
type Layout struct {
	ContainerPadding int
}

// parseThemeMode interprets the NIDO_THEME environment variable.
func parseThemeMode(env string) themeMode {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "light":
		return themeModeLight
	case "dark":
		return themeModeDark
	case "pink":
		return themeModePink
	case "high-contrast":
		return themeModeHighContrast
	case "matrix":
		return themeModeMatrix
	default:
		return themeModeAuto
	}
}

// shouldUse256Colors checks if we should use 256-color fallback.
// This happens when COLORTERM is not set to truecolor/24bit.
func shouldUse256Colors() bool {
	colorterm := os.Getenv("COLORTERM")
	colorterm = strings.ToLower(colorterm)

	// If COLORTERM indicates true color support, use full palette
	if colorterm == "truecolor" || colorterm == "24bit" {
		return false
	}

	// Check TERM for 256color support as minimum
	term := os.Getenv("TERM")
	if strings.Contains(term, "256color") {
		return true
	}

	// Default: assume true color is available on modern terminals
	// Most terminals in 2026 support true color
	return false
}

// Detect is an alias for Current() for backwards compatibility.
func Detect() Theme {
	return Current()
}

// ForceMode returns a theme with the specified mode, ignoring environment.
// Useful for testing or explicit user selection.
func ForceMode(dark bool) Theme {
	is256 := shouldUse256Colors()

	var palette Palette
	if is256 {
		palette = Palette256
	} else {
		palette = Dark
	}

	return Theme{
		Palette: palette,
		Is256:   is256,
		IsDark:  dark,
	}
}
