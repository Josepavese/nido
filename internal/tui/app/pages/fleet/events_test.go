package fleet

import (
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
