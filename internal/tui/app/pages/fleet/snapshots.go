package fleet

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Josepavese/nido/v4/internal/provider"
	"github.com/Josepavese/nido/v4/internal/tui/app/ops"
	"github.com/Josepavese/nido/v4/internal/tui/kit/widget"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func (f *Fleet) initSnapshotModals() {
	f.SnapshotList = widget.NewListModal("Snapshots", nil, 76, 18,
		func(item list.Item) tea.Cmd {
			if name, ok := item.(widget.SimpleListItem); ok && string(name) != "" {
				return f.promptSnapshotAction("restore")
			}
			return nil
		},
		func() tea.Cmd {
			f.closeSnapshotManager()
			return nil
		})

	f.SnapshotCreate = widget.NewFormModal("Create Disk Snapshot", func(values map[string]string) tea.Cmd {
		tag := values["tag"]
		if f.snapshotVM == "" || f.provider == nil {
			return nil
		}
		return ops.CreateSnapshot(f.provider, f.snapshotVM, tag)
	}, func() tea.Cmd {
		return f.reopenSnapshotList()
	})
	f.SnapshotCreate.Description = "Disk-only snapshot of a stopped VM; guest memory is not captured."
	f.SnapshotCreate.AddRow(&widget.FormEntry{
		Key:         "tag",
		Label:       "Tag",
		Placeholder: "before-upgrade",
		MaxChars:    128,
		Validator: func(value string) error {
			if strings.TrimSpace(value) == "" {
				return errors.New("snapshot tag is required")
			}
			return provider.ValidateSnapshotName(value)
		},
	})

	f.SnapshotConfirm = widget.NewModal("Snapshot", "", func() tea.Cmd {
		return f.submitSnapshotAction()
	}, func() tea.Cmd {
		return f.reopenSnapshotList()
	})
	f.SnapshotConfirm.SetLevel(widget.ModalLevelDanger)
}

func (f *Fleet) openSnapshotManager(vm string) tea.Cmd {
	if f.provider == nil {
		return nil
	}
	f.snapshotVM = vm
	f.snapshotPending = ""
	f.snapshotAction = ""
	f.SnapshotList.Title = snapshotTitle(vm, -1)
	f.SnapshotList.SetItems(nil)
	f.SnapshotList.Show()
	return ops.FetchSnapshots(f.provider, vm)
}

func (f *Fleet) onSnapshotList(msg ops.SnapshotListMsg) tea.Cmd {
	if msg.VM == "" || msg.VM != f.snapshotVM {
		return nil
	}
	if msg.Err != nil {
		f.showSnapshotError("List failed", msg.Err)
		return nil
	}
	items := make([]list.Item, 0, len(msg.Snapshots))
	for _, snapshot := range msg.Snapshots {
		items = append(items, widget.SimpleListItem(snapshot.Name))
	}
	f.SnapshotList.Title = snapshotTitle(msg.VM, len(items))
	f.SnapshotList.SetItems(items)
	if !f.SnapshotList.IsActive() {
		f.SnapshotList.Show()
	}
	return nil
}

func (f *Fleet) onSnapshotResult(msg ops.SnapshotResultMsg) tea.Cmd {
	if msg.VM == "" || msg.VM != f.snapshotVM {
		return nil
	}
	if msg.Err != nil {
		f.showSnapshotError(snapshotActionTitle(msg.Action)+" failed", msg.Err)
		return nil
	}
	f.snapshotPending = ""
	f.snapshotAction = ""
	f.SnapshotCreate.Hide()
	f.SnapshotConfirm.Hide()
	f.SnapshotList.Show()
	f.SnapshotList.SetItems(nil)
	f.SnapshotList.Title = snapshotTitle(f.snapshotVM, -1)
	return ops.FetchSnapshots(f.provider, f.snapshotVM)
}

func (f *Fleet) promptSnapshotAction(action string) tea.Cmd {
	if action != "restore" && action != "delete" {
		return nil
	}
	selected := f.SnapshotList.List.SelectedItem()
	if selected == nil {
		return nil
	}
	item, ok := selected.(widget.SimpleListItem)
	if !ok || string(item) == "" {
		return nil
	}
	f.snapshotAction = action
	f.snapshotPending = string(item)
	f.SnapshotList.Hide()
	verb := "Restore"
	if action == "delete" {
		verb = "Delete"
	}
	f.SnapshotConfirm.Title = verb + " Snapshot"
	f.SnapshotConfirm.Message = fmt.Sprintf("VM: %s\nSnapshot: %s\n\n%s this disk snapshot?", f.snapshotVM, f.snapshotPending, verb)
	f.SnapshotConfirm.Show()
	return nil
}

func (f *Fleet) submitSnapshotAction() tea.Cmd {
	vm, tag, action := f.snapshotVM, f.snapshotPending, f.snapshotAction
	if vm == "" || tag == "" || f.provider == nil {
		return nil
	}
	switch action {
	case "restore":
		return ops.RestoreSnapshot(f.provider, vm, tag)
	case "delete":
		return ops.DeleteSnapshot(f.provider, vm, tag)
	default:
		return nil
	}
}

func (f *Fleet) reopenSnapshotList() tea.Cmd {
	if f.snapshotVM == "" {
		return nil
	}
	f.SnapshotList.Title = snapshotTitle(f.snapshotVM, -1)
	f.SnapshotList.Show()
	return nil
}

func (f *Fleet) closeSnapshotManager() {
	f.SnapshotList.Hide()
	f.SnapshotCreate.Hide()
	f.SnapshotConfirm.Hide()
	f.snapshotVM = ""
	f.snapshotPending = ""
	f.snapshotAction = ""
}

func (f *Fleet) showSnapshotError(title string, err error) {
	f.closeSnapshotManager()
	f.ErrorModal.Title = "Snapshot Error"
	f.ErrorModal.Message = fmt.Sprintf("%s:\n%v", title, err)
	f.ErrorModal.Show()
}

func snapshotActionTitle(action string) string {
	if action == "" {
		return "Snapshot"
	}
	return strings.ToUpper(action[:1]) + action[1:]
}

func snapshotTitle(vm string, count int) string {
	title := fmt.Sprintf("Snapshots · %s · c create  r restore  d delete", vm)
	if count == 0 {
		title += "  (none)"
	}
	return title
}
