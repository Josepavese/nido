package build

import (
	"fmt"
	"github.com/Josepavese/nido/v5/internal/pkg/sysutil"
)

func NativeReleaseAsset() (string, string) {
	return fmt.Sprintf("nido-windows-%s.zip", sysutil.HostArchitecture()), "nido.exe"
}
