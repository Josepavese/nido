package config

import (
	appconfig "github.com/Josepavese/nido/internal/config"
	"github.com/Josepavese/nido/internal/tui/app/ops"
	fv "github.com/Josepavese/nido/internal/tui/kit/view"
	tea "github.com/charmbracelet/bubbletea"
	"testing"
)

func TestCacheSelectionUsesApplicationMessage(t *testing.T) {
	model := NewConfig(&appconfig.Config{})
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
