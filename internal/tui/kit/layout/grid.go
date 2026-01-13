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
	Header    Rect
	SubHeader Rect
	Body      Rect
	Footer    Rect
}

const (
	HeaderHeight    = 2 // 1 line tabs + 1 bottom border
	SubHeaderHeight = 2 // 1 line text + 1 padding/gap
	FooterHeight    = 1 // 1 line status bar
)

// CalculateGrid computes the grid zones for a given terminal size.
func CalculateGrid(width, height int) Grid {
	// Clamp inputs
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}

	// Re-approach: Strict Geometry from Top and Bottom
	// Header: Top
	// Footer: Bottom
	// Body: The middle

	// Reset loop for strict absolute positioning
	hRect := NewRect(0, 0, width, 0)
	shRect := NewRect(0, 0, width, 0)
	bRect := NewRect(0, 0, width, 0)
	fRect := NewRect(0, 0, width, 0)

	// Available vertical space
	availH := height

	// Header
	allocH := HeaderHeight
	if allocH > availH {
		allocH = availH
	}
	hRect = NewRect(0, 0, width, allocH)
	availH -= allocH

	// SubHeader
	allocSH := SubHeaderHeight
	if allocSH > availH {
		allocSH = availH
	}
	shRect = NewRect(0, hRect.Height, width, allocSH)
	availH -= allocSH

	// Footer (Bottom Up)
	allocF := FooterHeight
	if allocF > availH {
		allocF = availH
	}
	// Y position of footer is: TotalHeight - allocF
	// But if we condensed everything (e.g. h=4),
	// H=3, SH=1, Footer=0?
	// It's safer to stack them.
	// But "Grid" implies structural integrity.
	// If the screen is too small, Body should shrink to 0 first.
	// Then SubHeader/Footer.
	// Header usually has priority.

	// Let's stick to simple top-down stack for H+SH, and bottom-up for F.
	// Remaining checks are correct.

	fRect = NewRect(0, height-allocF, width, allocF) // Sticks to bottom
	if fRect.Y < hRect.Height+shRect.Height {
		// Overlap due to extreme small height?
		// Ensure Footer starts after SH
		fRect.Y = hRect.Height + shRect.Height
	}
	availH -= allocF

	// Body takes whatever is left between (Header+SubHeader) and Footer
	bodyY := hRect.Height + shRect.Height
	bodyH := fRect.Y - bodyY
	if bodyH < 0 {
		bodyH = 0
	}
	bRect = NewRect(0, bodyY, width, bodyH)

	return Grid{
		Header:    hRect,
		SubHeader: shRect,
		Body:      bRect,
		Footer:    fRect,
	}
}
