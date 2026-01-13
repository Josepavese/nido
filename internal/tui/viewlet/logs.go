package viewlet

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// Logs implements the Logs viewlet using a viewport.
type Logs struct {
	BaseViewlet
	viewport viewport.Model
}

// NewLogs returns a new Logs viewlet instance.
func NewLogs() *Logs {
	l := &Logs{}
	l.viewport = viewport.New(0, 0)
	l.viewport.YPosition = 0
	l.viewport.SetContent("Initializing logs...")
	return l
}

// Init initializes the viewlet.
func (l *Logs) Init() tea.Cmd {
	return nil
}

// Shortcuts returns the list of keyboard shortcuts available.
func (l *Logs) Shortcuts() []Shortcut {
	return DefaultShortcuts()
}

// Update handles messages.
func (l *Logs) Update(msg tea.Msg) (Viewlet, tea.Cmd) {
	var cmd tea.Cmd
	l.viewport, cmd = l.viewport.Update(msg)
	return l, cmd
}

// View renders the log viewport.
func (l *Logs) View() string {
	return l.viewport.View()
}

// Resize updates the viewport dimensions.
func (l *Logs) Resize(width, height int) {
	l.Width = width
	l.Height = height
	l.viewport.Width = width
	l.viewport.Height = height
}

// SetContent sets the content of the logs viewport.
func (l *Logs) SetContent(content string) {
	// If content is empty/new run, maybe show something else?
	// For now, raw.
	// Clean up newlines if needed or ensure wrapping?
	// Viewport handles wrapping if Style set?
	// Default viewport wraps.
	l.viewport.SetContent(content)
	// Auto-scroll to bottom on update?
	// Typically yes for logs.
	l.viewport.GotoBottom()
}

// AddLine adds a line to the logs.
func (l *Logs) AddLine(line string) {
	// Not implemented deeply as SetContent overwrites.
	// We'll rely on model passing full buffer or Viewport buffer management.
	// For now model tracks buffer string.
}
