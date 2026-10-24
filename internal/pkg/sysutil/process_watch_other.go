//go:build !linux && !darwin && !windows

package sysutil

import (
	"context"
	"fmt"
)

func WaitProcessExit(context.Context, int) error {
	return fmt.Errorf("native process exit observation is unsupported on this host")
}
