package runner

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Invocation describes a single CLI execution.
type Invocation struct {
	Command string
	Args    []string
	Timeout time.Duration
	Env     map[string]string
	Workdir string
	Stdin   string
}

// Result captures the outcome of a CLI invocation.
type Result struct {
	// ...
	Stdout    string
	Stderr    string
	ExitCode  int
	Duration  time.Duration
	TimedOut  bool
	StartTime time.Time
}

// Runner executes commands with optional timeout and env overrides.
type Runner struct {
	defaultEnv []string
}

// New creates a Runner preserving current environment as baseline.
func New() Runner {
	return Runner{
		defaultEnv: os.Environ(),
	}
}

// Exec executes a command and captures output.
func (r Runner) Exec(inv Invocation) Result {
	timeout := inv.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, inv.Command, inv.Args...)
	cmd.Dir = inv.Workdir

	// Wire up Stdin if present
	if inv.Stdin != "" {
		cmd.Stdin = strings.NewReader(inv.Stdin)
	}

	// Build env slice
	env := append([]string{}, r.defaultEnv...)
	for k, v := range inv.Env {
		env = append(env, k+"="+v)
	}
	cmd.Env = env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	res := Result{
		Stdout:    stdout.String(),
		Stderr:    stderr.String(),
		Duration:  duration,
		StartTime: start,
	}

	if ctx.Err() == context.DeadlineExceeded {
		res.TimedOut = true
		res.ExitCode = -1
		return res
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			res.ExitCode = exitErr.ExitCode()
		} else {
			res.ExitCode = -1
		}
		return res
	}

	res.ExitCode = 0
	return res
}
