package widget

import (
	"github.com/Josepavese/nido/internal/tui/kit/layout"
	"github.com/Josepavese/nido/internal/tui/kit/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ValidatorFunc returns an error if the input is invalid.
type ValidatorFunc func(string) error

// FormEntry represents a single input field definition.
type FormEntry struct {
	Key         string        // Unique identifier for result map
	Label       string        // Display label (above input)
	Value       string        // Initial/Current value
	Placeholder string        // Phantom text when empty
	Validator   ValidatorFunc // logic to validate on change/submit
	Width       int           // Optional fixed width (chars) for weighting
	MaxChars    int           // Optional character limit
	Filter      FilterFunc    // Optional: Block invalid characters immediately
}

// FormModal represents the state of the modal using Kit widgets.
type FormModal struct {
	Title       string
	Description string

	// Callbacks
	OnConfirm func(map[string]string) tea.Cmd
	OnCancel  func() tea.Cmd

	// Internal Kit Components
	form    *Form
	inputs  map[string]*Input
	rows    []Element // Stored to rebuild form if needed
	buttons *Row      // The action buttons row

	// State
	active bool
	width  int
	err    error
}

// NewFormModal creates a new form modal.
func NewFormModal(title string, onConfirm func(map[string]string) tea.Cmd, onCancel func() tea.Cmd) *FormModal {
	m := &FormModal{
		Title:     title,
		OnConfirm: onConfirm,
		OnCancel:  onCancel,
		width:     74,
		inputs:    make(map[string]*Input),
	}

	// Create Buttons (Confirm / Cancel)
	// We use a Row for them to handle Left/Right navigation
	confirmBtn := NewSubmitButton("", "CONFIRM", m.submit) // Empty label -> Centered
	confirmBtn.Centered = true                             // Force center style

	cancelBtn := NewButton("", "Cancel", m.cancel)
	cancelBtn.Centered = true
	cancelBtn.Role = RoleCancel // Styling hint

	// Button Row with weighting? Or just equal?
	// Let's use weights to give Confirm more prominence if needed, or equal.
	// Equal is fine for buttons side-by-side.
	m.buttons = NewRow(confirmBtn, cancelBtn)

	return m
}

// AddRow adds a row of entries to the modal.
// It converts FormEntry definitions into widget.Input and wraps them in a widget.Row.
func (m *FormModal) AddRow(entries ...*FormEntry) {
	var elements []Element
	var weights []int
	hasWeights := false

	for _, e := range entries {
		// Create Input Widget
		input := NewInput(e.Label, e.Placeholder, e.Validator)
		input.SetValue(e.Value)
		input.Filter = e.Filter // Apply filter
		if e.MaxChars > 0 {
			input.SetCharLimit(e.MaxChars)
		}
		// We don't set Width on Input directly here, Form/Row will handle it.
		// But we use e.Width for Row weighting.

		m.inputs[e.Key] = input
		elements = append(elements, input)

		if e.Width > 0 {
			weights = append(weights, e.Width)
			hasWeights = true
		} else {
			weights = append(weights, 1) // Default weight
		}
	}

	if len(elements) == 0 {
		return
	}

	// Wrap in Row if multiple or just generic consistency
	var rowElement Element
	if len(elements) > 1 {
		if hasWeights {
			rowElement = NewRowWithWeights(elements, weights)
		} else {
			rowElement = NewRow(elements...)
		}
	} else {
		// Single element, just add directly?
		// User requested "frecce verticali" behavior.
		// Form handles vertical.
		// If we wrap single element in Row, Row might trap Left/Right.
		// But that's fine.
		rowElement = elements[0]
	}

	m.rows = append(m.rows, rowElement)
	m.rebuildForm()
}

func (m *FormModal) rebuildForm() {
	// Assemble Form: [Rows..., ButtonRow]
	allElements := append([]Element{}, m.rows...)
	allElements = append(allElements, m.buttons)

	m.form = NewForm(allElements...)
	m.form.Spacing = 0
	// Width is managed by View() injection
}

// Show resets state and activates
func (m *FormModal) Show() tea.Cmd {
	m.active = true
	m.err = nil
	// Reset inputs to show placeholders
	for _, input := range m.inputs {
		input.SetValue("")
	}
	m.form.Blur()             // Ensure clean slate
	m.form.FocusIndex = -1    // Reset
	return m.form.NextField() // Select first and return focus command
}

// Hide closes the modal
func (m *FormModal) Hide() {
	m.active = false
	m.form.Blur()
}

// IsActive returns whether the modal is open
func (m *FormModal) IsActive() bool {
	return m.active
}

func (m *FormModal) cancel() tea.Cmd {
	m.Hide()
	if m.OnCancel != nil {
		return m.OnCancel()
	}
	return nil
}

func (m *FormModal) submit() tea.Cmd {
	// Form Validate handled by Button RoleSubmit usually,
	// but here we are in a custom closure.
	if !m.form.Validate() {
		return nil // Visual errors updated
	}

	// Collect results
	res := make(map[string]string)
	for key, input := range m.inputs {
		res[key] = input.Value()
	}

	m.Hide()
	if m.OnConfirm != nil {
		return m.OnConfirm(res)
	}
	return nil
}

func (m *FormModal) Update(msg tea.Msg) (*FormModal, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	// 1. Global Shortcuts
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, m.cancel()
		case "q":
			// Quit app if not typing
			if !m.form.HasActiveTextInput() {
				return m, tea.Quit
			}
		}
	}

	// 2. Delegate to Form
	// Form handles Tab/Shift+Tab, Up/Down, Enter (on buttons)
	var cmd tea.Cmd
	// We need to re-wrap Form because Update returns generic Element (interface)
	// mostly, but here m.form is *Form.
	// But Form.Update returns (*Form, Cmd)
	newForm, cmd := m.form.Update(msg)
	m.form = newForm

	return m, cmd
}

func (m *FormModal) View(parentWidth, parentHeight int) string {
	if !m.active {
		return ""
	}

	t := theme.Current()

	// 1. Content Rendering
	var content []string

	// Title
	titleStyle := t.Styles.Title.Foreground(t.Palette.Accent)
	content = append(content, titleStyle.Render(m.Title), "")

	// Description
	if m.Description != "" {
		content = append(content, t.Styles.TextDim.Render(m.Description), "")
	}

	// Form
	// Constrain width inside modal
	formWidth := m.width - 4 // Padding
	content = append(content, m.form.View(formWidth))

	// Frame
	inner := lipgloss.JoinVertical(lipgloss.Center, content...)

	dialogStyle := t.Styles.Border.
		BorderForeground(t.Palette.Accent).
		Padding(1, 2).
		Width(m.width).
		Align(lipgloss.Center)

	dialog := dialogStyle.Render(inner)

	return layout.PlaceOverlay(parentWidth, parentHeight, dialog)
}
