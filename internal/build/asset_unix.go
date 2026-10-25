//go:build !windows

package build

import (
	"fmt"
	"github.com/Josepavese/nido/v5/internal/pkg/sysutil"
	"runtime"
)

// NativeReleaseAsset belongs to artifact composition, not command handlers.
func NativeReleaseAsset() (string, string) {
	return fmt.Sprintf("nido-%s-%s.tar.gz", runtime.GOOS, sysutil.HostArchitecture()), "nido"
}
