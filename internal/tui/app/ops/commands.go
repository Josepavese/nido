// Package services provides command functions for the Nido TUI.
// These functions return tea.Cmd that can be dispatched by the model.
// Extracting commands here reduces model.go complexity and improves testability.
package ops

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Josepavese/nido/internal/config"
	"github.com/Josepavese/nido/internal/provider"
	tea "github.com/charmbracelet/bubbletea"
)

// --- Message Types ---

// VMListMsg contains the refreshed list of VMs.
type VMListMsg struct {
	Items []VMItem
	Err   error
}

// VMItem represents a VM in the fleet list.
type VMItem struct {
	Name    string
	State   string
	PID     int
	SSHPort int
	VNCPort int
	SSHUser string
}

// VMDetailMsg contains detailed VM information.
type VMDetailMsg struct {
	Name   string
	Detail provider.VMDetail
	Err    error
}

// OpResultMsg is the result of a VM operation.
type OpResultMsg struct {
	Op   string
	Err  error
	Path string // Optional: for templates
}

// LogMsg is an internal log message.
type LogMsg struct {
	Level string
	Text  string
}

// UpdateCheckMsg contains version check results.
type UpdateCheckMsg struct {
	Current string
	Latest  string
	Err     error
}

// ConfigSavedMsg confirms a config value was saved.
type ConfigSavedMsg struct {
	Key   string
	Value string
}

// VMDetailRequestMsg requests details for a VM.
type VMDetailRequestMsg struct {
	Name string
}

// VM Operation Constants
const (
	OpStart  = "start"
	OpStop   = "stop"
	OpDelete = "delete"
)

// RequestOpMsg requests a VM operation.
type RequestOpMsg struct {
	Op   string
	Name string
}

// --- VM Commands ---

// RefreshFleet fetches the current VM list from the provider.
func RefreshFleet(prov provider.VMProvider) tea.Cmd {
	return func() tea.Msg {
		vms, err := prov.List()
		if err != nil {
			return VMListMsg{Err: err}
		}

		// Sort VMs alphabetically by Name
		sort.Slice(vms, func(i, j int) bool {
			return strings.ToLower(vms[i].Name) < strings.ToLower(vms[j].Name)
		})

		items := make([]VMItem, 0, len(vms))
		for _, v := range vms {
			items = append(items, VMItem{
				Name:    v.Name,
				State:   v.State,
				PID:     v.PID,
				SSHPort: v.SSHPort,
				VNCPort: v.VNCPort,
				SSHUser: v.SSHUser,
			})
		}
		return VMListMsg{Items: items}
	}
}

// FetchVMInfo retrieves detailed information about a VM.
func FetchVMInfo(prov provider.VMProvider, name string) tea.Cmd {
	return func() tea.Msg {
		detail, err := prov.Info(name)
		return VMDetailMsg{Name: name, Detail: detail, Err: err}
	}
}

// SpawnVM creates a new VM from a template.
func SpawnVM(prov provider.VMProvider, name, template, userData string, gui bool) tea.Cmd {
	return func() tea.Msg {
		opts := provider.VMOptions{
			DiskPath:     template,
			UserDataPath: userData,
			Gui:          gui,
			SSHUser:      "",
		}
		err := prov.Spawn(name, opts)
		return OpResultMsg{Op: "spawn", Err: err}
	}
}

// StartVM starts a stopped VM.
func StartVM(prov provider.VMProvider, name string) tea.Cmd {
	return func() tea.Msg {
		err := prov.Start(name, provider.VMOptions{Gui: true})
		return OpResultMsg{Op: "start", Err: err}
	}
}

// StopVM stops a running VM.
func StopVM(prov provider.VMProvider, name string) tea.Cmd {
	return func() tea.Msg {
		err := prov.Stop(name, true)
		return OpResultMsg{Op: "stop", Err: err}
	}
}

// DeleteVM removes a VM.
func DeleteVM(prov provider.VMProvider, name string) tea.Cmd {
	return func() tea.Msg {
		err := prov.Delete(name)
		return OpResultMsg{Op: "delete", Err: err}
	}
}

// CreateTemplate creates a new template from a VM.
func CreateTemplate(prov provider.VMProvider, vmName, templateName string) tea.Cmd {
	return func() tea.Msg {
		path, err := prov.CreateTemplate(vmName, templateName)
		return OpResultMsg{Op: "create-template", Err: err, Path: path}
	}
}

// DeleteTemplate removes a template.
func DeleteTemplate(prov provider.VMProvider, name string) tea.Cmd {
	return func() tea.Msg {
		err := prov.DeleteTemplate(name)
		return OpResultMsg{Op: "delete-template", Err: err}
	}
}

// --- Config Commands ---

// CheckUpdate checks for available updates.
func CheckUpdate() tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("nido", "version").Output()
		if err != nil {
			return UpdateCheckMsg{Err: err}
		}
		current := strings.TrimSpace(string(out))
		// Extract version number (e.g., "Nido v4.3.6 (State: Evolved)" -> "v4.3.6")
		parts := strings.Fields(current)
		if len(parts) >= 2 {
			current = parts[1]
		}
		return UpdateCheckMsg{Current: current, Latest: current} // TODO: Check GitHub for latest
	}
}

// SaveConfig updates a config value and reloads the config.
func SaveConfig(cfg *config.Config, key, value string) tea.Cmd {
	return func() tea.Msg {
		// Find config file path
		home, _ := os.UserHomeDir()
		path := filepath.Join(home, ".nido", "config.env")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			cwd, _ := os.Getwd()
			path = filepath.Join(cwd, "config", "config.env")
		}

		err := config.UpdateConfig(path, key, value)
		if err != nil {
			return LogMsg{Level: "error", Text: fmt.Sprintf("Save failed: %v", err)}
		}

		// Reload config into memory
		newCfg, _ := config.LoadConfig(path)
		*cfg = *newCfg

		return ConfigSavedMsg{Key: key, Value: value}
	}
}
