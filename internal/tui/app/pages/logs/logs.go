package logs

import (
	"github.com/Josepavese/nido/internal/tui/kit/layout"
	view "github.com/Josepavese/nido/internal/tui/kit/view"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// Logs implements the Logs viewlet using a viewport.
type Logs struct {
	view.BaseViewlet
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
func (l *Logs) Shortcuts() []view.Shortcut {
	return view.DefaultShortcuts()
}

func (l *Logs) HandleMouse(x, y int, msg tea.MouseMsg) (view.Viewlet, tea.Cmd, bool) {
	// Viewport handles mouse natively in Update if we pass the message.
	// We just return true to indicate we handled it if it's within our rect.
	_, cmd := l.Update(msg)
	return l, cmd, true
}

// Update handles messages.
func (l *Logs) Update(msg tea.Msg) (view.Viewlet, tea.Cmd) {
	var cmd tea.Cmd
	l.viewport, cmd = l.viewport.Update(msg)
	return l, cmd
}

// View renders the log viewport.
func (l *Logs) View() string {
	return l.viewport.View()
}

// Resize updates the viewport dimensions.
func (l *Logs) Resize(r layout.Rect) {
	l.BaseViewlet.Resize(r)
	l.viewport.Width = r.Width
	l.viewport.Height = r.Height
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
