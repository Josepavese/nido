package widget

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/Josepavese/nido/v5/internal/tui/kit/layout"
	"github.com/Josepavese/nido/v5/internal/tui/kit/theme"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func assertFits(t *testing.T, content string, w, h int) {
	t.Helper()
	if !utf8.ValidString(content) {
		t.Fatal("invalid Unicode")
	}
	if lipgloss.Width(content) > w || lipgloss.Height(content) > h {
		t.Fatalf("overflow: %dx%d outside %dx%d\n%s", lipgloss.Width(content), lipgloss.Height(content), w, h, content)
	}
}
func TestResponsiveFieldsAndRows(t *testing.T) {
	for _, w := range []int{1, 8, 24, 40, 49, 60, 100} {
		t.Run(fmt.Sprint(w), func(t *testing.T) {
			input := NewInput("Mémoire 🐦", "", nil)
			input.SetValue(strings.Repeat("città 🪺", 20))
			row := NewRow(input, NewToggle("Linked Clones", true))
			content := row.View(w)
			assertFits(t, content, w, 20)
			if w == 40 && lipgloss.Height(content) != 6 {
				t.Fatal("narrow row did not reflow")
			}
			if w == 60 && lipgloss.Height(content) != 3 {
				t.Fatal("wide row unnecessarily stacked")
			}
			// Cursor's width agrees with the value's cell budget including validation.
			input.Error = "invalid"
			input.View(w)
			if input.InputModel.Width > max(1, w-layout.FieldFrameWidth-fieldLabelWidth(input.Label, w)-4) {
				t.Fatal("cursor overflows error indicator")
			}
		})
	}
}
func TestFormViewportFollowsNestedFocusAndMouse(t *testing.T) {
	a, b := NewInput("First", "", nil), NewInput("Second", "", nil)
	c := NewInput("Last", "", nil)
	form := NewForm(NewCard(theme.IconNest, "Draft", ""), NewRow(a, b), c)
	form.Focus()
	form.Viewport(40, 8)
	form.NextField() // second input in a stacked row
	view := form.Viewport(40, 8)
	if !strings.Contains(ansi.Strip(view), "Second") {
		t.Fatal("nested focus lost below viewport")
	}
	form.NextField()
	view = form.Viewport(40, 8)
	if !strings.Contains(ansi.Strip(view), "Last") {
		t.Fatal("last input lost")
	}
	if !strings.Contains(ansi.Strip(view), "pgup/pgdn") {
		t.Fatal("scroll affordance missing")
	}
	// Scrolled hit coordinates must focus the visible row, not a hidden one.
	_, handled := form.HandleMouse(3, 1, tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	if !handled {
		t.Fatal("visible form click not handled")
	}
	before := form.viewport.offset
	form.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	form.Viewport(40, 8)
	if form.viewport.offset >= before {
		t.Fatal("manual scroll snapped back to focus")
	}
	a.SetValue("draft")
	form.Viewport(100, 40)
	form.Viewport(40, 8)
	if a.Value() != "draft" {
		t.Fatal("resize discarded draft")
	}
}
func TestOverlayBoundsAndCancelHit(t *testing.T) {
	for _, size := range [][2]int{{120, 32}, {60, 13}, {40, 8}} {
		w, h := size[0], size[1]
		cancelled, confirmed := false, false
		modal := NewModal("Delete città 🐦", strings.Repeat("Important context. ", 20), func() tea.Cmd { confirmed = true; return nil }, func() tea.Cmd { cancelled = true; return nil })
		modal.Show()
		content := modal.View(w, h)
		assertFits(t, content, w, h)
		if !strings.Contains(ansi.Strip(content), "Yes") || !strings.Contains(ansi.Strip(content), "No") {
			t.Fatal("controls cropped")
		}
		modal.HandleMouse(modal.cancelHit.X, modal.cancelHit.Y, tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
		if !cancelled || confirmed {
			t.Fatal("cancel hit triggered wrong action")
		}
		modal.Show()
		modal.Update(tea.KeyMsg{Type: tea.KeyEnter})
		if confirmed {
			t.Fatal("confirmation no longer defaults to safe choice")
		}
		form := NewFormModal("Port", nil, nil)
		form.AddRow(&FormEntry{Key: "host", Label: "Host"}, &FormEntry{Key: "guest", Label: "Guest"})
		form.Show()
		assertFits(t, form.View(w, h), w, h)
		choices := NewListModal("Themes", []list.Item{SimpleListItem("First"), SimpleListItem("Second")}, 60, 20, nil, nil)
		choices.Show()
		assertFits(t, choices.View(w, h), w, h)
	}
}
func TestTabPixelsAndHitsHaveOneGeometry(t *testing.T) {
	for _, w := range []int{40, 60, 80, 120} {
		for active := range 5 {
			bar := RenderTabs(TabsRenderOptions{Routes: []string{"FLEET", "HATCHERY", "REGISTRY", "SYSTEM", "NIDO"}, ActiveIndex: active, Width: w, Height: layout.HeaderSize(w, 24)})
			assertFits(t, bar.Content, w, layout.HeaderSize(w, 24))
			for i, hit := range bar.Hits {
				if hit.Width == 0 || hit.X+hit.Width > w {
					t.Fatalf("tab %d unreachable at width %d", i, w)
				}
				if i > 0 && bar.Hits[i-1].X+bar.Hits[i-1].Width > hit.X {
					t.Fatal("tab hit overlap")
				}
			}
		}
	}
}
func TestFooterKeepsWholeHintsAndExit(t *testing.T) {
	bar := NewStatusBar(40, StatusBarStyles{})
	bar.SetItems([]StatusBarItem{{Key: "←/→", Label: "flow"}, {Key: "enter", Label: "open"}, {Key: "enter", Label: "duplicate"}, {Key: "long-binding", Label: "too long"}, {Key: "^C", Label: "quit"}})
	text := ansi.Strip(bar.View())
	assertFits(t, text, 40, 1)
	if !strings.Contains(text, "^C quit") || !strings.Contains(text, "enter open") || strings.Contains(text, "long-binding") || strings.Contains(text, "duplicate") {
		t.Fatalf("bad hints %q", text)
	}
	bar.SetWidth(120)
	if strings.Count(ansi.Strip(bar.View()), "enter") != 1 {
		t.Fatal("duplicate shortcut survived in wide footer")
	}
}
func TestSplitViewSinglePaneAndPreference(t *testing.T) {
	sidebar, detail := &mockViewlet{}, &mockViewlet{}
	split := NewSplitView(sidebar, detail, lipgloss.NewStyle())
	for _, w := range []int{40, 60, 80, 120} {
		split.Resize(layout.NewRect(0, 0, w, 18))
		if split.SinglePane() {
			if sidebar.width != w || detail.width != w {
				t.Fatal("single pane child has stale size")
			}
		} else if sidebar.width+detail.width+layout.PaneDividerWidth != w {
			t.Fatal("split has inconsistent divider width")
		}
	}
	old := theme.Width.Sidebar
	t.Cleanup(func() { theme.Width.Sidebar = old })
	theme.Width.Sidebar = 55
	split.Resize(layout.NewRect(0, 0, 90, 18))
	if !split.SinglePane() {
		t.Fatal("theme preference bypassed adaptive pane policy")
	}
}

func TestShortViewportShowsFocusedValueInsteadOfItsBorder(t *testing.T) {
	input := NewInput("Name", "", nil)
	input.SetValue("visible-draft")
	form := NewForm(NewCard(theme.IconNest, "Header", ""), input)
	form.Focus()
	for _, height := range []int{1, 2, 4, 12, 2} {
		rendered := ansi.Strip(form.Viewport(32, height))
		if !strings.Contains(rendered, "visible-draft") {
			t.Fatalf("focused value hidden at height %d: %q", height, rendered)
		}
	}
}

func TestCompactListKeepsChoicesAndPagination(t *testing.T) {
	choices := NewListModal("Choices", []list.Item{SimpleListItem("First"), SimpleListItem("Second"), SimpleListItem("Third"), SimpleListItem("Fourth"), SimpleListItem("Fifth")}, 60, 20, nil, nil)
	choices.Show()
	rendered := ansi.Strip(choices.View(40, 8))
	if !strings.Contains(rendered, "First") || !strings.Contains(rendered, "Second") {
		t.Fatalf("pagination consumed choices: %s", rendered)
	}
	choices.List.Select(4)
	rendered = ansi.Strip(choices.View(40, 8))
	if !strings.Contains(rendered, "Fifth") {
		t.Fatal("last choice disappeared after resize")
	}
}

func TestActionButtonsReflowByTheirContent(t *testing.T) {
	confirm := NewSubmitButton("", "CONFIRM", nil)
	cancel := NewButton("", "Cancel", nil)
	cancel.Centered = true
	row := NewRow(confirm, cancel)
	if lipgloss.Height(row.View(32)) != 3 {
		t.Fatal("short buttons unnecessarily stacked")
	}
	cancel.Focus()
	row.View(32)
	if rect := elementAnchor(row, 32); rect.X == 0 {
		t.Fatal("focus geometry disagrees with horizontal action row")
	}
}

func TestEmptyStateKeepsItsInvitation(t *testing.T) {
	for _, size := range [][2]int{{120, 32}, {40, 8}} {
		rendered := RenderEmptyState(theme.IconNest, "Ready to hatch?", "Select a source to begin incubation.", size[0], size[1])
		assertFits(t, rendered, size[0], size[1])
		if !strings.Contains(ansi.Strip(rendered), "Ready to hatch?") {
			t.Fatal("empty invitation lost")
		}
	}
}

func TestCardBalancesLongDescriptionAndTitle(t *testing.T) {
	card := NewCard(theme.IconSelfDestruct, "Self Destruct", strings.Repeat("Long description ", 10))
	for _, width := range []int{1, 8, 40, 60, 120} {
		content := card.View(width)
		assertFits(t, content, width, 3)
		if width >= 40 && !strings.Contains(ansi.Strip(content), "Self Destruct") {
			t.Fatal("description squeezed out page title")
		}
	}
}

type sectionItem string

func (s sectionItem) FilterValue() string { return string(s) }
func (s sectionItem) Title() string       { return string(s) }
func (s sectionItem) Icon() string        { return "" }
func (s sectionItem) IsAction() bool      { return false }
func (s sectionItem) Selectable() bool    { return false }

func TestSidebarSectionsDoNotTakeFocus(t *testing.T) {
	sidebar := NewSidebarList([]SidebarItem{sectionItem("First header"), SidebarItemString("First"), sectionItem("Second header"), SidebarItemString("Second"), sectionItem("Trailing header")}, 40, SidebarStyles{}, "")
	sidebar.Resize(layout.NewRect(0, 0, 40, 3))
	sidebar.Focus()
	sidebar.Select(1)
	for _, step := range []struct {
		key      tea.KeyType
		expected int
	}{{tea.KeyDown, 3}, {tea.KeyDown, 3}, {tea.KeyUp, 1}, {tea.KeyUp, 1}, {tea.KeyEnd, 3}, {tea.KeyHome, 1}} {
		sidebar.Update(tea.KeyMsg{Type: step.key})
		if sidebar.Index() != step.expected {
			t.Fatalf("%v selected section %d", step.key, sidebar.Index())
		}
	}
}
