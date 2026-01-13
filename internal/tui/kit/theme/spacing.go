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
	None  int
	XS    int
	SM    int
	MD    int
	LG    int
	XL    int
	scale int // internal multiplier
}{
	None:  0,
	XS:    1,
	SM:    2,
	MD:    4,
	LG:    6,
	XL:    8,
	scale: 1,
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
	TabMin      int // Minimum per-tab width
	ExitZone    int // Width of the clickable exit zone
}{
	Sidebar:     30,
	SidebarWide: 38,
	Label:       12,
	Button:      18,
	InputMin:    20,
	TabMin:      10,
	ExitZone:    6,
}

// Inset defines horizontal padding applied to main content containers.
// TotalContent represents left+right padding; subtract it from available
// width when sizing inner viewlets or cards.
var Inset = struct {
	TotalContent int
}{
	TotalContent: 4, // 2 cells per side
}

// Overrides allows runtime configuration of layout tokens.
type Overrides struct {
	SidebarWidth     int
	SidebarWideWidth int
	InsetContent     int
	TabMinWidth      int
	ExitZoneWidth    int
	GapScale         int
}

// ApplyOverrides mutates theme tokens from provided overrides, keeping
// existing values when an override is zero or negative.
func ApplyOverrides(o Overrides) {
	if o.SidebarWidth > 0 {
		Width.Sidebar = o.SidebarWidth
	}
	if o.SidebarWideWidth > 0 {
		Width.SidebarWide = o.SidebarWideWidth
	}
	if o.InsetContent > 0 {
		Inset.TotalContent = o.InsetContent
	}
	if o.TabMinWidth > 0 {
		Width.TabMin = o.TabMinWidth
	}
	if o.ExitZoneWidth > 0 {
		Width.ExitZone = o.ExitZoneWidth
	}
	if o.GapScale > 0 {
		Space.scale = o.GapScale
	}
}

// Gap returns a scaled spacing value.
func Gap(val int) int {
	if Space.scale <= 0 {
		return val
	}
	return val * Space.scale
}
