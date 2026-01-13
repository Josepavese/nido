// Package viewlet defines the interface for TUI view modules.
// Each viewlet (Fleet, Hatchery, Config, Logs, Help) implements this
// interface for consistent lifecycle and keyboard handling.
package viewlet

import (
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
	View() string

	// Resize updates the viewlet's dimensions.
	// Called when the terminal is resized.
	Resize(width, height int)

	// Shortcuts returns the list of keyboard shortcuts available in this viewlet.
	// Used by the status bar to display contextual keybindings.
	Shortcuts() []Shortcut

	// Focused returns whether the viewlet currently has keyboard focus.
	Focused() bool

	// Focus gives the viewlet keyboard focus.
	Focus() tea.Cmd

	// Blur removes keyboard focus from the viewlet.
	Blur()
}

// BaseViewlet provides common functionality for viewlets.
// Embed this in your viewlet implementation.
type BaseViewlet struct {
	Width   int
	Height  int
	focused bool
}

// Resize updates the viewlet's dimensions.
func (b *BaseViewlet) Resize(width, height int) {
	b.Width = width
	b.Height = height
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

// DefaultShortcuts returns common shortcuts shared across viewlets.
func DefaultShortcuts() []Shortcut {
	return []Shortcut{
		{Key: "q", Label: "quit"},
		{Key: "←/→", Label: "tabs"},
	}
}
