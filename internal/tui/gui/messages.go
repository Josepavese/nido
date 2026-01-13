// Package gui implements the Nido interactive TUI using Bubble Tea.
// This file contains all message types used for state updates.
package gui

import (
	"github.com/Josepavese/nido/internal/provider"
	"github.com/Josepavese/nido/internal/tui/viewlet"
	"github.com/charmbracelet/bubbles/list"
)

// --- Tab Types ---

type tab int

const (
	tabFleet tab = iota
	tabHatchery
	tabLogs
	tabConfig
	tabHelp
)

// --- Focus Types ---

type fleetFocus int

const (
	focusList fleetFocus = iota
)

type hatcheryFocus int

const (
	focusHatchSidebar hatcheryFocus = iota
	focusHatchForm
)

// --- Operation Types ---

type operation string

const (
	opNone           operation = ""
	opSpawn          operation = "spawn"
	opStart          operation = "start"
	opStop           operation = "stop"
	opDelete         operation = "delete"
	opRefresh        operation = "refresh"
	opInfo           operation = "info"
	opCreateTemplate operation = "create-template"
)

// --- Internal Messages ---

// tickMsg is sent on each animation tick.
type tickMsg struct{}

// vmListMsg contains the refreshed list of VMs.
type vmListMsg struct{ items []list.Item }

// logMsg is an internal log message.
type logMsg struct {
	level string
	text  string
}

// opResultMsg is the result of a VM operation.
type opResultMsg struct {
	op   operation
	err  error
	path string // Optional: for templates
}

// detailMsg contains detailed VM information.
type detailMsg struct {
	name   string
	detail provider.VMDetail
	err    error
}

// updateCheckMsg contains version check results.
type updateCheckMsg struct {
	current string
	latest  string
	err     error
}

// cacheListMsg contains the list of cached images.
type cacheListMsg struct {
	items []viewlet.CacheItem
	err   error
}

// cacheStatsMsg contains cache statistics.
type cacheStatsMsg struct {
	stats viewlet.CacheStats
	err   error
}

// cachePruneMsg is the result of a cache prune operation.
type cachePruneMsg struct {
	err error
}

// sourcesLoadedMsg contains the list of available sources.
type sourcesLoadedMsg struct {
	sources []string
	err     error
}

// configSavedMsg confirms a config value was saved.
type configSavedMsg struct{ key, value string }

// resetHighlightMsg resets UI highlight state.
type resetHighlightMsg struct {
	action string
}

// downloadProgressMsg contains download progress.
type downloadProgressMsg float64

// downloadFinishedMsg indicates a download completed.
type downloadFinishedMsg struct {
	name string
	path string
	err  error
}
