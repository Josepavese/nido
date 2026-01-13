package main

import (
	"fmt"
	"time"

	"github.com/Josepavese/nido/internal/tui/kit/shell"
	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	sh shell.Shell
}

func (m model) Init() tea.Cmd {
	return tea.EnterAltScreen
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.sh.Resize(msg.Width, msg.Height)

		// Update content to show dimensions
		m.sh.HeaderContent = fmt.Sprintf("HEADER (%dx%d)", msg.Width, 3)
		m.sh.SubHeaderContent = fmt.Sprintf("SUBHEADER (%dx%d)", msg.Width, 2)
		m.sh.FooterContent = fmt.Sprintf("FOOTER (%dx%d) - Resize me!", msg.Width, 1)

		// Fill body with pattern
		m.sh.BodyContent = "BODY CONTENT AREA\nShould flex to fill remaining space.\n" +
			time.Now().Format(time.RFC3339)
	}
	return m, nil
}

func (m model) View() string {
	return m.sh.View()
}

func main() {
	p := tea.NewProgram(model{sh: shell.NewShell()})
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
