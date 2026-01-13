// Package gui implements the Nido interactive TUI using Bubble Tea.
// This file contains key handling functions extracted from model.handleKey().
package gui

import (
	"github.com/Josepavese/nido/internal/tui/services"
	"github.com/Josepavese/nido/internal/tui/theme"
	"github.com/Josepavese/nido/internal/tui/viewlet"
	tea "github.com/charmbracelet/bubbletea"
)

// --- Tab-Specific Key Handlers ---

// handleFleetKeys handles keyboard input for the Fleet tab.
func (m model) handleFleetKeys(msg tea.KeyMsg) (model, tea.Cmd, bool) {
	switch msg.String() {
	case "enter":
		if sel := m.list.SelectedItem(); sel != nil {
			if _, ok := sel.(spawnItem); ok {
				m.activeTab = tabHatchery
				return m, nil, true
			}
			if item, ok := sel.(vmItem); ok {
				if item.state == "running" {
					m.loading = true
					m.op = opStop
					return m, services.StopVM(m.prov, item.name), true
				}
				m.loading = true
				m.op = opStart
				return m, services.StartVM(m.prov, item.name), true
			}
		}
	case "x":
		if sel := m.list.SelectedItem(); sel != nil {
			if item, ok := sel.(vmItem); ok {
				m.loading = true
				m.op = opStop
				return m, services.StopVM(m.prov, item.name), true
			}
		}
	case "delete":
		if sel := m.list.SelectedItem(); sel != nil {
			if item, ok := sel.(vmItem); ok {
				m.loading = true
				m.op = opDelete
				return m, services.DeleteVM(m.prov, item.name), true
			}
		}
	}
	// Fallback to list navigation
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd, true
}

// handleHatcheryKeys handles keyboard input for the Hatchery tab.
func (m model) handleHatcheryKeys(msg tea.KeyMsg) (model, tea.Cmd, bool) {
	if m.hatcheryFocus == focusHatchSidebar {
		switch msg.String() {
		case "right", "tab":
			m.hatcheryFocus = focusHatchForm
			return m, nil, true
		case "enter":
			m.hatcheryFocus = focusHatchForm
			return m, nil, true
		}
		var cmd tea.Cmd
		m.hatchery.Sidebar, cmd = m.hatchery.Sidebar.Update(msg)

		// Sync mode to viewlet
		mode := viewlet.HatcherySpawn
		if m.hatchery.Sidebar.Index() == 1 {
			mode = viewlet.HatcheryTemplate
		}
		m.hatcheryView.SetMode(mode)

		return m, cmd, true
	}

	// Form focus - delegate to viewlet
	updatedView, cmd := m.hatcheryView.Update(msg)
	if hView, ok := updatedView.(*viewlet.Hatchery); ok {
		m.hatcheryView = hView
	}

	// Handle explicit "Enter" for submission
	if msg.String() == "enter" {
		if m.hatcheryView.IsSubmitted() {
			newM, cmd := m.handleSubmitHatchery()
			return newM, cmd, true
		}

		if m.hatcheryView.IsSelecting() {
			action := 0 // Spawn
			if m.hatcheryView.Mode == viewlet.HatcheryTemplate {
				action = 1
			}
			m.loading = true
			return m, services.FetchSources(m.prov, services.SourceAction(action)), true
		}
	}
	return m, cmd, true
}

// handleConfigKeys handles keyboard input for the Config tab.
func (m model) handleConfigKeys(msg tea.KeyMsg) (model, tea.Cmd, bool) {
	var cmd tea.Cmd
	var v viewlet.Viewlet
	v, cmd = m.configView.Update(msg)
	m.configView = v.(*viewlet.Config)
	return m, cmd, true
}

// handleLogsKeys handles keyboard input for the Logs tab.
func (m model) handleLogsKeys(msg tea.KeyMsg) (model, tea.Cmd, bool) {
	var cmd tea.Cmd
	var v viewlet.Viewlet
	v, cmd = m.logsView.Update(msg)
	m.logsView = v.(*viewlet.Logs)
	return m, cmd, true
}

// handleGlobalKeys handles global keyboard shortcuts (quit, tab switching, refresh).
func (m model) handleGlobalKeys(msg tea.KeyMsg) (model, tea.Cmd, bool) {
	switch msg.String() {
	case m.keymap.Quit, "ctrl+c":
		if m.isInputFocused() {
			return m, nil, false
		}
		return m, tea.Quit, true
	case m.keymap.TabSelect[0]:
		if m.isInputFocused() {
			return m, nil, false
		}
		m.activeTab = tabFleet
		return m, nil, true
	case m.keymap.TabSelect[1]:
		if m.isInputFocused() {
			return m, nil, false
		}
		m.activeTab = tabHatchery
		return m, nil, true
	case m.keymap.TabSelect[2]:
		if m.isInputFocused() {
			return m, nil, false
		}
		m.activeTab = tabLogs
		return m, nil, true
	case m.keymap.TabSelect[3]:
		if m.isInputFocused() {
			return m, nil, false
		}
		m.activeTab = tabConfig
		return m, nil, true
	case m.keymap.TabSelect[4], "h":
		if m.isInputFocused() {
			return m, nil, false
		}
		m.activeTab = tabHelp
		return m, nil, true
	case m.keymap.Refresh:
		if m.isInputFocused() {
			return m, nil, false
		}
		m.loading = true
		m.op = opRefresh
		return m, services.RefreshFleet(m.prov), true
	}
	return m, nil, false
}

// handleNavigationKeys handles arrow key navigation between tabs.
func (m model) handleNavigationKeys(msg tea.KeyMsg) (model, tea.Cmd, bool) {
	if msg.String() != "left" && msg.String() != "right" {
		return m, nil, false
	}

	// Exception: In Hatchery AND focused on Form AND in input field
	if m.activeTab == tabHatchery && m.hatcheryFocus == focusHatchForm && m.hatcheryView.IsTyping() {
		return m, nil, false
	}
	// Exception: In Config AND focused on Form
	if m.activeTab == tabConfig && m.configView.Mode == viewlet.ConfigModeForm {
		return m, nil, false
	}

	// Perform Switch
	if msg.String() == "left" {
		m.activeTab = (m.activeTab - 1 + 5) % 5
	} else {
		m.activeTab = (m.activeTab + 1) % 5
	}

	// Reset focus when entering tabs
	if m.activeTab == tabHatchery {
		m.hatcheryFocus = focusHatchSidebar
	}
	return m, nil, true
}

// --- Mouse Handlers ---

// handleHeaderMouse handles mouse clicks on the header (tabs, exit button).
func (m model) handleHeaderMouse(msg tea.MouseMsg) (model, tea.Cmd, bool) {
	if msg.Y != 0 {
		return m, nil, false
	}

	// Exit Button Click (Rightmost configured chars)
	if msg.X >= m.width-theme.Width.ExitZone {
		return m, tea.Quit, true
	}

	// Tab Switching (5 tabs)
	availableWidth := m.width - headerReserve
	tabWidth := availableWidth / 5
	if tabWidth > 0 {
		clickIndex := msg.X / tabWidth
		if clickIndex >= 0 && clickIndex <= 4 {
			m.activeTab = tab(clickIndex)
			return m, nil, true
		}
	}
	return m, nil, false
}

// handleFleetMouse handles mouse clicks in Fleet tab.
func (m model) handleFleetMouse(msg tea.MouseMsg) (model, tea.Cmd) {
	// Sidebar click (sidebar width + border ≈ 24)
	if msg.X < 24 {
		row := msg.Y - 4 // Offset 4 (Header 2 + SubHeader 2)
		if row >= 0 {
			pageStart := m.list.Paginator.Page * m.list.Paginator.PerPage
			index := pageStart + row
			if index >= 0 && index < len(m.list.Items()) {
				m.list.Select(index)
				if sel := m.list.SelectedItem(); sel != nil {
					if v, ok := sel.(vmItem); ok {
						m.detailName = v.name
						return m, services.FetchVMInfo(m.prov, m.detailName)
					} else if _, ok := sel.(spawnItem); ok {
						m.activeTab = tabHatchery
						return m, nil
					}
				}
			} else if index >= len(m.list.Items()) && index <= len(m.list.Items())+3 {
				// Check for spawnItem wrap
				lastIdx := len(m.list.Items()) - 1
				if lastIdx >= 0 {
					if _, ok := m.list.Items()[lastIdx].(spawnItem); ok {
						m.list.Select(lastIdx)
						m.activeTab = tabHatchery
						return m, nil
					}
				}
			}
		}
	} else {
		// Main Area - delegate to Fleet Viewlet
		var v viewlet.Viewlet
		var cmd tea.Cmd
		v, cmd = m.fleetView.Update(msg)
		m.fleetView = v.(*viewlet.Fleet)
		return m, cmd
	}
	return m, nil
}

// handleHatcheryMouse handles mouse clicks in Hatchery tab.
func (m model) handleHatcheryMouse(msg tea.MouseMsg) (model, tea.Cmd) {
	if msg.X < 28 {
		// Sidebar Click
		row := msg.Y - 4
		if row >= 0 && row <= 1 {
			m.hatchery.Sidebar.Select(row)
			m.hatcheryFocus = focusHatchSidebar
			return m, nil
		}
	}
	return m, nil
}
