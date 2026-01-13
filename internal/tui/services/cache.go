// Package services provides command functions for the Nido TUI.
package services

import (
	"github.com/Josepavese/nido/internal/provider"
	tea "github.com/charmbracelet/bubbletea"
)

// --- Cache Message Types ---

// CacheItem represents a cached image.
type CacheItem struct {
	Name    string
	Version string
	Size    string
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
		var cacheItems []CacheItem
		for _, img := range items {
			cacheItems = append(cacheItems, CacheItem{
				Name:    img.Name,
				Version: img.Version,
				Size:    img.Size,
			})
		}
		return CacheListMsg{Items: cacheItems}
	}
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

// PruneCache removes unused cached images.
func PruneCache(prov provider.VMProvider) tea.Cmd {
	return func() tea.Msg {
		err := prov.CachePrune(true) // unused only
		return CachePruneMsg{Err: err}
	}
}
