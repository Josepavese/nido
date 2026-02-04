package image

// Blueprint defines the Single Source of Truth "recipe" for building an image locally.
// Unlike Catalog/Image entries which point to static files, Blueprints describe a process.
type Blueprint struct {
	// Metadata
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description" json:"description"`
	Version     string `yaml:"version" json:"version"` // Blueprint version, not OS version

	// Source Media
	ISOURL      string `yaml:"iso_url" json:"iso_url"`
	ISOChecksum string `yaml:"iso_checksum" json:"iso_checksum"` // Optional but recommended

	// Drivers & Tools (e.g., VirtIO for Windows)
	Drivers []Resource `yaml:"drivers" json:"drivers"`

	// Automation (e.g., autounattend.xml)
	// The key is the destination filename in the root of the build ISO/Floppy
	Scripts map[string]string `yaml:"scripts" json:"scripts"`

	// Machine Specs needed for the build process itself
	BuildSpecs BuildSpecs `yaml:"build_specs" json:"build_specs"`

	// Output
	OutputImage string `yaml:"output_image" json:"output_image"` // e.g. "windows-11-eval.qcow2"
	OutputSize  string `yaml:"output_size" json:"output_size"`   // e.g. "64G"
}

// Resource represents an external file needed for the build
type Resource struct {
	Name     string `yaml:"name" json:"name"`
	URL      string `yaml:"url" json:"url"`
	Checksum string `yaml:"checksum" json:"checksum"`
}

// BuildSpecs defines VM requirements for the installation phase
type BuildSpecs struct {
	CPU     int    `yaml:"cpu" json:"cpu"`
	Memory  string `yaml:"memory" json:"memory"`   // e.g. "4G"
	Disk    string `yaml:"disk" json:"disk"`       // e.g. "64G"
	Timeout string `yaml:"timeout" json:"timeout"` // Safety timeout
}
