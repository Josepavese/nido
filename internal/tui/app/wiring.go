package app

import (
	"context"
	"fmt"

	// Added import
	"github.com/Josepavese/nido/internal/config" // Global config
	"github.com/Josepavese/nido/internal/provider"

	"github.com/Josepavese/nido/internal/tui/kit/app"
	"github.com/Josepavese/nido/internal/tui/kit/shell"
	"github.com/Josepavese/nido/internal/tui/kit/theme"
	"github.com/Josepavese/nido/internal/tui/kit/view"
	"github.com/Josepavese/nido/internal/tui/kit/widget"

	"github.com/Josepavese/nido/internal/tui/app/ops"                     // Was services
	configpage "github.com/Josepavese/nido/internal/tui/app/pages/config" // Alias
	"github.com/Josepavese/nido/internal/tui/app/pages/fleet"
	"github.com/Josepavese/nido/internal/tui/app/pages/hatchery"
	"github.com/Josepavese/nido/internal/tui/app/pages/help"
	"github.com/Josepavese/nido/internal/tui/app/pages/logs"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// NidoApp wraps the generic Kit App to handle domain-specific logic.
type NidoApp struct {
	*app.App
	prov provider.VMProvider
}

// Update intercepts messages to handle Nido domain logic.
func (n *NidoApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case ops.RequestSpawnMsg:
		n.Shell.Loading = true
		n.Shell.Operation = "spawn"
		n.Shell.SwitchTo("fleet")
		return n, ops.SpawnVM(n.prov, msg.Name, msg.Source, msg.UserData, msg.GUI)

	case ops.RequestCreateTemplateMsg:
		n.Shell.Loading = true
		n.Shell.Operation = "create-template"
		return n, ops.CreateTemplate(n.prov, msg.Name, msg.Source)

	case ops.OpResultMsg:
		n.Shell.Loading = false
		if msg.Err != nil {
			n.Shell.Logs = append(n.Shell.Logs, fmt.Sprintf("Operation %s failed: %v", msg.Op, msg.Err))
		} else {
			n.Shell.Logs = append(n.Shell.Logs, fmt.Sprintf("Operation %s complete.", msg.Op))
			cmds = append(cmds, ops.RefreshFleet(n.prov))
		}

	case view.LogMsg:
		// Also capture generic log messages from viewlets
		n.Shell.Logs = append(n.Shell.Logs, fmt.Sprintf("[%s] %s", "LOG", msg.Text))

		// Sync Logs Viewlet content (?)
		// Just an example showing we can intercept.
	}

	// Delegate to Kit App
	newApp, cmd := n.App.Update(msg)
	n.App = newApp.(*app.App)
	cmds = append(cmds, cmd)

	// Sync Logs content to the viewlet (if we want to be explicit)
	// We might need access to lView instance here if we want to set content.
	// But lView is inside App.Shell.Routes.
	// For now, let's rely on Shell.Logs aggregation.

	return n, tea.Batch(cmds...)
}

// Run starts the Nido TUI.
func Run(ctx context.Context, prov provider.VMProvider, cfg *config.Config) error {
	// 1. Initialize Theme
	t := theme.Current()

	// 2. Initialize Kit App
	kitApp := app.NewApp()

	// Configure Shell Styles
	kitApp.Shell.HeaderContent = ""
	kitApp.Shell.Styles = shell.ShellStyles{
		Header:           lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, false, true, false).BorderForeground(t.Palette.SurfaceSubtle),
		SubHeaderContext: lipgloss.NewStyle().Foreground(t.Palette.AccentStrong).Bold(true),
		SubHeaderNav:     lipgloss.NewStyle().Foreground(t.Palette.TextDim).Italic(true),
		StatusBar: widget.StatusBarStyles{
			Key:    lipgloss.NewStyle().Foreground(t.Palette.Accent).Bold(true),
			Label:  lipgloss.NewStyle().Foreground(t.Palette.TextDim),
			Status: lipgloss.NewStyle().Foreground(t.Palette.TextDim),
		},
	}

	// 3. Initialize Pages
	fView := fleet.NewFleet(prov) // Fleet needs provider
	hView := hatchery.NewHatchery()
	cView := configpage.NewConfig(cfg)
	lView := logs.NewLogs()
	helpView := help.NewHelp()

	// 5. Register Routes
	kitApp.AddRoute("fleet", "FLEET", "Select a bird to inspect. Use ←/→ to navigate tabs.", fView)
	kitApp.AddRoute("hatchery", "HATCHERY", "Spawn birds. Tab cycles fields. Use ←/→ to navigate tabs.", hView)
	kitApp.AddRoute("logs", "LOGS", "System activity log. Use ←/→ to navigate tabs.", lView)
	kitApp.AddRoute("config", "CONFIG", "Modify Nido's core DNA. Use ←/→ to navigate tabs.", cView)
	kitApp.AddRoute("help", "HELP", "Shortcuts & documentation. Use ←/→ to navigate tabs.", helpView)

	kitApp.Shell.SwitchTo("fleet")

	// 6. Wrap in NidoApp
	nidoApp := &NidoApp{
		App:  kitApp,
		prov: prov,
	}

	// 7. Run
	p := tea.NewProgram(nidoApp, tea.WithContext(ctx), tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}
