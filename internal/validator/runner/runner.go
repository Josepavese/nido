package runner

import (
	"fmt"
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

	cmd := exec.Command(inv.Command, inv.Args...)
	cmd.Dir = inv.Workdir
	prepareCommand(cmd)

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

	stdoutFile, err := os.CreateTemp("", "nido-runner-stdout-*")
	if err != nil {
		return Result{Stderr: err.Error(), ExitCode: -1, StartTime: time.Now()}
	}
	defer os.Remove(stdoutFile.Name())
	stderrFile, err := os.CreateTemp("", "nido-runner-stderr-*")
	if err != nil {
		stdoutFile.Close()
		return Result{Stderr: err.Error(), ExitCode: -1, StartTime: time.Now()}
	}
	defer os.Remove(stderrFile.Name())
	cmd.Stdout = stdoutFile
	cmd.Stderr = stderrFile

	start := time.Now()
	err = cmd.Start()
	if err != nil {
		stdoutText, stderrText := readCommandOutput(stdoutFile, stderrFile)
		return Result{
			Stdout:    stdoutText,
			Stderr:    strings.TrimSpace(stderrText + "\n" + err.Error()),
			ExitCode:  -1,
			Duration:  time.Since(start),
			StartTime: start,
		}
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	timedOut := false
	select {
	case err = <-done:
	case <-time.After(timeout):
		timedOut = true
		terminateProcessTree(cmd.Process)
		select {
		case err = <-done:
		case <-time.After(5 * time.Second):
			err = fmt.Errorf("process did not exit after timeout")
		}
	}
	duration := time.Since(start)
	stdoutText, stderrText := readCommandOutput(stdoutFile, stderrFile)

	res := Result{
		Stdout:    stdoutText,
		Stderr:    stderrText,
		Duration:  duration,
		StartTime: start,
	}

	if timedOut {
		res.TimedOut = true
		res.ExitCode = -1
		timeoutMessage := fmt.Sprintf("command timed out after %s", timeout)
		if strings.TrimSpace(res.Stderr) == "" {
			res.Stderr = timeoutMessage
		} else {
			res.Stderr = strings.TrimRight(res.Stderr, "\r\n") + "\n" + timeoutMessage
		}
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

func readCommandOutput(stdoutFile, stderrFile *os.File) (string, string) {
	stdoutPath := stdoutFile.Name()
	stderrPath := stderrFile.Name()
	_ = stdoutFile.Close()
	_ = stderrFile.Close()
	stdout, _ := os.ReadFile(stdoutPath)
	stderr, _ := os.ReadFile(stderrPath)
	return string(stdout), string(stderr)
}
