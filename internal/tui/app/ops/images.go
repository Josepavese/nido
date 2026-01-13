// Package services provides command functions for the Nido TUI.
package ops

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/Josepavese/nido/internal/image"
	"github.com/Josepavese/nido/internal/provider"
	tea "github.com/charmbracelet/bubbletea"
)

// --- Image/Source Message Types ---

// SourcesLoadedMsg contains the list of available sources (templates/images/VMs).
type SourcesLoadedMsg struct {
	Sources []string
	Err     error
}

// SourceAction defines what type of sources to fetch.
type SourceAction int

const (
	// SourceActionSpawn fetches templates and cloud images for spawning VMs.
	SourceActionSpawn SourceAction = 0
	// SourceActionTemplate fetches VMs for creating templates.
	SourceActionTemplate SourceAction = 1
)

// --- Image Commands ---

// FetchSources retrieves available sources based on the action type.
// For SourceActionSpawn: returns templates and cloud images.
// For SourceActionTemplate: returns list of VMs.
func FetchSources(prov provider.VMProvider, action SourceAction) tea.Cmd {
	return func() tea.Msg {
		var srcList []string

		if action == SourceActionSpawn {
			// 1. Local Templates (from BackupDir)
			templates, err := prov.ListTemplates()
			if err != nil {
				return SourcesLoadedMsg{Err: err}
			}
			sort.Strings(templates)

			// 2. Cloud Images from Catalog (with registry distinction)
			cfg := prov.GetConfig()
			imagesDir := cfg.ImageDir
			if imagesDir == "" {
				home, _ := os.UserHomeDir()
				imagesDir = filepath.Join(home, ".nido", "images")
			}

			catalog, err := image.LoadCatalog(imagesDir, image.DefaultCacheTTL)
			if err != nil {
				// Catalog failed, but we can still show templates
				for _, tpl := range templates {
					srcList = append(srcList, fmt.Sprintf("[TEMPLATE] %s", tpl))
				}
				if len(srcList) == 0 {
					return SourcesLoadedMsg{Err: fmt.Errorf("no templates found (catalog unavailable)")}
				}
				return SourcesLoadedMsg{Sources: srcList}
			}

			// Build source list with proper labels
			// Templates first
			for _, tpl := range templates {
				srcList = append(srcList, fmt.Sprintf("[TEMPLATE] %s", tpl))
			}

			// Then Flavours (registry=nido), then Cloud (registry=official)
			var flavours, cloud []string
			for _, img := range catalog.Images {
				for _, ver := range img.Versions {
					entry := fmt.Sprintf("%s:%s", img.Name, ver.Version)
					if img.Registry == "nido" {
						flavours = append(flavours, entry)
					} else {
						cloud = append(cloud, entry)
					}
				}
			}

			sort.Strings(flavours)
			sort.Strings(cloud)

			for _, f := range flavours {
				srcList = append(srcList, fmt.Sprintf("[FLAVOUR] %s", f))
			}
			for _, c := range cloud {
				srcList = append(srcList, fmt.Sprintf("[CLOUD] %s", c))
			}

			if len(srcList) == 0 {
				return SourcesLoadedMsg{Err: fmt.Errorf("no images or templates found")}
			}

			return SourcesLoadedMsg{Sources: srcList}
		}

		// SourceActionTemplate: List VMs for creating templates
		vms, err := prov.List()
		if err != nil {
			return SourcesLoadedMsg{Err: err}
		}
		sort.Slice(vms, func(i, j int) bool {
			return vms[i].Name < vms[j].Name
		})
		for _, vm := range vms {
			srcList = append(srcList, fmt.Sprintf("[VM] %s", vm.Name))
		}

		if len(srcList) == 0 {
			return SourcesLoadedMsg{Err: fmt.Errorf("no images or templates found")}
		}

		return SourcesLoadedMsg{Sources: srcList}
	}
}
