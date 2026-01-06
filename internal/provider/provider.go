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
	IP      string
	SSHUser string
	SSHPort int
	VNCPort int
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

	// Prune removes all stopped VMs from the system.
	// Returns the count of VMs deleted.
	Prune() (int, error)

	// Connectivity

	// SSHCommand generates the SSH connection string for a VM.
	// Format: "ssh -p <port> <user>@<ip>"
	SSHCommand(name string) (string, error)

	// Health checks

	// Doctor runs system diagnostics and returns a report of checks performed.
	// Each string in the result contains a check name, status, and details.
	Doctor() []string
}
