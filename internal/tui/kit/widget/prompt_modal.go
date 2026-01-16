package widget

import (
	"strings"

	"github.com/Josepavese/nido/internal/tui/kit/layout"
	"github.com/Josepavese/nido/internal/tui/kit/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PromptModal is a reusable dialog with a title, message, and a single input field.
type PromptModal struct {
	Title    string
	Message  string
	Input    *Input
	OnSubmit func(string) tea.Cmd
	OnCancel func() tea.Cmd

	active bool
	width  int
}

// NewPromptModal creates a new specialized modal for single-value input.
func NewPromptModal(title, message, label, placeholder string) *PromptModal {
	return &PromptModal{
		Title:   title,
		Message: message,
		Input:   NewInput(label, placeholder, nil),
		width:   50,
	}
}

// Show opens the modal and focuses the input.
func (m *PromptModal) Show(initialValue string) tea.Cmd {
	m.active = true
	m.Input.SetValue(initialValue)
	m.Input.Error = ""
	return m.Input.Focus()
}

// Hide closes the modal.
func (m *PromptModal) Hide() {
	m.active = false
	m.Input.Blur()
}

// IsActive returns the current state.
func (m *PromptModal) IsActive() bool {
	return m.active
}

// Update handles input events.
func (m *PromptModal) Update(msg tea.Msg) (*PromptModal, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			m.Hide()
			if m.OnCancel != nil {
				return m, m.OnCancel()
			}
			return m, nil

		case "enter":
			// Validate if a validator is present
			if m.Input.Validator != nil {
				if err := m.Input.Validator(m.Input.Value()); err != nil {
					m.Input.Error = err.Error()
					return m, nil
				}
			}
			val := m.Input.Value()
			m.Hide()
			if m.OnSubmit != nil {
				return m, m.OnSubmit(val)
			}
			return m, nil
		}
	}

	_, cmd := m.Input.Update(msg)
	return m, cmd
}

// View renders the modal with an overlay.
func (m *PromptModal) View(parentWidth, parentHeight int) string {
	if !m.active {
		return ""
	}

	t := theme.Current()

	// 1. Style
	dialogStyle := t.Styles.Border.Copy().
		BorderForeground(t.Palette.Accent).
		Padding(1, 2).
		Width(m.width).
		Align(lipgloss.Center)

	// 2. Content
	titleView := t.Styles.Title.Render(strings.ToUpper(m.Title))
	messageView := t.Styles.Text.Render(m.Message)

	// Input (Width - Padding(4) - Border(2))
	inputView := m.Input.View(m.width - 6)

	// Hints
	hints := t.Styles.TextMuted.Copy().Italic(true).Render("Enter to confirm • Esc to cancel")

	content := lipgloss.JoinVertical(lipgloss.Center,
		titleView,
		" ",
		messageView,
		" ",
		inputView,
		" ",
		hints,
	)

	dialog := dialogStyle.Render(content)

	// 3. Overlay
	if parentWidth == 0 || parentHeight == 0 {
		return dialog
	}
	return layout.PlaceOverlay(parentWidth, parentHeight, dialog)
}
