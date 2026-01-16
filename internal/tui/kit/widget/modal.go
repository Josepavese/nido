package widget

import (
	"strings"

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

	// Layout state for hit detection
	lastW, lastH    int
	yesX, noX, btnY int
	okX, okY        int

	// Configuration
	SingleButton bool // If true, shows only one button (OK/Close)
	width        int
	height       int

	BorderColor  lipgloss.TerminalColor
	MessageAlign lipgloss.Position
}

func NewModal(title, message string, onConfirm, onCancel func() tea.Cmd) *Modal {
	return &Modal{
		Title:        title,
		Message:      message,
		OnConfirm:    onConfirm,
		OnCancel:     onCancel,
		width:        50,
		height:       10,
		MessageAlign: lipgloss.Center,
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
		MessageAlign: lipgloss.Center,
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
	borderColor := m.BorderColor
	if borderColor == nil {
		borderColor = t.Palette.Accent
		if m.SingleButton {
			borderColor = t.Palette.Error // Alerts usually imply errors/warnings
		}
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
		Width(effectiveWidth - horizontalBorder).
		Align(lipgloss.Center)

	// 2. Buttons
	buttons := ""
	if m.SingleButton {
		okStyle := lipgloss.NewStyle().Foreground(t.Palette.Background).Background(t.Palette.Error).Bold(true).Padding(0, 2)
		buttons = okStyle.Render("Close")
		m.btnY = 2 + (strings.Count(m.Message, "\n") + 1) + 1
		m.okX = (effectiveWidth - lipgloss.Width(buttons)) / 2
	} else {
		yesStyle := lipgloss.NewStyle().Foreground(t.Palette.TextDim)
		noStyle := lipgloss.NewStyle().Foreground(t.Palette.TextDim)

		activeStyle := lipgloss.NewStyle().Foreground(t.Palette.Background).Background(t.Palette.Accent).Bold(true).Padding(0, 1)

		if m.selectedIsYes {
			yesStyle = activeStyle
		} else {
			noStyle = activeStyle
		}

		yesLabel := "Yes"
		noLabel := "No"
		yesWidth := lipgloss.Width(yesStyle.Render(yesLabel))
		noWidth := lipgloss.Width(noStyle.Render(noLabel))
		spacing := 3

		buttons = lipgloss.JoinHorizontal(lipgloss.Center,
			yesStyle.Render(yesLabel),
			strings.Repeat(" ", spacing),
			noStyle.Render(noLabel),
		)

		// Record relative positions for mouse
		totalBtnsWidth := yesWidth + spacing + noWidth
		m.yesX = (effectiveWidth - totalBtnsWidth) / 2
		m.noX = m.yesX + yesWidth + spacing
		m.btnY = 2 + (strings.Count(m.Message, "\n") + 1) + 1
	}

	// 3. Layout
	titleStyle := t.Styles.Title.Copy()
	if m.SingleButton {
		titleStyle = titleStyle.Foreground(t.Palette.Error)
	}

	content := lipgloss.JoinVertical(lipgloss.Center,
		titleStyle.Render(m.Title),
		"",
		lipgloss.PlaceHorizontal(contentWidth, m.MessageAlign, t.Styles.Text.Render(m.Message)),
		"",
		buttons,
	)

	dialog := dialogStyle.Render(content)

	m.lastW = parentWidth
	m.lastH = parentHeight

	return layout.PlaceOverlay(parentWidth, parentHeight, dialog)
}

func (m *Modal) HandleMouse(x, y int, msg tea.MouseMsg) (tea.Cmd, bool) {
	if !m.active || msg.Type != tea.MouseLeft {
		return nil, false
	}

	effectiveWidth := m.width
	if m.lastW > 0 && effectiveWidth > m.lastW-2 {
		effectiveWidth = m.lastW - 2
	}

	dialogX := (m.lastW - effectiveWidth) / 2
	dialogY := (m.lastH - m.height) / 2

	lx := x - dialogX
	ly := y - dialogY

	if lx < 0 || ly < 0 || lx >= effectiveWidth || ly >= m.height {
		return nil, false
	}

	if ly >= m.btnY-1 && ly <= m.btnY+1 {
		if m.SingleButton {
			m.Hide()
			if m.OnConfirm != nil {
				return m.OnConfirm(), true
			}
			return nil, true
		} else {
			if lx >= m.yesX-1 && lx <= m.yesX+4 {
				m.selectedIsYes = true
				m.Hide()
				if m.OnConfirm != nil {
					return m.OnConfirm(), true
				}
				return nil, true
			}
			if lx >= m.noX-1 && lx <= m.noX+4 {
				m.selectedIsYes = false
				m.Hide()
				if m.OnCancel != nil {
					return m.OnCancel(), true
				}
				return nil, true
			}
		}
	}

	return nil, true
}
