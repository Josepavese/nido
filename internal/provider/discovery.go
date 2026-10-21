package provider

import "github.com/Josepavese/nido/internal/hardware"

func (p *QemuProvider) ListAccelerators() ([]hardware.Accelerator, error) {
	return p.Hardware.ListAccelerators()
}

func (p *QemuProvider) Virtualization() hardware.Virtualization { return p.Hardware.Virtualization() }
