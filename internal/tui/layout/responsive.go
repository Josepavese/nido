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

// HeaderHeight is the fixed height for the header section.
const HeaderHeight = 2

// TabbarHeight is the fixed height for the tab bar.
const TabbarHeight = 1

// FooterHeight is the fixed height for the footer/status bar.
const FooterHeight = 1

// SpacingOverhead is the total vertical space used by fixed elements.
// Header(2) + SubHeader(2) + Footer(1) + 3 Gaps = 8
const SpacingOverhead = 8

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
