package fleet

import (
	"context"
	"testing"

	"github.com/Josepavese/nido/v4/internal/provider"
	"github.com/Josepavese/nido/v4/internal/tui/app/ops"
	"github.com/Josepavese/nido/v4/internal/tui/kit/widget"
	tea "github.com/charmbracelet/bubbletea"
)

type snapshotTestProvider struct {
	provider.VMProvider
	snapshots []provider.DiskSnapshot
	calls     []string
}

func (p *snapshotTestProvider) SnapshotList(context.Context, string) ([]provider.DiskSnapshot, error) {
	p.calls = append(p.calls, "list")
	return p.snapshots, nil
}

func (p *snapshotTestProvider) SnapshotCreate(_ context.Context, _ string, snapshot string) (provider.DiskSnapshot, error) {
	p.calls = append(p.calls, "create:"+snapshot)
	return provider.DiskSnapshot{Name: snapshot, ID: 1}, nil
}

func (p *snapshotTestProvider) SnapshotRestore(_ context.Context, _ string, snapshot string) error {
	p.calls = append(p.calls, "restore:"+snapshot)
	return nil
}

func (p *snapshotTestProvider) SnapshotDelete(_ context.Context, _ string, snapshot string) error {
	p.calls = append(p.calls, "delete:"+snapshot)
	return nil
}

func fleetWithState(prov provider.VMProvider, state string) *Fleet {
	f := NewFleet(prov)
	f.Sidebar.SetItems([]widget.SidebarItem{FleetItem{Name: "vm-a", State: state}})
	return f
}

func TestTUIHRequestsGuestShutdownNotForcedStop(t *testing.T) {
	f := fleetWithState(nil, "running")
	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if cmd == nil {
		t.Fatal("shutdown shortcut emitted no command")
	}
	msg, ok := cmd().(ops.RequestOpMsg)
	if !ok || msg.Op != ops.OpShutdown || msg.Name != "vm-a" {
		t.Fatalf("shutdown shortcut = %#v", msg)
	}
	if msg.Op == ops.OpStop {
		t.Fatal("shutdown shortcut used forced stop")
	}
}

func TestTUISnapshotsRequireStoppedVM(t *testing.T) {
	prov := &snapshotTestProvider{}
	f := fleetWithState(prov, "running")
	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if cmd != nil {
		t.Fatal("running VM snapshots opened a fetch")
	}
	if !f.ErrorModal.IsActive() {
		t.Fatal("running VM snapshots did not explain the precondition")
	}
	if len(prov.calls) != 0 {
		t.Fatalf("running VM touched snapshots: %v", prov.calls)
	}
}

func TestTUISnapshotListCreateRestoreDelete(t *testing.T) {
	prov := &snapshotTestProvider{snapshots: []provider.DiskSnapshot{{Name: "before-upgrade", ID: 1}}}
	f := fleetWithState(prov, "stopped")

	_, fetch := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if fetch == nil {
		t.Fatal("stopped VM did not fetch snapshots")
	}
	if _, refresh := f.Update(fetch()); refresh != nil {
		t.Fatalf("snapshot list unexpectedly emitted %v", refresh)
	}
	if !f.SnapshotList.IsActive() || len(f.SnapshotList.List.Items()) != 1 {
		t.Fatalf("snapshot list = active %v items %d", f.SnapshotList.IsActive(), len(f.SnapshotList.List.Items()))
	}

	_, _ = f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if f.SnapshotList.IsActive() || !f.SnapshotCreate.IsActive() {
		t.Fatal("create shortcut did not replace list with form")
	}
	create := f.SnapshotCreate.OnConfirm(map[string]string{"tag": "before upgrade"})
	if create == nil {
		t.Fatal("create form emitted no command")
	}
	createResult, ok := create().(ops.SnapshotResultMsg)
	if !ok || createResult.Action != "create" || createResult.Snapshot != "before upgrade" {
		t.Fatalf("create result = %#v", createResult)
	}
	_, refresh := f.Update(createResult)
	if refresh == nil {
		t.Fatal("snapshot mutation did not refresh list")
	}
	f.Update(refresh())
	if !f.SnapshotList.IsActive() || len(f.SnapshotList.List.Items()) != 1 {
		t.Fatalf("post-create list = active %v items %d", f.SnapshotList.IsActive(), len(f.SnapshotList.List.Items()))
	}
	f.SnapshotList.List.Select(0)
	if selected := f.SnapshotList.List.SelectedItem(); selected == nil {
		t.Fatalf("no selected item after Select: %#v", f.SnapshotList.List.Items())
	}
	_, _ = f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if !f.SnapshotConfirm.IsActive() || f.snapshotAction != "restore" || f.snapshotPending != "before-upgrade" {
		t.Fatalf("restore prompt = active %v action %q snapshot %q", f.SnapshotConfirm.IsActive(), f.snapshotAction, f.snapshotPending)
	}
	restore := f.SnapshotConfirm.OnConfirm()
	if restore == nil {
		t.Fatal("restore confirmation emitted no command")
	}
	restoreResult, ok := restore().(ops.SnapshotResultMsg)
	if !ok || restoreResult.Action != "restore" {
		t.Fatalf("restore result = %#v", restoreResult)
	}
	if _, refresh := f.Update(restoreResult); refresh == nil {
		t.Fatal("restore did not refresh list")
	} else {
		f.Update(refresh())
	}
	f.SnapshotList.List.Select(0)
	_, _ = f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if !f.SnapshotConfirm.IsActive() || f.snapshotAction != "delete" {
		t.Fatalf("delete prompt = active %v action %q", f.SnapshotConfirm.IsActive(), f.snapshotAction)
	}
	deleteCmd := f.SnapshotConfirm.OnConfirm()
	if deleteCmd == nil {
		t.Fatal("delete confirmation emitted no command")
	}
	deleteResult, ok := deleteCmd().(ops.SnapshotResultMsg)
	if !ok || deleteResult.Action != "delete" || deleteResult.Snapshot != "before-upgrade" {
		t.Fatalf("delete result = %#v", deleteResult)
	}
	if len(prov.calls) < 6 {
		t.Fatalf("provider calls = %v", prov.calls)
	}
}

func TestTUISnapshotResultCannotRetargetAnotherVM(t *testing.T) {
	prov := &snapshotTestProvider{}
	f := fleetWithState(prov, "stopped")
	_, fetch := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	f.Update(fetch())
	f.snapshotVM = "other-vm"
	if cmd := f.onSnapshotList(ops.SnapshotListMsg{VM: "vm-a", Snapshots: []provider.DiskSnapshot{{Name: "late"}}}); cmd != nil {
		t.Fatal("stale snapshot list emitted a command")
	}
}
