package sysutil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// Command is a portable, owned child process. Wait is required after Start,
// including after Kill. It also satisfies interactive terminal executors.
type Command interface {
	Run() error
	Output() ([]byte, error)
	CombinedOutput() ([]byte, error)
	Start() error
	Wait() error
	Kill() error
	PID() int
	SetStdin(io.Reader)
	SetStdout(io.Writer)
	SetStderr(io.Writer)
}

// NewCommandWithEnvironment overrides named environment entries for one owned
// child without changing the caller's process-wide environment.
func NewCommandWithEnvironment(ctx context.Context, environment map[string]string, name string, args ...string) Command {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = os.Environ()
	for key, value := range environment {
		// os/exec keeps the last occurrence, including Windows key semantics.
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	return &nativeCommand{command: cmd}
}

type CommandError struct {
	Code  int
	cause error
}

func (e *CommandError) Error() string { return e.cause.Error() }
func (e *CommandError) Unwrap() error { return e.cause }
func ExitCode(err error) int {
	var e *CommandError
	if errors.As(err, &e) {
		return e.Code
	}
	return 1
}
func commandError(err error) error {
	if err == nil {
		return nil
	}
	code := 1
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		code = exit.ExitCode()
	}
	return &CommandError{Code: code, cause: err}
}

type nativeCommand struct{ command *exec.Cmd }

func NewCommand(name string, args ...string) Command {
	return NewCommandContext(context.Background(), name, args...)
}
func NewCommandContext(ctx context.Context, name string, args ...string) Command {
	return &nativeCommand{command: exec.CommandContext(ctx, name, args...)}
}
func (c *nativeCommand) Run() error { return commandError(c.command.Run()) }
func (c *nativeCommand) Output() ([]byte, error) {
	b, e := c.command.Output()
	return b, commandError(e)
}
func (c *nativeCommand) CombinedOutput() ([]byte, error) {
	b, e := c.command.CombinedOutput()
	return b, commandError(e)
}
func (c *nativeCommand) Start() error { return commandError(c.command.Start()) }
func (c *nativeCommand) Wait() error  { return commandError(c.command.Wait()) }
func (c *nativeCommand) Kill() error {
	if c.command.Process == nil {
		return fmt.Errorf("command has not started")
	}
	return commandError(c.command.Process.Kill())
}
func (c *nativeCommand) PID() int {
	if c.command.Process == nil {
		return 0
	}
	return c.command.Process.Pid
}
func (c *nativeCommand) SetStdin(r io.Reader)  { c.command.Stdin = r }
func (c *nativeCommand) SetStdout(w io.Writer) { c.command.Stdout = w }
func (c *nativeCommand) SetStderr(w io.Writer) { c.command.Stderr = w }
func RunSelf(args ...string) ([]byte, error) {
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return NewCommand(executable, args...).CombinedOutput()
}
