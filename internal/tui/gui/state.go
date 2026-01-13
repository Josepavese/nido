// Package gui implements the Nido interactive TUI using Bubble Tea.
// This file contains state holder structs for organizing model state.
package gui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// --- List Item Types ---

// vmItem represents a VM in the fleet sidebar.
type vmItem struct {
	name    string
	state   string
	pid     int
	sshPort int
	vncPort int
	sshUser string
}

func (i vmItem) Title() string {
	indicator := "🔴"
	if i.state == "running" {
		indicator = "🟢"
	}
	name := i.name
	// Truncate name if too long for 18-char sidebar:
	// Sidebar Width (18) - Indicator (2) - Space (1) - Padding (2) = 13 chars safe
	if len(name) > 13 {
		name = name[:12] + "..."
	}
	return fmt.Sprintf("%s %s", indicator, name)
}
func (i vmItem) Description() string { return i.state }
func (i vmItem) FilterValue() string { return i.name }
func (i vmItem) String() string      { return i.Title() }

// spawnItem is the special "Spawn new bird" item in the fleet sidebar.
type spawnItem struct{}

func (i spawnItem) Title() string       { return "+ Spawn new bird (VM)" }
func (i spawnItem) Description() string { return "" }
func (i spawnItem) FilterValue() string { return "" }
func (i spawnItem) String() string      { return i.Title() }

// hatchTypeItem represents a hatchery action type.
type hatchTypeItem struct {
	title string
	desc  string
}

func (i hatchTypeItem) String() string      { return i.title }
func (i hatchTypeItem) Title() string       { return i.title }
func (i hatchTypeItem) Description() string { return i.desc }
func (i hatchTypeItem) FilterValue() string { return i.title }

// listItem is a simple string item for lists.
type listItem string

func (i listItem) String() string      { return string(i) }
func (i listItem) FilterValue() string { return string(i) }
func (i listItem) Title() string       { return string(i) }
func (i listItem) Description() string { return "Source Image / Template" }

// --- List Delegates ---

// customDelegate for Sidebar items to prevent padding shifts.
type customDelegate struct{}

func (d customDelegate) Height() int                             { return 1 }
func (d customDelegate) Spacing() int                            { return 0 }
func (d customDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d customDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	// Check for Spawn Item first
	if _, ok := listItem.(spawnItem); ok {
		str := "+ Spawn new bird (VM)"
		if index == m.Index() {
			fmt.Fprint(w, hatchButtonActiveStyle.Render(str))
		} else {
			fmt.Fprint(w, hatchButtonStyle.Render(str))
		}
		return
	}

	str, ok := listItem.(fmt.Stringer)
	if !ok {
		return
	}

	// Check if this item is selected
	if index == m.Index() {
		fmt.Fprint(w, sidebarItemSelectedStyle.Render(str.String()))
	} else {
		fmt.Fprint(w, sidebarItemStyle.Render(str.String()))
	}
}

// --- Hatchery State ---

// hatcheryState holds the legacy hatchery sidebar state.
type hatcheryState struct {
	Sidebar list.Model
}
