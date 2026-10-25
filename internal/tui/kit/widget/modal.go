package widget

import (
	"github.com/charmbracelet/x/ansi"
	"strings"

	"github.com/Josepavese/nido/v4/internal/tui/kit/layout"
	"github.com/Josepavese/nido/v4/internal/tui/kit/theme"
	"github.com/Josepavese/nido/v4/internal/tui/kit/view"
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

	confirmHit, cancelHit layout.Rect
	messageView           ScrollView

	// Configuration
	SingleButton bool // If true, shows only one button (OK/Close)
	width        int
	height       int

	BorderColor  lipgloss.TerminalColor
	MessageAlign lipgloss.Position

	Level ModalLevel
}

// ModalLevel defines the semantic severity of the modal
type ModalLevel int

const (
	ModalLevelNormal  ModalLevel = iota // Default: Uses Accent color
	ModalLevelDanger                    // Destructive: Uses Error color
	ModalLevelSuccess                   // Positive: Uses Success color
)

func NewModal(title, message string, onConfirm, onCancel func() tea.Cmd) *Modal {
	return &Modal{
		Title:        title,
		Message:      message,
		OnConfirm:    onConfirm,
		OnCancel:     onCancel,
		width:        layout.ModalWidth,
		height:       layout.ModalHeight,
		MessageAlign: lipgloss.Center,
		Level:        ModalLevelNormal,
	}
}

func NewAlertModal(title, message string, onDismiss func() tea.Cmd) *Modal {
	return &Modal{
		Title:        title,
		Message:      message,
		OnConfirm:    onDismiss, // We treat Dismiss as Confirm (OK) behavior
		SingleButton: true,
		Level:        ModalLevelDanger,
		width:        layout.AlertWidth, // Slightly wider for error messages
		height:       layout.ModalHeight,
		MessageAlign: lipgloss.Center,
	}
}

func (m *Modal) SetWidth(w int)        { m.width = w }
func (m *Modal) SetHeight(h int)       { m.height = h }
func (m *Modal) SetLevel(l ModalLevel) { m.Level = l }

func (m *Modal) Show() {
	m.active = true
	m.messageView = ScrollView{}
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
	case tea.MouseMsg:
		cmd, _ := m.HandleMouse(msg.X, msg.Y, msg)
		return m, cmd
	case tea.KeyMsg:
		switch msg.String() {
		case "pgup":
			m.messageView.Scroll(-m.messageView.height)
		case "pgdown":
			m.messageView.Scroll(m.messageView.height)
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
	color := m.BorderColor
	if color == nil {
		color = t.Palette.Accent
		if m.Level == ModalLevelDanger {
			color = t.Palette.Error
		}
		if m.Level == ModalLevelSuccess {
			color = t.Palette.Success
		}
	}
	// Wrap before measuring; never crop away confirmation/cancellation controls.
	d := measureDialog(parentWidth, parentHeight, m.width, parentHeight)
	message := ansi.Wrap(m.Message, max(1, d.inner.Width), "")
	naturalH := lipgloss.Height(message) + layout.DialogBorder + layout.DialogPaddingY*2 + layout.DialogHeadingHeight + layout.DialogControlsHeight + theme.Space.SM
	d = measureDialog(parentWidth, parentHeight, m.width, max(m.height, naturalH))
	title := t.Styles.Title.Foreground(color).Render(ansi.Truncate(m.Title, d.inner.Width, "…"))
	buttonStyle := lipgloss.NewStyle().Foreground(t.Palette.TextDim).Padding(0, 1)
	selectedStyle := buttonStyle.Foreground(t.Palette.Background).Background(color).Bold(true)
	yes, no := buttonStyle, selectedStyle
	if m.selectedIsYes {
		yes, no = selectedStyle, buttonStyle
	}
	var buttons string
	confirmW, cancelW := 0, 0
	if m.SingleButton {
		buttons = selectedStyle.Render("Close")
		confirmW = lipgloss.Width(buttons)
	} else {
		confirm := yes.Render("Yes")
		cancel := no.Render("No")
		confirmW, cancelW = lipgloss.Width(confirm), lipgloss.Width(cancel)
		buttons = confirm + strings.Repeat(" ", layout.DialogButtonGap) + cancel
	}
	messageH := max(0, d.inner.Height-layout.DialogHeadingHeight-layout.DialogControlsHeight)
	body := m.messageView.Render(d.inner.Width, messageH, lipgloss.NewStyle().Width(d.inner.Width).Align(m.MessageAlign).Render(t.Styles.TextDim.Render(message)), layout.Rect{})
	content := lipgloss.JoinVertical(lipgloss.Center, layout.Center(d.inner.Width, title), body, layout.Center(d.inner.Width, buttons))
	btnX := d.inner.X + (d.inner.Width-lipgloss.Width(buttons))/2
	btnY := d.inner.Y + d.inner.Height - 1
	m.confirmHit = layout.NewRect(btnX, btnY, confirmW, 1)
	m.cancelHit = layout.NewRect(btnX+confirmW+layout.DialogButtonGap, btnY, cancelW, 1)
	return d.render(parentWidth, parentHeight, content, color)
}
func (m *Modal) HandleMouse(x, y int, msg tea.MouseMsg) (tea.Cmd, bool) {
	if !m.active {
		return nil, false
	}
	if msg.Action == tea.MouseActionPress {
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.messageView.Scroll(-1)
			return nil, true
		case tea.MouseButtonWheelDown:
			m.messageView.Scroll(1)
			return nil, true
		case tea.MouseButtonLeft:
			if m.confirmHit.Contains(x, y) {
				m.Hide()
				if m.OnConfirm != nil {
					return m.OnConfirm(), true
				}
			} else if m.cancelHit.Contains(x, y) {
				m.Hide()
				if m.OnCancel != nil {
					return m.OnCancel(), true
				}
			}
		}
	}
	return nil, true
}

func (m *Modal) Shortcuts() []view.Shortcut {
	if m.SingleButton {
		return []view.Shortcut{{Key: "enter/esc", Label: "close"}, {Key: "pgup/pgdn", Label: "scroll"}}
	}
	return []view.Shortcut{{Key: "tab", Label: "choose"}, {Key: "enter", Label: "select"}, {Key: "esc", Label: "cancel"}, {Key: "pgup/pgdn", Label: "scroll"}}
}
