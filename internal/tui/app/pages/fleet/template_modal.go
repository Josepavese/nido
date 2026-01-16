package fleet

import (
	"fmt"

	"github.com/Josepavese/nido/internal/tui/app/ops"
	widget "github.com/Josepavese/nido/internal/tui/kit/widget"
	tea "github.com/charmbracelet/bubbletea"
)

// CreateTemplateModal handles the UI for creating a template from a VM.
type CreateTemplateModal struct {
	VMName         string
	KnownTemplates []string

	// Component
	Modal *widget.PromptModal
}

// NewCreateTemplateModal creates the modal.
func NewCreateTemplateModal() *CreateTemplateModal {
	m := &CreateTemplateModal{
		Modal: widget.NewPromptModal("Create Template", "", "Template Name", "e.g. customized-ubuntu"),
	}

	// Connect validation and submission
	m.Modal.Input.Validator = m.Validate
	m.Modal.OnSubmit = m.HandleSubmit

	return m
}

// Validate checks for duplicates and empty values.
func (m *CreateTemplateModal) Validate(s string) error {
	if s == "" {
		return fmt.Errorf("required")
	}
	for _, t := range m.KnownTemplates {
		if t == s {
			return fmt.Errorf("exists")
		}
	}
	return nil
}

// HandleSubmit constructs the final operation message.
func (m *CreateTemplateModal) HandleSubmit(name string) tea.Cmd {
	return func() tea.Msg {
		// NOTE: Name is VMName, Source is TemplateName (Target) as per wiring.go convention
		return ops.RequestCreateTemplateMsg{Name: m.VMName, Source: name}
	}
}

// Show opens the modal for the given VM.
func (m *CreateTemplateModal) Show(vmName string, known []string) tea.Cmd {
	m.VMName = vmName
	m.KnownTemplates = known
	m.Modal.Message = fmt.Sprintf("Create a template from '%s'", vmName)
	return m.Modal.Show("")
}

// Hide closes the modal.
func (m *CreateTemplateModal) Hide() {
	m.Modal.Hide()
}

// IsActive returns the state.
func (m *CreateTemplateModal) IsActive() bool {
	return m.Modal.IsActive()
}

// Update handles input events.
func (m *CreateTemplateModal) Update(msg tea.Msg) tea.Cmd {
	_, cmd := m.Modal.Update(msg)
	return cmd
}

// View renders the modal overlay.
func (m *CreateTemplateModal) View(parentWidth, parentHeight int) string {
	return m.Modal.View(parentWidth, parentHeight)
}
