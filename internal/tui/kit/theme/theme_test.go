package theme

import (
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestDarkPaletteComplete verifies all Dark palette fields are populated.
func TestDarkPaletteComplete(t *testing.T) {
	p := Dark

	// Check that no AdaptiveColor has empty values
	checks := []struct {
		name  string
		color lipgloss.AdaptiveColor
	}{
		{"Background", p.Background},
		{"Surface", p.Surface},
		{"SurfaceSubtle", p.SurfaceSubtle},
		{"Text", p.Text},
		{"TextDim", p.TextDim},
		{"TextMuted", p.TextMuted},
		{"Accent", p.Accent},
		{"AccentStrong", p.AccentStrong},
		{"Success", p.Success},
		{"Warning", p.Warning},
		{"Error", p.Error},
		{"Focus", p.Focus},
		{"Hover", p.Hover},
		{"Disabled", p.Disabled},
	}

	for _, c := range checks {
		if c.color.Light == "" || c.color.Dark == "" {
			t.Errorf("Dark.%s has empty color value (Light=%q, Dark=%q)",
				c.name, c.color.Light, c.color.Dark)
		}
	}
}

// TestPalette256Complete verifies 256-color fallback palette is populated.
func TestPalette256Complete(t *testing.T) {
	p := Palette256

	checks := []struct {
		name  string
		color lipgloss.AdaptiveColor
	}{
		{"Background", p.Background},
		{"Surface", p.Surface},
		{"Text", p.Text},
		{"Accent", p.Accent},
		{"Success", p.Success},
		{"Warning", p.Warning},
		{"Error", p.Error},
	}

	for _, c := range checks {
		if c.color.Light == "" || c.color.Dark == "" {
			t.Errorf("Palette256.%s has empty color value", c.name)
		}
	}
}

// TestSpaceValues verifies spacing scale values are correct.
func TestSpaceValues(t *testing.T) {
	if Space.None != 0 {
		t.Errorf("Space.None = %d, want 0", Space.None)
	}
	if Space.XS != 1 {
		t.Errorf("Space.XS = %d, want 1", Space.XS)
	}
	if Space.SM != 2 {
		t.Errorf("Space.SM = %d, want 2", Space.SM)
	}
	if Space.MD != 4 {
		t.Errorf("Space.MD = %d, want 4", Space.MD)
	}
	if Space.LG != 6 {
		t.Errorf("Space.LG = %d, want 6", Space.LG)
	}
	if Space.XL != 8 {
		t.Errorf("Space.XL = %d, want 8", Space.XL)
	}
}

// TestWidthValues verifies width constants match expected values.
func TestWidthValues(t *testing.T) {
	if Width.Sidebar != 30 {
		t.Errorf("Width.Sidebar = %d, want 30", Width.Sidebar)
	}
	if Width.Label != 12 {
		t.Errorf("Width.Label = %d, want 12", Width.Label)
	}
}

// TestCurrentReturnsValidTheme verifies Current() returns a usable theme.
func TestCurrentReturnsValidTheme(t *testing.T) {
	theme := Current()

	// Should have a valid palette
	if theme.Palette.Text.Dark == "" && theme.Palette.Text.Light == "" {
		t.Error("Current() returned theme with empty Text color")
	}
}

// TestNIDO_THEME_Override verifies environment variable overrides work.
func TestNIDO_THEME_Override(t *testing.T) {
	tests := []struct {
		env      string
		wantDark bool
	}{
		{"dark", true},
		{"DARK", true},
		{"light", false},
		{"LIGHT", false},
		{"auto", true}, // auto defaults to dark if detection fails
		{"", true},     // empty defaults to auto -> dark
	}

	for _, tc := range tests {
		t.Run("NIDO_THEME="+tc.env, func(t *testing.T) {
			os.Setenv("NIDO_THEME", tc.env)
			defer os.Unsetenv("NIDO_THEME")

			theme := Current()
			if theme.IsDark != tc.wantDark {
				t.Errorf("NIDO_THEME=%q: IsDark = %v, want %v",
					tc.env, theme.IsDark, tc.wantDark)
			}
		})
	}
}

// TestForceMode verifies explicit mode selection works.
func TestForceMode(t *testing.T) {
	dark := ForceMode(true)
	if !dark.IsDark {
		t.Error("ForceMode(true) should return dark theme")
	}

	light := ForceMode(false)
	if light.IsDark {
		t.Error("ForceMode(false) should return light theme")
	}
}
