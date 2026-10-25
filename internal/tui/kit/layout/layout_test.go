package layout

import (
	"testing"
)

func TestNewRect(t *testing.T) {
	tests := []struct {
		name         string
		x, y, w, h   int
		wantW, wantH int
	}{
		{"normal", 0, 0, 10, 10, 10, 10},
		{"negative width", 0, 0, -5, 10, 0, 10},
		{"negative height", 0, 0, 10, -5, 10, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRect(tt.x, tt.y, tt.w, tt.h)
			if r.Width != tt.wantW || r.Height != tt.wantH {
				t.Errorf("NewRect() = %v, want w=%d h=%d", r, tt.wantW, tt.wantH)
			}
		})
	}
}

func TestRect_Inner(t *testing.T) {
	r := NewRect(0, 0, 100, 100)
	inner := r.Inner(10, 10, 10, 10)

	if inner.X != 10 || inner.Y != 10 {
		t.Errorf("Inner X/Y mismatch: got (%d,%d) want (10,10)", inner.X, inner.Y)
	}
	if inner.Width != 80 || inner.Height != 80 {
		t.Errorf("Inner W/H mismatch: got %dx%d want 80x80", inner.Width, inner.Height)
	}

	// Test padding exceeding dimensions
	tiny := NewRect(0, 0, 5, 5)
	innerTiny := tiny.Inner(5, 5, 5, 5)
	if innerTiny.Width != 0 || innerTiny.Height != 0 {
		t.Errorf("Inner zero clamp failed: got %v", innerTiny)
	}
}

func TestCalculateGrid(t *testing.T) {
	for _, w := range []int{0, 1, 25, 40, 60, 80, 120, 160} {
		for h := 0; h <= 50; h++ {
			for _, tasks := range []int{0, 3, 300} {
				g := CalculateGrid(w, h, tasks)
				zones := []Rect{g.Header, g.ActionStack, g.Body, g.Footer}
				y := 0
				for _, r := range zones {
					if r.Y != y || r.Width != w || r.Height < 0 {
						t.Fatalf("bad rectangle %v at %dx%d", r, w, h)
					}
					y += r.Height
				}
				if y != h {
					t.Fatalf("grid uses %d rows, available %d", y, h)
				}
				if h > 0 && g.Footer.Height != 1 {
					t.Fatal("exit hint starved")
				}
				if h >= MinHeight && g.Body.Height < 1 {
					t.Fatal("body starved by tasks")
				}
			}
		}
	}
}
func TestFormWidthNeverExceedsPane(t *testing.T) {
	for w := 0; w <= 200; w++ {
		for _, inset := range []int{0, 4, 20, 200} {
			got := FormWidth(w, inset)
			if got < 0 || got > w || got > FormMaxWidth {
				t.Fatalf("invalid form width %d from %d", got, w)
			}
		}
	}
}
