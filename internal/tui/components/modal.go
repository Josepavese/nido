package components

import (
	"github.com/Josepavese/nido/internal/tui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Modal represents a centered overlay dialog.
type Modal struct {
	Title   string
	Content string
	Actions []ModalAction
	visible bool
	width   int
}

// ModalAction represents a button in the modal.
type ModalAction struct {
	Label    string
	Key      string
	Primary  bool
	Callback func() tea.Cmd
}

// NewModal creates a new modal dialog.
func NewModal(title string, width int) Modal {
	return Modal{
		Title: title,
		width: width,
	}
}

// Show makes the modal visible.
func (m *Modal) Show() {
	m.visible = true
}

// Hide makes the modal invisible.
func (m *Modal) Hide() {
	m.visible = false
}

// Visible returns whether the modal is shown.
func (m Modal) Visible() bool {
	return m.visible
}

// SetContent updates the modal body text.
func (m *Modal) SetContent(content string) {
	m.Content = content
}

// SetActions sets the modal buttons.
func (m *Modal) SetActions(actions []ModalAction) {
	m.Actions = actions
}

// View renders the modal if visible.
func (m Modal) View() string {
	if !m.visible {
		return ""
	}

	t := theme.Current()

	// Title bar
	titleStyle := lipgloss.NewStyle().
		Foreground(t.Palette.AccentStrong).
		Bold(true).
		Width(m.width).
		Align(lipgloss.Center).
		Padding(0, 1)

	// Content area
	contentStyle := lipgloss.NewStyle().
		Foreground(t.Palette.Text).
		Width(m.width).
		Padding(1, 2)

	// Action buttons
	buttonStyle := lipgloss.NewStyle().
		Foreground(t.Palette.TextDim).
		Padding(0, 2)

	primaryButtonStyle := buttonStyle.
		Foreground(t.Palette.Accent).
		Bold(true)

	// Build action row
	var actions string
	for i, action := range m.Actions {
		if i > 0 {
			actions += "  "
		}
		btn := buttonStyle
		if action.Primary {
			btn = primaryButtonStyle
		}
		actions += btn.Render("[" + action.Key + "] " + action.Label)
	}

	actionsRow := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		Render(actions)

	// Combine sections
	inner := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(m.Title),
		contentStyle.Render(m.Content),
		actionsRow,
	)

	// Modal border
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Palette.SurfaceSubtle).
		Padding(1, 0)

	return modalStyle.Render(inner)
}

// ConfirmModal creates a confirmation dialog.
func ConfirmModal(title, message string) Modal {
	m := NewModal(title, 40)
	m.SetContent(message)
	m.SetActions([]ModalAction{
		{Label: "Cancel", Key: "esc"},
		{Label: "Confirm", Key: "enter", Primary: true},
	})
	return m
}

// AlertModal creates an alert dialog with single OK button.
func AlertModal(title, message string) Modal {
	m := NewModal(title, 40)
	m.SetContent(message)
	m.SetActions([]ModalAction{
		{Label: "OK", Key: "enter", Primary: true},
	})
	return m
}
