package main

import (
	"fmt"
	"strings"

	"github.com/Josepavese/nido/internal/tui/kit/shell"
	"github.com/charmbracelet/lipgloss"
)

// Mock Viewlet
type mockViewlet struct{}

func (m *mockViewlet) Init()                                             {}
func (m *mockViewlet) Update(msg interface{}) (interface{}, interface{}) { return nil, nil }
func (m *mockViewlet) View() string                                      { return "BODY CONTENT" }
func (m *mockViewlet) Resize(r interface{})                              {}

func main() {
	// Setup minimalist shell
	s := shell.NewShell()

	// Add routes
	s.AddRoute(shell.Route{Key: "fleet", Title: "FLEET", Description: "Fleet Desc", Viewlet: &mockViewlet{}})
	s.AddRoute(shell.Route{Key: "registry", Title: "REGISTRY", Description: "Reg Desc", Viewlet: &mockViewlet{}})
	s.SwitchTo("fleet")

	// Set Styles (mimic wiring.go)
	// We use exact styles from wiring.go
	// Since we can't import internal/theme if it requires full init, we mock colors.
	// But let's try to be close.
	styleHeader := lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, false, true, false)

	s.Styles = shell.ShellStyles{
		Header: styleHeader,
	}

	// Case 1: Idle (No ActionStack)
	// Initialize with size
	width := 80
	height := 24
	s.Resize(width, height)

	fmt.Println("--- CASE 1: IDLE SHELL ---")
	viewIdle := s.View()
	printFirstLines(viewIdle, 5)

	// Dump grid info if we can access it (we can't easily, private)
	// But we can check if Header part is visible.

	// Case 2: Active Action (Simulate)
	fmt.Println("\n--- CASE 2: ACTIVE ACTION ---")
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
