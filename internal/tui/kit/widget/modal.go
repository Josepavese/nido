package widget

import (
	"github.com/Josepavese/nido/internal/tui/kit/layout"
	"github.com/Josepavese/nido/internal/tui/kit/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Modal represents a popup dialog for confirmation or blocking interactions.
// Modal represents a popup dialog for confirmation or blocking interactions.
type Modal struct {
	Title     string
	Message   string
	OnConfirm func() tea.Cmd
	OnCancel  func() tea.Cmd

	active        bool
	selectedIsYes bool // true = Yes/Confirm, false = No/Cancel

	// Configuration
	SingleButton bool // If true, shows only one button (OK/Close)
	width        int
	height       int
}

func NewModal(title, message string, onConfirm, onCancel func() tea.Cmd) *Modal {
	return &Modal{
		Title:     title,
		Message:   message,
		OnConfirm: onConfirm,
		OnCancel:  onCancel,
		width:     50,
		height:    10,
	}
}

func NewAlertModal(title, message string, onDismiss func() tea.Cmd) *Modal {
	return &Modal{
		Title:        title,
		Message:      message,
		OnConfirm:    onDismiss, // We treat Dismiss as Confirm (OK) behavior
		SingleButton: true,
		width:        60, // Slightly wider for error messages
		height:       10,
	}
}

func (m *Modal) Show() {
	m.active = true
	m.selectedIsYes = false // Default to safe option (No/Cancel), unless SingleButton
	if m.SingleButton {
		m.selectedIsYes = true // Only one option, so it's selected
	}
}

func (m *Modal) Hide() {
	m.active = false
}

func (m *Modal) IsActive() bool {
	return m.active
}

func (m *Modal) Update(msg tea.Msg) (*Modal, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "right", "h", "l", "tab", "shift+tab":
			if !m.SingleButton {
				m.selectedIsYes = !m.selectedIsYes
			}
		case "enter", " ":
			m.Hide()
			if m.SingleButton {
				if m.OnConfirm != nil {
					return m, m.OnConfirm()
				}
				return m, nil
			}

			if m.selectedIsYes {
				if m.OnConfirm != nil {
					return m, m.OnConfirm()
				}
			} else {
				if m.OnCancel != nil {
					return m, m.OnCancel()
				}
			}
		case "esc":
			m.Hide()
			if m.SingleButton {
				if m.OnConfirm != nil { // Alerts close with OK/Dismiss logic on Esc too
					return m, m.OnConfirm()
				}
			} else {
				if m.OnCancel != nil {
					return m, m.OnCancel()
				}
			}
		}
	}
	return m, nil
}

func (m *Modal) View(parentWidth, parentHeight int) string {
	if !m.active {
		return ""
	}

	t := theme.Current()

	// 1. Content Styling
	borderColor := t.Palette.Accent
	if m.SingleButton {
		borderColor = t.Palette.Error // Alerts usually imply errors/warnings
	}

	horizontalPadding := 4 // 2 left + 2 right
	horizontalBorder := 2  // 1 left + 1 right

	// Dynamic Width Adjustment (Responsive)
	// If parent is smaller than desired width, shrink to fit parent - margin
	effectiveWidth := m.width
	if parentWidth > 0 && effectiveWidth > parentWidth-2 {
		effectiveWidth = parentWidth - 2
		if effectiveWidth < 20 { // Minimum viable width
			effectiveWidth = 20
		}
	}

	contentWidth := effectiveWidth - horizontalPadding - horizontalBorder
	if contentWidth < 1 {
		contentWidth = 1
	}

	dialogStyle := t.Styles.Border.Copy().
		BorderForeground(borderColor).
		Padding(1, 2).
		Width(contentWidth).
		Align(lipgloss.Center)

	// 2. Buttons
	buttons := ""
	if m.SingleButton {
		okStyle := lipgloss.NewStyle().Foreground(t.Palette.Background).Background(t.Palette.Error).Bold(true).Padding(0, 2)
		buttons = okStyle.Render("Close")
	} else {
		yesStyle := lipgloss.NewStyle().Foreground(t.Palette.TextDim)
		noStyle := lipgloss.NewStyle().Foreground(t.Palette.TextDim)

		activeStyle := lipgloss.NewStyle().Foreground(t.Palette.Background).Background(t.Palette.Accent).Bold(true).Padding(0, 1)

		if m.selectedIsYes {
			yesStyle = activeStyle
		} else {
			noStyle = activeStyle
		}

		buttons = lipgloss.JoinHorizontal(lipgloss.Center,
			yesStyle.Render("Yes"),
			"   ",
			noStyle.Render("No"),
		)
	}

	// 3. Layout
	titleStyle := t.Styles.Title.Copy()
	if m.SingleButton {
		titleStyle = titleStyle.Foreground(t.Palette.Error)
	}

	content := lipgloss.JoinVertical(lipgloss.Center,
		titleStyle.Render(m.Title),
		"",
		t.Styles.Text.Render(m.Message),
		"",
		buttons,
	)

	dialog := dialogStyle.Render(content)

	// 4. Overlay Center
	if parentWidth == 0 || parentHeight == 0 {
		return dialog
	}

	return layout.PlaceOverlay(parentWidth, parentHeight, dialog)
}
