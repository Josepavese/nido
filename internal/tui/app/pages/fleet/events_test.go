package fleet

import (
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
