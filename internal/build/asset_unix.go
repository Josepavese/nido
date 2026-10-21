//go:build !windows

package build

import (
	"fmt"
	"runtime"
)

// NativeReleaseAsset belongs to artifact composition, not command handlers.
func NativeReleaseAsset() (string, string) {
	return fmt.Sprintf("nido-%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH), "nido"
}
