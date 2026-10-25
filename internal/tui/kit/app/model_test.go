package app

import (
	"github.com/Josepavese/nido/v5/internal/tui/kit/layout"
	"github.com/Josepavese/nido/v5/internal/tui/kit/view"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"strings"
	"testing"
)

type inputPage struct {
	view.BaseViewlet
	keys    int
	updates int
}

func (p *inputPage) Update(msg tea.Msg) (view.Viewlet, tea.Cmd) {
	p.updates++
	if _, ok := msg.(tea.KeyMsg); ok {
		p.keys++
	}
	return p, nil
}
func (p *inputPage) Init() tea.Cmd            { return nil }
func (p *inputPage) View() string             { return "draft" }
func (p *inputPage) HasActiveTextInput() bool { return true }
func TestResizeSuspendsOnlyInputAndRecovers(t *testing.T) {
	a := NewApp()
	page := &inputPage{}
	a.AddRoute("draft", "DRAFT", "", page)
	a.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	a.Update(tea.KeyMsg{Type: tea.KeyRight})
	if page.keys != 1 {
		t.Fatal("caret key intercepted")
	}
	a.Update(tea.WindowSizeMsg{Width: 20, Height: 5})
	a.Update(tea.KeyMsg{Type: tea.KeyEnter})
	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if page.keys != 1 {
		t.Fatal("hidden input accepted")
	}
	updates := page.updates
	a.Update(struct{}{})
	if page.updates <= updates {
		t.Fatal("background update blocked")
	}
	if lipgloss.Width(a.View()) > 20 || lipgloss.Height(a.View()) > 5 {
		t.Fatal("resize notice overflow")
	}
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if !strings.Contains(a.View(), "draft") {
		t.Fatal("recovery lost route")
	}
	a.Update(tea.WindowSizeMsg{Width: 0, Height: 0})
	if a.View() != "" {
		t.Fatal("zero size renders outside bounds")
	}
	if layout.IsViable(20, 5) {
		t.Fatal("invalid fixture")
	}
}
func TestMouseRoutesUseRenderedTabs(t *testing.T) {
	a := NewApp()
	first, second := &inputPage{}, &inputPage{}
	a.AddRoute("a", "ONE", "", first)
	a.AddRoute("b", "SECOND", "", second)
	a.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	a.Update(tea.MouseMsg{X: 7, Y: 0, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	if a.Shell.ActiveViewlet() != second {
		t.Fatal("rendered second tab not clickable")
	}
}

func TestQuitControlCodesSurviveEscapeModifier(t *testing.T) {
	for _, key := range []tea.KeyType{tea.KeyCtrlC, tea.KeyCtrlQ} {
		for _, width := range []int{12, 40, 120} {
			a := NewApp()
			a.Update(tea.WindowSizeMsg{Width: width, Height: 10})
			calls := 0
			a.OnQuit = func() { calls++ }
			_, cmd := a.Update(tea.KeyMsg{Type: key, Alt: true})
			if cmd == nil || calls != 1 {
				t.Fatal("modified quit key did not reach lifecycle")
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Fatal("modified control key was not an exit")
			}
		}
	}
}
