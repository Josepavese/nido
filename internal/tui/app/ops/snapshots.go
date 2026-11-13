package ops

import (
	"context"

	"github.com/Josepavese/nido/v4/internal/provider"
	tea "github.com/charmbracelet/bubbletea"
)

// SnapshotListMsg carries the disk snapshots for one VM into the UI owner
// thread. The VM identity prevents a late result from retargeting a newer dialog.
type SnapshotListMsg struct {
	VM        string
	Snapshots []provider.DiskSnapshot
	Err       error
}

// SnapshotResultMsg reports one completed snapshot mutation.
type SnapshotResultMsg struct {
	VM       string
	Action   string
	Snapshot string
	Err      error
}

func FetchSnapshots(prov provider.VMProvider, name string) tea.Cmd {
	return func() tea.Msg {
		snapshots, err := prov.SnapshotList(context.Background(), name)
		return SnapshotListMsg{VM: name, Snapshots: snapshots, Err: err}
	}
}

func CreateSnapshot(prov provider.VMProvider, name, snapshot string) tea.Cmd {
	return func() tea.Msg {
		_, err := prov.SnapshotCreate(context.Background(), name, snapshot)
		return SnapshotResultMsg{VM: name, Action: "create", Snapshot: snapshot, Err: err}
	}
}

func RestoreSnapshot(prov provider.VMProvider, name, snapshot string) tea.Cmd {
	return func() tea.Msg {
		err := prov.SnapshotRestore(context.Background(), name, snapshot)
		return SnapshotResultMsg{VM: name, Action: "restore", Snapshot: snapshot, Err: err}
	}
}

func DeleteSnapshot(prov provider.VMProvider, name, snapshot string) tea.Cmd {
	return func() tea.Msg {
		err := prov.SnapshotDelete(context.Background(), name, snapshot)
		return SnapshotResultMsg{VM: name, Action: "delete", Snapshot: snapshot, Err: err}
	}
}
