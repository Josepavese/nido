package provider

import "github.com/Josepavese/nido/internal/config"

// VMStatus represents basic information about a VM.
type VMStatus struct {
	Name    string
	State   string
	PID     int
	SSHPort int
	VNCPort int
	SSHUser string
}

// VMOptions defines parameters for creating/starting a VM.
type VMOptions struct {
	MemoryMB     int
	VCPUs        int
	DiskPath     string
	UserDataPath string
	Gui          bool
	SSHUser      string
}

// VMDetail contains comprehensive data about a VM.
type VMDetail struct {
	Name    string
	State   string
	PID     int
	IP      string
	SSHUser string
	SSHPort int
	VNCPort int
	// DiskPath is the absolute path to the VM disk image.
	DiskPath string
	// DiskMissing indicates the disk file is missing on disk.
	DiskMissing bool
	// BackingPath is the backing file path if the disk uses one.
	BackingPath string
	// BackingMissing indicates the backing file is missing on disk.
	BackingMissing bool
}

// CachedImage represents a cached cloud image.
type CachedImage struct {
	Name    string
	Version string
	Size    string
}

// CacheInfoResult contains cache statistics.
type CacheInfoResult struct {
	Count     int
	TotalSize string
}

// VMProvider defines the contract for OS-specific hypervisor management.
// Implementations handle VM lifecycle, storage, and connectivity operations.
type VMProvider interface {
	// Lifecycle operations

	// Spawn creates a new VM from a template and starts it.
	// If opts.DiskPath is empty, uses the default template from config.
	Spawn(name string, opts VMOptions) error

	// Start boots up a stopped VM. Returns nil if already running.
	// If gui is true, enables the graphical interface.
	Start(name string, opts VMOptions) error

	// Stop halts a running VM. If graceful is true, sends ACPI shutdown signal.
	Stop(name string, graceful bool) error

	// Delete permanently removes a VM and its disk image.
	Delete(name string) error

	// Information queries

	// List returns status of all VMs (running and stopped).
	List() ([]VMStatus, error)

	// Info retrieves detailed information about a specific VM.
	Info(name string) (VMDetail, error)

	// GetConfig returns the current provider configuration.
	GetConfig() config.Config

	// Storage operations

	// CreateDisk creates a new qcow2 disk image, optionally from a template.
	CreateDisk(name string, size string, templatePath string) error

	// CreateTemplate archives a VM into a compressed template for reuse.
	// Returns the path to the created template file.
	CreateTemplate(vmName string, templateName string) (string, error)

	// ListTemplates returns names of all available templates in cold storage.
	ListTemplates() ([]string, error)

	// ListImages returns names/tags of all available cloud images in cache.
	ListImages() ([]string, error)

	// GetUsedBackingFiles identifies all backing files currently in use by VMs.
	GetUsedBackingFiles() ([]string, error)

	// DeleteTemplate removes a template from cold storage.
	DeleteTemplate(name string) error

	// Prune removes all stopped VMs from the system.
	// Returns the count of VMs deleted.
	Prune() (int, error)

	// Cache operations

	// ListCachedImages returns all cached cloud images.
	ListCachedImages() ([]CachedImage, error)

	// CacheInfo returns statistics about the image cache.
	CacheInfo() (CacheInfoResult, error)

	// CachePrune removes cached images. If unusedOnly is true, only removes
	// images not used by any VM.
	CachePrune(unusedOnly bool) error

	// Connectivity

	// SSHCommand generates the SSH connection string for a VM.
	// Format: "ssh -p <port> <user>@<ip>"
	SSHCommand(name string) (string, error)

	// Health checks

	// Doctor runs system diagnostics and returns a report of checks performed.
	// Each string in the result contains a check name, status, and details.
	Doctor() []string
}
