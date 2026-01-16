package ops

// RequestSpawnMsg requests a VM spawn.
type RequestSpawnMsg struct {
	Name     string
	Source   string
	IsFile   bool
	UserData string
	GUI      bool
}

// RequestCreateTemplateMsg requests a template creation.
type RequestCreateTemplateMsg struct {
	Name   string
	Source string
}

// RequestDeleteTemplateMsg requests a template deletion.
type RequestDeleteTemplateMsg struct {
	Name string
}

// RequestTemplateListMsg requests the list of templates.
type RequestTemplateListMsg struct{}

// --- Registry Messages ---

// RegistryImage represents a remote image available for download.
type RegistryImage struct {
	Name        string
	Version     string
	Registry    string // "nido" or "official"
	Description string
	Size        string // Estimated if available
}

// RegistryListMsg contains the list of available remote images.
type RegistryListMsg struct {
	Images []RegistryImage
	Err    error
}

// RequestPullMsg requests an image pull.
type RequestPullMsg struct {
	Image string // "ubuntu:24.04"
}

// RequestDeleteImageMsg requests deletion of a cached image.
type RequestDeleteImageMsg struct {
	Name    string
	Version string
}

// RequestPruneMsg requests pruning of unused images.
type RequestPruneMsg struct {
	UnusedOnly bool
}
