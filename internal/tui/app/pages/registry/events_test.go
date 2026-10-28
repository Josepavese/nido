package registry

import (
	"github.com/Josepavese/nido/v4/internal/tui/kit/layout"
	"github.com/Josepavese/nido/v4/internal/tui/kit/widget"
	"testing"

	"github.com/Josepavese/nido/v4/internal/tui/app/ops"
	tea "github.com/charmbracelet/bubbletea"
)

func TestBackgroundRemovalKeepsConfirmedImageIdentity(t *testing.T) {
	r := NewRegistry(nil)
	a := RegistryItem{Name: "original", Version: "1", IsLocal: true}
	r.Update(ops.CacheListMsg{Items: []ops.CacheItem{{Name: a.Name, Version: a.Version}}})
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

func TestRegistryOpenAndPullRemainDistinct(t *testing.T) {
	r := NewRegistry(nil)
	r.DetailView.UpdateItem(RegistryItem{Name: "bird", Version: "1"})
	r.Resize(layout.NewRect(0, 0, 120, 30))
	_, cmd := r.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil || cmd().(ops.RequestPullMsg).Image != "bird:1" {
		t.Fatal("wide Enter lost its pull action")
	}
	r.Resize(layout.NewRect(0, 0, 60, 13))
	_, cmd = r.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if r.MasterDetail.ActiveFocus != widget.FocusDetail {
		t.Fatal("compact Enter did not open detail")
	}
	if cmd != nil {
		if _, pull := cmd().(ops.RequestPullMsg); pull {
			t.Fatal("opening detail also pulled an image")
		}
	}
	_, cmd = r.Update(tea.KeyMsg{Type: tea.KeyEnter})
	// The command is batched by MasterDetail, so inspect its children.
	found := false
	var inspect func(tea.Cmd)
	inspect = func(c tea.Cmd) {
		if c == nil {
			return
		}
		switch msg := c().(type) {
		case ops.RequestPullMsg:
			found = msg.Image == "bird:1"
		case tea.BatchMsg:
			for _, child := range msg {
				inspect(child)
			}
		}
	}
	inspect(cmd)
	if !found {
		t.Fatal("detail Enter did not pull")
	}
}

func TestRefreshPreservesSelectedArchitectureAndProvider(t *testing.T) {
	r := NewRegistry(nil)
	r.remoteItems = []RegistryItem{
		{Name: "bird", Version: "1", Arch: "amd64", Provider: "official"},
		{Name: "bird", Version: "1", Arch: "arm64", Provider: "official"},
		{Name: "bird", Version: "1", Arch: "arm64", Provider: "nido"},
	}
	r.rebuildSidebar()
	for index, item := range r.Sidebar.Items() {
		candidate, ok := item.(RegistryItem)
		if !ok {
			continue
		}
		r.Sidebar.Select(index)
		r.DetailView.UpdateItem(candidate)
		r.Update(ops.CacheListMsg{})
		if got := r.Sidebar.SelectedItem(); got != candidate || r.DetailView.Item != candidate {
			t.Fatalf("refresh replaced %#v with %#v (detail %#v)", candidate, got, r.DetailView.Item)
		}
	}
}

func TestRefreshDuringNavigationUsesHighlightedItem(t *testing.T) {
	r := NewRegistry(nil)
	r.remoteItems = []RegistryItem{{Name: "first"}, {Name: "second"}, {Name: "third"}}
	r.rebuildSidebar()
	r.Resize(layout.NewRect(0, 0, 120, 30))
	r.Focus()
	old := r.Sidebar.Select(r.Sidebar.Index())
	r.Update(tea.KeyMsg{Type: tea.KeyDown}) // Selection notification is still queued.
	selected := r.Sidebar.SelectedItem().(RegistryItem)
	if selected.Name != "second" {
		t.Fatalf("Down did not select second: %v", selected)
	}
	r.Update(ops.CacheListMsg{})
	if r.Sidebar.SelectedItem() != selected || r.DetailView.Item != selected {
		t.Fatal("background refresh moved selection backwards")
	}
	// An older asynchronous notification must not replace the current detail.
	r.Update(old())
	if r.DetailView.Item != selected {
		t.Fatal("late selection notification replaced current detail")
	}
}
