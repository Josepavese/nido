package hardware

import (
	"context"
	"strings"
	"time"
)

const hypervisorProbeTimeout = 3 * time.Second

func (h *nativeHost) Virtualization() Virtualization {
	ctx, cancel := context.WithTimeout(context.Background(), hypervisorProbeTimeout)
	defer cancel()
	out, err := h.io.Run(ctx, "sysctl", "-n", "kern.hv_support")
	if err == nil && strings.TrimSpace(string(out)) == "1" {
		return Virtualization{Accelerator: "hvf", CPU: "host"}
	}
	return Virtualization{CPU: "qemu64"}
}
