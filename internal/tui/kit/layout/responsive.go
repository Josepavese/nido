package layout

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
const (
	NarrowMax  = 99  // < 100 cols
	RegularMax = 140 // 100-140 cols
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

// Dimensions holds the current terminal dimensions and derived layout values.
type Dimensions struct {
	// Raw terminal size
	Width  int
	Height int

	// Derived values
	Breakpoint   Breakpoint
	SidebarWidth int
	ContentWidth int
	BodyHeight   int
}

// SpacingOverhead is the total vertical space used by fixed elements.
// Matches grid.go definitions: Header(3) + SubHeader(2) + Footer(1) = 6.
const SpacingOverhead = HeaderHeight + SubHeaderHeight + FooterHeight

// Calculate computes layout dimensions based on terminal size and sidebar preferences.
// It is pure and does not depend on global themes.
func Calculate(width, height int, narrowSidebarW, regularSidebarW, wideSidebarW int) Dimensions {
	bp := Detect(width)

	// Calculate sidebar width based on breakpoint
	var sidebarW int
	switch bp {
	case Narrow:
		sidebarW = narrowSidebarW
	case Regular:
		sidebarW = regularSidebarW
	case Wide:
		sidebarW = wideSidebarW
	}

	// Calculate body height (available for viewlet content)
	bodyHeight := height - SpacingOverhead
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	// Calculate content width (Width - Sidebar - Borders/Gap logic).
	// This logic was "Terminal - sidebar - 3" in legacy.
	// We will preserve that simple heuristic here.
	contentW := width - sidebarW - 3
	if contentW < 0 {
		contentW = 0
	}

	return Dimensions{
		Width:        width,
		Height:       height,
		Breakpoint:   bp,
		SidebarWidth: sidebarW,
		ContentWidth: contentW,
		BodyHeight:   bodyHeight,
	}
}

// MinWidth is the minimum supported terminal width.
const MinWidth = 60

// MinHeight is the minimum supported terminal height.
const MinHeight = 15

// IsViable returns true if the terminal is large enough for the TUI.
func (d Dimensions) IsViable() bool {
	return d.Width >= MinWidth && d.Height >= MinHeight
}

// TooSmallMessage returns a message to display when terminal is too small.
func TooSmallMessage(width, height int) string {
	return "🪺 Terminal too small. Need at least 60×15."
}
