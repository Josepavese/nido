// Package services provides command functions for the Nido TUI.
package ops

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Josepavese/nido/internal/builder"
	"github.com/Josepavese/nido/internal/pkg/sysutil"
	"github.com/Josepavese/nido/internal/provider"
	tea "github.com/charmbracelet/bubbletea"
)

// --- Cache Message Types ---

// CacheItem represents a cached image.
type CacheItem struct {
	Name          string
	Version       string
	Size          string
	Kind          string
	DeleteName    string
	DeleteVersion string
}

// CacheStats contains cache statistics.
type CacheStats struct {
	TotalImages int
	TotalSize   string
}

// CacheListMsg contains the list of cached images.
type CacheListMsg struct {
	Items []CacheItem
	Err   error
}

// CacheStatsMsg contains cache statistics.
type CacheStatsMsg struct {
	Stats CacheStats
	Err   error
}

// CachePruneMsg is the result of a cache prune operation.
type CachePruneMsg struct {
	Err error
}

// --- Cache Commands ---

// ListCache retrieves the list of cached images.
func ListCache(prov provider.VMProvider) tea.Cmd {
	return func() tea.Msg {
		items, err := prov.ListCachedImages()
		if err != nil {
			return CacheListMsg{Err: err}
		}
		blueprints := cachedBlueprintsByOutput(prov)
		var cacheItems []CacheItem
		for _, img := range items {
			item := CacheItem{
				Name:          img.Name,
				Version:       img.Version,
				Size:          img.Size,
				Kind:          "image",
				DeleteName:    img.Name,
				DeleteVersion: img.Version,
			}
			if bp, ok := blueprints[cacheFilename(img.Name, img.Version)]; ok {
				item.Name = bp.DisplayName
				if item.Name == "" {
					item.Name = bp.Name
				}
				item.Version = bp.Version
				item.Kind = "blueprint"
			}
			cacheItems = append(cacheItems, item)
		}
		return CacheListMsg{Items: cacheItems}
	}
}

func cachedBlueprintsByOutput(prov provider.VMProvider) map[string]builder.BlueprintInfo {
	out := map[string]builder.BlueprintInfo{}
	if prov == nil {
		return out
	}
	cfg := prov.GetConfig()
	home, _ := sysutil.UserHome()
	nidoDir := filepath.Join(home, ".nido")
	imageDir := cfg.ImageDir
	if imageDir == "" {
		imageDir = filepath.Join(nidoDir, "images")
	}
	cwd, _ := os.Getwd()
	blueprints, err := builder.ListBlueprints(cwd, nidoDir, imageDir)
	if err != nil {
		return out
	}
	for _, bp := range blueprints {
		if bp.OutputImage != "" && bp.Built {
			out[bp.OutputImage] = bp
		}
	}
	return out
}

func cacheFilename(name, version string) string {
	if version == "" {
		return fmt.Sprintf("%s.qcow2", name)
	}
	return fmt.Sprintf("%s-%s.qcow2", name, version)
}

// FetchCacheStats retrieves cache statistics.
func FetchCacheStats(prov provider.VMProvider) tea.Cmd {
	return func() tea.Msg {
		info, err := prov.CacheInfo()
		if err != nil {
			return CacheStatsMsg{Err: err}
		}
		return CacheStatsMsg{Stats: CacheStats{
			TotalImages: info.Count,
			TotalSize:   info.TotalSize,
		}}
	}
}

// PruneStats contains the result of a prune operation.
type PruneStats struct {
	Count     int
	Reclaimed int64
}

// PruneCache removes unused cached images.
func PruneCache(prov provider.VMProvider) tea.Cmd {
	return func() tea.Msg {
		count, reclaimed, err := prov.CachePrune(true) // unused only
		return OpResultMsg{
			Op:  "prune",
			Err: err,
			Data: PruneStats{
				Count:     count,
				Reclaimed: reclaimed,
			},
		}
	}
}

// DeleteCacheImage removes a specific cached image.
func DeleteCacheImage(prov provider.VMProvider, name, version string) tea.Cmd {
	opName := fmt.Sprintf("delete %s", name)
	return func() tea.Msg {
		err := prov.CacheRemove(name, version)
		return OpResultMsg{Op: opName, Err: err}
	}
}
