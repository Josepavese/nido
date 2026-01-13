// Package theme provides the Nido TUI design system.
// It defines color palettes, spacing scales, and typography styles
// that adapt to terminal capabilities (dark/light, true color/256c).
//
// Usage:
//
//	t := theme.Current()
//	style := lipgloss.NewStyle().Foreground(t.Palette.Accent)
//
// The theme automatically detects terminal capabilities and can be
// overridden via the NIDO_THEME environment variable (light|dark|auto).
package theme

import "github.com/charmbracelet/lipgloss"

// Palette holds all color tokens for the design system.
// Uses AdaptiveColor for automatic light/dark switching based on
// terminal background detection.
type Palette struct {
	// Base surfaces
	Background    lipgloss.AdaptiveColor
	Surface       lipgloss.AdaptiveColor
	SurfaceSubtle lipgloss.AdaptiveColor

	// Text hierarchy
	Text      lipgloss.AdaptiveColor
	TextDim   lipgloss.AdaptiveColor
	TextMuted lipgloss.AdaptiveColor

	// Accent colors
	Accent       lipgloss.AdaptiveColor
	AccentStrong lipgloss.AdaptiveColor

	// Semantic colors
	Success lipgloss.AdaptiveColor
	Warning lipgloss.AdaptiveColor
	Error   lipgloss.AdaptiveColor

	// Interactive states
	Focus    lipgloss.AdaptiveColor
	Hover    lipgloss.AdaptiveColor
	Disabled lipgloss.AdaptiveColor
}

// Dark is the default dark theme palette.
// Designed for terminals with dark backgrounds.
var Dark = Palette{
	// Base surfaces - dark blue-gray tones
	Background:    lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#0F1216"},
	Surface:       lipgloss.AdaptiveColor{Light: "#F5F7FA", Dark: "#151A21"},
	SurfaceSubtle: lipgloss.AdaptiveColor{Light: "#E8ECF0", Dark: "#1A2028"},

	// Text hierarchy
	Text:      lipgloss.AdaptiveColor{Light: "#1A1A1A", Dark: "#E6EBF2"},
	TextDim:   lipgloss.AdaptiveColor{Light: "#666666", Dark: "#707D8C"},
	TextMuted: lipgloss.AdaptiveColor{Light: "#999999", Dark: "#4A5568"},

	// Accent - Nido blue
	Accent:       lipgloss.AdaptiveColor{Light: "#3BA3E6", Dark: "#76C7FF"},
	AccentStrong: lipgloss.AdaptiveColor{Light: "#2B7CB8", Dark: "#3BA3E6"},

	// Semantic colors
	Success: lipgloss.AdaptiveColor{Light: "#2DA866", Dark: "#3DDC84"},
	Warning: lipgloss.AdaptiveColor{Light: "#D4940A", Dark: "#F5C26B"},
	Error:   lipgloss.AdaptiveColor{Light: "#D93F4C", Dark: "#F06D79"},

	// Interactive states
	Focus:    lipgloss.AdaptiveColor{Light: "#3BA3E6", Dark: "#76C7FF"},
	Hover:    lipgloss.AdaptiveColor{Light: "#E0F0FF", Dark: "#1E2A38"},
	Disabled: lipgloss.AdaptiveColor{Light: "#CCCCCC", Dark: "#3A3A3A"},
}

// Light is the light theme palette.
// Designed for terminals with light backgrounds.
var Light = Palette{
	// Base surfaces
	Background:    lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#FFFFFF"},
	Surface:       lipgloss.AdaptiveColor{Light: "#F5F7FA", Dark: "#F5F7FA"},
	SurfaceSubtle: lipgloss.AdaptiveColor{Light: "#E8ECF0", Dark: "#E8ECF0"},

	// Text hierarchy
	Text:      lipgloss.AdaptiveColor{Light: "#1A1A1A", Dark: "#1A1A1A"},
	TextDim:   lipgloss.AdaptiveColor{Light: "#666666", Dark: "#666666"},
	TextMuted: lipgloss.AdaptiveColor{Light: "#999999", Dark: "#999999"},

	// Accent
	Accent:       lipgloss.AdaptiveColor{Light: "#3BA3E6", Dark: "#3BA3E6"},
	AccentStrong: lipgloss.AdaptiveColor{Light: "#2B7CB8", Dark: "#2B7CB8"},

	// Semantic colors
	Success: lipgloss.AdaptiveColor{Light: "#2DA866", Dark: "#2DA866"},
	Warning: lipgloss.AdaptiveColor{Light: "#D4940A", Dark: "#D4940A"},
	Error:   lipgloss.AdaptiveColor{Light: "#D93F4C", Dark: "#D93F4C"},

	// Interactive states
	Focus:    lipgloss.AdaptiveColor{Light: "#3BA3E6", Dark: "#3BA3E6"},
	Hover:    lipgloss.AdaptiveColor{Light: "#E0F0FF", Dark: "#E0F0FF"},
	Disabled: lipgloss.AdaptiveColor{Light: "#CCCCCC", Dark: "#CCCCCC"},
}

// Palette256 provides 256-color fallback for legacy terminals.
// Uses ANSI color codes (0-255) for maximum compatibility.
var Palette256 = Palette{
	Background:    lipgloss.AdaptiveColor{Light: "231", Dark: "233"}, // white / dark gray
	Surface:       lipgloss.AdaptiveColor{Light: "255", Dark: "235"}, // bright white / gray
	SurfaceSubtle: lipgloss.AdaptiveColor{Light: "254", Dark: "236"}, // light gray / dark gray

	Text:      lipgloss.AdaptiveColor{Light: "232", Dark: "255"}, // black / white
	TextDim:   lipgloss.AdaptiveColor{Light: "243", Dark: "245"}, // gray / light gray
	TextMuted: lipgloss.AdaptiveColor{Light: "248", Dark: "240"}, // light gray / dark gray

	Accent:       lipgloss.AdaptiveColor{Light: "33", Dark: "117"}, // blue
	AccentStrong: lipgloss.AdaptiveColor{Light: "27", Dark: "75"},  // bright blue

	Success: lipgloss.AdaptiveColor{Light: "34", Dark: "84"},   // green
	Warning: lipgloss.AdaptiveColor{Light: "172", Dark: "221"}, // orange/yellow
	Error:   lipgloss.AdaptiveColor{Light: "160", Dark: "210"}, // red

	Focus:    lipgloss.AdaptiveColor{Light: "33", Dark: "117"},  // blue
	Hover:    lipgloss.AdaptiveColor{Light: "153", Dark: "237"}, // light blue / dark
	Disabled: lipgloss.AdaptiveColor{Light: "250", Dark: "240"}, // gray
}
