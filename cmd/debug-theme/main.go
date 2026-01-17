package main

import (
	"fmt"

	"github.com/Josepavese/nido/internal/tui/kit/theme"
	"github.com/Josepavese/nido/internal/tui/kit/widget"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	modal    *widget.ListModal
	quitting bool
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
	}
	// Always keep modal active
	m.modal.Show()
	_, cmd := m.modal.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.quitting {
		return "Bye!"
	}
	// Simulate the rendering logic in Config.View
	w, h := 80, 24

	// Direct render check
	modalView := m.modal.View()
	if modalView == "" {
		return fmt.Sprintf("ERROR: Modal.View() returned empty string! Active: %v", m.modal.Active)
	}

	// Layout check
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, modalView)
}

func main() {
	// 1. Force theme load (just in case)
	_ = theme.Current()

	// 2. Mock Items
	items := []list.Item{
		widget.SimpleListItem("Theme A"),
		widget.SimpleListItem("Theme B"),
	}

	// 3. Create Modal
	modal := widget.NewListModal(
		"Debug Theme Selector",
		items,
		40, 20,
		nil, nil,
	)
	modal.Show()

	// 4. Render
	view := modal.View()
	fmt.Printf("View Length: %d\n", len(view))
	fmt.Printf("View Content:\n%q\n", view)

	// 5. Test Lipgloss Place
	placed := lipgloss.Place(80, 24, lipgloss.Center, lipgloss.Center, view)
	fmt.Printf("Placed Length: %d\n", len(placed))
}
