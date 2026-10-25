package layout

// Breakpoint describes the horizontal density of the terminal.
type Breakpoint int

const (
	Narrow Breakpoint = iota
	Regular
	Wide
)

// Cell dimensions owned by the kit. Theme preferences may change sidebar widths,
// but cannot force a pane or a form beyond the available rectangle.
const (
	NarrowMax        = 99
	RegularMax       = 140
	MinWidth         = 40
	MinHeight        = 10
	CompactHeight    = 20
	FormMaxWidth     = 60
	DetailMinWidth   = 40
	PaneDividerWidth = 1
	RowGap           = 1
	FieldFrameWidth  = 4 // two borders and one padding cell per side
	FieldMinWidth    = 24
	DialogMargin     = 2
	DialogPaddingX   = 2
	DialogPaddingY   = 1
	DialogBorder     = 2
)

const (
	FieldLabelGap        = 1
	InputCursorWidth     = 1
	DialogButtonGap      = 3
	DialogHeadingHeight  = 1
	DialogControlsHeight = 1
	ModalWidth           = 50
	AlertWidth           = 60
	ModalHeight          = 10
	FormModalWidth       = 74
	ScrollHintHeight     = 1
	TaskCardHeight       = 3
)

func Detect(width int) Breakpoint {
	if width <= NarrowMax {
		return Narrow
	}
	if width <= RegularMax {
		return Regular
	}
	return Wide
}
func (b Breakpoint) String() string {
	switch b {
	case Narrow:
		return "Narrow"
	case Regular:
		return "Regular"
	case Wide:
		return "Wide"
	}
	return "Unknown"
}

func IsViable(width, height int) bool { return width >= MinWidth && height >= MinHeight }

// FormWidth resolves the single form measure used by every page. In cramped
// panes the decorative inset gives way to the content.
func FormWidth(available, inset int) int {
	return max(0, min(FormMaxWidth, max(min(available, DetailMinWidth), available-inset)))
}

func SidebarWidth(width, preferred, regular, wide int) int {
	if preferred <= 0 {
		preferred = regular
		if Detect(width) == Wide {
			preferred = wide
		}
	}
	return max(0, preferred)
}

func SinglePane(width, sidebar int) bool { return width < sidebar+PaneDividerWidth+DetailMinWidth }
