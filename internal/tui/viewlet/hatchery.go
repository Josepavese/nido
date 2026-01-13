package viewlet

import (
	"strings"

	"github.com/Josepavese/nido/internal/tui/layout"
	"github.com/Josepavese/nido/internal/tui/theme"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HatcheryMode represents the current hatchery operation mode.
type HatcheryMode int

const (
	HatcherySpawn HatcheryMode = iota
	HatcheryTemplate
)

// Hatchery implements the Hatchery viewlet for spawning VMs and creating templates.
type Hatchery struct {
	BaseViewlet

	Mode        HatcheryMode
	inputs      []textinput.Model
	sourceList  list.Model
	focusIndex  int
	isSelecting bool

	sources    []string
	sourceIdx  int
	guiEnabled bool

	// Output channels/callbacks could go here, or handled via messages
	SpawnSource    string
	TemplateSource string
}

// NewHatchery creates a new Hatchery viewlet.
func NewHatchery() *Hatchery {
	// Initialize Inputs
	nameInput := textinput.New()
	nameInput.Placeholder = "vm-name"
	nameInput.CharLimit = 50
	nameInput.Width = theme.Width.SidebarWide // Use wide width as default for input
	nameInput.Focus()                         // Initial focus

	// Initialize Source List
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.Styles.SelectedTitle = theme.Current().Styles.SidebarItemSelected
	delegate.Styles.NormalTitle = theme.Current().Styles.SidebarItem

	l := list.New([]list.Item{}, delegate, 40, 10)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)

	return &Hatchery{
		Mode:       HatcherySpawn,
		inputs:     []textinput.Model{nameInput},
		sourceList: l,
		focusIndex: 0,
		guiEnabled: true,
	}
}

// Init initializes the Hatchery viewlet.
func (h *Hatchery) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for the Hatchery viewlet.
func (h *Hatchery) Update(msg tea.Msg) (Viewlet, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// If selecting source (modal)
	if h.isSelecting {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "esc" {
				h.isSelecting = false
				return h, nil
			}
			if msg.String() == "enter" {
				sel := h.sourceList.SelectedItem()
				if sel != nil {
					if i, ok := sel.(item); ok {
						if h.Mode == HatcherySpawn {
							h.SpawnSource = string(i)
						} else {
							h.TemplateSource = string(i)
						}
					}
				}
				h.isSelecting = false
				h.focusIndex++ // Move to next field
				return h, nil
			}
		}
		h.sourceList, cmd = h.sourceList.Update(msg)
		return h, cmd
	}

	// Normal Form Interaction
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			h.focusIndex++
			h.updateFocus()
		case "shift+tab", "up":
			h.focusIndex--
			h.updateFocus()
		case "enter":
			if h.focusIndex == 1 { // Source field
				h.isSelecting = true
				h.sourceList.SetHeight(10) // Ensure visible height
				// Re-populate list here if needed, or rely on SetSources
				return h, nil
			} else if h.focusIndex == 2 && h.Mode == HatcherySpawn {
				h.ToggleGUI() // Enter also toggles
			}
		case " ":
			if h.focusIndex == 2 && h.Mode == HatcherySpawn {
				h.ToggleGUI()
			} else if h.focusIndex == 1 {
				h.isSelecting = true
			}
		}
	}

	// Update inputs
	if h.focusIndex == 0 {
		h.inputs[0], cmd = h.inputs[0].Update(msg)
		cmds = append(cmds, cmd)
	}

	return h, tea.Batch(cmds...)
}

func (h *Hatchery) View() string {
	t := theme.Current()

	// Modal Overlay for Source Selection
	if h.isSelecting {
		return lipgloss.Place(h.Width, h.Height,
			lipgloss.Center, lipgloss.Center,
			theme.Current().Styles.SidebarItemSelected.Render(h.sourceList.View()),
		)
	}

	titleStyle := lipgloss.NewStyle().Foreground(t.Palette.AccentStrong).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(t.Palette.TextDim).Width(10)
	activeLabelStyle := lipgloss.NewStyle().Foreground(t.Palette.Accent).Width(10).Bold(true)

	// Mode Header
	modeLabel := "🐣 SPAWN VM"
	if h.Mode == HatcheryTemplate {
		modeLabel = "📦 CREATE TEMPLATE"
	}
	header := titleStyle.Render(modeLabel)

	// Form Construction
	var form strings.Builder

	// Field 1: Name
	lStyle := labelStyle
	if h.focusIndex == 0 {
		lStyle = activeLabelStyle
	}
	form.WriteString(lipgloss.JoinHorizontal(lipgloss.Left, lStyle.Render("Name"), h.inputs[0].View()) + "\n\n")

	// Field 2: Source
	lStyle = labelStyle
	valStyle := lipgloss.NewStyle().Foreground(t.Palette.TextDim)
	src := h.SpawnSource
	if h.Mode == HatcheryTemplate {
		src = h.TemplateSource
	}
	if src == "" {
		src = "Select..."
	}

	if h.focusIndex == 1 {
		lStyle = activeLabelStyle
		valStyle = valStyle.Foreground(t.Palette.Accent)
	}
	form.WriteString(lipgloss.JoinHorizontal(lipgloss.Left, lStyle.Render("Source"), valStyle.Render(src)) + "\n\n")

	// Field 3: Options (Spawn only)
	if h.Mode == HatcherySpawn {
		lStyle = labelStyle
		if h.focusIndex == 2 {
			lStyle = activeLabelStyle
		}
		toggle := h.renderToggle("GUI", h.guiEnabled)
		form.WriteString(lipgloss.JoinHorizontal(lipgloss.Left, lStyle.Render("Options"), toggle) + "\n\n")
	}

	// Submit Button
	btnText := "[ START INCUBATION ]"
	if h.Mode == HatcheryTemplate {
		btnText = "[ FREEZE TEMPLATE ]"
	}
	btnStyle := lipgloss.NewStyle().Foreground(t.Palette.TextDim)

	targetIdx := 3
	if h.Mode == HatcheryTemplate {
		targetIdx = 2
	}

	if h.focusIndex == targetIdx {
		btnStyle = lipgloss.NewStyle().Foreground(t.Palette.AccentStrong).Bold(true)
	}
	form.WriteString(btnStyle.Render(btnText))

	// Padding
	content := form.String()

	// Create the stacked view (Header + Content)
	stack := layout.VStack(theme.Space.XS, header, content)

	// Ensure full height occupancy to push footer down
	// We use PlaceVertical with Top alignment.
	// If height is invalid, just return stack.
	if h.Height > 0 {
		return lipgloss.PlaceVertical(h.Height, lipgloss.Top, stack)
	}
	return stack
}

// renderToggle renders a toggle switch.
func (h *Hatchery) renderToggle(label string, enabled bool) string {
	t := theme.Current()

	indicator := "○"
	style := lipgloss.NewStyle().Foreground(t.Palette.TextDim)
	if enabled {
		indicator = "●"
		style = lipgloss.NewStyle().Foreground(t.Palette.Success)
	}
	return style.Render(indicator + " Enable VNC")
}

func (h *Hatchery) updateFocus() {
	max := 3
	if h.Mode == HatcheryTemplate {
		max = 2 // Name, Source, Button
	}

	if h.focusIndex > max {
		h.focusIndex = 0
	} else if h.focusIndex < 0 {
		h.focusIndex = max
	}

	// Inputs
	if h.focusIndex == 0 {
		h.inputs[0].Focus()
	} else {
		h.inputs[0].Blur()
	}
}

// Helper list item wrapper
type item string

func (i item) FilterValue() string { return string(i) }
func (i item) Title() string       { return string(i) }
func (i item) Description() string { return "" }

// SetSources updates the available sources (images/templates/VMs).
func (h *Hatchery) SetSources(sources []string) {
	h.sources = sources
	items := make([]list.Item, len(sources))
	for i, s := range sources {
		items[i] = item(s)
	}
	h.sourceList.SetItems(items)
}

// SetMode changes the hatchery mode.
func (h *Hatchery) SetMode(mode HatcheryMode) {
	h.Mode = mode
}

// ToggleGUI toggles the GUI flag.
func (h *Hatchery) ToggleGUI() {
	h.guiEnabled = !h.guiEnabled
}

// GetValues returns the current form values.
func (h *Hatchery) GetValues() (name, source string, gui bool) {
	name = h.inputs[0].Value()
	if h.Mode == HatcherySpawn {
		source = h.SpawnSource
	} else {
		source = h.TemplateSource
	}
	gui = h.guiEnabled
	return
}

// Resize updates the viewlet dimensions.
func (h *Hatchery) Resize(width, height int) {
	h.BaseViewlet.Resize(width, height)
	h.inputs[0].Width = width - 20
	h.sourceList.SetWidth(width - 10)
}

// Shortcuts returns Hatchery-specific shortcuts.
func (h *Hatchery) Shortcuts() []Shortcut {
	return []Shortcut{
		{Key: "Tab", Label: "next field"},
		{Key: "Space", Label: "select"},
		{Key: "↵", Label: "hatch"},
	}
}

// Validate validates all form fields.
func (h *Hatchery) Validate() bool {
	// Simple validation for names
	return h.inputs[0].Value() != ""
}

// IsSubmitted returns true if the focus is on the submit button.
func (h *Hatchery) IsSubmitted() bool {
	targetIdx := 3
	if h.Mode == HatcheryTemplate {
		targetIdx = 2
	}
	return h.focusIndex == targetIdx
}

// IsSelecting returns true if the source selection modal is open.
func (h *Hatchery) IsSelecting() bool {
	return h.isSelecting
}

// IsTyping returns true if the focus is on a text input field.
func (h *Hatchery) IsTyping() bool {
	return h.focusIndex == 0
}
