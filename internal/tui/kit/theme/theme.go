package theme

import (
	"os"
	"strings"

	"github.com/Josepavese/nido/internal/pkg/sysutil"

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

// Current returns the active theme based on:
// 1. NIDO_THEME environment variable (light|dark|auto)
// 2. Terminal background detection (if auto)
// 3. 256-color fallback if true color is not available
//
// Note: The Palette uses AdaptiveColor which automatically selects
// Light or Dark values based on lipgloss.HasDarkBackground().
// The IsDark flag is for reference only; the actual color selection
// happens in lipgloss when rendering.
// Registry of available themes
var (
	registry        = make(map[string]Palette)
	activeThemeName = "auto"

	// Built-in themes (registered in init)
	BuiltinThemes = []string{"Dark", "Light", "Pink", "High Contrast", "Matrix"}
)

func init() {
	Register("Dark", Dark)
	Register("Light", Light)
	Register("Pink", Pink)
	Register("High Contrast", HighContrast)
	Register("Matrix", Matrix)
}

// Register adds a new theme to the registry.
func Register(name string, p Palette) {
	registry[strings.ToLower(name)] = p
}

// SetTheme activates a specific theme by name.
func SetTheme(name string) {
	if name == "" {
		return
	}
	activeThemeName = strings.ToLower(name)
}

// AvailableThemes returns a list of all registered theme names.
func AvailableThemes() []string {
	var names []string
	for k := range registry {
		// capitalize for display?
		names = append(names, k) // we store lower, maybe return original case if we stored it?
	}
	return names // simple for now
}

// GetPalette returns the palette for a registered theme.
// Returns default Dark palette if name not found.
func GetPalette(name string) Palette {
	if p, ok := registry[strings.ToLower(name)]; ok {
		return p
	}
	return Dark
}

// Current returns the active theme.
func Current() Theme {
	// 1. Resolve Mode/Name
	mode := activeThemeName
	env := os.Getenv("NIDO_THEME")
	if env != "" {
		mode = strings.ToLower(env)
	}

	// 2. Check Registry
	if p, ok := registry[mode]; ok {
		isDark := mode == "dark"
		return buildTheme(p, isDark)
	}

	// 3. Fallback / Auto Logic
	if mode == "auto" || mode == "" {
		if lipgloss.HasDarkBackground() {
			return buildTheme(Dark, true)
		}
		return buildTheme(Light, false)
	}

	// 4. Unknown theme -> Fallback to Dark
	return buildTheme(Dark, true)
}

func buildTheme(p Palette, isDark bool) Theme {
	is256 := shouldUse256Colors()
	if is256 {
		p = Palette256
	}

	styles := Styles{
		SidebarItem:         lipgloss.NewStyle().Foreground(p.TextDim),
		SidebarItemSelected: lipgloss.NewStyle().Foreground(p.Accent).Bold(true),
		Text:                lipgloss.NewStyle().Foreground(p.Text),
		TextDim:             lipgloss.NewStyle().Foreground(p.TextDim),
		TextMuted:           lipgloss.NewStyle().Foreground(p.TextMuted),
		Accent:              lipgloss.NewStyle().Foreground(p.Accent),
		AccentStrong:        lipgloss.NewStyle().Foreground(p.AccentStrong).Bold(true),
		Success:             lipgloss.NewStyle().Foreground(p.Success),
		Error:               lipgloss.NewStyle().Foreground(p.Error),
		Warning:             lipgloss.NewStyle().Foreground(p.Warning),
		ButtonActive:        lipgloss.NewStyle().Foreground(p.Background).Background(p.Accent).Bold(true).Padding(0, 2),

		Label: lipgloss.NewStyle().Foreground(p.TextDim),
		Value: lipgloss.NewStyle().Foreground(p.TextDim),
		Title: lipgloss.NewStyle().Foreground(p.Accent).Bold(true),
		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(p.SurfaceHighlight),
	}

	layout := Layout{ContainerPadding: 2}

	return Theme{
		Palette: p,
		Is256:   is256,
		IsDark:  isDark, // Approximation, accurate only if using standard adaptive colors
		Styles:  styles,
		Layout:  layout,
	}
}

// LoadUserThemes attempts to load themes.json from ~/.nido
func LoadUserThemes() error {
	home, err := sysutil.UserHome()
	if err != nil {
		return err
	}

	// Check standard path
	path := home + "/.nido/themes.json"

	// Check if install dir has one? (Skipped for now per verification report risk)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // No user themes, no error
	}

	themes, err := LoadThemes(path)
	if err != nil {
		return err // Malformed file is an error worth reporting
	}

	for _, t := range themes {
		Register(t.Name, t.ToPalette())
	}
	return nil
}

// Layout holds metrics for UI spacing and sizing.
type Layout struct {
	ContainerPadding int
}

// parseThemeMode interprets the NIDO_THEME environment variable.

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
