package hatchery

import (
	"testing"

	"github.com/Josepavese/nido/v4/internal/tui/app/ops"
)

func TestBackgroundRefreshPreservesDraftAndPinsTemplateConfirmation(t *testing.T) {
	h := NewHatchery(nil)
	h.Update(ops.SourcesLoadedMsg{Sources: []string{"[TEMPLATE] selected", "[TEMPLATE] other"}})
	h.Incubator.input.SetValue("draft-vm")
	h.Incubator.rawArgsInput.SetValue("-S")
	form := h.Incubator.Form
	h.Update(ops.TemplateUsageMsg{Name: "selected"})
	h.Update(ops.SourcesLoadedMsg{Sources: []string{"[TEMPLATE] other", "[TEMPLATE] selected"}})
	if h.Incubator.Form != form || h.Incubator.input.Value() != "draft-vm" || h.Incubator.rawArgsInput.Value() != "-S" || h.Incubator.SelectedSource.Label != "selected" {
		t.Fatal("background source update overwrote draft")
	}
	h.Update(ops.SourcesLoadedMsg{Sources: []string{"[TEMPLATE] other"}})
	if h.Incubator.input.Value() != "draft-vm" || !h.Incubator.spawnBtn.Disabled {
		t.Fatal("removed source destroyed draft or remained usable")
	}
	cmd := h.ConfirmDelete.OnConfirm()
	if cmd == nil {
		t.Fatal("lost confirmed target")
	}
	if msg := cmd().(ops.RequestDeleteTemplateMsg); msg.Name != "selected" {
		t.Fatal("confirmation changed target", msg)
	}
	h.Update(ops.VMListMsg{Items: []ops.VMItem{{Name: "another-process-vm"}}})
	if !h.ExistingVMs["another-process-vm"] {
		t.Fatal("modal blocked background fleet validation")
	}
}
