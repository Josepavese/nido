package builder

import "github.com/Josepavese/nido/internal/image"

// SourcesConfig represents the root of sources.yaml
type SourcesConfig struct {
	Sources []Source `yaml:"sources"`
}

// Source represents a provider like Ubuntu or Alpine
type Source struct {
	Name        string     `yaml:"name"`
	Provider    string     `yaml:"provider"`
	Description string     `yaml:"description"`
	Homepage    string     `yaml:"homepage"`
	Strategies  []Strategy `yaml:"strategies"`
}

// Strategy defines how to fetch images for a provider
type Strategy struct {
	Type     string   `yaml:"type"`     // e.g., "ubuntu-cloud", "generic"
	BaseURL  string   `yaml:"base_url"` // Root URL for scanning
	Versions []string `yaml:"versions"` // Versions to scan (e.g. ["24.04", "22.04"])

	// Generic strategy fields
	TemplateURL  string `yaml:"template_url,omitempty"`  // "{base_url}/{version}/file.img"
	ChecksumURL  string `yaml:"checksum_url,omitempty"`  // URL to SUMS file
	ChecksumType string `yaml:"checksum_type,omitempty"` // sha256 or sha512
	Regex        string `yaml:"regex,omitempty"`         // Regex to match filename in directory listing or SUMS file
	Format       string `yaml:"format,omitempty"`        // qcow2 or raw
}

// Fetcher defines the interface for different strategies
type Fetcher interface {
	// Fetch returns a list of versions (with images) found for this strategy
	Fetch(source Source, strategy Strategy) ([]image.Version, error)
}
