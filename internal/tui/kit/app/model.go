package app

import (
	"context"

	"github.com/Josepavese/nido/internal/tui/kit/shell"
	"github.com/Josepavese/nido/internal/tui/kit/view"
	tea "github.com/charmbracelet/bubbletea"
)

// App is the generic TUI application controller.
type App struct {
	Shell shell.Shell

	// Hooks
	OnQuit func()
}

// NewApp creates a new generic App.
func NewApp() *App {
	return &App{
		Shell: shell.NewShell(),
	}
}

// Init handles initial commands.
func (a *App) Init() tea.Cmd {
	return a.Shell.Init()
}

// Update handles the main loop.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Check for blocking viewlet (Modal active)
	blocking := false
	if v := a.Shell.ActiveViewlet(); v != nil {
		if b, ok := v.(interface{ IsModalActive() bool }); ok {
			blocking = b.IsModalActive()
		}
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "ctrl+q":
			if a.OnQuit != nil {
				a.OnQuit()
			}
			return a, tea.Quit
		// GLOBAL NAV: Intercept Arrow Keys for Tab Switching
		case "right":
			if !blocking {
				a.Shell.NextTab()
				return a, nil
			}
		case "left":
			if !blocking {
				a.Shell.PrevTab()
				return a, nil
			}
		}
	case tea.WindowSizeMsg:
		a.Shell.Resize(msg.Width, msg.Height)
		// We might need to repaint explicitly if View() relies on it,
		// but standard Bubble Tea loop handles it.
	}

	// Delegate to Shell
	shellCmd := a.Shell.Update(msg)

	// Delegate to Active Viewlet
	var vCmd tea.Cmd
	if v := a.Shell.ActiveViewlet(); v != nil {
		_, vCmd = v.Update(msg)
	}

	return a, tea.Batch(cmd, shellCmd, vCmd)
}

// View delegates rendering to the Shell.
func (a *App) View() string {
	return a.Shell.View()
}

// Run starts the application.
func (a *App) Run(ctx context.Context) error {
	p := tea.NewProgram(a, tea.WithContext(ctx), tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

// AddRoute registers a new viewlet route.
func (a *App) AddRoute(key, title, hint string, v view.Viewlet) {
	a.Shell.AddRoute(shell.Route{
		Key:     key,
		Title:   title,
		Hint:    hint,
		Viewlet: v,
	})
}
