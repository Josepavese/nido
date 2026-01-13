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
	// Constants validation
	// Header=2, SubHeader=2, Footer=1. Total Chrome = 5.

	tests := []struct {
		name        string
		w, h        int
		wantBodyH   int
		wantFooterY int
	}{
		{"Standard (80x24)", 80, 24, 19, 23}, // 24 - 5 = 19
		{"Exact Chrome (80x5)", 80, 5, 0, 4}, // Body is 0
		{"Too Small (80x4)", 80, 4, 0, 3},    // Footer is at y=3 (last line), Header takes 3, SubHeader takes 1?
		// Logic trace: H=3 (avail 1), SH=1 (avail 0), F=0? Let's check test result.
		{"Zero Height", 80, 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := CalculateGrid(tt.w, tt.h)

			if g.Body.Height != tt.wantBodyH {
				t.Errorf("Body Height mismatch: got %d, want %d", g.Body.Height, tt.wantBodyH)
			}

			// Footer should always be at the bottom if enough space
			if tt.h >= 6 {
				if g.Footer.Y != tt.wantFooterY {
					t.Errorf("Footer Y mismatch: got %d, want %d", g.Footer.Y, tt.wantFooterY)
				}
			}

			// Sanity check: Total height usage
			// Sum of all heights should <= tt.h
			totalUsed := g.Header.Height + g.SubHeader.Height + g.Body.Height + g.Footer.Height
			if totalUsed > tt.h {
				t.Errorf("Grid overflow: usage %d > total %d", totalUsed, tt.h)
			}
		})
	}
}
