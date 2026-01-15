package main

import (
	"fmt"
	"strings"

	"github.com/Josepavese/nido/internal/tui/kit/layout"
	"github.com/Josepavese/nido/internal/tui/kit/shell"
	"github.com/Josepavese/nido/internal/tui/kit/view"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Mock Viewlet
type mockViewlet struct {
	view.BaseViewlet
}

func (m *mockViewlet) Init() tea.Cmd                              { return nil }
func (m *mockViewlet) Update(msg tea.Msg) (view.Viewlet, tea.Cmd) { return m, nil }
func (m *mockViewlet) View() string                               { return "BODY CONTENT" }
func (m *mockViewlet) Resize(r layout.Rect)                       { m.BaseViewlet.Resize(r) }

// BaseViewlet handles Shortcuts, Focused, Focus, Blur, HandleMouse

func main() {
	// Setup minimalist shell
	s := shell.NewShell()

	// Add routes
	s.AddRoute(shell.Route{Key: "fleet", Title: "FLEET", Hint: "Fleet Hint", Viewlet: &mockViewlet{}})
	s.AddRoute(shell.Route{Key: "registry", Title: "REGISTRY", Hint: "Reg Hint", Viewlet: &mockViewlet{}})
	s.SwitchTo("fleet")

	// Set Styles (mimic wiring.go)
	// We use exact styles from wiring.go
	styleHeader := lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, false, true, false)

	// We need to set a Foreground if we want to mimic visibility check, but let's assume default color exists.
	// We set TextDim equivalent (grey)
	styleNav := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Italic(true)

	s.Styles = shell.ShellStyles{
		Header:       styleHeader,
		SubHeaderNav: styleNav,
	}

	// Case 1: Idle (No ActionStack)
	// Initialize with size
	width := 80
	height := 24
	s.Resize(width, height)

	fmt.Println("--- CASE 1: IDLE SHELL (No ActionStack) ---")
	viewIdle := s.View()
	printFirstLines(viewIdle, 5)

	// Case 2: Active Action (Simulate)
	fmt.Println("\n--- CASE 2: ACTIVE ACTION (With ActionStack) ---")
	s.StartAction("Pulling Image...")
	// Resize again to trigger layout recalc with stack
	s.Resize(width, height)

	viewActive := s.View()
	printFirstLines(viewActive, 8)
}

func printFirstLines(s string, n int) {
	lines := strings.Split(s, "\n")
	for i := 0; i < n && i < len(lines); i++ {
		fmt.Printf("[%d] %q\n", i, lines[i])
	}
}
