package sysutil

import (
	"context"
	"fmt"
	"io"
	"time"
)

// RunTimedCommand owns cancellation and reaping before callers retire files.
func RunTimedCommand(name string, args []string, timeout time.Duration, output io.Writer) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	command := NewCommandContext(ctx, name, args...)
	if output != nil {
		command.SetStdout(output)
		command.SetStderr(output)
	}
	start := time.Now()
	err := command.Run()
	elapsed := time.Since(start)
	if ctx.Err() != nil {
		return elapsed, fmt.Errorf("command timed out after %s: %w", timeout, ctx.Err())
	}
	return elapsed, err
}
