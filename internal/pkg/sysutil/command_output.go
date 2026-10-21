package sysutil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
)

// ReadCommandOutput owns the child until Wait completes, including cancellation
// when the consumer fails. Consumers receive bytes, never a native process handle.
func ReadCommandOutput(ctx context.Context, name string, args []string, consume func(io.Reader) error) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		pipe.Close()
		return err
	}
	readErr := consume(pipe)
	if readErr != nil {
		cancel()
	} else {
		_, readErr = io.Copy(io.Discard, pipe)
	}
	waitErr := cmd.Wait()
	if waitErr != nil {
		waitErr = fmt.Errorf("%s failed: %w", name, waitErr)
	}
	return errors.Join(readErr, waitErr)
}
