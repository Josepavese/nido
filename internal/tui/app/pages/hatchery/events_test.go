package hatchery

import (
	tea "github.com/charmbracelet/bubbletea"
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

func TestSourceRefreshUsesHighlightAndRejectsLateNotifications(t *testing.T) {
	h := NewHatchery(nil)
	h.Update(ops.SourcesLoadedMsg{Sources: []string{"[TEMPLATE] first", "[CLOUD] second", "[TEMPLATE] second"}})
	old := h.Sidebar.Select(0)
	h.Sidebar.Select(2) // The detail notification has not run yet.
	h.Update(ops.SourcesLoadedMsg{Sources: []string{"[TEMPLATE] second\trenamed", "[TEMPLATE] first", "[CLOUD] second"}})
	h.Update(old())
	selected := h.Sidebar.SelectedItem().(SourceItem)
	if selected.Type != "TEMPLATE" || selected.Label != "second" || h.Incubator.SelectedSource.Type != selected.Type || h.Incubator.SelectedSource.Label != selected.Label {
		t.Fatal("refresh or late notification selected the wrong source", selected, h.Incubator.SelectedSource)
	}
	h.Incubator.input.SetValue("keep-my-draft")
	h.Update(ops.SourcesLoadedMsg{Sources: []string{"[CLOUD] second", "[TEMPLATE] second\tnew title"}})
	if h.Incubator.input.Value() != "keep-my-draft" || h.Incubator.SelectedSource.Display != "new title" {
		t.Fatal("label refresh reset draft")
	}
}

func TestRepeatedSourceRefreshCannotReplaceRemovedSourceDraft(t *testing.T) {
	h := NewHatchery(nil)
	h.Update(ops.SourcesLoadedMsg{Sources: []string{"[TEMPLATE] selected", "[TEMPLATE] other"}})
	h.Incubator.input.SetValue("keep-draft")
	for range 2 {
		h.Update(ops.SourcesLoadedMsg{Sources: []string{"[TEMPLATE] other"}})
	}
	if h.Incubator.input.Value() != "keep-draft" || !h.Incubator.spawnBtn.Disabled {
		t.Fatal("refresh selected a fallback and discarded the removed source draft")
	}
	h.Sidebar.Focus()
	h.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if h.Incubator.SelectedSource.Label != "other" || h.Incubator.spawnBtn.Disabled {
		t.Fatal("explicitly opening the available source did not select it")
	}
}
