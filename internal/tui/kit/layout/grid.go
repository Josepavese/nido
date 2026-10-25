package layout

// Grid allocates chrome without starving the body or losing the exit hint.
type Grid struct {
	Header, ActionStack, Body, Footer Rect
}

const (
	HeaderHeight        = 4
	CompactHeaderHeight = 1
	FooterHeight        = 1
)

func HeaderSize(width, height int) int {
	if Detect(width) == Narrow || height < CompactHeight {
		return CompactHeaderHeight
	}
	return HeaderHeight
}
func CalculateGrid(width, height, actionStackHeight int) Grid {
	width, height = max(0, width), max(0, height)
	footerH := min(FooterHeight, height)
	headerH := min(HeaderSize(width, height), height-footerH)
	// Running tasks can use at most a third of the remaining space. Keep a body
	// line even for an extreme size; the shell presents the resize state there.
	remaining := height - footerH - headerH
	stackH := min(max(0, actionStackHeight), remaining/3)
	return Grid{
		Header:      NewRect(0, 0, width, headerH),
		ActionStack: NewRect(0, headerH, width, stackH),
		Body:        NewRect(0, headerH+stackH, width, remaining-stackH),
		Footer:      NewRect(0, height-footerH, width, footerH),
	}
}
