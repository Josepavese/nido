package image

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Josepavese/nido/internal/cli"
)

const (
	// CatalogURL is the remote catalog location on GitHub
	CatalogURL = "https://raw.githubusercontent.com/Josepavese/nido/main/registry/images.json"

	// DefaultCacheTTL is the default cache time-to-live (6 hours)
	DefaultCacheTTL = 6 * time.Hour

	// CatalogCacheFile is the filename for the cached catalog
	CatalogCacheFile = ".catalog.json"
)

// LoadCatalogFromFile loads a catalog from a specific file path.
func LoadCatalogFromFile(path string) (*Catalog, error) {
	return loadFromFile(path)
}

// LoadCatalog loads the image catalog from cache or remote source.
// It implements a cache-first strategy with TTL:
// 1. If cache exists and is fresh (< TTL), use it
// 2. Otherwise, try to fetch from remote
// 3. If remote fails, fall back to stale cache (if available)
//
// This ensures offline functionality while keeping catalog reasonably up-to-date.
func LoadCatalog(cacheDir string, ttl time.Duration) (*Catalog, error) {
	cachePath := filepath.Join(cacheDir, CatalogCacheFile)

	// Check if cache exists and is fresh
	if stat, err := os.Stat(cachePath); err == nil {
		age := time.Since(stat.ModTime())
		if age < ttl {
			// Cache is fresh, use it
			return loadFromFile(cachePath)
		}
	}

	// Try to fetch from remote
	catalog, err := fetchRemote(CatalogURL)
	if err != nil {
		// Remote fetch failed, try stale cache as fallback
		if c, cacheErr := loadFromFile(cachePath); cacheErr == nil {
			return c, nil
		}
		return nil, fmt.Errorf("failed to load catalog: %w", err)
	}

	// Save to cache for future use
	if err := saveToFile(catalog, cachePath); err != nil {
		// Non-fatal: we have the catalog, just couldn't cache it
		if !cli.IsJSONMode() {
			fmt.Fprintf(os.Stderr, "Warning: failed to cache catalog: %v\n", err)
		}
	}

	return catalog, nil
}

// FindImage locates an image and version in the catalog.
// If version is empty, returns the first version (usually latest).
// Returns the Image, Version, and any error encountered.
func (c *Catalog) FindImage(name, version string) (*Image, *Version, error) {
	for i := range c.Images {
		if c.Images[i].Name == name {
			img := &c.Images[i]

			// If no version specified, return first version
			if version == "" {
				if len(img.Versions) == 0 {
					return nil, nil, fmt.Errorf("image %s has no versions", name)
				}
				return img, &img.Versions[0], nil
			}

			// Search for exact version or alias match
			for j := range img.Versions {
				v := &img.Versions[j]
				if v.Version == version {
					return img, v, nil
				}
				// Check aliases
				for _, alias := range v.Aliases {
					if alias == version {
						return img, v, nil
					}
				}
			}

			return nil, nil, fmt.Errorf("version %s not found for image %s", version, name)
		}
	}

	return nil, nil, fmt.Errorf("image %s not found in catalog", name)
}

// HasVersion checks if a specific image version exists in the catalog.
// Used by registry builder to detect new versions.
func (c *Catalog) HasVersion(imageName, version string) bool {
	_, _, err := c.FindImage(imageName, version)
	return err == nil
}

// GetCachedImages returns a list of all cached image files.
// Scans the cache directory and returns metadata for each cached image.
func (c *Catalog) GetCachedImages(cacheDir string) ([]CachedImage, error) {
	var cached []CachedImage

	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".qcow2") {
			continue
		}

		// Parse filename: <name>-<version>.qcow2
		name := strings.TrimSuffix(entry.Name(), ".qcow2")
		parts := strings.Split(name, "-")
		if len(parts) < 2 {
			continue // Skip malformed filenames
		}

		imageName := parts[0]
		imageVersion := strings.Join(parts[1:], "-")

		info, err := entry.Info()
		if err != nil {
			continue
		}

		fullPath := filepath.Join(cacheDir, entry.Name())
		cached = append(cached, CachedImage{
			Name:    imageName,
			Version: imageVersion,
			Path:    fullPath,
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}

	return cached, nil
}

// GetCacheStats calculates statistics about the image cache.
// Returns total count, size, and age information.
func (c *Catalog) GetCacheStats(cacheDir string) (*CacheStats, error) {
	cached, err := c.GetCachedImages(cacheDir)
	if err != nil {
		return nil, err
	}

	if len(cached) == 0 {
		return &CacheStats{}, nil
	}

	stats := &CacheStats{
		TotalImages: len(cached),
		OldestImage: cached[0].ModTime,
		NewestImage: cached[0].ModTime,
	}

	for _, img := range cached {
		stats.TotalSize += img.Size
		if img.ModTime.Before(stats.OldestImage) {
			stats.OldestImage = img.ModTime
		}
		if img.ModTime.After(stats.NewestImage) {
			stats.NewestImage = img.ModTime
		}
	}

	return stats, nil
}

// RemoveCachedImage removes a specific cached image file.
// Returns an error if the image is not found or cannot be deleted.
func (c *Catalog) RemoveCachedImage(cacheDir, name, version string) error {
	filename := fmt.Sprintf("%s-%s.qcow2", name, version)
	path := filepath.Join(cacheDir, filename)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("cached image not found: %s:%s", name, version)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to remove cached image: %w", err)
	}

	return nil
}

// PruneCache removes cached images based on the specified criteria.
// If unusedOnly is true, only removes images not currently used by any VM.
// Returns the number of images removed and any error encountered.
func (c *Catalog) PruneCache(cacheDir string, unusedOnly bool, activeVMs []string) (int, error) {
	cached, err := c.GetCachedImages(cacheDir)
	if err != nil {
		return 0, err
	}

	removed := 0
	for _, img := range cached {
		// If unusedOnly is true, check if image is in use
		if unusedOnly {
			inUse := false
			for _, vm := range activeVMs {
				// Check if VM disk path matches this cached image
				if strings.Contains(vm, img.Name) && strings.Contains(vm, img.Version) {
					inUse = true
					break
				}
			}
			if inUse {
				continue
			}
		}

		// Remove the image
		if err := os.Remove(img.Path); err != nil {
			// Log error but continue with other images
			if !cli.IsJSONMode() {
				fmt.Fprintf(os.Stderr, "Warning: failed to remove %s: %v\n", img.Path, err)
			}
			continue
		}
		removed++
	}

	return removed, nil
}

// Internal helper functions

func loadFromFile(path string) (*Catalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read catalog file: %w", err)
	}

	var catalog Catalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("failed to parse catalog: %w", err)
	}

	// Compute human-readable sizes
	catalog.computeSizes()

	return &catalog, nil
}

func fetchRemote(url string) (*Catalog, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch catalog: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var catalog Catalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("failed to parse catalog: %w", err)
	}

	// Compute human-readable sizes
	catalog.computeSizes()

	return &catalog, nil
}

func saveToFile(catalog *Catalog, path string) error {
	data, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal catalog: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write catalog: %w", err)
	}

	return nil
}

// computeSizes calculates human-readable size strings for all versions.
// This is called after loading/fetching the catalog.
func (c *Catalog) computeSizes() {
	for i := range c.Images {
		for j := range c.Images[i].Versions {
			v := &c.Images[i].Versions[j]
			v.SizeHuman = FormatBytes(v.SizeBytes)
		}
	}
}

// FormatBytes converts bytes to human-readable format (KB, MB, GB, etc.).
// Exported for use in CLI and other packages.
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
