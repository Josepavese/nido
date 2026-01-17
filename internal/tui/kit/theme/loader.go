package theme

import (
	"encoding/json"
	"os"

	"github.com/charmbracelet/lipgloss"
)

// ThemeDefinition mirrors the JSON structure for external themes.
type ThemeDefinition struct {
	Name    string            `json:"name"`
	Mode    string            `json:"mode"` // "dark" or "light" (base)
	Palette PaletteDefinition `json:"palette"`
}

// PaletteDefinition maps JSON keys to the Palette struct.
type PaletteDefinition struct {
	Background       AdaptiveColorDef `json:"background"`
	Surface          AdaptiveColorDef `json:"surface"`
	SurfaceSubtle    AdaptiveColorDef `json:"surface_subtle"`
	SurfaceHighlight AdaptiveColorDef `json:"surface_highlight"`

	Text      AdaptiveColorDef `json:"text"`
	TextDim   AdaptiveColorDef `json:"text_dim"`
	TextMuted AdaptiveColorDef `json:"text_muted"`

	Accent       AdaptiveColorDef `json:"accent"`
	AccentStrong AdaptiveColorDef `json:"accent_strong"`

	Success AdaptiveColorDef `json:"success"`
	Warning AdaptiveColorDef `json:"warning"`
	Error   AdaptiveColorDef `json:"error"`

	Focus    AdaptiveColorDef `json:"focus"`
	Hover    AdaptiveColorDef `json:"hover"`
	Disabled AdaptiveColorDef `json:"disabled"`
}

type AdaptiveColorDef struct {
	Light string `json:"light"`
	Dark  string `json:"dark"`
}

// ToPalette converts the JSON definition to a native Palette.
func (td ThemeDefinition) ToPalette() Palette {
	p := Palette{
		Background:       td.Palette.Background.toAdaptive(),
		Surface:          td.Palette.Surface.toAdaptive(),
		SurfaceSubtle:    td.Palette.SurfaceSubtle.toAdaptive(),
		SurfaceHighlight: td.Palette.SurfaceHighlight.toAdaptive(),
		Text:             td.Palette.Text.toAdaptive(),
		TextDim:          td.Palette.TextDim.toAdaptive(),
		TextMuted:        td.Palette.TextMuted.toAdaptive(),
		Accent:           td.Palette.Accent.toAdaptive(),
		AccentStrong:     td.Palette.AccentStrong.toAdaptive(),
		Success:          td.Palette.Success.toAdaptive(),
		Warning:          td.Palette.Warning.toAdaptive(),
		Error:            td.Palette.Error.toAdaptive(),
		Focus:            td.Palette.Focus.toAdaptive(),
		Hover:            td.Palette.Hover.toAdaptive(),
		Disabled:         td.Palette.Disabled.toAdaptive(),
	}
	return p
}

func (ac AdaptiveColorDef) toAdaptive() lipgloss.AdaptiveColor {
	return lipgloss.AdaptiveColor{Light: ac.Light, Dark: ac.Dark}
}

// LoadThemes reads a list of themes from a JSON file.
func LoadThemes(path string) ([]ThemeDefinition, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var themesWrapper struct {
		Themes []ThemeDefinition `json:"themes"`
	}

	if err := json.Unmarshal(content, &themesWrapper); err != nil {
		return nil, err
	}

	return themesWrapper.Themes, nil
}
