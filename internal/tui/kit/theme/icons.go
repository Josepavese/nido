package theme

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Application Icons (Single Source of Truth)
const (
	// Tab/Page Icons
	IconRegistry = "🛸" // Registry Page (Mothership)
	IconHatchery = "🥐" // Hatchery / Sources (Breadcrumbs / Pica-pane)
	IconFleet    = "🦅" // Fleet Page (Birds in Flight)
	IconSystem   = "💾" // System Page (Floppy Disk / Nerdy)

	// Asset Types
	IconTemplate  = "🧬" // VM Template
	IconFlavour   = "🍦" // Nido Flavour (Pre-configured)
	IconPackage   = "🥚" // Generic Cloud Image / Distro
	IconBlueprint = "📐" // Build recipe for a local image

	// Storage Locations
	IconCache = "👾"  // Local Disk (Alien Tech)
	IconCloud = "☁️" // Remote Registry

	// Actions & States
	IconUnknown      = "❓"
	IconError        = "❌"
	IconCheck        = "✅"  // Success / Update
	IconDoctor       = "🩺"  // Doctor
	IconWarning      = "⚠️" // Warning
	IconSelfDestruct = "🍳"  // The Nest is Cooked

	// Fleet States
	IconBird   = "🐦" // Running
	IconSleep  = "💤" // Stopped
	IconLayout = "🎨" // Appearance / Rice
)

// IconForType maps a string type (e.g., from API or config) to an icon.
func IconForType(t string) string {
	switch t {
	case "TEMPLATE":
		return IconTemplate
	case "FLAVOUR":
		return IconFlavour
	case "CLOUD":
		return IconPackage
	case "BLUEPRINT":
		return IconBlueprint
	case "CACHE", "LOCAL":
		return IconCache
	case "REMOTE":
		return IconCloud
	default:
		return IconUnknown
	}
}

// RenderIcon returns a standardized, fixed-width string for any icon.
// It uses a "Trailing Space" strategy to save space.
// Target Width: 3 cells.
func RenderIcon(icon string) string {
	if icon == "" {
		return "   " // 3 spaces
	}

	// 1. Start with the icon flush left
	s := icon

	// 2. Measure current width
	w := lipgloss.Width(s)

	// 3. Pad to Target (3) with spaces to the right
	target := 3
	if w < target {
		s += strings.Repeat(" ", target-w)
	}

	return s
}
