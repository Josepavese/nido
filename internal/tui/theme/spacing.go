package theme

// Space defines the spacing scale in terminal cells.
// Use these values for consistent padding, margins, and gaps.
//
// The scale follows a geometric progression for visual harmony:
//   - None: 0 (no spacing)
//   - XS: 1 (tight)
//   - SM: 2 (compact)
//   - MD: 4 (default)
//   - LG: 6 (comfortable)
//   - XL: 8 (spacious)
var Space = struct {
	None int
	XS   int
	SM   int
	MD   int
	LG   int
	XL   int
}{
	None: 0,
	XS:   1,
	SM:   2,
	MD:   4,
	LG:   6,
	XL:   8,
}

// Radius defines border radius presets.
// Note: Terminal support for rounded corners is limited;
// these are primarily for lipgloss.RoundedBorder() sizing.
var Radius = struct {
	None int
	SM   int
	MD   int
	LG   int
}{
	None: 0,
	SM:   1,
	MD:   2,
	LG:   4,
}

// Width defines common width presets for layout elements.
// These replace hardcoded magic numbers throughout the codebase.
var Width = struct {
	Sidebar     int // Default sidebar width
	SidebarWide int // Sidebar in wide breakpoint
	Label       int // Form label width
	Button      int // Standard button width
	InputMin    int // Minimum input field width
}{
	Sidebar:     18,
	SidebarWide: 28,
	Label:       12,
	Button:      18,
	InputMin:    20,
}
