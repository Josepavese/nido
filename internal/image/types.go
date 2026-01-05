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
	Versions    []Version `json:"versions"`
}

// Version represents a specific version of a VM image.
// Includes download URL, checksum for verification, and metadata.
type Version struct {
	Version      string   `json:"version"`
	Aliases      []string `json:"aliases,omitempty"` // e.g., ["latest", "lts"]
	Arch         string   `json:"arch"`              // e.g., "amd64"
	URL          string   `json:"url"`
	ChecksumType string   `json:"checksum_type"` // "sha256" or "sha512"
	Checksum     string   `json:"checksum"`
	SizeBytes    int64    `json:"size_bytes"`
	Format       string   `json:"format"` // "qcow2"
}
