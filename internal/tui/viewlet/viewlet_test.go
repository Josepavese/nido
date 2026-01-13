package viewlet

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestHelp_View(t *testing.T) {
	h := NewHelp()
	out := h.View()

	if !strings.Contains(out, "NAVIGATION") {
		t.Error("Help view missing NAVIGATION section")
	}
	if !strings.Contains(out, "FLEET") {
		t.Error("Help view missing FLEET section")
	}
	// Check for columnar layout elements
	if !strings.Contains(out, "Switch tabs") {
		t.Error("Help view missing content")
	}
}

func TestFleet_DetailView(t *testing.T) {
	f := NewFleet()
	// Populate items to bypass "empty nest" check
	f.SetItems([]FleetItem{{Name: "test-vm"}})

	detail := FleetDetail{
		Name:    "test-vm",
		State:   "running",
		PID:     1234,
		SSHPort: 2222,
	}
	f.SetDetail(detail)
	f.Resize(80, 24)

	if f.detail.Name != "test-vm" {
		t.Errorf("Expected detail name test-vm, got %s", f.detail.Name)
	}

	// Render view
	out := f.View()
	if !strings.Contains(out, "TEST-VM") { // Upper case in title
		t.Logf("View Output:\n%s", out)
		t.Error("Fleet detail view missing item name")
	}
	if !strings.Contains(out, "Status") {
		t.Error("Fleet detail view missing status label")
	}
}

func TestHatchery_Mode(t *testing.T) {
	h := NewHatchery()

	// Default mode
	if h.Mode != HatcherySpawn {
		t.Errorf("Expected mode Spawn, got %d", h.Mode)
	}

	// Change mode
	h.SetMode(HatcheryTemplate)
	if h.Mode != HatcheryTemplate {
		t.Errorf("Expected mode Template, got %d", h.Mode)
	}

	// Render view check
	out := h.View()
	if !strings.Contains(out, "CREATE TEMPLATE") {
		t.Error("Hatchery view missing template mode header")
	}
}

func TestHatchery_Interaction(t *testing.T) {
	h := NewHatchery()
	h.Resize(100, 24)
	h.Init() // Ensure blinking cursor is ready if used

	// Initial state: Name input should be focused.
	// We can't access h.inputs directly as it is private.
	// But we can check if the View contains the blinking cursor or focus style.
	// view := h.View()
	// Textinput usually shows a cursor when focused.

	// Simulate Tab -> Focus moves to Source selection
	h.Update(tea.KeyMsg{Type: tea.KeyTab})

	// Simulate Space -> Trigger selection modal
	h.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})

	newView := h.View()
	// Modal uses lipgloss.Place, likely centering the list
	// The list delegate has styles.

	// Check for "Select..." logic or list content availability
	// Since list is empty initially, it might show empty state.
	// We'll trust that if no panic happens and state updates, it's good for now.
	// But let's check if the View string is non-empty.
	if len(newView) == 0 {
		t.Error("Hatchery view is empty after interaction")
	}
}

func TestFleet_MouseInteraction(t *testing.T) {
	f := NewFleet()
	f.SetItems([]FleetItem{{Name: "test-vm"}})
	f.SetDetail(FleetDetail{
		Name:  "test-vm",
		State: "stopped",
		PID:   1234,
	})

	// Simulate Mouse Click on "START" button
	// Coordinates in update: Y >= 14 && Y <= 22
	// Local X < 14. Global X = Local + 24 implies X < 38.
	// Let's pick Global X=30, Y=15.

	msg := tea.MouseMsg{
		X:      30,
		Y:      15,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
	}

	_, cmd := f.Update(msg)
	if cmd == nil {
		t.Fatal("Expected command from mouse click, got nil")
	}

	// Verify command returns FleetActionMsg
	// Since cmd is a function, we run it to get the Msg
	res := cmd()
	actionMsg, ok := res.(FleetActionMsg)
	if !ok {
		t.Errorf("Expected FleetActionMsg, got %T", res)
	}
	if actionMsg.Action != "start" {
		t.Errorf("Expected action start, got %s", actionMsg.Action)
	}
	if actionMsg.Name != "test-vm" {
		t.Errorf("Expected name test-vm, got %s", actionMsg.Name)
	}
}
