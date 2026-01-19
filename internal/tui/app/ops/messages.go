package ops

import (
	"github.com/Josepavese/nido/internal/provider"
)

// RequestSpawnMsg requests a VM spawn.
type RequestSpawnMsg struct {
	Name     string
	Source   string
	IsFile   bool
	UserData string
	GUI      bool
	Ports    []provider.PortForward
}

// RequestCreateTemplateMsg requests a template creation.
type RequestCreateTemplateMsg struct {
	Name   string
	Source string
}

// RequestDeleteTemplateMsg requests a template deletion.
type RequestDeleteTemplateMsg struct {
	Name  string
	Force bool
}

// RequestTemplateListMsg requests the list of templates.
type RequestTemplateListMsg struct{}

// TemplateUsageMsg contains the result of a dependency check.
type TemplateUsageMsg struct {
	Name   string
	InUse  bool
	UsedBy []string
	Err    error
}

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

// RequestUpdateMsg requests a version check.
type RequestUpdateMsg struct {
	Manual bool
}

// RequestCacheMsg requests cache statistics.
type RequestCacheMsg struct{}

// RequestApplyUpdateMsg requests an actual binary update.
type RequestApplyUpdateMsg struct{}
