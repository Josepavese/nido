package app

import (
	"context"
	"testing"
	"time"

	"github.com/Josepavese/nido/v4/internal/config"
	"github.com/Josepavese/nido/v4/internal/events"
	"github.com/Josepavese/nido/v4/internal/hardware"
	"github.com/Josepavese/nido/v4/internal/provider"
	"github.com/Josepavese/nido/v4/internal/tui/app/ops"
	"github.com/Josepavese/nido/v4/internal/tui/app/pages/fleet"
	kit "github.com/Josepavese/nido/v4/internal/tui/kit/app"
	"github.com/Josepavese/nido/v4/internal/tui/kit/widget"
)

func TestEventFlowContinuesBehindGlobalAndFleetModals(t *testing.T) {
	p := provider.NewQemuProvider(t.TempDir(), &config.Config{}, hardware.New())
	f := fleet.NewFleet(p)
	model := &NidoApp{App: kit.NewApp(), prov: p, Fleet: f, eventContext: context.Background(), ErrorModal: widget.NewAlertModal("fixture", "", nil)}
	model.ErrorModal.Show()
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
	model.Update(ops.VMListMsg{RequestedAt: requested.Add(time.Second)})
	if f.Sidebar.SelectedItem() != nil {
		t.Fatal("external deletion did not clear selection")
	}
}
