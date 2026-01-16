package main

import (
	"fmt"

	"github.com/Josepavese/nido/internal/tui/kit/app"
	"github.com/Josepavese/nido/internal/tui/kit/layout"
	"github.com/Josepavese/nido/internal/tui/kit/view"
	tea "github.com/charmbracelet/bubbletea"
	// We can't import wiring directly if it's main? No wiring is in internal/tui/app
	// But we can replicate the logic.
)

func main() {
	// 1. Initialize Kit App
	kitApp := app.NewApp()

	// 2. Add Routes (Simulate wiring.go)
	// We use dummy viewlets
	dummyView := &DummyViewlet{}
	kitApp.AddRoute("fleet", "FLEET", "Hint", dummyView)
	kitApp.AddRoute("hatchery", "HATCHERY", "Hint", dummyView)

	fmt.Printf("Routes in KitApp Shell: %d\n", kitApp.Shell.RouteCount())

	// 3. Wrap in NidoApp (Simulate wiring struct)
	nidoApp := &NidoApp{
		App: kitApp,
	}

	fmt.Printf("Routes in NidoApp Shell: %d\n", nidoApp.App.Shell.RouteCount())

	// Test Pointer Logic
	kitApp.AddRoute("extra", "EXTRA", "Hint", dummyView)
	fmt.Printf("Routes in NidoApp Shell after Add to KitApp: %d\n", nidoApp.App.Shell.RouteCount())
}

type NidoApp struct {
	*app.App
}

// Dummy Viewlet
type DummyViewlet struct{}

func (d *DummyViewlet) Init() tea.Cmd                              { return nil }
func (d *DummyViewlet) Update(msg tea.Msg) (view.Viewlet, tea.Cmd) { return d, nil }
func (d *DummyViewlet) View() string                               { return "View" }
func (d *DummyViewlet) Resize(r layout.Rect)                       {}
func (d *DummyViewlet) HandleMouse(x, y int, msg tea.MouseMsg) (view.Viewlet, tea.Cmd, bool) {
	return d, nil, false
}
func (d *DummyViewlet) Shortcuts() []view.Shortcut { return nil }
func (d *DummyViewlet) IsModalActive() bool        { return false }
func (d *DummyViewlet) HasActiveInput() bool       { return false }
func (d *DummyViewlet) Blur()                      {}
func (d *DummyViewlet) Focus() tea.Cmd             { return nil }
func (d *DummyViewlet) Focused() bool              { return false }
func (d *DummyViewlet) Focusable() bool            { return false }

// Need helper to access routes in shell if they are private
// But Shell export routes? No "routes" field is private (lowercase).
// shell.go: routes []Route
