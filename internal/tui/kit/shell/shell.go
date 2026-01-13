package shell

import (
	"fmt"
	"strings"

	"github.com/Josepavese/nido/internal/tui/kit/layout"
	viewlet "github.com/Josepavese/nido/internal/tui/kit/view"
	"github.com/Josepavese/nido/internal/tui/kit/widget"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MouseZone represents a zone in the shell grid.
type MouseZone int

const (
	ZoneNone MouseZone = iota
	ZoneHeader
	ZoneSubHeader
	ZoneBody
	ZoneFooter
)

// Route defines a viewlet and its associated metadata for the shell chrome.
type Route struct {
	Key     string
	Title   string // e.g., "GENETIC CONFIG"
	Hint    string // e.g., "Modify Nido's core DNA."
	Viewlet viewlet.Viewlet
}

// ShellStyles defines the appearance of the shell widget.
type ShellStyles struct {
	Header           lipgloss.Style
	SubHeaderContext lipgloss.Style
	SubHeaderNav     lipgloss.Style
	StatusBar        widget.StatusBarStyles
}

// Shell is the generic container for the TUI.
type Shell struct {
	Width  int
	Height int

	// Chrome Content (set these directly)
	HeaderContent    string
	SubHeaderContent string
	FooterContent    string

	// Viewlet Management
	routes        []Route
	routesByKey   map[string]viewlet.Viewlet
	activeKey     string
	activeViewlet viewlet.Viewlet
	activeRoute   Route

	// Layout State
	grid layout.Grid

	// Status & Logging
	Loading   bool
	Operation string
	Progress  float64
	Logs      []string

	// Styles
	Styles ShellStyles
}

// NewShell creates a new shell.
func NewShell() Shell {
	return Shell{
		routesByKey: make(map[string]viewlet.Viewlet),
	}
}

// AddRoute registers a route with the shell.
func (s *Shell) AddRoute(r Route) {
	s.routes = append(s.routes, r)
	s.routesByKey[r.Key] = r.Viewlet
	// Auto-select first one if none selected
	if s.activeKey == "" {
		s.SwitchTo(r.Key)
	}
}

// SwitchTo activates a viewlet by key.
func (s *Shell) SwitchTo(key string) {
	if v, ok := s.routesByKey[key]; ok {
		s.activeKey = key
		s.activeViewlet = v
		// Find and sync active route
		for _, r := range s.routes {
			if r.Key == key {
				s.activeRoute = r
				break
			}
		}
		// Ensure it's sized correctly immediately (if shell has size)
		if s.grid.Body.Width > 0 {
			v.Resize(s.grid.Body)
		}
	}
}

// Init initializes the shell.
func (s *Shell) Init() tea.Cmd {
	return nil
}

// NextTab cycles to the next tab.
func (s *Shell) NextTab() {
	if len(s.routes) <= 1 {
		return
	}
	for i, r := range s.routes {
		if r.Key == s.activeKey {
			nextIdx := (i + 1) % len(s.routes)
			s.SwitchTo(s.routes[nextIdx].Key)
			return
		}
	}
	s.SwitchTo(s.routes[0].Key)
}

// PrevTab cycles to the previous tab.
func (s *Shell) PrevTab() {
	if len(s.routes) <= 1 {
		return
	}
	for i, r := range s.routes {
		if r.Key == s.activeKey {
			prevIdx := i - 1
			if prevIdx < 0 {
				prevIdx = len(s.routes) - 1
			}
			s.SwitchTo(s.routes[prevIdx].Key)
			return
		}
	}
	s.SwitchTo(s.routes[0].Key)
}

// Update handles shell-level messages (Status, Logs, SwitchTab).
func (s *Shell) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global Navigation
		// Global Navigation
		// Handled by App (model.go) now to prevent viewlet conflict.
		// Keeping switch here mostly for future shell-specific shortcuts if needed.
		switch msg.String() {
		// case "ctrl+right", "tab":
		// 	s.NextTab()
		// 	return nil
		// case "ctrl+left", "shift+tab":
		// 	s.PrevTab()
		// 	return nil
		}

	case viewlet.StatusMsg:
		s.Loading = msg.Loading
		s.Operation = msg.Operation
		s.Progress = msg.Progress
	case viewlet.LogMsg:
		s.Logs = append(s.Logs, msg.Text)
	case viewlet.SwitchTabMsg:
		// Support programmatic switching (e.g., from a hotkey inside a viewlet)
		// For now, we assume the Msg contains the target Key or Index.
		// Since message definition in kit/view might rely on index, we map it?
		// Current definition: TabIndex int.
		if msg.TabIndex >= 0 && msg.TabIndex < len(s.routes) {
			s.SwitchTo(s.routes[msg.TabIndex].Key)
		}
	}
	return nil
}

// HandleMouse processes mouse events using grid-aware localization.
// It returns the hit zone, localized coordinates within that zone, a command,
// and a boolean indicating if the active viewlet handled the event.

// SetStatus manually updates the status bar state.
func (s *Shell) SetStatus(loading bool, op string, progress float64) {
	s.Loading = loading
	s.Operation = op
	s.Progress = progress
}

// ActiveViewlet returns the current viewlet.
func (s *Shell) ActiveViewlet() viewlet.Viewlet {
	return s.activeViewlet
}

// Resize updates dimensions and propagates to active viewlet.
func (s *Shell) Resize(w, h int) {
	s.Width = w
	s.Height = h

	// Calculate strict layout
	s.grid = layout.CalculateGrid(w, h)

	// Propagate to active viewlet
	if s.activeViewlet != nil {
		s.activeViewlet.Resize(s.grid.Body)
	}

	// Note: We could resize ALL viewlets here if we wanted them background-ready,
	// but resizing the active one is the priority.
}

// View aligns all content using the Grid Engine.
func (s *Shell) View() string {
	if s.Width == 0 || s.Height == 0 {
		return "Initializing Shell..."
	}

	// 1. Render Chrome
	// We use the cached grid from Resize()
	header := place(s.grid.Header, s.renderHeader())
	subHeader := place(s.grid.SubHeader, s.renderSubHeader())
	footer := place(s.grid.Footer, s.renderFooter())

	// 2. Render Body
	// If we have an active viewlet, use it. Otherwise use static string.
	var bodyContent string
	if s.activeViewlet != nil {
		bodyContent = s.activeViewlet.View()
	} else {
		bodyContent = "No Active Viewlet"
	}
	body := place(s.grid.Body, bodyContent)

	// 3. Stack them
	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		subHeader,
		body,
		footer,
	)
}

func (s *Shell) renderHeader() string {
	// Tabs
	var tabs []string
	for _, r := range s.routes {
		style := lipgloss.NewStyle().Padding(0, 1).Foreground(s.Styles.StatusBar.Label.GetForeground())
		if r.Key == s.activeKey {
			style = style.Bold(true).
				Foreground(s.Styles.SubHeaderContext.GetForeground()).
				Reverse(true)
		}
		tabs = append(tabs, style.Render(r.Key))
	}
	startTabs := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	// Spacer (Push tabs to right if desired, or keep left)
	// User requested "Remove Branding", implying simpler look.
	// Let's keep tabs on the LEFT for standard navigation feel, or RIGHT?
	// Previous code put them on right with spacer.
	// "return lipgloss.JoinHorizontal(lipgloss.Top, branding, spacer, startTabs)"
	// If we remove branding and spacer, they are on LEFT.
	// Let's stick to LEFT alignment for simpler TUI.

	// If we want them on RIGHT:
	// availWidth := s.grid.Header.Width - lipgloss.Width(startTabs)
	// if availWidth < 0 { availWidth = 0 }
	// spacer := strings.Repeat(" ", availWidth)
	// return lipgloss.JoinHorizontal(lipgloss.Top, spacer, startTabs)

	// Returning Left Aligned Tabs:
	return startTabs
}

func (s *Shell) renderSubHeader() string {
	if s.activeRoute.Key == "" {
		return ""
	}

	style := lipgloss.NewStyle().PaddingLeft(1).PaddingRight(1)
	context := s.Styles.SubHeaderContext.Render(s.activeRoute.Title)
	nav := s.Styles.SubHeaderNav.Render(s.activeRoute.Hint)

	return style.Render(fmt.Sprintf("%s  %s", context, nav))
}

func (s *Shell) renderFooter() string {
	if s.FooterContent != "" {
		return s.FooterContent
	}

	// Build generic status bar based on shell state
	sb := widget.NewStatusBar(s.Width, s.Styles.StatusBar)

	if s.Loading {
		sb.SetStatus(fmt.Sprintf("EXECUTING %s...", strings.ToUpper(s.Operation)))
	} else {
		sb.SetStatus("NOMINAL")
	}

	// Add shortcuts from active viewlet
	if s.activeViewlet != nil {
		var sbItems []widget.StatusBarItem
		for _, sc := range s.activeViewlet.Shortcuts() {
			sbItems = append(sbItems, widget.StatusBarItem{
				Key:   sc.Key,
				Label: sc.Label,
			})
		}
		sb.SetItems(sbItems)
	}

	return sb.View()
}

// HandleMouse processes mouse events using grid-aware localization.
// It returns the hit zone, localized coordinates within that zone, a command,
// and a boolean indicating if the active viewlet handled the event.
func (s *Shell) HandleMouse(msg tea.MouseMsg) (MouseZone, int, int, tea.Cmd, bool) {
	if s.grid.Header.Contains(msg.X, msg.Y) {
		lx := msg.X - s.grid.Header.X
		ly := msg.Y - s.grid.Header.Y

		// Tab Click Detection (Left Aligned)
		if msg.Type == tea.MouseLeft {
			cursorX := 0
			for _, r := range s.routes {
				// Match style logic from renderHeader
				style := lipgloss.NewStyle().Padding(0, 1)
				if r.Key == s.activeKey {
					style = style.Bold(true)
				}
				w := lipgloss.Width(style.Render(r.Key))

				if lx >= cursorX && lx < cursorX+w {
					s.SwitchTo(r.Key)
					return ZoneHeader, lx, ly, nil, true
				}
				cursorX += w
			}
		}

		return ZoneHeader, lx, ly, nil, false
	}

	if s.grid.SubHeader.Contains(msg.X, msg.Y) {
		lx := msg.X - s.grid.SubHeader.X
		ly := msg.Y - s.grid.SubHeader.Y
		return ZoneSubHeader, lx, ly, nil, false
	}

	if s.grid.Body.Contains(msg.X, msg.Y) {
		lx := msg.X - s.grid.Body.X
		ly := msg.Y - s.grid.Body.Y
		if s.activeViewlet != nil {
			newV, cmd, handled := s.activeViewlet.HandleMouse(lx, ly, msg)
			if handled {
				s.activeViewlet = newV
				return ZoneBody, lx, ly, cmd, true
			}
		}
		return ZoneBody, lx, ly, nil, false
	}

	if s.grid.Footer.Contains(msg.X, msg.Y) {
		lx := msg.X - s.grid.Footer.X
		ly := msg.Y - s.grid.Footer.Y
		return ZoneFooter, lx, ly, nil, false
	}

	return ZoneNone, msg.X, msg.Y, nil, false
}

// place forces content into a Rect.
func place(r layout.Rect, content string) string {
	if r.Height == 0 || r.Width == 0 {
		return ""
	}

	// Use Lipgloss to size it exactly.
	style := lipgloss.NewStyle().
		Width(r.Width).
		Height(r.Height).
		MaxHeight(r.Height).
		MaxWidth(r.Width)

	return style.Render(content)
}

// DebugPlace helps visualize layout bounds
func DebugPlace(r layout.Rect, content string) string {
	if r.Height == 0 || r.Width == 0 {
		return ""
	}
	style := lipgloss.NewStyle().
		Width(r.Width - 2).
		Height(r.Height - 2).
		Border(lipgloss.NormalBorder())

	return style.Render(content)
}
