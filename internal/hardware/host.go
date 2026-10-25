// Package hardware owns host virtualization and PCI device semantics. Business
// consumers receive capabilities and the Host contract, never native handles.
package hardware

import (
	"context"
	"errors"
	"github.com/Josepavese/nido/v4/internal/machine"
	"github.com/Josepavese/nido/v4/internal/pkg/sysutil"
)

var ErrUnsupported = errors.New("hardware capability unsupported on this host")

// Virtualization reports the available accelerator and CPU model. An empty
// accelerator means software emulation; it never claims hardware acceleration.
type Virtualization struct {
	Accelerator string `json:"accelerator"`
	CPU         string `json:"cpu"`
}

func (v Virtualization) Accelerated() bool {
	return v.Accelerator != "" && v.Accelerator != machine.AcceleratorTCG
}

func (v Virtualization) QEMUArgs() []string {
	if v.Accelerator == "" {
		return nil
	}
	return []string{"-accel", v.Accelerator}
}

type Host interface {
	Virtualization(guest string) Virtualization
	ListAccelerators() ([]Accelerator, error)
	PreparePassthrough(context.Context, string, string, string) error
	RestorePassthrough(context.Context, string, string, string) error
}

type nativeHost struct{ io sysutil.HostIO }

// New is the native composition factory. The selected backend is build-specific.
func New() Host { return &nativeHost{io: sysutil.NativeHostIO{}} }

// Accelerator represents a PCI device candidate for passthrough.
type Accelerator struct {
	ID         string // PCI ID (e.g., "0000:00:1f.3")
	Vendor     string // Vendor Name/ID
	Device     string // Device Name/ID
	Class      string // Device Class (e.g., "VGA compatible controller")
	IOMMUGroup string // IOMMU Group ID
	IsIsolated bool   // True if this is the only device in the group (or others are bridges)
	IsSafe     bool   // True if Isolated AND not the primary GPU
	Warning    string // Reason why it might not be safe
}

// SoftwareVirtualization selects a CPU model belonging to the guest ISA.
func SoftwareVirtualization(guest string) Virtualization {
	profile, err := machine.For(guest)
	if err != nil {
		return Virtualization{}
	}
	return Virtualization{Accelerator: machine.AcceleratorTCG, CPU: profile.SoftwareCPU}
}
