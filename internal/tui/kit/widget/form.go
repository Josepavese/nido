package widget

import (
	"github.com/Josepavese/nido/internal/tui/kit/theme"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FormField represents a single input field with label and validation.
type FormField struct {
	Label       string
	Placeholder string
	Input       textinput.Model
	Validator   func(string) error
	Error       string
	focused     bool
}

// NewFormField creates a new form field.
func NewFormField(label, placeholder string, validator func(string) error) FormField {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Prompt = ""
	ti.CharLimit = 100

	return FormField{
		Label:       label,
		Placeholder: placeholder,
		Input:       ti,
		Validator:   validator,
	}
}

// Focus gives the field keyboard focus.
func (f *FormField) Focus() tea.Cmd {
	f.focused = true
	return f.Input.Focus()
}

// Blur removes keyboard focus.
func (f *FormField) Blur() {
	f.focused = false
	f.Input.Blur()
}

// Focused returns whether the field has focus.
func (f FormField) Focused() bool {
	return f.focused
}

// Value returns the current input value.
func (f FormField) Value() string {
	return f.Input.Value()
}

// SetValue sets the input value.
func (f *FormField) SetValue(s string) {
	f.Input.SetValue(s)
}

// Validate runs the validator and updates the error state.
func (f *FormField) Validate() bool {
	if f.Validator == nil {
		f.Error = ""
		return true
	}

	err := f.Validator(f.Input.Value())
	if err != nil {
		f.Error = err.Error()
		return false
	}

	f.Error = ""
	return true
}

// Update handles messages for the field.
func (f FormField) Update(msg tea.Msg) (FormField, tea.Cmd) {
	var cmd tea.Cmd
	f.Input, cmd = f.Input.Update(msg)

	// Real-time validation
	if f.Validator != nil {
		f.Validate()
	}

	return f, cmd
}

// View renders the form field.
func (f FormField) View() string {
	t := theme.Current()

	labelStyle := lipgloss.NewStyle().
		Foreground(t.Palette.TextDim).
		Width(theme.Width.Label)

	if f.focused {
		labelStyle = labelStyle.
			Foreground(t.Palette.Accent).
			Bold(true)
	}

	inputStyle := lipgloss.NewStyle().
		Foreground(t.Palette.Text)

	errorStyle := lipgloss.NewStyle().
		Foreground(t.Palette.Error)

	label := labelStyle.Render(f.Label)
	input := inputStyle.Render(f.Input.View())

	result := lipgloss.JoinHorizontal(lipgloss.Top, label, input)

	if f.Error != "" {
		errorMsg := errorStyle.Render("⚠ " + f.Error)
		result = lipgloss.JoinVertical(lipgloss.Left, result, errorMsg)
	}

	return result
}

// Form manages a collection of FormFields.
type Form struct {
	Fields     []FormField
	FocusIndex int
}

// NewForm creates a form with the given fields.
func NewForm(fields ...FormField) Form {
	return Form{
		Fields:     fields,
		FocusIndex: 0,
	}
}

// Focus focuses the current field.
func (f *Form) Focus() tea.Cmd {
	if len(f.Fields) == 0 {
		return nil
	}
	return f.Fields[f.FocusIndex].Focus()
}

// NextField moves focus to the next field.
func (f *Form) NextField() tea.Cmd {
	if len(f.Fields) == 0 {
		return nil
	}

	f.Fields[f.FocusIndex].Blur()
	f.FocusIndex = (f.FocusIndex + 1) % len(f.Fields)
	return f.Fields[f.FocusIndex].Focus()
}

// PrevField moves focus to the previous field.
func (f *Form) PrevField() tea.Cmd {
	if len(f.Fields) == 0 {
		return nil
	}

	f.Fields[f.FocusIndex].Blur()
	f.FocusIndex--
	if f.FocusIndex < 0 {
		f.FocusIndex = len(f.Fields) - 1
	}
	return f.Fields[f.FocusIndex].Focus()
}

// Validate validates all fields and returns true if all pass.
func (f *Form) Validate() bool {
	valid := true
	for i := range f.Fields {
		if !f.Fields[i].Validate() {
			valid = false
		}
	}
	return valid
}

// Update handles messages for the form.
func (f Form) Update(msg tea.Msg) (Form, tea.Cmd) {
	if len(f.Fields) == 0 {
		return f, nil
	}

	var cmd tea.Cmd
	f.Fields[f.FocusIndex], cmd = f.Fields[f.FocusIndex].Update(msg)
	return f, cmd
}

// View renders all form fields vertically.
func (f Form) View() string {
	var fields []string
	for _, field := range f.Fields {
		fields = append(fields, field.View())
	}
	return lipgloss.JoinVertical(lipgloss.Left, fields...)
}
