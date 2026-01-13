package layout

import "github.com/Josepavese/nido/internal/tui/theme"

// Breakpoint represents terminal width categories for responsive layouts.
type Breakpoint int

const (
	// Narrow is for terminals < 100 columns.
	// Sidebar may be hidden or stacked vertically.
	Narrow Breakpoint = iota

	// Regular is for terminals 100-140 columns.
	// Standard two-column layout with sidebar.
	Regular

	// Wide is for terminals > 140 columns.
	// Extended sidebar, optional additional panels.
	Wide
)

// Thresholds for breakpoint detection.
// These are empirically validated values from the plan.
const (
	NarrowMax  = 99  // < 100 cols
	RegularMax = 140 // 100-140 cols
	// Wide = > 140 cols
)

// Detect returns the appropriate breakpoint for the given terminal width.
func Detect(width int) Breakpoint {
	if width < 100 {
		return Narrow
	}
	if width <= RegularMax {
		return Regular
	}
	return Wide
}

// String returns a human-readable name for the breakpoint.
func (b Breakpoint) String() string {
	switch b {
	case Narrow:
		return "Narrow"
	case Regular:
		return "Regular"
	case Wide:
		return "Wide"
	default:
		return "Unknown"
	}
}

// SidebarWidth returns the appropriate sidebar width for the breakpoint.
func (b Breakpoint) SidebarWidth() int {
	switch b {
	case Narrow:
		return 0 // Hidden in narrow mode
	case Regular:
		return theme.Width.Sidebar
	case Wide:
		return theme.Width.SidebarWide
	default:
		return theme.Width.Sidebar
	}
}

// ShowSidebar returns whether the sidebar should be visible.
func (b Breakpoint) ShowSidebar() bool {
	return b != Narrow
}

// ContentWidth calculates available content width given terminal width.
// Accounts for sidebar and borders.
func (b Breakpoint) ContentWidth(terminalWidth int) int {
	sidebar := b.SidebarWidth()
	if sidebar == 0 {
		return terminalWidth - 2 // Just borders
	}
	// Terminal - sidebar - border (1) - padding (2)
	return terminalWidth - sidebar - 3
}
