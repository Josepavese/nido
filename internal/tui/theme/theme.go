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
}

// Styles holds common reusable styles
type Styles struct {
	SidebarItem         lipgloss.Style
	SidebarItemSelected lipgloss.Style
}

// themeMode represents the user's theme preference
type themeMode int

const (
	themeModeAuto themeMode = iota
	themeModeDark
	themeModeLight
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
	if is256 {
		palette = Palette256
	} else if isDark {
		palette = Dark
	} else {
		palette = Light
	}

	styles := Styles{
		SidebarItem:         lipgloss.NewStyle().Foreground(palette.TextDim),
		SidebarItemSelected: lipgloss.NewStyle().Foreground(palette.Accent).Bold(true),
	}

	return Theme{
		Palette: palette,
		Is256:   is256,
		IsDark:  isDark,
		Styles:  styles,
	}
}

// parseThemeMode interprets the NIDO_THEME environment variable.
func parseThemeMode(env string) themeMode {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "light":
		return themeModeLight
	case "dark":
		return themeModeDark
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
