package widget

import (
	"github.com/Josepavese/nido/internal/tui/kit/layout"
	"github.com/Josepavese/nido/internal/tui/kit/theme"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Element is the interface for all form components.
type Element interface {
	View(width int) string
	Update(msg tea.Msg) (Element, tea.Cmd)
	Focus() tea.Cmd
	Blur()
	Focused() bool
	Focusable() bool
	SetWidth(int)
}

// --- 1. Card Element (Moved to card.go) ---

// --- 2. Input Element (Wrapper for textinput) ---
type Input struct {
	Label      string
	InputModel textinput.Model
	Validator  func(string) error
	Error      string
	Disabled   bool // New: Read-only mode
	Compact    bool // New: Render without borders (single line)
	focused    bool
	width      int
}

func NewInput(label, placeholder string, validator func(string) error) *Input {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Prompt = ""
	ti.CharLimit = 256
	return &Input{Label: label, InputModel: ti, Validator: validator}
}
func (i *Input) Focusable() bool { return !i.Disabled }
func (i *Input) Focused() bool   { return i.focused }
func (i *Input) Focus() tea.Cmd {
	if i.Disabled {
		return nil
	}
	i.focused = true
	return i.InputModel.Focus()
}
func (i *Input) Blur()             { i.focused = false; i.InputModel.Blur() }
func (i *Input) SetWidth(w int)    { i.width = w; i.updateInputWidth() }
func (i *Input) Value() string     { return i.InputModel.Value() }
func (i *Input) SetValue(s string) { i.InputModel.SetValue(s) }
func (i *Input) updateInputWidth() {
	// Sync strictly with RenderBoxedField logic
	labelWidth := theme.Width.Label
	if i.width > 0 && i.width < 34 { // Increased threshold slightly for label compatibility
		labelWidth = 10 // Compact
	}

	// Width - Box(4) - Label - ValidationPadding(1ish)
	// If Error is present, Validation adds " 🔺" (len 2)
	validationWidth := 1 // Default safety padding
	if i.Error != "" {
		validationWidth = 3 // 2 for icon + 1 extra safety
	}

	inner := i.width - 4 - labelWidth - validationWidth
	if inner < 1 {
		inner = 1
	}
	i.InputModel.Width = inner
}
func (i *Input) Update(msg tea.Msg) (Element, tea.Cmd) {
	if i.Disabled {
		return i, nil
	}
	var cmd tea.Cmd
	i.InputModel, cmd = i.InputModel.Update(msg)

	// Re-validate on update to ensure Error state is fresh for Width calc
	if i.Validator != nil {
		prevErr := i.Error
		if err := i.Validator(i.Value()); err != nil {
			i.Error = err.Error()
		} else {
			i.Error = ""
		}
		// If error state changed, we might need to update width,
		// but View() will handle it if we call updateInputWidth there.
		if prevErr != i.Error {
			i.updateInputWidth() // Proactive update
		}
	}
	return i, cmd
}
func (i *Input) View(width int) string {
	// Update width if changed or just to ensure Error state is respected
	if width != 0 && width != i.width {
		i.width = width
	}
	// Always enforce width constraints before rendering View
	i.updateInputWidth()

	if i.Compact {
		t := theme.Current()
		labelWidth := theme.Width.Label
		if width > 0 && width < 30 {
			labelWidth = 8
		}

		labelStyle := t.Styles.Label.Copy().
			Width(labelWidth).
			PaddingRight(1)

		valueStyle := t.Styles.Value.Copy()
		if i.Error != "" {
			valueStyle = valueStyle.Foreground(t.Palette.Error)
		}

		return lipgloss.JoinHorizontal(lipgloss.Left,
			labelStyle.Render(i.Label),
			valueStyle.Render(i.InputModel.Value()),
		)
	}

	return RenderBoxedField(i.Label, i.InputModel.View(), i.Error, i.focused && !i.Disabled, i.width, lipgloss.Left)
}

// --- 3. Toggle Element ---
type Toggle struct {
	Label    string
	Checked  bool
	Disabled bool
	focused  bool
	width    int
}

func NewToggle(label string, initial bool) *Toggle {
	return &Toggle{Label: label, Checked: initial}
}
func (t *Toggle) Focusable() bool { return !t.Disabled }
func (t *Toggle) Focused() bool   { return t.focused }
func (t *Toggle) Focus() tea.Cmd {
	if t.Disabled {
		return nil
	}
	t.focused = true
	return nil
}
func (t *Toggle) Blur()          { t.focused = false }
func (t *Toggle) SetWidth(w int) { t.width = w }
func (t *Toggle) Toggle() {
	if t.Disabled {
		return
	}
	t.Checked = !t.Checked
}
func (t *Toggle) Update(msg tea.Msg) (Element, tea.Cmd) {
	if t.Disabled {
		return t, nil
	}
	if msg, ok := msg.(tea.KeyMsg); ok && t.focused {
		if msg.String() == " " || msg.String() == "enter" {
			t.Toggle()
		}
	}
	return t, nil
}
func (t *Toggle) View(width int) string {
	if width == 0 {
		width = t.width
	}
	state := "[ OFF ]"
	st := lipgloss.NewStyle().Foreground(theme.Current().Palette.TextDim)
	if t.Checked {
		state = "[ ON ]"
		st = lipgloss.NewStyle().Foreground(theme.Current().Palette.Success).Bold(true)
	}
	return RenderBoxedField(t.Label, st.Render(state), "", t.focused && !t.Disabled, width, lipgloss.Left)
}

// --- 4. Action Button Element ---
type ButtonRole int

const (
	RoleNormal ButtonRole = iota
	RoleSubmit
	RoleCancel
)

type Button struct {
	Label    string
	Text     string
	Action   func() tea.Cmd
	Role     ButtonRole
	Disabled bool
	focused  bool
	width    int
}

func NewButton(label, text string, action func() tea.Cmd) *Button {
	return &Button{Label: label, Text: text, Action: action, Role: RoleNormal}
}

func NewSubmitButton(label, text string, action func() tea.Cmd) *Button {
	return &Button{Label: label, Text: text, Action: action, Role: RoleSubmit}
}
func (b *Button) Focusable() bool { return !b.Disabled }
func (b *Button) Focused() bool   { return b.focused }
func (b *Button) Focus() tea.Cmd {
	if b.Disabled {
		return nil
	}
	b.focused = true
	return nil
}
func (b *Button) Blur()          { b.focused = false }
func (b *Button) SetWidth(w int) { b.width = w }
func (b *Button) Update(msg tea.Msg) (Element, tea.Cmd) {
	if b.Disabled {
		return b, nil
	}
	if msg, ok := msg.(tea.KeyMsg); ok && b.focused {
		if msg.String() == "enter" || msg.String() == " " {
			if b.Action != nil {
				return b, b.Action()
			}
		}
	}
	return b, nil
}
func (b *Button) View(width int) string {
	if width == 0 {
		width = b.width
	}
	t := theme.Current()
	contStyle := lipgloss.NewStyle().Foreground(t.Palette.TextDim)
	if b.focused {
		contStyle = lipgloss.NewStyle().Foreground(t.Palette.Accent).Bold(true)
	}
	// Pass center alignment for buttons and empty label if label is "SAVE" or specifically requested
	displayLabel := b.Label
	align := lipgloss.Left
	if b.Role == RoleSubmit {
		displayLabel = "" // Centered buttons usually don't need a label
		align = lipgloss.Center
	}
	return RenderBoxedField(displayLabel, contStyle.Render(b.Text), "", b.focused, width, align)
}

// --- Smart Form ---
type Form struct {
	Elements   []Element
	FocusIndex int
	Spacing    int
	Width      int
}

func (f *Form) HasActiveInput() bool {
	res := false
	for _, el := range f.Elements {
		if input, ok := el.(*Input); ok {
			if input.Focused() {
				res = true
				break
			}
		}
	}
	return res
}

func (f *Form) Validate() bool {
	valid := true
	for _, el := range f.Elements {
		if input, ok := el.(*Input); ok {
			if input.Validator != nil {
				if err := input.Validator(input.Value()); err != nil {
					valid = false
					input.Error = err.Error()
				}
			}
		}
	}
	return valid
}

func NewForm(elements ...Element) *Form {
	f := &Form{Elements: elements, Spacing: 0}
	f.FocusIndex = -1
	f.nextFocus(1)
	return f
}

func (f *Form) nextFocus(dir int) {
	start := f.FocusIndex
	cnt := len(f.Elements)
	if cnt == 0 {
		return
	}

	for i := 1; i <= cnt; i++ {
		idx := (start + i*dir) % cnt
		if idx < 0 {
			idx += cnt
		}

		if f.Elements[idx].Focusable() {
			if btn, ok := f.Elements[idx].(*Button); ok {
				if btn.Role == RoleSubmit && !f.Validate() {
					continue
				}
			}
			f.FocusIndex = idx
			return
		}
	}
}

func (f *Form) Focus() tea.Cmd {
	// Auto-select first element if none selected
	if f.FocusIndex < 0 || f.FocusIndex >= len(f.Elements) {
		f.nextFocus(1)
	}

	if f.FocusIndex >= 0 && f.FocusIndex < len(f.Elements) {
		return f.Elements[f.FocusIndex].Focus()
	}
	return nil
}

func (f *Form) NextField() tea.Cmd {
	if f.FocusIndex >= 0 {
		f.Elements[f.FocusIndex].Blur()
	}
	f.nextFocus(1)
	return f.Focus()
}

func (f *Form) PrevField() tea.Cmd {
	if f.FocusIndex >= 0 {
		f.Elements[f.FocusIndex].Blur()
	}
	f.nextFocus(-1)
	return f.Focus()
}

func (f *Form) Blur() {
	for _, el := range f.Elements {
		el.Blur()
	}
	f.FocusIndex = -1 // Reset internal focus tracking
}

func (f *Form) Update(msg tea.Msg) (*Form, tea.Cmd) {
	var cmds []tea.Cmd

	if kmsg, ok := msg.(tea.KeyMsg); ok {
		switch kmsg.String() {
		case "up":
			cmds = append(cmds, f.PrevField())
			return f, tea.Batch(cmds...)
		case "down":
			cmds = append(cmds, f.NextField())
			return f, tea.Batch(cmds...)
		}
	}

	if f.FocusIndex >= 0 && f.FocusIndex < len(f.Elements) {
		el, cmd := f.Elements[f.FocusIndex].Update(msg)
		f.Elements[f.FocusIndex] = el
		cmds = append(cmds, cmd)
	}
	return f, tea.Batch(cmds...)
}

func (f *Form) View(width int) string {
	if width == 0 {
		width = f.Width
	}
	var views []string
	for _, el := range f.Elements {
		if width > 0 {
			el.SetWidth(width)
		}
		views = append(views, el.View(width))
	}
	return layout.VStack(f.Spacing, views...)
}

// Shared Renderer
func RenderBoxedField(label, content, errorMsg string, focused bool, width int, align lipgloss.Position) string {
	t := theme.Current()

	// 1. Determine Label Width
	labelWidth := theme.Width.Label
	if label == "" {
		labelWidth = 0
	} else if width > 0 && width < 34 {
		labelWidth = 10 // Compact label for small boxes (enough for "SSH Port")
	}

	// 2. Define Styles
	labelStyle := t.Styles.Label.Copy().
		Width(labelWidth).
		MaxWidth(labelWidth).
		Align(lipgloss.Left)

	contentStyle := t.Styles.Value.Copy()

	borderColor := t.Palette.SurfaceHighlight
	validation := ""
	if errorMsg != "" {
		borderColor = t.Palette.Error
		validation = lipgloss.NewStyle().Foreground(t.Palette.Error).Render(" 🔺")
	} else if focused {
		borderColor = t.Palette.Accent
		labelStyle = labelStyle.Foreground(t.Palette.Accent).Bold(true)
	}

	// 3. Calculate Available Space for Content
	// Box Overhead: 2 (Borders) + 2 (Padding) = 4
	innerWidth := width - 4
	if innerWidth < 0 {
		innerWidth = 0
	}

	// 3. Render and Truncate Label
	renderedLabel := ""
	if label != "" {
		if lipgloss.Width(label) > labelWidth {
			if labelWidth > 3 {
				label = label[:labelWidth-2] + "." // Compact truncation
			} else {
				label = label[:labelWidth]
			}
		}
		renderedLabel = labelStyle.Render(label)
	}

	valWidth := lipgloss.Width(validation)
	contentAvail := innerWidth - labelWidth - valWidth
	if contentAvail < 0 {
		contentAvail = 0
	}

	// 4. Truncate Content if necessary (Strict)
	contentWidth := lipgloss.Width(content)
	if contentWidth > contentAvail {
		truncateLen := contentAvail
		if truncateLen > 3 {
			content = content[:truncateLen-3] + "..."
		} else if truncateLen > 0 {
			content = content[:truncateLen]
		} else {
			content = ""
		}
	}

	// 5. Compose Inner String (Force precisely innerWidth)
	middleBlock := lipgloss.PlaceHorizontal(contentAvail, align, contentStyle.Render(content))

	inner := lipgloss.JoinHorizontal(lipgloss.Top,
		renderedLabel,
		middleBlock,
		validation,
	)

	// 6. Final Box Render
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1)

	if width > 0 {
		boxStyle = boxStyle.Width(width - 2)
	}

	return boxStyle.Render(inner)
}
