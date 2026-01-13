package layout

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

// HeaderHeight is the fixed height for the header section (tabs row).
const HeaderHeight = 1

// TabbarHeight is the fixed height for the subheader/context row.
const TabbarHeight = 1

// FooterHeight is the fixed height for the footer/status bar.
const FooterHeight = 1

// SpacingOverhead is the total vertical space used by fixed elements when
// stacking header, subheader, body, footer with a single blank line gap
// between each section:
// Header(1) + Gap(1) + SubHeader(1) + Gap(1) + Body + Gap(1) + Footer(1) = 6
const SpacingOverhead = HeaderHeight + TabbarHeight + FooterHeight + 3

// Calculate returns Dimensions with all layout values computed
// from the given terminal width and height.
func Calculate(width, height int) Dimensions {
	bp := Detect(width)

	// Calculate body height (available for viewlet content)
	bodyHeight := height - SpacingOverhead
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	return Dimensions{
		Width:        width,
		Height:       height,
		Breakpoint:   bp,
		SidebarWidth: bp.SidebarWidth(),
		ContentWidth: bp.ContentWidth(width),
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
