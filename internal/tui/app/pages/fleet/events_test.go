package fleet

import (
	"github.com/Josepavese/nido/v4/internal/hardware"
	"github.com/Josepavese/nido/v4/internal/provider"
	"github.com/Josepavese/nido/v4/internal/tui/kit/layout"
	"github.com/Josepavese/nido/v4/internal/tui/kit/widget"
	"testing"

	"github.com/Josepavese/nido/v4/internal/tui/app/ops"
	tea "github.com/charmbracelet/bubbletea"
)

func TestBackgroundDetailCannotRetargetDeleteConfirmation(t *testing.T) {
	f := NewFleet(nil)
	f.detail.Name = "original"
	f.Update(tea.KeyMsg{Type: tea.KeyDelete})
	f.detail.Name = "replacement"
	cmd := f.ConfirmDelete.OnConfirm()
	if cmd == nil {
		t.Fatal("lost original confirmation")
	}
	if msg := cmd().(ops.RequestOpMsg); msg.Name != "original" {
		t.Fatal("confirmation switched VM", msg)
	}
}

func TestCompactFleetEnterOnlyOpensAndMouseKeepsPage(t *testing.T) {
	f := NewFleet(nil)
	f.Resize(layout.NewRect(0, 0, 60, 13))
	f.Sidebar.SetItems([]widget.SidebarItem{FleetItem{Name: "bird", State: "stopped"}})
	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil || f.MasterDetail.ActiveFocus != widget.FocusDetail {
		t.Fatal("opening compact fleet detail emitted an action")
	}
	f.Resize(layout.NewRect(0, 0, 120, 30))
	f.View()
	page, _, _ := f.HandleMouse(35, 2, tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	if page != f {
		t.Fatal("mouse replaced Fleet with a child layout")
	}
}

func TestFleetRefreshPreservesHighlightedVMAndRejectsLateNotifications(t *testing.T) {
	f := NewFleet(nil)
	f.Update(ops.VMListMsg{Items: []ops.VMItem{{Name: "first", State: "stopped"}, {Name: "second", State: "stopped"}}})
	old := f.Sidebar.Select(0)
	f.Sidebar.Select(1)
	f.Update(ops.VMListMsg{Items: []ops.VMItem{{Name: "second", State: "running"}, {Name: "first", State: "stopped"}}})
	if selected := f.Sidebar.SelectedItem().(FleetItem); selected.Name != "second" || selected.State != "running" {
		t.Fatal("refresh replaced highlighted VM", selected)
	}
	if _, cmd := f.Update(old()); cmd != nil {
		t.Fatal("late event requested an obsolete VM detail")
	}
	other := NewFleet(nil)
	other.Sidebar.SetItems([]widget.SidebarItem{FleetItem{Name: "foreign"}})
	if _, cmd := f.Update(other.Sidebar.Select(0)()); cmd != nil {
		t.Fatal("foreign list retargeted fleet")
	}
}

type acceleratorProvider struct{ provider.VMProvider }

func (acceleratorProvider) ListAccelerators() ([]hardware.Accelerator, error) {
	return []hardware.Accelerator{{ID: "fixture-gpu"}}, nil
}

func TestAcceleratorCommandOnlyPublishesData(t *testing.T) {
	f := NewFleet(acceleratorProvider{})
	first := f.OpenAcceleratorModal("first")
	second := f.OpenAcceleratorModal("second")
	result := second()
	if f.ModalAccel.IsActive() || len(f.ModalAccel.List.Items()) != 0 {
		t.Fatal("asynchronous command mutated UI state")
	}
	f.Update(result)
	f.Update(first())
	if !f.ModalAccel.IsActive() || f.pendingAccelVM != "second" || len(f.ModalAccel.List.Items()) != 1 {
		t.Fatal("obsolete accelerator result retargeted dialog")
	}
}
