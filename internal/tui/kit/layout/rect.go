package layout

import "fmt"

// Rect represents a geometric rectangle in term-space.
// Origin (0,0) is top-left.
type Rect struct {
	X      int
	Y      int
	Width  int
	Height int
}

// NewRect creates a new Rect.
func NewRect(x, y, w, h int) Rect {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	return Rect{X: x, Y: y, Width: w, Height: h}
}

// String returns a debug string representation.
func (r Rect) String() string {
	return fmt.Sprintf("(%d,%d %dx%d)", r.X, r.Y, r.Width, r.Height)
}

// Inner returns a new Rect shrunk by padding.
// If padding exceeds dimensions, resulting width/height will be 0.
func (r Rect) Inner(top, right, bottom, left int) Rect {
	newX := r.X + left
	newY := r.Y + top
	newW := r.Width - left - right
	newH := r.Height - top - bottom

	return NewRect(newX, newY, newW, newH)
}

// Contains returns true if the coordinate (x, y) is within the rectangle.
func (r Rect) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.Width && y >= r.Y && y < r.Y+r.Height
}
