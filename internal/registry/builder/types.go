package builder

import "github.com/Josepavese/nido/v4/internal/image"

// SourcesConfig represents the root of sources.yaml
type SourcesConfig struct {
	Sources []Source `yaml:"sources"`
}

// Source represents a provider like Ubuntu or Alpine
type Source struct {
	Name        string     `yaml:"name"`
	Registry    string     `yaml:"registry,omitempty"` // official, nido, etc.
	Provider    string     `yaml:"provider"`
	Description string     `yaml:"description"`
	Optional    bool       `yaml:"optional,omitempty"` // Sources without published assets may be absent.
	Homepage    string     `yaml:"homepage"`
	SSHUser     string     `yaml:"ssh_user,omitempty"`
	Strategies  []Strategy `yaml:"strategies"`
}

// Strategy defines how to fetch images for a provider
type Strategy struct {
	Architectures []string `yaml:"architectures"`
	architecture  string
	Type          string   `yaml:"type"`               // e.g., "ubuntu-cloud", "generic", "github-release"
	BaseURL       string   `yaml:"base_url"`           // Root URL for scanning
	Versions      []string `yaml:"versions,omitempty"` // Versions to scan (e.g. ["24.04", "22.04"])

	Discovery *Discovery `yaml:"discovery,omitempty"`

	// Generic strategy fields
	TemplateURL  string `yaml:"template_url,omitempty"`  // "{base_url}/{version}/file.img"
	ChecksumURL  string `yaml:"checksum_url,omitempty"`  // URL to SUMS file
	ChecksumType string `yaml:"checksum_type,omitempty"` // sha256 or sha512
	Regex        string `yaml:"regex,omitempty"`         // Regex to match filename in directory listing or SUMS file
	Format       string `yaml:"format,omitempty"`        // qcow2 or raw

	// GitHub Release strategy fields
	Owner string `yaml:"owner,omitempty"`
	Repo  string `yaml:"repo,omitempty"`
	Tag   string `yaml:"tag,omitempty"` // If empty, scans all releases
}

// Fetcher defines the interface for different strategies
type Fetcher interface {
	// Fetch returns a list of versions (with images) found for this strategy
	Fetch(source Source, strategy Strategy) ([]image.Version, error)
}

// Discovery selects versions from upstream metadata, rather than a maintained allowlist.
// URL and policy are owned by sources.yaml; provider parsers own their wire formats.
type Discovery struct {
	URL      string   `yaml:"url"`
	Channels []string `yaml:"channels,omitempty"` // Debian stable/oldstable Release files.
	Pattern  string   `yaml:"pattern,omitempty"`  // Generic directory: first capture is a stable version.
	Limit    int      `yaml:"limit,omitempty"`    // Generic directory: keep this many newest stable versions.
}
