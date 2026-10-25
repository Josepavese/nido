package shell

import (
	"fmt"
	"strings"
	"time"

	"github.com/Josepavese/nido/v4/internal/tui/kit/layout"
	"github.com/Josepavese/nido/v4/internal/tui/kit/theme"
	viewlet "github.com/Josepavese/nido/v4/internal/tui/kit/view"
	"github.com/Josepavese/nido/v4/internal/tui/kit/widget"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MouseZone represents a zone in the shell grid.
type MouseZone int

const (
	ZoneNone MouseZone = iota
	ZoneHeader
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
	Header      lipgloss.Style
	StatusBar   widget.StatusBarStyles
	BorderColor lipgloss.TerminalColor
}

// Shell is the generic container for the TUI.
type Shell struct {
	Width  int
	Height int

	// Chrome Content (set these directly)
	HeaderContent string
	FooterContent string

	// Viewlet Management
	routes        []Route
	routesByKey   map[string]viewlet.Viewlet
	activeKey     string
	activeViewlet viewlet.Viewlet
	activeRoute   Route

	// Layout State
	grid        layout.Grid
	ActionStack *widget.ActionStack // New Widget

	// Status
	Loading   bool
	Operation string
	Progress  float64

	// Styles
	Styles ShellStyles
}

// UpdateStyles refreshes the shell's styles from the current theme.
func (s *Shell) UpdateStyles() {
	t := theme.Current()
	s.Styles = ShellStyles{
		Header: lipgloss.NewStyle().Background(t.Palette.Surface).Border(lipgloss.NormalBorder(), false, false, true, false).BorderForeground(t.Palette.SurfaceSubtle),
		StatusBar: widget.StatusBarStyles{
			Key:    t.Styles.Label.Bold(true),
			Label:  t.Styles.Label,
			Status: t.Styles.Label,
		},
		BorderColor: t.Palette.SurfaceHighlight,
	}
}

// NewShell creates a new shell.
func NewShell() Shell {
	return Shell{
		routesByKey: make(map[string]viewlet.Viewlet),
		ActionStack: widget.NewActionStack(),
	}
}

// AddRoute registers a route with the shell.
// AddRoute registers a route with the shell. If the key already exists, it updates the existing route.
func (s *Shell) AddRoute(r Route) {
	// Check if already exists in slice to prevent duplicate tabs
	for i, existing := range s.routes {
		if existing.Key == r.Key {
			s.routes[i] = r
			s.routesByKey[r.Key] = r.Viewlet
			return
		}
	}

	s.routes = append(s.routes, r)
	s.routesByKey[r.Key] = r.Viewlet
	// Auto-select first one if none selected
	if s.activeKey == "" {
		s.SwitchTo(r.Key)
	}
}

// StartAction starts a unified global action and returns its ID and init Cmd
func (s *Shell) StartAction(msg string) (string, tea.Cmd) {
	id := fmt.Sprintf("act-%d", time.Now().UnixNano())
	cmd := s.ActionStack.Add(id, msg)
	s.RecalculateLayout() // Force resize to show stack
	return id, cmd
}

// FinishAction removes an action from the stack
func (s *Shell) FinishAction(id string) {
	s.ActionStack.Remove(id)
	s.RecalculateLayout() // Force resize to hide stack if empty
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

// Init initializes the shell and all registered viewlets.
func (s *Shell) Init() tea.Cmd {
	var cmds []tea.Cmd
	// Initialize ActionStack (starts spinners)
	cmds = append(cmds, s.ActionStack.Init())

	for _, r := range s.routes {
		if r.Viewlet != nil {
			cmds = append(cmds, r.Viewlet.Init())
		}
	}
	return tea.Batch(cmds...)
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
	var cmds []tea.Cmd

	// 1. Forward updates to ActionStack (Spinner ticks, etc.)
	cmds = append(cmds, s.ActionStack.Update(msg))

	switch msg := msg.(type) {
	case tea.KeyMsg:
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
		// Convert old StatusMsg to ActionStack calls
		s.Loading = msg.Loading
		s.Operation = msg.Operation
		s.Progress = msg.Progress

		if s.Loading {
			// Start/Update Action
			cmd := s.ActionStack.Add("global", s.Operation)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			s.ActionStack.UpdateProgress("global", s.Progress)
			s.RecalculateLayout()
		} else {
			// Stop Action
			s.ActionStack.Remove("global")
			s.RecalculateLayout()
		}

	case viewlet.SwitchTabMsg:
		// Support programmatic switching (e.g., from a hotkey inside a viewlet)
		// For now, we assume the Msg contains the target Key or Index.
		// Since message definition in kit/view might rely on index, we map it?
		// Current definition: TabIndex int.
		if msg.TabIndex >= 0 && msg.TabIndex < len(s.routes) {
			s.SwitchTo(s.routes[msg.TabIndex].Key)
		}
	}
	return tea.Batch(cmds...)
}

// HandleMouse processes mouse events using grid-aware localization.
// It returns the hit zone, localized coordinates within that zone, a command,
// and a boolean indicating if the active viewlet handled the event.

// UpdateAction synchronizes an action's state with the stack and returns an init Cmd if new.
func (s *Shell) UpdateAction(id, message string, progress float64) tea.Cmd {
	cmd := s.ActionStack.Add(id, message)
	s.ActionStack.UpdateProgress(id, progress)
	s.RecalculateLayout()
	return cmd
}

// SetStatus manually updates the status bar state and syncs with ActionStack.
func (s *Shell) SetStatus(loading bool, op string, progress float64) {
	s.Loading = loading
	s.Operation = op
	s.Progress = progress

	if loading {
		s.ActionStack.Add("global", op)
		s.ActionStack.UpdateProgress("global", progress)
	} else {
		s.ActionStack.Remove("global")
	}
	s.RecalculateLayout()
}

// ActiveViewlet returns the current viewlet.
func (s *Shell) ActiveViewlet() viewlet.Viewlet {
	return s.activeViewlet
}

// RecalculateLayout triggers a grid recalculation based on current state (e.g. stack height)
func (s *Shell) RecalculateLayout() {
	if s.Width > 0 && s.Height > 0 {
		s.Resize(s.Width, s.Height)
	}
}

// Resize updates dimensions and propagates to active viewlet.
func (s *Shell) Resize(w, h int) {
	s.Width = w
	s.Height = h

	// Calculate strict layout with dynamic stack height
	stackHeight := 0
	if s.ActionStack != nil {
		stackHeight = s.ActionStack.Height()
	}

	s.grid = layout.CalculateGrid(w, h, stackHeight)

	// Propagate to active viewlet
	if s.activeViewlet != nil {
		s.activeViewlet.Resize(s.grid.Body)
	}

	// Note: We could resize ALL viewlets here if we wanted them background-ready,
	// but resizing the active one is the priority.
}

// View aligns all content using the Grid Engine.
func (s *Shell) View() string {
	s.UpdateStyles()

	if s.Width <= 0 || s.Height <= 0 {
		return ""
	}
	if !layout.IsViable(s.Width, s.Height) {
		return widget.RenderResizeNotice(s.Width, s.Height)
	}

	// 1. Render Chrome
	// We use the cached grid from Resize()
	header := place(s.grid.Header, s.renderHeader())

	// Action stack owns its rendering; the shell only assigns a rectangle.
	var stack string
	if s.grid.ActionStack.Height > 0 {
		stack = place(s.grid.ActionStack, s.ActionStack.Viewport(s.grid.ActionStack.Width, s.grid.ActionStack.Height))
	}

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

	// Empty zones must not contribute phantom newline rows.
	var zones []string
	for _, zone := range []string{header, stack, body, footer} {
		if zone != "" {
			zones = append(zones, zone)
		}
	}
	return layout.Fit(s.Width, s.Height, lipgloss.JoinVertical(lipgloss.Left, zones...))
}

func (s *Shell) tabs() widget.TabBar {
	var titles []string
	activeIndex := 0
	for i, r := range s.routes {
		titles = append(titles, r.Title)
		if r.Key == s.activeKey {
			activeIndex = i
		}
	}
	t := theme.Current()
	return widget.RenderTabs(widget.TabsRenderOptions{
		Routes: titles, ActiveIndex: activeIndex, Width: s.Width, Height: s.grid.Header.Height,
		HighlightColor: t.Palette.Accent, InactiveColor: t.Palette.TextDim, BorderColor: t.Palette.SurfaceHighlight,
	})
}
func (s *Shell) renderHeader() string { return s.tabs().Content }

func (s *Shell) renderFooter() string {
	if s.FooterContent != "" {
		return s.FooterContent
	}

	// Build generic status bar based on shell state
	sb := widget.NewStatusBar(s.Width, s.Styles.StatusBar)

	if s.Loading {
		sb.SetStatus(fmt.Sprintf("EXECUTING %s...", strings.ToUpper(s.Operation)))
	} else {
		sb.SetStatus("")
	}

	var sbItems []widget.StatusBarItem
	// 2. Viewlet Shortcuts
	if s.activeViewlet != nil {
		for _, sc := range s.activeViewlet.Shortcuts() {
			sbItems = append(sbItems, widget.StatusBarItem{
				Key:   prettifyKey(sc.Key),
				Label: sc.Label,
			})
		}
	}

	sbItems = append(sbItems, widget.StatusBarItem{Key: s.navigationKey(), Label: "flow"})
	// 3. System Exit (Always last)
	sbItems = append(sbItems, widget.StatusBarItem{
		Key:   "^C",
		Label: "quit",
	})

	sb.SetItems(sbItems)

	return sb.View()
}

func prettifyKey(k string) string {
	switch strings.ToLower(k) {
	case "enter":
		return "↵"
	case "tab":
		return "⭾"
	case "shift+tab":
		return "⇧⭾"
	case "up", "arrow up", "↑/↓":
		return "↑/↓" // Already pretty if passed in this way, but just in case
	case "left", "arrow left", "←/→":
		return "←/→"
	case "right", "arrow right":
		return "→"
	case "delete", "backspace":
		return "⌫"
	case "esc":
		return "⎋"
	case "ctrl+c", "ctrl+q":
		return "^C"
	case "enter/space":
		return "↵/␣"
	case "space":
		return "␣"
	default:
		return k
	}
}

// HandleMouse processes mouse events using grid-aware localization.
// It returns the hit zone, localized coordinates within that zone, a command,
// and a boolean indicating if the active viewlet handled the event.
func (s *Shell) HandleMouse(msg tea.MouseMsg) (MouseZone, int, int, tea.Cmd, bool) {
	if s.grid.Header.Contains(msg.X, msg.Y) && (s.activeViewlet == nil || !s.activeViewlet.IsModalActive()) {
		lx := msg.X - s.grid.Header.X
		ly := msg.Y - s.grid.Header.Y

		if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
			for i, hit := range s.tabs().Hits {
				if hit.Contains(lx, ly) {
					s.SwitchTo(s.routes[i].Key)
					return ZoneHeader, lx, ly, nil, true
				}
			}
		}
		return ZoneHeader, lx, ly, nil, false
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

	return layout.Fit(r.Width, r.Height, content)
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

// RouteCount returns the number of registered routes (Debug).
func (s *Shell) RouteCount() int {
	return len(s.routes)
}

// GridHeaderHeight returns the calculated grid header height (Debug).
func (s *Shell) GridHeaderHeight() int {
	return s.grid.Header.Height
}

func (s *Shell) navigationKey() string {
	if s.activeViewlet != nil && s.activeViewlet.HasActiveTextInput() {
		return "ctrl+←/→"
	}
	return "←/→"
}
