package app

import (
	"context"
	"fmt"
	"strings"

	// Added import
	"github.com/Josepavese/nido/internal/config" // Global config
	"github.com/Josepavese/nido/internal/provider"

	"github.com/Josepavese/nido/internal/tui/kit/app"
	"github.com/Josepavese/nido/internal/tui/kit/shell"
	"github.com/Josepavese/nido/internal/tui/kit/theme"
	"github.com/Josepavese/nido/internal/tui/kit/widget"

	"github.com/Josepavese/nido/internal/tui/app/ops" // Was services
	"github.com/Josepavese/nido/internal/tui/app/pages/about"
	configpage "github.com/Josepavese/nido/internal/tui/app/pages/config" // Alias
	"github.com/Josepavese/nido/internal/tui/app/pages/fleet"
	"github.com/Josepavese/nido/internal/tui/app/pages/hatchery"
	"github.com/Josepavese/nido/internal/tui/app/pages/registry"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// NidoApp wraps the generic Kit App to handle domain-specific logic.
type NidoApp struct {
	*app.App
	prov          provider.VMProvider
	Hatchery      *hatchery.Hatchery // Keep reference for background updates
	Registry      *registry.Registry // Keep reference for background updates
	Config        *configpage.Config // Keep reference for background updates
	activeActions map[string]string  // Map OpName -> ActionID
}

func (n *NidoApp) Init() tea.Cmd {
	return tea.Batch(n.App.Init(), ops.RefreshFleet(n.prov))
}

// Update intercepts messages to handle Nido domain logic.
func (n *NidoApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case ops.SourcesLoadedMsg:
		// Broadcast to Hatchery even if not active
		if n.Hatchery != nil {
			_, cmd := n.Hatchery.Update(msg)
			cmds = append(cmds, cmd)
		}

	case ops.RegistryListMsg, ops.CacheListMsg, ops.CachePruneMsg:
		// Broadcast to Registry even if not active
		if n.Registry != nil {
			_, cmd := n.Registry.Update(msg)
			cmds = append(cmds, cmd)
		}
		// Also broadcast CacheStats to Config if needed, but CacheListMsg is redundant for Config now
		// unless we want it to refresh as well.

	case ops.UpdateCheckMsg, ops.CacheStatsMsg:
		if n.Config != nil {
			_, cmd := n.Config.Update(msg)
			cmds = append(cmds, cmd)
		}

	case ops.RequestSpawnMsg:
		opName := "spawn"
		n.Shell.Operation = opName
		n.Shell.SwitchTo("fleet")
		id, cmd := n.Shell.StartAction(fmt.Sprintf("Spawning %s", msg.Name))
		n.activeActions[opName] = id
		return n, tea.Batch(cmd, ops.SpawnVM(n.prov, msg.Name, msg.Source, msg.UserData, msg.GUI, msg.Ports))

	case ops.RequestCreateTemplateMsg:
		opName := "create-template"
		n.Shell.Operation = opName
		id, cmd := n.Shell.StartAction(fmt.Sprintf("Creating template %s", msg.Name))
		// Set indeterminate progress (no bar) because qemu-img doesn't report it easily
		n.Shell.ActionStack.UpdateProgress(id, -1)
		n.activeActions[opName] = id
		return n, tea.Batch(cmd, ops.CreateTemplate(n.prov, msg.Name, msg.Source))

	case ops.RequestPullMsg:
		opName := fmt.Sprintf("pull %s", msg.Image)
		n.Shell.Operation = opName
		id, cmd := n.Shell.StartAction(fmt.Sprintf("Pulling %s", msg.Image))
		n.activeActions[opName] = id
		return n, tea.Batch(cmd, ops.PullImage(n.prov, msg.Image))

	case ops.RequestDeleteImageMsg:
		opName := fmt.Sprintf("delete %s", msg.Name)
		n.Shell.Operation = opName
		id, cmd := n.Shell.StartAction(fmt.Sprintf("Deleting %s", msg.Name))
		n.activeActions[opName] = id
		return n, tea.Batch(cmd, ops.DeleteCacheImage(n.prov, msg.Name, msg.Version))

	case ops.RequestPruneMsg:
		opName := "prune"
		n.Shell.Operation = opName
		id, cmd := n.Shell.StartAction("Pruning cache")
		n.activeActions[opName] = id
		return n, tea.Batch(cmd, ops.PruneCache(n.prov))

	case ops.RequestDeleteTemplateMsg:
		opName := "delete-template"
		n.Shell.Operation = opName
		id, cmd := n.Shell.StartAction(fmt.Sprintf("Deleting template %s", msg.Name))
		n.activeActions[opName] = id
		return n, tea.Batch(cmd, ops.DeleteTemplate(n.prov, msg.Name, msg.Force))

	case ops.RequestUpdateMsg:
		return n, ops.CheckUpdate(msg.Manual)

	case ops.RequestCacheMsg:
		return n, ops.FetchCacheStats(n.prov)

	case ops.RequestApplyUpdateMsg:
		opName := "update-nido"
		n.Shell.Operation = opName
		id, cmd := n.Shell.StartAction("Ascending to next evolutionary state")
		n.Shell.ActionStack.UpdateProgress(id, -1)
		n.activeActions[opName] = id
		return n, tea.Batch(cmd, ops.ApplyUpdate())

	case ops.ApplyUpdateMsg:
		if id, ok := n.activeActions["update-nido"]; ok {
			n.Shell.FinishAction(id)
			delete(n.activeActions, "update-nido")
		}
		// Broadcast to Config for notification modal
		if n.Config != nil {
			_, cmd := n.Config.Update(msg)
			cmds = append(cmds, cmd)
		}
		if msg.Err != nil {
			return n, tea.Batch(cmds...)
		}
		// Success
		return n, tea.Batch(cmds...)

	case ops.RequestUninstallMsg:
		// We don't really need a long running action tracker because
		// if it works, we die immediately.
		n.Shell.Operation = "uninstall"
		return n, ops.UninstallCmd()

	case ops.ProgressMsg:
		if msg.Result != nil {
			// Finished: Handle final result logic
			return n.Update(*msg.Result)
		}
		// Running: Update Status Bar & Action Stack
		var cmd tea.Cmd
		lookupKey := msg.Status.Operation
		if msg.OpName != "" {
			lookupKey = msg.OpName
		}

		if id, ok := n.activeActions[lookupKey]; ok {
			// Use the display string from StatusMsg as the card message
			cmd = n.Shell.UpdateAction(id, msg.Status.Operation, msg.Status.Progress)
		} else {
			// Fallback: update global status if mismatch or generic
			n.Shell.SetStatus(msg.Status.Loading, msg.Status.Operation, msg.Status.Progress)
		}
		// Continue Loop
		return n, tea.Batch(cmd, msg.Next)

	case ops.VMDetailRequestMsg:
		return n, ops.FetchVMInfo(n.prov, msg.Name)

	case ops.RequestTemplateListMsg:
		return n, ops.FetchTemplatesList(n.prov)

	case ops.RequestOpMsg:
		n.Shell.Operation = msg.Op

		// "Fast" Operations: Don't show global ActionStack.
		// These are handled partially by local spinners (Fleet) or are instant enough.
		// - Start/Stop: Local spinner in Fleet sidebar.
		// - Delete: Still showing global for now as per user request (it feels heavy).
		// - Delete Template: Fast action from Hatchery, kept fast.
		isFastOp := false
		if msg.Op == ops.OpStart || msg.Op == ops.OpStop || msg.Op == "delete-template" {
			isFastOp = true
		}

		var id string
		var cmd tea.Cmd

		// Only trigger Global Action Card if NOT fast op
		if !isFastOp {
			id, cmd = n.Shell.StartAction(fmt.Sprintf("%s %s", strings.ToUpper(msg.Op), msg.Name))
			n.activeActions[msg.Op] = id
		}

		switch msg.Op {
		case ops.OpStart:
			return n, tea.Batch(cmd, ops.StartVM(n.prov, msg.Name))
		case ops.OpStop:
			return n, tea.Batch(cmd, ops.StopVM(n.prov, msg.Name))
		case ops.OpDelete:
			return n, tea.Batch(cmd, ops.DeleteVM(n.prov, msg.Name))
		}
		// If unknown op, cancel immediately
		if !isFastOp {
			n.Shell.FinishAction(id)
			delete(n.activeActions, msg.Op)
		}

	case ops.OpResultMsg:
		// 1. Finish Action
		if id, ok := n.activeActions[msg.Op]; ok {
			n.Shell.FinishAction(id)
			delete(n.activeActions, msg.Op)
		} else {
			// Fallback: If we can't find exact match (e.g. race?), try to clear global loading?
			// But we don't have global loading anymore.
			// Just log warning if needed, or ignore.
		}
		cmds = append(cmds, ops.RefreshFleet(n.prov))
		// Also refresh sources (e.g. after template delete/create)
		cmds = append(cmds, ops.FetchSources(n.prov, ops.SourceActionSpawn, false, true))

		// Check if we need to refresh Registry (Prune/Delete/Pull)
		// Heuristic: if op starts with "pull", "delete", "prune"
		if msg.Op == "prune" || strings.HasPrefix(msg.Op, "delete ") || strings.HasPrefix(msg.Op, "pull ") {
			// Forward as CachePruneMsg to trigger Registry refresh
			if n.Registry != nil {
				// We misuse CachePruneMsg slightly as a "Something Changed" signal
				_, cmd := n.Registry.Update(ops.CachePruneMsg{})
				cmds = append(cmds, cmd)
			}
		}

	case tea.QuitMsg:
		return n, tea.Quit
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
	if cfg.Theme != "" {
		theme.SetTheme(cfg.Theme)
	}
	t := theme.Current()

	// 2. Initialize Kit App
	kitApp := app.NewApp()

	// Configure Shell Styles
	kitApp.Shell.HeaderContent = ""
	kitApp.Shell.Styles = shell.ShellStyles{
		// Header: Deep background, subtle bottom border in the same grey palette
		Header:           lipgloss.NewStyle().Background(t.Palette.Surface).Border(lipgloss.NormalBorder(), false, false, true, false).BorderForeground(t.Palette.SurfaceSubtle),
		SubHeaderContext: t.Styles.Label.Copy().Bold(true),
		// SubHeaderNav: subtle grey
		SubHeaderNav: t.Styles.Label.Copy(),
		StatusBar: widget.StatusBarStyles{
			Key:    t.Styles.Label.Copy().Bold(true),
			Label:  t.Styles.Label.Copy(),
			Status: t.Styles.Label.Copy(),
		},
		BorderColor: t.Palette.SurfaceHighlight,
	}

	// 3. Initialize Pages
	fView := fleet.NewFleet(prov) // Fleet needs provider
	hView := hatchery.NewHatchery(prov)
	cView := configpage.NewConfig(cfg)
	rView := registry.NewRegistry(prov)

	// 5. Register Routes
	kitApp.AddRoute("fleet", "FLEET", "Select a bird to inspect.", fView)
	kitApp.AddRoute("hatchery", "HATCHERY", "Spawn birds.", hView)
	kitApp.AddRoute("registry", "REGISTRY", "Manage images & templates.", rView)
	kitApp.AddRoute("config", "SYSTEM", "Modify Nido's core DNA.", cView)
	kitApp.AddRoute("about", "NIDO", "The Nest.", about.NewAbout())

	kitApp.Shell.SwitchTo("fleet")

	// 6. Wrap in NidoApp
	nidoApp := &NidoApp{
		App:           kitApp,
		prov:          prov,
		Hatchery:      hView,
		Registry:      rView,
		Config:        cView,
		activeActions: make(map[string]string),
	}

	// 7. Run
	p := tea.NewProgram(nidoApp, tea.WithContext(ctx), tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}
