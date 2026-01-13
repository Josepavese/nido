package widget

import (
	"time"

	"github.com/Josepavese/nido/internal/tui/kit/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ToastVariant defines the toast style.
type ToastVariant int

const (
	ToastInfo ToastVariant = iota
	ToastSuccess
	ToastWarning
	ToastError
)

// Toast represents a temporary notification message.
type Toast struct {
	Message  string
	Variant  ToastVariant
	Duration time.Duration
	visible  bool
}

// ToastDismissMsg signals that a toast should be hidden.
type ToastDismissMsg struct{}

// NewToast creates a toast with default 3s duration.
func NewToast(message string, variant ToastVariant) Toast {
	return Toast{
		Message:  message,
		Variant:  variant,
		Duration: 3 * time.Second,
		visible:  true,
	}
}

// Show returns a command to display and auto-dismiss the toast.
func (t Toast) Show() tea.Cmd {
	return tea.Tick(t.Duration, func(time.Time) tea.Msg {
		return ToastDismissMsg{}
	})
}

// View renders the toast if visible.
func (t Toast) View() string {
	if !t.visible {
		return ""
	}

	th := theme.Current()
	style := lipgloss.NewStyle().
		Padding(0, 2).
		Bold(true)

	switch t.Variant {
	case ToastSuccess:
		style = style.Foreground(th.Palette.Success)
	case ToastWarning:
		style = style.Foreground(th.Palette.Warning)
	case ToastError:
		style = style.Foreground(th.Palette.Error)
	default:
		style = style.Foreground(th.Palette.Accent)
	}

	icon := "ℹ"
	switch t.Variant {
	case ToastSuccess:
		icon = "✓"
	case ToastWarning:
		icon = "⚠"
	case ToastError:
		icon = "✕"
	}

	return style.Render(icon + " " + t.Message)
}

// Visible returns whether the toast is currently shown.
func (t Toast) Visible() bool {
	return t.visible
}

// Dismiss hides the toast.
func (t *Toast) Dismiss() {
	t.visible = false
}
