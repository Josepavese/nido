package sysutil

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestEventProcessHelper(t *testing.T) {
	if os.Getenv("NIDO_PROCESS_WATCH_HELPER") != "1" {
		return
	}
	io.Copy(io.Discard, os.Stdin)
	os.Exit(0)
}
func TestNativeProcessExitAndCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestEventProcessHelper$")
	command.Env = append(os.Environ(), "NIDO_PROCESS_WATCH_HELPER=1")
	input, err := command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- WaitProcessExit(ctx, command.Process.Pid) }()
	input.Close()
	if err := command.Wait(); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	stopped, stop := context.WithCancel(context.Background())
	cancelDone := make(chan error, 1)
	go func() { cancelDone <- WaitProcessExit(stopped, os.Getpid()) }()
	stop()
	select {
	case err := <-cancelDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("process observer did not cancel")
	}
}
