// Package gui implements the Nido interactive TUI using Bubble Tea.
// This file contains message handler functions extracted from model.Update().
package gui

import (
	"fmt"
	"strings"
	"time"

	"os"
	"path/filepath"

	"github.com/Josepavese/nido/internal/image"
	"github.com/Josepavese/nido/internal/provider"
	"github.com/Josepavese/nido/internal/tui/services"
	"github.com/Josepavese/nido/internal/tui/viewlet"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// --- Log Helpers ---

// appendLog adds a timestamped log entry and updates the logs view.
func (m *model) appendLog(text string) {
	m.logs = append(m.logs, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), text))
	m.logsView.SetContent(strings.Join(m.logs, "\n"))
}

// appendLogf adds a formatted timestamped log entry.
func (m *model) appendLogf(format string, args ...interface{}) {
	m.appendLog(fmt.Sprintf(format, args...))
}

// --- Message Handlers ---

// handleDownloadProgress updates download progress state.
func (m model) handleDownloadProgress(msg downloadProgressMsg) (model, tea.Cmd) {
	if m.downloading {
		m.downloadProgress = float64(msg)
		return m, waitForDownloadProgress(m.downloadChan)
	}
	return m, nil
}

// handleDownloadFinished processes download completion.
func (m model) handleDownloadFinished(msg downloadFinishedMsg) (model, tea.Cmd) {
	m.downloading = false
	m.loading = false
	if msg.err != nil {
		m.appendLogf("Download failed: %v", msg.err)
		return m, nil
	}
	m.appendLogf("Download complete for %s.", msg.name)

	// Resume Spawn
	name, _, gui := m.hatcheryView.GetValues()
	m.activeTab = tabFleet
	m.op = opSpawn
	m.loading = true
	return m, services.SpawnVM(m.prov, name, msg.path, "", gui)
}

// handleVMListMsg updates the VM list state.
func (m model) handleVMListMsg(msg services.VMListMsg) (model, []tea.Cmd) {
	var cmds []tea.Cmd

	if msg.Err != nil {
		m.appendLogf("List failed: %v", msg.Err)
		m.loading = false
		return m, nil
	}

	// Convert services.VMItem to local vmItem (list.Item)
	items := make([]interface{}, 0, len(msg.Items)+1)
	for _, v := range msg.Items {
		items = append(items, vmItem{
			name:    v.Name,
			state:   v.State,
			pid:     v.PID,
			sshPort: v.SSHPort,
			vncPort: v.VNCPort,
			sshUser: v.SSHUser,
		})
	}
	// Append keyhandlers.spawnItem
	items = append(items, spawnItem{})

	// SetItems expects []list.Item, so we need to cast or rely on interface
	// bubbletea list.SetItems takes []list.Item
	listItems := make([]list.Item, len(items))
	for i, it := range items {
		listItems[i] = it.(list.Item)
	}

	m.list.SetItems(listItems)
	m.page.SetTotalPages((len(items) + m.page.PerPage - 1) / m.page.PerPage)
	m.loading = false
	m.op = opNone

	// Sync data to fleetView viewlet
	fleetItems := make([]viewlet.FleetItem, 0, len(msg.Items))
	for _, v := range msg.Items {
		fleetItems = append(fleetItems, viewlet.FleetItem{
			Name:    v.Name,
			State:   v.State,
			PID:     v.PID,
			SSHPort: v.SSHPort,
			VNCPort: v.VNCPort,
			SSHUser: v.SSHUser,
		})
	}
	m.fleetView.SetItems(fleetItems)

	// Refresh detail if we have one
	if m.detailName != "" {
		cmds = append(cmds, services.FetchVMInfo(m.prov, m.detailName))
	} else if len(msg.Items) > 0 {
		// Initial selection
		if sel := m.list.SelectedItem(); sel != nil {
			if v, ok := sel.(vmItem); ok {
				m.detailName = v.name
				cmds = append(cmds, services.FetchVMInfo(m.prov, m.detailName))
			}
		}
	}

	return m, cmds
}

// handleDetailMsg updates VM detail state.
func (m model) handleDetailMsg(msg services.VMDetailMsg) model {
	if msg.Err != nil {
		if msg.Name == m.detailName {
			m.detailName = ""
			m.detail = provider.VMDetail{}
		}
		m.appendLogf("Info failed: %v", msg.Err)
	} else if msg.Name == m.detailName {
		m.detail = msg.Detail
	}
	return m
}

// handleSourcesLoadedMsg processes loaded sources.
func (m model) handleSourcesLoadedMsg(msg services.SourcesLoadedMsg) model {
	m.loading = false
	if msg.Err != nil {
		m.appendLogf("Failed to load sources: %v", msg.Err)
	} else {
		m.hatcheryView.SetSources(msg.Sources)
	}
	return m
}

// handleResetHighlightMsg resets UI highlight state.
func (m model) handleResetHighlightMsg(msg resetHighlightMsg) model {
	if msg.action == "ssh" {
		m.highlightSSH = false
	} else {
		m.highlightVNC = false
	}
	return m
}

// handleLogMsg adds a log entry.
func (m model) handleLogMsg(msg services.LogMsg) model {
	m.appendLog(msg.Text)
	return m
}

// handleOpResultMsg processes operation results.
func (m model) handleOpResultMsg(msg services.OpResultMsg) (model, tea.Cmd) {
	m.loading = false
	if msg.Err != nil {
		m.appendLogf("Operation %s failed: %v", msg.Op, msg.Err)
	} else {
		m.appendLogf("Operation %s complete.", msg.Op)
	}
	m.op = opNone
	return m, services.RefreshFleet(m.prov)
}

// handleConfigSavedMsg processes config save confirmation.
func (m model) handleConfigSavedMsg(msg services.ConfigSavedMsg) model {
	m.loading = false
	m.appendLogf("Config %s updated to %s", msg.Key, msg.Value)
	m.configView.RefreshItems()
	return m
}

// handleUpdateCheckMsg processes version check results.
func (m model) handleUpdateCheckMsg(msg services.UpdateCheckMsg) model {
	if msg.Err != nil {
		m.appendLogf("Update check failed: %v", msg.Err)
		m.configView.SetUpdateStatus("", "", false)
	} else {
		m.configView.SetUpdateStatus(msg.Current, msg.Latest, false)
		m.appendLogf("Version check complete: %s", msg.Current)
	}
	return m
}

// handleCacheListMsg processes cache list results.
func (m model) handleCacheListMsg(msg services.CacheListMsg) model {
	m.loading = false
	if msg.Err != nil {
		m.appendLogf("Cache list failed: %v", msg.Err)
	} else {
		// Convert services.CacheItem to viewlet.CacheItem
		items := make([]viewlet.CacheItem, len(msg.Items))
		for i, x := range msg.Items {
			items[i] = viewlet.CacheItem{
				Name:    x.Name,
				Version: x.Version,
				Size:    x.Size,
			}
		}
		m.configView.SetCacheList(items)
	}
	return m
}

// handleCacheStatsMsg processes cache stats results.
func (m model) handleCacheStatsMsg(msg services.CacheStatsMsg) model {
	m.loading = false
	if msg.Err != nil {
		m.appendLogf("Cache info failed: %v", msg.Err)
	} else {
		// Convert services.CacheStats to viewlet.CacheStats
		stats := viewlet.CacheStats{
			TotalImages: msg.Stats.TotalImages,
			TotalSize:   msg.Stats.TotalSize,
		}
		m.configView.SetCacheStats(stats)
	}
	return m
}

// handleCachePruneMsg processes cache prune results.
func (m model) handleCachePruneMsg(msg services.CachePruneMsg) (model, []tea.Cmd) {
	var cmds []tea.Cmd
	m.loading = false
	if msg.Err != nil {
		m.appendLogf("Cache prune failed: %v", msg.Err)
	} else {
		m.appendLog("Cache pruned successfully")
		cmds = append(cmds, services.ListCache(m.prov), services.FetchCacheStats(m.prov))
	}
	return m, cmds
}

// handleSubmitHatchery handles the form submission logic extracted from model.go
func (m model) handleSubmitHatchery() (model, tea.Cmd) {
	name, source, gui := m.hatcheryView.GetValues()
	isSpawn := m.hatcheryView.Mode == viewlet.HatcherySpawn

	// Input Validation
	if name == "" {
		m.appendLog("Hatchery: Name is required!")
		return m, nil
	}
	if source == "" {
		m.appendLog("Hatchery: Source is required!")
		return m, nil
	}

	m.loading = true
	m.activeTab = tabFleet // Switch back to view progress

	if isSpawn {
		// SPAWN
		m.op = opSpawn

		// Resolve Source Path
		realSource := source
		if strings.Contains(source, "[IMAGE]") {
			// Extract Name:Version
			tag := strings.TrimPrefix(source, "[IMAGE] ")
			tag = strings.TrimSpace(tag)

			// Resolve image directory
			imgDir := m.cfg.ImageDir
			if imgDir == "" {
				home, _ := os.UserHomeDir()
				imgDir = filepath.Join(home, ".nido", "images")
			}

			// Parse tag
			parts := strings.Split(tag, ":")
			if len(parts) == 2 {
				name, ver := parts[0], parts[1]
				imgPath := filepath.Join(imgDir, fmt.Sprintf("%s-%s.qcow2", name, ver))

				// Check if exists
				if _, err := os.Stat(imgPath); os.IsNotExist(err) {
					// Need download!
					catalog, err := image.LoadCatalog(imgDir, image.DefaultCacheTTL)
					if err == nil {
						_, verEntry, err := catalog.FindImage(name, ver)
						if err == nil {
							// START ASYNC DOWNLOAD
							m.downloading = true
							m.downloadProgress = 0
							m.downloadChan = make(chan float64)
							m.appendLogf("Starting download for %s:%s...", name, ver)

							// Return batch: start download routine AND start listener routine
							return m, tea.Batch(
								m.downloadImageCmd(verEntry.URL, imgPath, name, verEntry.SizeBytes, m.downloadChan),
								waitForDownloadProgress(m.downloadChan),
							)
						}
					}
					// If catalog/image not found, proceed and let spawn fail naturally or use fallback
				}

				realSource = imgPath
			} else {
				// Fallback for simple names if any (legacy flat files?)
				realSource = filepath.Join(imgDir, tag)
			}
		} else if strings.Contains(source, "[TEMPLATE]") {
			realSource = strings.TrimPrefix(source, "[TEMPLATE] ")
			realSource = strings.TrimSpace(realSource)
		}

		return m, services.SpawnVM(m.prov, name, realSource, "", gui)
	} else {
		// CREATE TEMPLATE
		m.op = "create-template"
		vmName := strings.TrimPrefix(source, "[VM] ")
		vmName = strings.TrimSpace(vmName)
		return m, services.CreateTemplate(m.prov, vmName, name)
	}
}
