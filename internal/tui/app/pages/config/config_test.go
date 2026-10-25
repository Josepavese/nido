package config

import (
	"errors"
	appconfig "github.com/Josepavese/nido/v4/internal/config"
	"github.com/Josepavese/nido/v4/internal/tui/app/ops"
	"github.com/Josepavese/nido/v4/internal/tui/kit/layout"
	fv "github.com/Josepavese/nido/v4/internal/tui/kit/view"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
)

func TestCacheSelectionUsesApplicationMessage(t *testing.T) {
	model := NewConfig(&appconfig.Config{}, nil)
	_, command := model.Update(fv.SelectionMsg{Item: ConfigItem{Key: "CACHE"}})
	if command == nil {
		t.Fatal("cache selection did not request stats")
	}
	// Update returns a batch; inspect the requested action rather than a local
	// compatibility type that the application's operation router cannot handle.
	message := command()
	var hasRequest func(tea.Msg) bool
	hasRequest = func(message tea.Msg) bool {
		if _, ok := message.(ops.RequestCacheMsg); ok {
			return true
		}
		if batch, ok := message.(tea.BatchMsg); ok {
			for _, child := range batch {
				if child != nil && hasRequest(child()) {
					return true
				}
			}
		}
		return false
	}
	found := hasRequest(message)

	if !found {
		t.Fatal("application cache request missing")
	}
}

func TestWideOverlayMouseUsesPageCoordinates(t *testing.T) {
	model := NewConfig(&appconfig.Config{}, nil)
	model.Resize(layout.NewRect(0, 0, 120, 30))
	model.Pages.SwitchTo("SETTINGS")
	page, ok := model.Pages.ActivePage().(*ConfigPageGlobalForm)
	if !ok {
		t.Fatalf("unexpected settings page %T", model.Pages.ActivePage())
	}
	page.Modal.Show()
	rendered := ansi.Strip(model.View())
	clicked := false
	for y, line := range strings.Split(rendered, "\n") {
		if x := strings.Index(line, "No"); x >= 0 {
			model.HandleMouse(x, y, tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
			clicked = true
			break
		}
	}
	if !clicked || page.Modal.IsActive() {
		t.Fatal("full-page modal was hit-tested in detail-pane coordinates")
	}
}

func TestDiagnosticFailureUsesVisibleOverlay(t *testing.T) {
	model := NewConfig(&appconfig.Config{}, nil)
	model.Pages.SwitchTo("DOCTOR")
	page := model.Pages.ActivePage().(*ConfigPageDoctor)
	page.Focus()
	page.Update(ops.DoctorResultMsg{Err: errors.New("diagnostic fixture failure")})
	model.Resize(layout.NewRect(0, 0, 40, 8))
	rendered := ansi.Strip(model.View())
	t.Log("\n" + rendered)
	if !page.IsModalActive() || !strings.Contains(rendered, "Diagnostics failed") || !strings.Contains(rendered, "Close") {
		t.Fatal("diagnostic error is hidden outside form viewport")
	}
}
