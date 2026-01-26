package image

import "time"

// Catalog represents the image registry catalog.
// It contains metadata about available VM images from various sources.
type Catalog struct {
	SchemaVersion string    `json:"schema_version"`
	UpdatedAt     time.Time `json:"updated_at"`
	Images        []Image   `json:"images"`
}

// Image represents a single VM image in the catalog.
// Each image can have multiple versions (e.g., Ubuntu 24.04, 22.04).
type Image struct {
	Name        string    `json:"name"`
	Registry    string    `json:"registry"` // "official" or "nido"
	Description string    `json:"description"`
	Homepage    string    `json:"homepage,omitempty"`
	SSHUser     string    `json:"ssh_user,omitempty"`
	Versions    []Version `json:"versions"`
}

// Version represents a specific version of a VM image.
// Includes download URL, checksum for verification, and metadata.
type Version struct {
	Version        string   `json:"version"`
	Aliases        []string `json:"aliases,omitempty"` // e.g., ["latest", "lts"]
	Arch           string   `json:"arch"`              // e.g., "amd64"
	URL            string   `json:"url"`
	ChecksumType   string   `json:"checksum_type"` // "sha256" or "sha512"
	Checksum       string   `json:"checksum"`
	SizeBytes      int64    `json:"size_bytes"`
	SizeHuman      string   `json:"size,omitempty"` // e.g., "1.2 GB"
	Format         string   `json:"format"`         // "qcow2"
	PartURLs       []string `json:"part_urls,omitempty"`
	KernelURL      string   `json:"kernel_url,omitempty"`
	KernelChecksum string   `json:"kernel_checksum,omitempty"`
	InitrdURL      string   `json:"initrd_url,omitempty"`
	InitrdChecksum string   `json:"initrd_checksum,omitempty"`
	Cmdline        string   `json:"cmdline,omitempty"`
}

// CachedImage represents a cached image file on disk.
// Used for cache management operations.
type CachedImage struct {
	Name    string    // Image name (e.g., "ubuntu")
	Version string    // Version (e.g., "24.04")
	Path    string    // Full path to cached file
	Size    int64     // File size in bytes
	ModTime time.Time // Last modification time
}

// CacheStats provides statistics about the image cache.
// Used for displaying cache information to users.
type CacheStats struct {
	TotalImages int       // Total number of cached images
	TotalSize   int64     // Total size in bytes
	OldestImage time.Time // Modification time of oldest image
	NewestImage time.Time // Modification time of newest image
}
