package provider

import (
	"github.com/Josepavese/nido/v4/internal/hardware"
)

func (p *QemuProvider) ListAccelerators() ([]hardware.Accelerator, error) {
	return p.Hardware.ListAccelerators()
}

func (p *QemuProvider) Virtualization(guest string) hardware.Virtualization {
	return p.Hardware.Virtualization(guest)
}
