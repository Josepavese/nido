package hardware

import (
	"context"
	"github.com/Josepavese/nido/v4/internal/machine"
	"strings"
	"time"
)

const hypervisorProbeTimeout = 3 * time.Second

func (h *nativeHost) Virtualization(guest string) Virtualization {
	if guest != h.io.Architecture() {
		return SoftwareVirtualization(guest)
	}
	ctx, cancel := context.WithTimeout(context.Background(), hypervisorProbeTimeout)
	defer cancel()
	out, err := h.io.Run(ctx, "sysctl", "-n", "kern.hv_support")
	if err == nil && strings.TrimSpace(string(out)) == "1" {
		return Virtualization{Accelerator: "hvf", CPU: machine.CPUHost}
	}
	return SoftwareVirtualization(guest)
}
