package hatchery

import (
	"fmt"

	"github.com/Josepavese/nido/internal/tui/app/ops"
	"github.com/Josepavese/nido/internal/tui/kit/layout"
	"github.com/Josepavese/nido/internal/tui/kit/theme"
	fv "github.com/Josepavese/nido/internal/tui/kit/view"
	widget "github.com/Josepavese/nido/internal/tui/kit/widget"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	dimStyle     = lipgloss.NewStyle().Foreground(theme.Current().Palette.TextDim)
	successStyle = lipgloss.NewStyle().Foreground(theme.Current().Palette.Success)
)

// HatcheryMode defines the active incubator mode.
type HatcheryMode int

const (
	HatcherySpawn HatcheryMode = iota
	HatcheryTemplate
)

// HatchTypeItem represents a sidebar menu item in the incubator.
type HatchTypeItem struct {
	ID       int
	TitleStr string
	DescStr  string
}

func (i HatchTypeItem) Title() string       { return i.TitleStr }
func (i HatchTypeItem) Description() string { return i.DescStr }
func (i HatchTypeItem) FilterValue() string { return i.TitleStr }
func (i HatchTypeItem) String() string      { return i.TitleStr }
func (i HatchTypeItem) Icon() string        { return "" }
func (i HatchTypeItem) IsAction() bool      { return false }

// Hatchery implements the Incubator viewlet.
type Hatchery struct {
	fv.BaseViewlet

	// State
	Mode        HatcheryMode
	inputs      []textinput.Model
	focusIndex  int
	isSelecting bool

	// Data
	SpawnSource    string
	TemplateSource string
	sourceList     list.Model
	sources        []string // Cache for internal logic

	// Components
	Sidebar      *widget.SidebarList
	MasterDetail *widget.MasterDetail
	Pages        *widget.PageManager

	// Services
	guiEnabled bool
}

// NewHatchery returns a new Hatchery viewlet.
func NewHatchery() *Hatchery {
	h := &Hatchery{
		Mode: HatcherySpawn,
	}

	// Inputs (0: Name)
	h.inputs = make([]textinput.Model, 1)
	ti := textinput.New()
	ti.Placeholder = "bird-name"
	ti.CharLimit = 32
	h.inputs[0] = ti

	// Source List (Modal)
	h.sourceList = list.New(nil, list.NewDefaultDelegate(), 0, 0)
	h.sourceList.Title = "Select Source"

	// Sidebar Items
	items := []widget.SidebarItem{
		HatchTypeItem{TitleStr: "SPAWN VM", DescStr: "Create a new bird"},
		HatchTypeItem{TitleStr: "TEMPLATES", DescStr: "Manage genetic code"},
	}

	// Sidebar
	t := theme.Current()
	styles := widget.SidebarStyles{
		Normal:   t.Styles.SidebarItem,
		Selected: t.Styles.SidebarItemSelected,
		Dim:      lipgloss.NewStyle().Foreground(t.Palette.TextDim),
		Action:   t.Styles.SidebarItemSelected.Copy().Background(t.Palette.Accent).Foreground(t.Palette.Background),
	}
	h.Sidebar = widget.NewSidebarList(items, theme.Width.Sidebar, styles, "🐣 ")

	// Pages
	h.Pages = widget.NewPageManager()
	h.Pages.AddPage("SPAWN VM", &HatcheryPageSpawn{Parent: h})
	h.Pages.AddPage("TEMPLATES", &HatcheryPageTemplate{Parent: h})

	// MasterDetail Wiring
	border := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(t.Palette.SurfaceSubtle)

	h.MasterDetail = widget.NewMasterDetail(h.Sidebar, h.Pages, border)

	// Sync initial state
	h.Pages.SwitchTo("SPAWN VM")
	h.Sidebar.Select(0)

	return h
}

// Init initializes the hatchery and its pages.
func (h *Hatchery) Init() tea.Cmd {
	return h.MasterDetail.Init()
}

// Update handles messages.
func (h *Hatchery) Update(msg tea.Msg) (fv.Viewlet, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case fv.SelectionMsg:
		if msg.Index == 0 {
			h.Mode = HatcherySpawn
			h.Pages.SwitchTo("SPAWN")
		} else if msg.Index == 1 {
			h.Mode = HatcheryTemplate
			h.Pages.SwitchTo("TEMPLATE")
			cmds = append(cmds, h.Focus())
		}
	case ops.SourcesLoadedMsg:
		if msg.Err != nil {
			return h, func() tea.Msg { return fv.LogMsg{Text: fmt.Sprintf("Failed to load sources: %v", msg.Err)} }
		}
		h.SetSources(msg.Sources)
		return h, func() tea.Msg { return fv.StatusMsg{Loading: false, Operation: "load-sources"} }
	}

	if h.isSelecting {
		newSourceList, cmd := h.sourceList.Update(msg)
		h.sourceList = newSourceList
		cmds = append(cmds, cmd)

		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
			if sel := h.sourceList.SelectedItem(); sel != nil {
				if h.Mode == HatcherySpawn {
					h.SpawnSource = sel.FilterValue()
				} else {
					h.TemplateSource = sel.FilterValue()
				}
				h.isSelecting = false
			}
		} else if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "esc" {
			h.isSelecting = false
		}
		return h, tea.Batch(cmds...)
	}

	// Delegate to MasterDetail
	newMD, cmd := h.MasterDetail.Update(msg)
	h.MasterDetail = newMD.(*widget.MasterDetail)
	cmds = append(cmds, cmd)

	return h, tea.Batch(cmds...)
}

// View translates the page view and applies overlay.
func (h *Hatchery) View() string {
	if h.isSelecting {
		return lipgloss.Place(h.Width(), h.Height(),
			lipgloss.Center, lipgloss.Center,
			theme.Current().Styles.SidebarItemSelected.Render(h.sourceList.View()),
		)
	}
	return h.MasterDetail.View()
}

func (h *Hatchery) Resize(r layout.Rect) {
	h.BaseViewlet.Resize(r)
	h.MasterDetail.Resize(r)
	h.sourceList.SetSize(r.Width-10, r.Height-10)
}

func (h *Hatchery) Shortcuts() []fv.Shortcut {
	return h.MasterDetail.Shortcuts()
}

func (h *Hatchery) SetSources(sources []string) {
	h.sources = sources
	items := make([]list.Item, len(sources))
	for i, s := range sources {
		items[i] = widget.SidebarItemString(s)
	}
	h.sourceList.SetItems(items)
}

func (h *Hatchery) renderToggle(label string, enabled bool) string {
	if enabled {
		return successStyle.Render("[ ON ]")
	}
	return dimStyle.Render("[ OFF ]")
}

func (h *Hatchery) SetMode(mode HatcheryMode) {
	h.Mode = mode
	if mode == HatcherySpawn {
		h.Pages.SwitchTo("SPAWN VM")
		h.Sidebar.Select(0)
	} else {
		h.Pages.SwitchTo("TEMPLATES")
		h.Sidebar.Select(1)
	}
}

func (h *Hatchery) HandleMouse(x, y int, msg tea.MouseMsg) (fv.Viewlet, tea.Cmd, bool) {
	if h.isSelecting {
		// Modal handling remains local
		return h, nil, false
	}
	return h.MasterDetail.HandleMouse(x, y, msg)
}

// GUI Bridge Methods
func (h *Hatchery) IsSelecting() bool { return h.isSelecting }
func (h *Hatchery) IsTyping() bool {
	// If focusing on the name input
	return h.focusIndex == 0
}
func (h *Hatchery) IsSubmitted() bool {
	return h.focusIndex == (len(h.inputs) + 1) // Simple heuristic
}
func (h *Hatchery) GetValues() (string, string, bool) {
	src := h.SpawnSource
	if h.Mode == HatcheryTemplate {
		src = h.TemplateSource
	}
	return h.inputs[0].Value(), src, h.guiEnabled
}
func (h *Hatchery) SetSubmitted(b bool) {
	// No-op for now or handle reset
}
