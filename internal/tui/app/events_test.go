package app

import (
	"context"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
	"time"

	"github.com/Josepavese/nido/v4/internal/config"
	"github.com/Josepavese/nido/v4/internal/events"
	"github.com/Josepavese/nido/v4/internal/hardware"
	"github.com/Josepavese/nido/v4/internal/provider"
	"github.com/Josepavese/nido/v4/internal/tui/app/ops"
	"github.com/Josepavese/nido/v4/internal/tui/app/pages/fleet"
	"github.com/Josepavese/nido/v4/internal/tui/app/pages/hatchery"
	kit "github.com/Josepavese/nido/v4/internal/tui/kit/app"
	"github.com/Josepavese/nido/v4/internal/tui/kit/widget"
)

func TestEventFlowContinuesBehindGlobalAndFleetModals(t *testing.T) {
	p := provider.NewQemuProvider(t.TempDir(), &config.Config{}, hardware.New())
	f := fleet.NewFleet(p)
	h := hatchery.NewHatchery(p)
	model := &NidoApp{App: kit.NewApp(), prov: p, Fleet: f, Hatchery: h, eventContext: context.Background(), ErrorModal: widget.NewAlertModal("fixture", "", nil)}
	model.ErrorModal.Show()
	h.ConfirmDelete.Show()
	f.ErrorModal.Show()
	requested := time.Now()
	if _, cmd := model.Update(ops.EventBatchMsg{Events: []events.Event{{Source: events.Filesystem, Kind: events.Changed, Resources: []string{events.Fleet}}}}); cmd == nil {
		t.Fatal("event refresh stopped behind modal")
	}
	model.Update(ops.VMListMsg{RequestedAt: requested, Items: []ops.VMItem{{Name: "new-vm", State: "stopped"}}})
	if f.Sidebar.SelectedItem() == nil {
		t.Fatal("background fleet result lost behind modal")
	}
	model.Update(ops.VMListMsg{RequestedAt: requested.Add(-time.Second)})
	if f.Sidebar.SelectedItem() == nil {
		t.Fatal("stale response overwrote current fleet")
	}
	if !h.ExistingVMs["new-vm"] {
		t.Fatal("stale response overwrote hatchery VM validation")
	}
	model.Update(ops.VMListMsg{RequestedAt: requested.Add(time.Second)})
	if f.Sidebar.SelectedItem() != nil {
		t.Fatal("external deletion did not clear selection")
	}
}

func TestGlobalErrorStillAllowsExit(t *testing.T) {
	for _, key := range []tea.KeyType{tea.KeyCtrlC, tea.KeyCtrlQ} {
		model := &NidoApp{App: kit.NewApp(), ErrorModal: widget.NewAlertModal("Error", "fixture", nil)}
		model.ErrorModal.Show()
		_, cmd := model.Update(tea.KeyMsg{Type: key, Alt: true})
		if cmd == nil {
			t.Fatal("global error swallowed exit")
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatal("exit did not reach kit lifecycle")
		}
	}
}

func TestGlobalOverlayUsesShellGeometry(t *testing.T) {
	for _, size := range [][2]int{{120, 36}, {40, 10}} {
		model := &NidoApp{App: kit.NewApp(), ErrorModal: widget.NewAlertModal("Error", "fixture", nil)}
		model.Shell.Resize(size[0], size[1])
		model.ErrorModal.Show()
		text := ansi.Strip(model.View())
		if !strings.Contains(text, "^C quit") || strings.Contains(text, "flow") {
			t.Fatal("global overlay lost its modal footer")
		}
		for y, line := range strings.Split(text, "\n") {
			if index := strings.Index(line, "Close"); index >= 0 {
				x := ansi.StringWidth(line[:index])
				model.Update(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
				break
			}
		}
		if model.ErrorModal.IsActive() {
			t.Fatal("global close click did not follow shell coordinates")
		}
	}
}
