package components

import (
	"github.com/Josepavese/nido/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// Badge renders a small styled text label.
type Badge struct {
	Text    string
	Variant BadgeVariant
}

// BadgeVariant defines the visual style of a badge.
type BadgeVariant int

const (
	BadgeDefault BadgeVariant = iota
	BadgeSuccess
	BadgeWarning
	BadgeError
	BadgeInfo
	BadgeMuted
)

// Render returns the styled badge string.
func (b Badge) Render() string {
	t := theme.Current()
	style := lipgloss.NewStyle().
		Padding(0, 1).
		Bold(true)

	switch b.Variant {
	case BadgeSuccess:
		style = style.Foreground(t.Palette.Success)
	case BadgeWarning:
		style = style.Foreground(t.Palette.Warning)
	case BadgeError:
		style = style.Foreground(t.Palette.Error)
	case BadgeInfo:
		style = style.Foreground(t.Palette.Accent)
	case BadgeMuted:
		style = style.Foreground(t.Palette.TextDim)
	default:
		style = style.Foreground(t.Palette.Text)
	}

	return style.Render(b.Text)
}

// StatusBadge creates a badge from VM state.
func StatusBadge(state string) Badge {
	switch state {
	case "running":
		return Badge{Text: "●", Variant: BadgeSuccess}
	case "stopped", "shutoff":
		return Badge{Text: "○", Variant: BadgeMuted}
	case "paused":
		return Badge{Text: "◐", Variant: BadgeWarning}
	case "error":
		return Badge{Text: "✕", Variant: BadgeError}
	default:
		return Badge{Text: "?", Variant: BadgeMuted}
	}
}

// StateBadge creates a colored status indicator.
func StateBadge(state string) string {
	return StatusBadge(state).Render()
}
