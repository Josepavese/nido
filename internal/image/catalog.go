package image

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
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
		// Remote failed, try cache as fallback (even if stale)
		if _, statErr := os.Stat(cachePath); statErr == nil {
			return loadFromFile(cachePath)
		}
		return nil, fmt.Errorf("failed to load catalog: remote unreachable and no cache available: %w", err)
	}

	// Save to cache for future use
	if err := saveToFile(catalog, cachePath); err != nil {
		// Log warning but don't fail (catalog is still usable)
		fmt.Fprintf(os.Stderr, "Warning: failed to cache catalog: %v\n", err)
	}

	return catalog, nil
}

// loadFromFile loads and validates a catalog from a local file.
func loadFromFile(path string) (*Catalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read catalog: %w", err)
	}

	var catalog Catalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("failed to parse catalog JSON: %w", err)
	}

	// Validate schema version
	if catalog.SchemaVersion != "1" {
		return nil, fmt.Errorf("unsupported catalog schema version: %s (expected: 1)", catalog.SchemaVersion)
	}

	return &catalog, nil
}

// fetchRemote downloads the catalog from the remote URL.
func fetchRemote(url string) (*Catalog, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var catalog Catalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("invalid JSON from remote: %w", err)
	}

	// Validate schema version
	if catalog.SchemaVersion != "1" {
		return nil, fmt.Errorf("unsupported remote catalog schema: %s", catalog.SchemaVersion)
	}

	return &catalog, nil
}

// saveToFile saves a catalog to a local file with pretty formatting.
func saveToFile(catalog *Catalog, path string) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	data, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal catalog: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// FindImage finds an image by name and version in the catalog.
// Supports version aliases (e.g., "latest", "lts", "noble").
// If version is empty, defaults to "latest".
//
// Returns the Image and specific Version, or an error if not found.
func (c *Catalog) FindImage(name, version string) (*Image, *Version, error) {
	// Find image by name
	var img *Image
	for i := range c.Images {
		if c.Images[i].Name == name {
			img = &c.Images[i]
			break
		}
	}

	if img == nil {
		return nil, nil, fmt.Errorf("image not found: %s", name)
	}

	// If no version specified, use "latest"
	if version == "" {
		version = "latest"
	}

	// Find version (exact match or alias)
	for i := range img.Versions {
		v := &img.Versions[i]

		// Exact version match
		if v.Version == version {
			return img, v, nil
		}

		// Alias match
		for _, alias := range v.Aliases {
			if alias == version {
				return img, v, nil
			}
		}
	}

	return nil, nil, fmt.Errorf("version not found: %s:%s", name, version)
}
