package sysutil

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Josepavese/nido/v4/internal/machine"
)

const architectureProbeTimeout = 3 * time.Second

// HostArchitecture is the host ISA, which can differ from the executable ISA
// under Rosetta or Windows application emulation. Discovery is process-lived.
var HostArchitecture = sync.OnceValue(func() string {
	ctx, cancel := context.WithTimeout(context.Background(), architectureProbeTimeout)
	defer cancel()
	observed := runtime.GOARCH
	if runtime.GOOS == "windows" {
		if native := os.Getenv("PROCESSOR_ARCHITEW6432"); native != "" {
			observed = native
		} else if native := os.Getenv("PROCESSOR_ARCHITECTURE"); native != "" {
			observed = native
		}
	} else {
		if out, err := exec.CommandContext(ctx, "uname", "-m").Output(); err == nil {
			observed = strings.TrimSpace(string(out))
		}
		if runtime.GOOS == "darwin" {
			if out, err := exec.CommandContext(ctx, "sysctl", "-n", "hw.optional.arm64").Output(); err == nil && strings.TrimSpace(string(out)) == "1" {
				observed = machine.ARM64
			}
		}
	}
	arch, _ := machine.Normalize(observed)
	return arch
})

func (NativeHostIO) Architecture() string { return HostArchitecture() }
