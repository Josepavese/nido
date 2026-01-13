package layout

import (
	"testing"
)

func TestHStack(t *testing.T) {
	tests := []struct {
		name     string
		gap      int
		items    []string
		contains string
	}{
		{"empty", 0, []string{}, ""},
		{"single", 0, []string{"hello"}, "hello"},
		{"two items", 2, []string{"a", "b"}, "a"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := HStack(tc.gap, tc.items...)
			if tc.contains != "" && len(result) == 0 {
				t.Errorf("HStack returned empty, expected to contain %q", tc.contains)
			}
		})
	}
}

func TestVStack(t *testing.T) {
	result := VStack(0, "line1", "line2")
	if result == "" {
		t.Error("VStack returned empty string")
	}
}

func TestGrid(t *testing.T) {
	items := []string{"a", "b", "c", "d"}
	result := Grid(2, 1, items)
	if result == "" {
		t.Error("Grid returned empty string")
	}
}

func TestDetectBreakpoint(t *testing.T) {
	tests := []struct {
		width    int
		expected Breakpoint
	}{
		{50, Narrow},
		{99, Narrow},
		{100, Regular},
		{140, Regular},
		{141, Wide},
		{200, Wide},
	}

	for _, tc := range tests {
		result := Detect(tc.width)
		if result != tc.expected {
			t.Errorf("Detect(%d) = %v, want %v", tc.width, result, tc.expected)
		}
	}
}

func TestBreakpointSidebarWidth(t *testing.T) {
	if Narrow.SidebarWidth() != 0 {
		t.Errorf("Narrow.SidebarWidth() = %d, want 0", Narrow.SidebarWidth())
	}
	if Regular.SidebarWidth() != 18 {
		t.Errorf("Regular.SidebarWidth() = %d, want 18", Regular.SidebarWidth())
	}
	if Wide.SidebarWidth() != 28 {
		t.Errorf("Wide.SidebarWidth() = %d, want 28", Wide.SidebarWidth())
	}
}

func TestBreakpointShowSidebar(t *testing.T) {
	if Narrow.ShowSidebar() {
		t.Error("Narrow.ShowSidebar() should be false")
	}
	if !Regular.ShowSidebar() {
		t.Error("Regular.ShowSidebar() should be true")
	}
	if !Wide.ShowSidebar() {
		t.Error("Wide.ShowSidebar() should be true")
	}
}

func TestCalculate(t *testing.T) {
	dim := Calculate(120, 40)

	if dim.Width != 120 {
		t.Errorf("Width = %d, want 120", dim.Width)
	}
	if dim.Height != 40 {
		t.Errorf("Height = %d, want 40", dim.Height)
	}
	if dim.Breakpoint != Regular {
		t.Errorf("Breakpoint = %v, want Regular", dim.Breakpoint)
	}
	if dim.SidebarWidth != 18 {
		t.Errorf("SidebarWidth = %d, want 18", dim.SidebarWidth)
	}
	if dim.BodyHeight != 32 { // 40 - 8 = 32
		t.Errorf("BodyHeight = %d, want 32", dim.BodyHeight)
	}
}

func TestDimensionsIsViable(t *testing.T) {
	small := Calculate(50, 10)
	if small.IsViable() {
		t.Error("50x10 should not be viable")
	}

	ok := Calculate(80, 20)
	if !ok.IsViable() {
		t.Error("80x20 should be viable")
	}
}

func TestPadding(t *testing.T) {
	result := Pad(1, 2, 1, 2, "test")
	if result == "" {
		t.Error("Pad returned empty string")
	}

	result = PadH(2, "test")
	if result == "" {
		t.Error("PadH returned empty string")
	}

	result = PadV(1, "test")
	if result == "" {
		t.Error("PadV returned empty string")
	}
}

func TestWidthAndHeight(t *testing.T) {
	result := Width(20, "test")
	if result == "" {
		t.Error("Width returned empty string")
	}

	result = Height(5, "test")
	if result == "" {
		t.Error("Height returned empty string")
	}
}

func TestAlignment(t *testing.T) {
	result := Center(20, "test")
	if result == "" {
		t.Error("Center returned empty string")
	}

	result = Right(20, "test")
	if result == "" {
		t.Error("Right returned empty string")
	}
}
