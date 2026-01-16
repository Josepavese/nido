package view

import (
	"github.com/Josepavese/nido/internal/tui/kit/layout"
	tea "github.com/charmbracelet/bubbletea"
)

// Shortcut represents a keyboard shortcut with its key and description.
type Shortcut struct {
	Key   string // e.g., "↵", "del", "n"
	Label string // e.g., "start/stop", "delete", "spawn"
}

// Viewlet is the interface that all TUI views must implement.
// It provides a consistent lifecycle for initialization, updates,
// rendering, and keyboard handling.
type Viewlet interface {
	// Init returns a command to run when the viewlet is first initialized
	// or when it becomes active.
	Init() tea.Cmd

	// Update handles messages and returns the updated viewlet and any command.
	Update(msg tea.Msg) (Viewlet, tea.Cmd)

	// View renders the viewlet content as a string.
	// The content MUST fit within the Rect provided by Resize.
	View() string

	// Resize updates the viewlet's dimensions.
	// Called when the terminal is resized.
	Resize(r layout.Rect)

	// Shortcuts returns the list of keyboard shortcuts available in this viewlet.
	// Used by the status bar to display contextual keybindings.
	Shortcuts() []Shortcut

	// Focused returns whether the viewlet currently has keyboard focus.
	Focused() bool

	// Focus gives the viewlet keyboard focus.
	Focus() tea.Cmd

	// Blur removes keyboard focus from the viewlet.
	Blur()

	// HandleMouse processes mouse events at localized coordinates (x, y).
	// Returns the updated viewlet, a command, and a boolean indicating if the event was handled.
	HandleMouse(x, y int, msg tea.MouseMsg) (Viewlet, tea.Cmd, bool)

	// IsModalActive returns whether a modal dialog is currently blocking interaction.
	IsModalActive() bool

	// HasActiveTextInput returns whether the viewlet currently has an active text input focused (blocks 'q').
	HasActiveTextInput() bool

	// HasActiveFocus returns whether any interactive element (button, toggle, input) is focused (blocks 'esc' quit).
	HasActiveFocus() bool

	// Focusable returns whether the viewlet can accept keyboard focus.
	Focusable() bool
}

// BaseViewlet provides common functionality for viewlets.
// Embed this in your viewlet implementation.
type BaseViewlet struct {
	Rect     layout.Rect
	measured bool
	focused  bool
}

// Resize updates the viewlet's dimensions.
func (b *BaseViewlet) Resize(r layout.Rect) {
	b.Rect = r
	b.measured = true
}

// Width returns the current width.
func (b *BaseViewlet) Width() int {
	return b.Rect.Width
}

// Height returns the current height.
func (b *BaseViewlet) Height() int {
	return b.Rect.Height
}

// Focused returns whether the viewlet has focus.
func (b *BaseViewlet) Focused() bool {
	return b.focused
}

// Focus gives the viewlet keyboard focus.
func (b *BaseViewlet) Focus() tea.Cmd {
	b.focused = true
	return nil
}

// Blur removes keyboard focus.
func (b *BaseViewlet) Blur() {
	b.focused = false
}

// Shortcuts returns default common shortcuts.
func (b *BaseViewlet) Shortcuts() []Shortcut {
	return DefaultShortcuts()
}

// HandleMouse provides a default no-op implementation.
func (b *BaseViewlet) HandleMouse(x, y int, msg tea.MouseMsg) (Viewlet, tea.Cmd, bool) {
	return nil, nil, false
}

// IsModalActive provides a default implementation (no modal).
func (b *BaseViewlet) IsModalActive() bool {
	return false
}

// HasActiveTextInput provides a default implementation.
func (b *BaseViewlet) HasActiveTextInput() bool {
	return false
}

// HasActiveFocus provides a default implementation.
func (b *BaseViewlet) HasActiveFocus() bool {
	return false
}

// Focusable provides a default implementation (true for most views).
func (b *BaseViewlet) Focusable() bool {
	return true
}

// DefaultShortcuts returns common shortcuts shared across viewlets.
func DefaultShortcuts() []Shortcut {
	return nil
}

// SelectionMsg is emitted by components (like SidebarList) when their selection changes.
type SelectionMsg struct {
	Index int
	Item  interface{}
}

// SwitchTabMsg is emitted by viewlets to request a tab change.
type SwitchTabMsg struct {
	TabIndex int
}

// StatusMsg is used to update the global shell status bar.
type StatusMsg struct {
	Loading   bool
	Operation string
	Progress  float64
}
