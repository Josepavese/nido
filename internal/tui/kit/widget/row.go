package widget

import (
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
			r.Elements[r.FocusIndex].Blur()
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
			r.Elements[r.FocusIndex].Blur()
			r.FocusIndex = i
			return true
		}
	}
	// No earlier elements
	return false
}

func (r *Row) Update(msg tea.Msg) (Element, tea.Cmd) {
	var cmd tea.Cmd

	// Horizontal Navigation
	if kmsg, ok := msg.(tea.KeyMsg); ok {
		switch kmsg.String() {
		case "left":
			if r.FocusIndex > 0 {
				r.Elements[r.FocusIndex].Blur()
				r.FocusIndex--
				return r, r.Elements[r.FocusIndex].Focus()
			}
		case "right":
			if r.FocusIndex < len(r.Elements)-1 {
				r.Elements[r.FocusIndex].Blur()
				r.FocusIndex++
				return r, r.Elements[r.FocusIndex].Focus()
			}
		}
	}

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

func (r *Row) SetWidth(w int) {
	r.width = w
	if len(r.Elements) == 0 {
		return
	}

	// Calculate spacing: 1 char between each element
	numGaps := len(r.Elements) - 1
	totalSpacing := numGaps

	// Available width after accounting for spacing
	availableWidth := w - totalSpacing
	if availableWidth < 0 {
		availableWidth = 0
	}

	// Use weighted distribution if weights are provided
	if len(r.Weights) == len(r.Elements) {
		totalWeight := 0
		for _, weight := range r.Weights {
			totalWeight += weight
		}

		// Distribute width proportionally and handle rounding
		remainingWidth := availableWidth
		for i, el := range r.Elements {
			if i == len(r.Elements)-1 {
				// Last element gets all remaining width (handles rounding)
				el.SetWidth(remainingWidth)
			} else {
				elementWidth := (availableWidth * r.Weights[i]) / totalWeight
				el.SetWidth(elementWidth)
				remainingWidth -= elementWidth
			}
		}
	} else {
		// Equal distribution with rounding compensation
		baseWidth := availableWidth / len(r.Elements)
		remainder := availableWidth % len(r.Elements)

		for i, el := range r.Elements {
			elementWidth := baseWidth
			// Distribute remainder across first elements
			if i < remainder {
				elementWidth++
			}
			el.SetWidth(elementWidth)
		}
	}
}

func (r *Row) View(width int) string {
	if width > 0 {
		r.SetWidth(width)
	}

	var views []string
	for _, el := range r.Elements {
		views = append(views, el.View(0)) // Width already set via SetWidth
	}

	// Add spacing between elements
	spacer := " "
	var spacedViews []string
	for i, v := range views {
		spacedViews = append(spacedViews, v)
		if i < len(views)-1 {
			spacedViews = append(spacedViews, spacer)
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, spacedViews...)
}
