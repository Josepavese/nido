package build

import (
	"fmt"
	"runtime"
)

func NativeReleaseAsset() (string, string) {
	return fmt.Sprintf("nido-windows-%s.zip", runtime.GOARCH), "nido.exe"
}
