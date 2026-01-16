package widget

import (
	"github.com/Josepavese/nido/internal/tui/kit/layout"
	viewlet "github.com/Josepavese/nido/internal/tui/kit/view"
	tea "github.com/charmbracelet/bubbletea"
)

// PageManager is a container viewlet that manages multiple pages (viewlets).
// It switches between them based on string keys and delegates all lifecycle methods
// to the currently active page.
type PageManager struct {
	viewlet.BaseViewlet
	Pages  map[string]viewlet.Viewlet
	Active string
}

// NewPageManager creates a new PageManager.
func NewPageManager() *PageManager {
	return &PageManager{
		Pages: make(map[string]viewlet.Viewlet),
	}
}

// AddPage adds a page to the manager.
func (pm *PageManager) AddPage(key string, v viewlet.Viewlet) {
	pm.Pages[key] = v
}

// SwitchTo switches the active page.
func (pm *PageManager) SwitchTo(key string) tea.Cmd {
	pm.Active = key
	if p, ok := pm.Pages[key]; ok {
		// Ensure initial size is correct if pm already has size
		p.Resize(layout.NewRect(0, 0, pm.Width(), pm.Height()))
		return p.Init()
	}
	return nil
}

// Init initializes the active page.
func (pm *PageManager) Init() tea.Cmd {
	if p, ok := pm.Pages[pm.Active]; ok {
		return p.Init()
	}
	return nil
}

// Update delegates to all pages if necessary (for background tasks)
// or just the active one for UI interaction.
// Nido standard: Generic messages (Services, Ticks) go to everyone.
// UI messages go to Active.
func (pm *PageManager) Update(msg tea.Msg) (viewlet.Viewlet, tea.Cmd) {
	var cmds []tea.Cmd

	// Optimization: If it's a specific page switch message (could be custom), handle it here.

	// Propagate to all pages for background updates (Services, Ticks, etc.)
	// BUT, UI specific messages (Key, Mouse) ONLY to active.
	isUIMsg := false
	switch msg.(type) {
	case tea.KeyMsg, tea.MouseMsg:
		isUIMsg = true
	}

	for k, p := range pm.Pages {
		if isUIMsg && k != pm.Active {
			continue
		}
		newV, cmd := p.Update(msg)
		pm.Pages[k] = newV
		cmds = append(cmds, cmd)
	}

	return pm, tea.Batch(cmds...)
}

// View renders the active page.
func (pm *PageManager) View() string {
	if p, ok := pm.Pages[pm.Active]; ok {
		return p.View()
	}
	return ""
}

// Resize resizes all pages to match the manager's dimensions.
func (pm *PageManager) Resize(r layout.Rect) {
	pm.BaseViewlet.Resize(r)
	for _, p := range pm.Pages {
		p.Resize(r)
	}
}

// HandleMouse delegates to the active page.
func (pm *PageManager) HandleMouse(x, y int, msg tea.MouseMsg) (viewlet.Viewlet, tea.Cmd, bool) {
	if p, ok := pm.Pages[pm.Active]; ok {
		return p.HandleMouse(x, y, msg)
	}
	return pm, nil, false
}

// Shortcuts delegates to the active page.
func (pm *PageManager) Shortcuts() []viewlet.Shortcut {
	if p, ok := pm.Pages[pm.Active]; ok {
		return p.Shortcuts()
	}
	return nil
}

func (pm *PageManager) Focused() bool {
	if p, ok := pm.Pages[pm.Active]; ok {
		return p.Focused()
	}
	return false
}

func (pm *PageManager) Focus() tea.Cmd {
	if p, ok := pm.Pages[pm.Active]; ok {
		return p.Focus()
	}
	return nil
}

func (pm *PageManager) Blur() {
	if p, ok := pm.Pages[pm.Active]; ok {
		p.Blur()
	}
}

// ActivePage returns the currently active viewlet.
func (pm *PageManager) ActivePage() viewlet.Viewlet {
	return pm.Pages[pm.Active]
}

func (pm *PageManager) IsModalActive() bool {
	if p, ok := pm.Pages[pm.Active]; ok {
		return p.IsModalActive()
	}
	return false
}

func (pm *PageManager) HasActiveInput() bool {
	res := false
	if p, ok := pm.Pages[pm.Active]; ok {
		res = p.HasActiveInput()
	}
	return res
}

func (pm *PageManager) Focusable() bool {
	if p, ok := pm.Pages[pm.Active]; ok {
		return p.Focusable()
	}
	return false
}
