package app

import (
	"context"
	"fmt"
	"github.com/Josepavese/nido/v4/internal/events"
	"strings"
	"time"

	// Added import
	"github.com/Josepavese/nido/v4/internal/config" // Global config
	"github.com/Josepavese/nido/v4/internal/image"
	"github.com/Josepavese/nido/v4/internal/provider"

	"github.com/Josepavese/nido/v4/internal/tui/kit/app"
	"github.com/Josepavese/nido/v4/internal/tui/kit/layout"
	"github.com/Josepavese/nido/v4/internal/tui/kit/shell"
	"github.com/Josepavese/nido/v4/internal/tui/kit/theme"
	"github.com/Josepavese/nido/v4/internal/tui/kit/view"
	"github.com/Josepavese/nido/v4/internal/tui/kit/widget"

	"github.com/Josepavese/nido/v4/internal/tui/app/ops" // Was services
	"github.com/Josepavese/nido/v4/internal/tui/app/pages/about"
	configpage "github.com/Josepavese/nido/v4/internal/tui/app/pages/config" // Alias
	"github.com/Josepavese/nido/v4/internal/tui/app/pages/fleet"
	"github.com/Josepavese/nido/v4/internal/tui/app/pages/hatchery"
	"github.com/Josepavese/nido/v4/internal/tui/app/pages/registry"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// NidoApp wraps the generic Kit App to handle domain-specific logic.
type NidoApp struct {
	*app.App
	prov            provider.VMProvider
	eventContext    context.Context
	eventStream     <-chan events.Event
	configPath      string
	latestFleetRead time.Time
	Fleet           *fleet.Fleet
	Hatchery        *hatchery.Hatchery // Keep reference for background updates
	Registry        *registry.Registry // Keep reference for background updates
	Config          *configpage.Config // Keep reference for background updates
	activeActions   map[string]string  // Map OpName -> ActionID
	ErrorModal      *widget.Modal      // Global alert modal
}

func (n *NidoApp) Init() tea.Cmd {
	return tea.Batch(n.App.Init(), ops.RefreshFleet(n.prov), ops.WaitEvents(n.eventContext, n.eventStream))
}

// Update intercepts messages to handle Nido domain logic.
func (n *NidoApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && (key.String() == "ctrl+c" || key.String() == "ctrl+q") {
		_, cmd := n.App.Update(msg)
		return n, cmd
	}
	if n.Shell.Width > 0 && !layout.IsViable(n.Shell.Width, n.Shell.Height) && view.IsInputMessage(msg) {
		_, cmd := n.App.Update(msg)
		return n, cmd
	}
	// Background invalidations must flow even while a modal owns keyboard input.
	if batch, ok := msg.(ops.EventBatchMsg); ok {
		resources := map[string]bool{}
		for _, e := range batch.Events {
			if n.Fleet != nil {
				n.Fleet.ObserveEvent(e)
			}
			if e.Kind == events.Unavailable {
				if e.Epoch == "" {
					n.Shell.SetStatus(false, "Event stream reconnecting", 0)
				} else {
					n.Shell.SetStatus(false, "Event observation unavailable", 0)
				}
				continue
			}
			if e.Kind == events.Resync {
				n.Shell.SetStatus(false, "Event stream connected", 0)
				for _, uri := range []string{events.Fleet, events.Templates, events.Images, events.Cache, events.Config} {
					resources[uri] = true
				}
			}
			if e.Kind != events.Started {
				for _, uri := range e.Resources {
					resources[uri] = true
				}
			}
		}
		cmds := []tea.Cmd{ops.WaitEvents(n.eventContext, n.eventStream)}
		if resources[events.Config] {
			if err := n.prov.ReloadConfig(n.configPath); err != nil {
				n.Shell.SetStatus(false, fmt.Sprintf("Config refresh: %v", err), 0)
			} else if n.Config != nil {
				n.Config.ApplyConfig(n.prov.GetConfig())
			}
		}
		if resources[events.Fleet] {
			cmds = append(cmds, ops.RefreshFleet(n.prov))
		}
		if resources[events.Templates] || resources[events.Images] || resources[events.Cache] || resources[events.Config] {
			cmds = append(cmds, ops.FetchSources(n.prov, ops.SourceActionSpawn, false, true), ops.FetchRegistryImages(n.prov, false), ops.FetchCacheStats(n.prov))
			if n.Registry != nil {
				_, cmd := n.Registry.Update(ops.CachePruneMsg{})
				cmds = append(cmds, cmd)
			}
		}
		return n, tea.Batch(cmds...)
	}
	// Feed fleet data even when another route is active or a modal is open.
	if list, ok := msg.(ops.VMListMsg); ok {
		if list.RequestedAt.Before(n.latestFleetRead) {
			return n, nil
		}
		n.latestFleetRead = list.RequestedAt
	}
	if isFleetData(msg) && n.Fleet != nil {
		_, cmd := n.Fleet.Update(msg)
		if _, ok := msg.(ops.VMListMsg); !ok {
			return n, cmd
		}
		if n.Hatchery != nil {
			_, hcmd := n.Hatchery.Update(msg)
			return n, tea.Batch(cmd, hcmd)
		}
		return n, cmd
	}
	// 0. Modal Interception
	if n.ErrorModal.IsActive() && view.IsInputMessage(msg) {
		if mouse, ok := msg.(tea.MouseMsg); ok {
			n.Shell.Overlay = n.ErrorModal
			_, _, _, cmd, _ := n.Shell.HandleMouse(mouse)
			return n, cmd
		}
		newModal, cmd := n.ErrorModal.Update(msg)
		n.ErrorModal = newModal
		return n, cmd
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case ops.ConfigSavedMsg:
		if msg.Err != nil {
			n.Shell.SetStatus(false, fmt.Sprintf("Config save: %v", msg.Err), 0)
		} else {
			if err := n.prov.ReloadConfig(n.configPath); err != nil {
				n.Shell.SetStatus(false, err.Error(), 0)
			} else if n.Config != nil {
				n.Config.ApplyConfig(msg.Config)
			}
		}
	case ops.SourcesLoadedMsg, ops.VMListMsg:
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
		return n, tea.Batch(cmd, ops.SpawnVM(n.prov, msg.Name, msg.Source, msg.SourceType, msg.UserData, msg.GUI, msg.MemoryMB, msg.VCPUs, msg.Ports, msg.RawQemuArgs, msg.Accelerators))

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
		return n, ops.UninstallCmd(n.prov.EventPublisher())

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
		// Fleet data refresh is driven by the shared event stream.

		// 4. Handle Errors
		if msg.Err != nil {
			n.Shell.SetStatus(false, fmt.Sprintf("ERROR: %v", msg.Err), 0)
			// Show prominent modal for failed dangerous ops
			if msg.Op == "prune" || msg.Op == "delete-template" || strings.HasPrefix(msg.Op, "delete") || strings.HasPrefix(msg.Op, "pull") {
				n.ErrorModal.Title = "Operation Failed"
				n.ErrorModal.Message = fmt.Sprintf("Error during %s:\n%v", msg.Op, msg.Err)
				n.ErrorModal.Show()
			}
		} else if msg.Op != "prune" {
			n.Shell.SetStatus(false, fmt.Sprintf("SUCCESS: %s completed", msg.Op), 0)
		}

		// Local feedback and selection; data invalidations come from events.
		_, blueprintSpawn := msg.Data.(ops.BlueprintSpawnResult)
		if msg.Op == "prune" || strings.HasPrefix(msg.Op, "delete") || strings.HasPrefix(msg.Op, "pull") || blueprintSpawn {
			// UX: If this was a pull, auto-select the new image in Hatchery
			if strings.HasPrefix(msg.Op, "pull ") && n.Hatchery != nil {
				imgName := strings.TrimPrefix(msg.Op, "pull ")
				n.Hatchery.SetPendingSelection(imgName)
			}
			if blueprintSpawn && n.Hatchery != nil {
				result := msg.Data.(ops.BlueprintSpawnResult)
				n.Hatchery.SetPendingSelection(result.Blueprint)
			}

			// Special Feedback for Prune
			if msg.Op == "prune" && msg.Err == nil {
				stats, ok := msg.Data.(ops.PruneStats)
				if !ok {
					n.Shell.SetStatus(false, "Prune completed.", 0)
				} else {
					if stats.Count == 0 {
						n.Shell.SetStatus(false, "Nothing to prune from cache.", 0)
					} else {
						n.Shell.SetStatus(false, fmt.Sprintf("Pruned %d images (%s reclaimed).", stats.Count, image.FormatBytes(stats.Reclaimed)), 0)
					}
				}
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

// View delegates to Shell but overlays global modals
func (n *NidoApp) View() string {
	n.Shell.Overlay = n.ErrorModal
	return n.App.View()
}

// Run starts the Nido TUI.
func Run(ctx context.Context, prov provider.VMProvider, cfg *config.Config, source events.Source, configPath string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var stream <-chan events.Event
	if source != nil {
		stream = source.Watch(ctx)
	}
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
		Header: lipgloss.NewStyle().Background(t.Palette.Surface).Border(lipgloss.NormalBorder(), false, false, true, false).BorderForeground(t.Palette.SurfaceSubtle),
		StatusBar: widget.StatusBarStyles{
			Key:    t.Styles.Label.Bold(true),
			Label:  t.Styles.Label,
			Status: t.Styles.Label,
		},
		BorderColor: t.Palette.SurfaceHighlight,
	}

	// 3. Initialize Pages
	fView := fleet.NewFleet(prov) // Fleet needs provider
	hView := hatchery.NewHatchery(prov)
	cView := configpage.NewConfig(cfg, prov.EventPublisher())
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
		eventContext:  ctx,
		eventStream:   stream,
		configPath:    configPath,
		Fleet:         fView,
		Hatchery:      hView,
		Registry:      rView,
		Config:        cView,
		activeActions: make(map[string]string),
	}

	// 7. Initialize Global Modal
	nidoApp.ErrorModal = widget.NewAlertModal("Error", "", nil)
	nidoApp.Shell.Overlay = nidoApp.ErrorModal

	// 8. Run
	p := tea.NewProgram(nidoApp, tea.WithContext(ctx), tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	cancel()
	if stream != nil {
		for range stream {
		}
	}
	return err
}

func isFleetData(msg tea.Msg) bool {
	switch msg.(type) {
	case ops.VMListMsg, ops.VMDetailMsg, ops.TemplateListMsg:
		return true
	}
	return false
}
