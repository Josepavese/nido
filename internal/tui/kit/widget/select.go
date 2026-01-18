package widget

import (
	"github.com/Josepavese/nido/internal/tui/kit/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Select is a form widget that triggers an action (like opening a modal)
// It looks like an Input but is read-only and has a chevron.
type Select struct {
	Label    string
	Value    string
	focused  bool
	Disabled bool

	// OnActivate is called when the user hits Enter while focused
	OnActivate func() tea.Cmd
	Multiline  bool
}

func NewSelect(label, value string, onActivate func() tea.Cmd) *Select {
	return &Select{
		Label:      label,
		Value:      value,
		OnActivate: onActivate,
	}
}

func (s *Select) Update(msg tea.Msg) (Element, tea.Cmd) {
	if s.Disabled {
		return s, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if s.focused {
			switch msg.String() {
			case "enter", "space":
				if s.OnActivate != nil {
					return s, s.OnActivate()
				}
			}
		}
	}
	return s, nil
}

func (s *Select) Focus() tea.Cmd {
	s.focused = true
	return nil
}

func (s *Select) Blur() {
	s.focused = false
}

func (s *Select) Focused() bool {
	return s.focused
}

func (s *Select) Focusable() bool {
	return !s.Disabled
}

func (s *Select) HandleMouse(x int, y int, msg tea.MouseMsg) (tea.Cmd, bool) {
	if s.Disabled {
		return nil, false
	}
	// For now, assume if the mouse click reached this element (via Form delegation),
	// it is within bounds.
	// We check for Left Click.
	if msg.Type == tea.MouseLeft {
		s.focused = true
		if s.OnActivate != nil {
			return s.OnActivate(), true
		}
		return nil, true
	}
	return nil, false
}

func (s *Select) SetWidth(w int) {
	// Optional width handling
}

func (s *Select) View(width int) string {
	t := theme.Current()
	var borderColor lipgloss.TerminalColor = nil
	if s.focused {
		borderColor = t.Palette.Focus
	}

	// Apply Value style to text and chevron
	val := t.Styles.Value.Render(s.Value + " ▼")
	return RenderBoxedField(
		s.Label,
		val,
		"",
		s.focused,
		width,
		lipgloss.Left,
		borderColor,
		s.Multiline,
	)
}

func (s *Select) SetValue(v string) {
	s.Value = v
}
