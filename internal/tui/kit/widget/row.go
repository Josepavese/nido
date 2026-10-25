package widget

import (
	"github.com/Josepavese/nido/v4/internal/tui/kit/layout"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Row represents a horizontal group of Elements.
type Row struct {
	Elements   []Element
	Weights    []int
	FocusIndex int
	width      int
}

// NewRow creates a new Row with equal width distribution.
func NewRow(elements ...Element) *Row {
	return &Row{Elements: elements, FocusIndex: -1}
}

// NewRowWithWeights creates a new Row with specified column weights.
func NewRowWithWeights(elements []Element, weights []int) *Row {
	return &Row{Elements: elements, Weights: weights, FocusIndex: -1}
}

// Interface compliance for Form Element

func (r *Row) Focus() tea.Cmd {
	if r.FocusIndex < 0 {
		r.FocusIndex = 0
	}
	// Try to focus current index
	if r.FocusIndex < len(r.Elements) {
		if r.Elements[r.FocusIndex].Focusable() {
			return r.Elements[r.FocusIndex].Focus()
		}
	}
	// Fallback scanning
	for i, el := range r.Elements {
		if el.Focusable() {
			r.FocusIndex = i
			return el.Focus()
		}
	}
	return nil
}

func (r *Row) Blur() {
	for _, el := range r.Elements {
		el.Blur()
	}
	r.FocusIndex = -1
}

func (r *Row) Focused() bool {
	for _, el := range r.Elements {
		if el.Focused() {
			return true
		}
	}
	return false
}

func (r *Row) Focusable() bool {
	for _, el := range r.Elements {
		if el.Focusable() {
			return true
		}
	}
	return false
}

func (r *Row) HasActiveTextInput() bool {
	for _, el := range r.Elements {
		// Check for direct Input
		if input, ok := el.(*Input); ok {
			if input.Focused() {
				return true
			}
		}
		// Recursive check for containers
		if container, ok := el.(interface{ HasActiveTextInput() bool }); ok {
			if container.HasActiveTextInput() {
				return true
			}
		}
	}
	return false
}

// CollectInputs returns all Input fields within this Row (recursive).
func (r *Row) CollectInputs() []*Input {
	var inputs []*Input
	for _, el := range r.Elements {
		if input, ok := el.(*Input); ok {
			inputs = append(inputs, input)
		}
		if collector, ok := el.(interface{ CollectInputs() []*Input }); ok {
			inputs = append(inputs, collector.CollectInputs()...)
		}
	}
	return inputs
}

// CollectButtons returns all Button fields within this Row (recursive).
func (r *Row) CollectButtons() []*Button {
	var btns []*Button
	for _, el := range r.Elements {
		if btn, ok := el.(*Button); ok {
			btns = append(btns, btn)
		}
		if collector, ok := el.(interface{ CollectButtons() []*Button }); ok {
			btns = append(btns, collector.CollectButtons()...)
		}
	}
	return btns
}

// Navigator Implementation

func (r *Row) Next() bool {
	start := r.FocusIndex
	if start < 0 {
		start = -1
	}

	// Try to find next focusable element
	for i := start + 1; i < len(r.Elements); i++ {
		if r.Elements[i].Focusable() {
			if r.FocusIndex >= 0 && r.FocusIndex < len(r.Elements) {
				r.Elements[r.FocusIndex].Blur()
			}
			r.FocusIndex = i
			return true
		}
	}
	// No more elements
	return false
}

func (r *Row) Prev() bool {
	start := r.FocusIndex
	if start > len(r.Elements) {
		start = len(r.Elements)
	}

	// Try to find prev focusable element
	for i := start - 1; i >= 0; i-- {
		if r.Elements[i].Focusable() {
			if r.FocusIndex >= 0 && r.FocusIndex < len(r.Elements) {
				r.Elements[r.FocusIndex].Blur()
			}
			r.FocusIndex = i
			return true
		}
	}
	// No earlier elements
	return false
}

func (r *Row) Update(msg tea.Msg) (Element, tea.Cmd) {
	var cmd tea.Cmd

	// Update only the focused element (standard for forms) or all?
	// Row usually updates all because some might show status?
	// But mostly we only care about the active one receiving input.
	// However, if we broadcast to all, typing in one input might affect others?
	// Safest matches Form logic: Update active if index valid.
	if r.FocusIndex >= 0 && r.FocusIndex < len(r.Elements) {
		idx := r.FocusIndex
		el, c := r.Elements[idx].Update(msg)
		r.Elements[idx] = el
		cmd = c
	} else {
		// If no focus, update none? Or update all?
		// For passive display elements inside a row, they might need updates (spinner?)
		// But Form only updates focused element usually.
		// Let's stick to active-only for input safety.
	}

	return r, cmd
}

// rectangles is shared by drawing, focus following and mouse hit detection.
func (r *Row) rectangles(width int) []layout.Rect {
	count := len(r.Elements)
	if count == 0 {
		return nil
	}
	available := max(0, width-(count-1)*layout.RowGap)
	weights := make([]int, count)
	total := 0
	for i := range weights {
		weights[i] = 1
		if len(r.Weights) == count && r.Weights[i] > 0 {
			weights[i] = r.Weights[i]
		}
		total += weights[i]
	}
	widths := make([]int, count)
	remaining := available
	stacked := false
	for i := range widths {
		widths[i] = available * weights[i] / total
		if i == count-1 {
			widths[i] = remaining
		}
		remaining -= widths[i]
		stacked = stacked || widths[i] < elementMinWidth(r.Elements[i])
	}
	rects := make([]layout.Rect, count)
	x, y := 0, 0
	for i, el := range r.Elements {
		w := widths[i]
		if stacked {
			w = width
		}
		el.SetWidth(w)
		h := lipgloss.Height(el.View(w))
		rects[i] = layout.NewRect(x, y, w, h)
		if stacked {
			y += h
		} else {
			x += w + layout.RowGap
		}
	}
	return rects
}
func (r *Row) SetWidth(w int) { r.width = max(0, w); r.rectangles(r.width) }
func (r *Row) View(width int) string {
	if width > 0 {
		r.width = width
	}
	rects := r.rectangles(r.width)
	var views []string
	for i, el := range r.Elements {
		views = append(views, el.View(rects[i].Width))
	}
	if len(rects) > 1 && rects[1].Y > 0 {
		return layout.VStack(0, views...)
	}
	return layout.HStack(layout.RowGap, views...)
}
func (r *Row) HandleMouse(x, y int, msg tea.MouseMsg) (tea.Cmd, bool) {
	for i, rect := range r.rectangles(r.width) {
		el := r.Elements[i]
		if !rect.Contains(x, y) || !el.Focusable() {
			continue
		}
		if r.FocusIndex >= 0 && r.FocusIndex < len(r.Elements) {
			if r.FocusIndex >= 0 && r.FocusIndex < len(r.Elements) {
				r.Elements[r.FocusIndex].Blur()
			}
		}
		r.FocusIndex = i
		cmd := el.Focus()
		if clickable, ok := el.(interface {
			HandleMouse(int, int, tea.MouseMsg) (tea.Cmd, bool)
		}); ok {
			action, handled := clickable.HandleMouse(x-rect.X, y-rect.Y, msg)
			return tea.Batch(cmd, action), handled
		}
		return cmd, true
	}
	return nil, false
}
func elementAnchor(el Element, width int) layout.Rect {
	if !el.Focused() {
		return layout.Rect{}
	}
	if row, ok := el.(*Row); ok {
		for i, rect := range row.rectangles(width) {
			if row.Elements[i].Focused() {
				child := elementAnchor(row.Elements[i], rect.Width)
				child.X += rect.X
				child.Y += rect.Y
				return child
			}
		}
	}
	return layout.NewRect(0, 0, width, lipgloss.Height(el.View(width)))
}

// Buttons need their label and frame, not the editing space of a text field.
func elementMinWidth(el Element) int {
	if button, ok := el.(*Button); ok && (button.Centered || button.Role == RoleSubmit) {
		return min(layout.FieldMinWidth, lipgloss.Width(button.Text)+layout.FieldFrameWidth)
	}
	return layout.FieldMinWidth
}
