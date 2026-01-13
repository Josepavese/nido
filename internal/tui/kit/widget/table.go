// Package components provides reusable UI components for the Nido TUI.
// Components wrap Bubbles widgets with Nido-specific styling and behavior.
package widget

import (
	"github.com/Josepavese/nido/internal/tui/kit/theme"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Table wraps bubbles/table with Nido theming and additional features.
// This adapter isolates the codebase from bubbles/table API changes.
type Table struct {
	inner   table.Model
	theme   theme.Theme
	focused bool
}

// TableColumn defines a column in the table.
type TableColumn struct {
	Title string
	Width int
}

// TableRow represents a row of data.
type TableRow []string

// NewTable creates a new themed table.
func NewTable(columns []TableColumn, rows []TableRow, height int) Table {
	t := theme.Current()

	// Convert to bubbles columns
	cols := make([]table.Column, len(columns))
	for i, c := range columns {
		cols[i] = table.Column{
			Title: c.Title,
			Width: c.Width,
		}
	}

	// Convert to bubbles rows
	tableRows := make([]table.Row, len(rows))
	for i, r := range rows {
		tableRows[i] = table.Row(r)
	}

	// Create table with Nido styling
	tbl := table.New(
		table.WithColumns(cols),
		table.WithRows(tableRows),
		table.WithFocused(true),
		table.WithHeight(height),
	)

	// Apply theme styles
	s := table.DefaultStyles()
	s.Header = s.Header.
		Foreground(t.Palette.AccentStrong).
		Bold(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(t.Palette.SurfaceSubtle).
		BorderBottom(true)

	s.Selected = s.Selected.
		Foreground(t.Palette.Text).
		Background(t.Palette.Surface).
		Bold(true)

	s.Cell = s.Cell.
		Foreground(t.Palette.Text)

	tbl.SetStyles(s)

	return Table{
		inner:   tbl,
		theme:   t,
		focused: true,
	}
}

// Update handles messages for the table.
func (t Table) Update(msg tea.Msg) (Table, tea.Cmd) {
	var cmd tea.Cmd
	t.inner, cmd = t.inner.Update(msg)
	return t, cmd
}

// View renders the table.
func (t Table) View() string {
	return t.inner.View()
}

// SelectedRow returns the currently selected row index.
func (t Table) SelectedRow() int {
	return t.inner.Cursor()
}

// SelectedRowData returns the data of the currently selected row.
func (t Table) SelectedRowData() TableRow {
	row := t.inner.SelectedRow()
	if row == nil {
		return nil
	}
	return TableRow(row)
}

// SetRows updates the table rows.
func (t *Table) SetRows(rows []TableRow) {
	tableRows := make([]table.Row, len(rows))
	for i, r := range rows {
		tableRows[i] = table.Row(r)
	}
	t.inner.SetRows(tableRows)
}

// SetHeight sets the table height.
func (t *Table) SetHeight(h int) {
	t.inner.SetHeight(h)
}

// SetWidth sets the table width.
func (t *Table) SetWidth(w int) {
	t.inner.SetWidth(w)
}

// Focus gives the table keyboard focus.
func (t *Table) Focus() {
	t.focused = true
	t.inner.Focus()
}

// Blur removes keyboard focus from the table.
func (t *Table) Blur() {
	t.focused = false
	t.inner.Blur()
}

// Focused returns whether the table has focus.
func (t Table) Focused() bool {
	return t.focused
}

// RowCount returns the number of rows in the table.
func (t Table) RowCount() int {
	return len(t.inner.Rows())
}

// GotoTop moves selection to the first row.
func (t *Table) GotoTop() {
	t.inner.GotoTop()
}

// GotoBottom moves selection to the last row.
func (t *Table) GotoBottom() {
	t.inner.GotoBottom()
}

// MoveUp moves selection up by n rows.
func (t *Table) MoveUp(n int) {
	t.inner.MoveUp(n)
}

// MoveDown moves selection down by n rows.
func (t *Table) MoveDown(n int) {
	t.inner.MoveDown(n)
}
