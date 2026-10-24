package registry

import (
	"testing"

	"github.com/Josepavese/nido/v4/internal/tui/app/ops"
	tea "github.com/charmbracelet/bubbletea"
)

func TestBackgroundRemovalKeepsConfirmedImageIdentity(t *testing.T) {
	r := NewRegistry(nil)
	a := RegistryItem{Name: "original", Version: "1", IsLocal: true}
	r.DetailView.UpdateItem(a)
	r.Update(tea.KeyMsg{Type: tea.KeyDelete})
	r.Update(ops.CacheListMsg{Items: []ops.CacheItem{{Name: "replacement", Version: "2"}}})
	if len(r.localItems) != 1 || r.DetailView.Item.Name != "" {
		t.Fatal("modal blocked background removal")
	}
	cmd := r.ConfirmDelete.OnConfirm()
	if cmd == nil {
		t.Fatal("lost original confirmation")
	}
	if msg := cmd().(ops.RequestDeleteImageMsg); msg.Name != "original" || msg.Version != "1" {
		t.Fatal("confirmation switched image", msg)
	}
}
