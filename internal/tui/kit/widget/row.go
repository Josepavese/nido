package widget

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Row represents a horizontal group of Elements.
type Row struct {
	Elements []Element
	Weights  []int
	width    int
}

// NewRow creates a new Row with equal width distribution.
func NewRow(elements ...Element) *Row {
	return &Row{Elements: elements}
}

// NewRowWithWeights creates a new Row with specified column weights.
func NewRowWithWeights(elements []Element, weights []int) *Row {
	return &Row{Elements: elements, Weights: weights}
}

// Interface compliance for Form Element

func (r *Row) Focus() tea.Cmd {
	// Focusing a row focuses its first focusable element
	for _, el := range r.Elements {
		if el.Focusable() {
			return el.Focus()
		}
	}
	return nil
}

func (r *Row) Blur() {
	for _, el := range r.Elements {
		el.Blur()
	}
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

func (r *Row) Update(msg tea.Msg) (Element, tea.Cmd) {
	var cmds []tea.Cmd
	for i, el := range r.Elements {
		newEl, cmd := el.Update(msg)
		r.Elements[i] = newEl
		cmds = append(cmds, cmd)
	}
	return r, tea.Batch(cmds...)
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
