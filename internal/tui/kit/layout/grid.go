package layout

// Grid defines the standard 4-zone layout of the Nido TUI shell.
//
//	+-----------------------+
//	| HEADER (Fixed)        |
//	+-----------------------+
//	| SUBHEADER (Fixed)     |
//	+-----------------------+
//	|                       |
//	| BODY (Flex)           |
//	|                       |
//	+-----------------------+
//	| FOOTER (Fixed)        |
//	+-----------------------+
type Grid struct {
	Header      Rect
	ActionStack Rect // Dynamic zone (restored)
	SubHeader   Rect
	Body        Rect
	Footer      Rect
}

const (
	HeaderHeight    = 4 // 1 line padding top + 3 lines bubble tabs
	SubHeaderHeight = 0 // Removed as per user request (redundant with footer)
	FooterHeight    = 1 // 1 line status bar
)

// CalculateGrid computes the grid zones for a given terminal size.
// actionStackHeight is the requested height for the dynamic action stack zone.
func CalculateGrid(width, height int, actionStackHeight int) Grid {
	// Clamp inputs
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	if actionStackHeight < 0 {
		actionStackHeight = 0
	}

	// Re-approach: Strict Geometry from Top and Bottom
	// Header: Top
	// Footer: Bottom
	// Body: The middle

	// Reset loop for strict absolute positioning
	hRect := NewRect(0, 0, width, 0)
	asRect := NewRect(0, 0, width, 0)
	shRect := NewRect(0, 0, width, 0)
	bRect := NewRect(0, 0, width, 0)
	fRect := NewRect(0, 0, width, 0)

	// Available vertical space
	availH := height

	// 1. Header (Fixed, Priority 1)
	allocH := HeaderHeight
	if allocH > availH {
		allocH = availH
	}
	hRect = NewRect(0, 0, width, allocH)
	availH -= allocH

	// 2. Action Stack (Dynamic, Priority 2)
	// Stacks immediately below Header
	allocAS := actionStackHeight
	if allocAS > availH {
		allocAS = availH
	}
	asRect = NewRect(0, hRect.Height, width, allocAS)
	availH -= allocAS

	// 3. SubHeader (Fixed, Priority 3 - Currently 0)
	allocSH := SubHeaderHeight
	if allocSH > availH {
		allocSH = availH
	}
	shRect = NewRect(0, hRect.Height+asRect.Height, width, allocSH)
	availH -= allocSH

	// 4. Footer (Fixed, Priority 4)
	allocF := FooterHeight
	if allocF > availH {
		allocF = availH
	}
	// Footer sticks to bottom
	fRect = NewRect(0, height-allocF, width, allocF)

	// Safety check: ensure Footer doesn't overlap top content
	topContentBottom := hRect.Height + asRect.Height + shRect.Height
	if fRect.Y < topContentBottom {
		fRect.Y = topContentBottom
	}
	availH -= allocF

	// 5. Body (Flex, Priority 5)
	// Body takes whatever is left between Top Content and Footer
	bodyY := topContentBottom
	bodyH := fRect.Y - bodyY
	if bodyH < 0 {
		bodyH = 0
	}
	bRect = NewRect(0, bodyY, width, bodyH)

	return Grid{
		Header:      hRect,
		ActionStack: asRect,
		SubHeader:   shRect,
		Body:        bRect,
		Footer:      fRect,
	}
}
