package provider

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Josepavese/nido/v4/internal/events"
	"github.com/Josepavese/nido/v4/internal/pkg/identifier"
)

// Shutdown defaults and bounds are owned here so CLI, MCP and TUI consumers
// cannot invent different guest-poweroff budgets.
const (
	DefaultShutdownTimeout = 60 * time.Second
	MaxShutdownTimeout     = 5 * time.Minute

	shutdownPollInterval   = 200 * time.Millisecond
	shutdownDetectTimeout  = 10 * time.Second
	shutdownRequestTimeout = 15 * time.Second
	shutdownDetailLimit    = 256
	shutdownAlreadyStopped = "already_stopped"
	shutdownModeGuest      = "guest"
	shutdownPOSIXCommand   = "sync; if [ \"$(id -u)\" = 0 ]; then prefix=; elif command -v sudo >/dev/null 2>&1; then prefix=\"sudo -n\"; else prefix=; fi; if command -v shutdown >/dev/null 2>&1; then $prefix shutdown -h now; else $prefix poweroff; fi"
	shutdownWindowsCommand = "shutdown /s /t 0"
	shutdownPOSIXDetection = "uname -s"
)

// ShutdownResult describes a guest-requested poweroff without implying that a
// forced process stop occurred. Mode is "guest" when the guest completed the
// request and "already_stopped" when there was no running process to ask.
// Method records the request channel: "ssh" or "acpi".
type ShutdownResult struct {
	Name       string `json:"name"`
	Mode       string `json:"mode"`
	Method     string `json:"method,omitempty"`
	PID        int    `json:"pid,omitempty"`
	DurationMS int64  `json:"duration_ms"`
}

type shutdownDeps struct {
	findSSH func() (string, error)
	runSSH  func(ctx context.Context, argv []string, timeout time.Duration) (stdout, stderr []byte, err error)
	isAlive func(name string, pid int) bool
	now     func() time.Time
	sleep   func(ctx context.Context, d time.Duration) error
}

type guestShutdownTarget struct {
	name    string
	pid     int
	sshPort int
	sshUser string
}

func defaultShutdownDeps(p *QemuProvider) shutdownDeps {
	// readinessDeps and shutdownDeps intentionally share the same guest
	// observation functions; convert instead of copying every field.
	return shutdownDeps(defaultReadinessDeps(p))
}

func normalizeShutdownTimeout(timeout time.Duration) (time.Duration, error) {
	if timeout <= 0 {
		return DefaultShutdownTimeout, nil
	}
	if timeout > MaxShutdownTimeout {
		return 0, fmt.Errorf("shutdown timeout must not exceed %s", MaxShutdownTimeout)
	}
	return timeout, nil
}

// Shutdown requests a guest OS poweroff over SSH and waits only for the owned
// QEMU process to exit. It intentionally has no force fallback: callers that
// need termination must call Stop explicitly, so a guest that cannot complete
// shutdown is never reported as a clean one.
func (p *QemuProvider) Shutdown(ctx context.Context, name string, timeout time.Duration) (ShutdownResult, error) {
	if err := identifier.Validate(name); err != nil {
		return ShutdownResult{}, err
	}
	timeout, err := normalizeShutdownTimeout(timeout)
	if err != nil {
		return ShutdownResult{}, err
	}
	if ctx == nil {
		ctx = context.Background()
	}

	finish := p.operation(events.VMStop, name)
	var resultErr error
	defer func() { finish(resultErr) }()

	state, err := p.loadState(name)
	if err != nil {
		resultErr = fmt.Errorf("VM %q: %w", name, err)
		return ShutdownResult{}, resultErr
	}

	deps := defaultShutdownDeps(p)
	start := deps.now()
	result := ShutdownResult{Name: name}
	pid := p.runningPID(name)
	if pid > 0 && processAlive(pid) && !deps.isAlive(name, pid) {
		resultErr = fmt.Errorf("PID %d no longer belongs to VM %q", pid, name)
		return result, resultErr
	}
	if pid == 0 || !processAlive(pid) {
		if err := p.releaseStoppedVM(name); err != nil {
			resultErr = err
			return result, err
		}
		result.Mode = shutdownAlreadyStopped
		result.PID = pid
		result.DurationMS = deps.now().Sub(start).Milliseconds()
		return result, nil
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	target := guestShutdownTarget{
		name:    name,
		pid:     pid,
		sshPort: state.SSHPort,
		sshUser: state.SSHUser,
	}
	method, err := requestGuestShutdownWithFallback(waitCtx, target, deps, func(ctx context.Context) error {
		return requestQEMUPowerdown(ctx, p.qmpControlPath(name))
	})
	if err != nil {
		resultErr = err
		return result, err
	}
	if err := p.releaseStoppedVM(name); err != nil {
		resultErr = err
		return result, err
	}

	result.Mode = shutdownModeGuest
	result.Method = method
	result.PID = pid
	result.DurationMS = deps.now().Sub(start).Milliseconds()
	return result, nil
}

// requestGuestShutdownWithFallback keeps the SSH path first, then uses the
// provider-owned QMP control endpoint for an ACPI poweroff when SSH could not
// even deliver the request. A timeout after delivery never falls back: the
// caller must still see a running VM and choose an explicit stop.
func requestGuestShutdownWithFallback(ctx context.Context, target guestShutdownTarget, deps shutdownDeps, qmpPowerdown func(context.Context) error) (string, error) {
	if err := requestGuestShutdown(ctx, target, deps); err == nil {
		return "ssh", nil
	} else {
		if ctx.Err() != nil {
			return "", err
		}
		if qmpErr := qmpPowerdown(ctx); qmpErr != nil {
			return "", fmt.Errorf("%v; ACPI/QMP fallback unavailable: %v", err, qmpErr)
		}
		if waitErr := waitVMExit(ctx, target, deps); waitErr != nil {
			return "", fmt.Errorf("ACPI/QMP powerdown requested but %v", waitErr)
		}
		return "acpi", nil
	}
}

// qmpControlPath returns the provider-owned control socket created by QEMU at
// start. It is empty on hosts whose backend has no controllable QMP endpoint.
func (p *QemuProvider) qmpControlPath(name string) string {
	return qemuQMPPath(filepath.Join(p.RootDir, "run"), name)
}

// runningPID reads the same owned PID artifact Stop uses, falling back to the
// persisted state for recovery after a partially cleaned run directory.
func (p *QemuProvider) runningPID(name string) int {
	pid := 0
	if data, err := os.ReadFile(filepath.Join(p.RootDir, "run", name+".pid")); err == nil {
		fmt.Sscanf(string(data), "%d", &pid)
	}
	if pid == 0 {
		if state, err := p.loadState(name); err == nil {
			pid = state.PID
		}
	}
	return pid
}

func requestGuestShutdown(ctx context.Context, target guestShutdownTarget, deps shutdownDeps) error {
	if target.sshPort <= 0 {
		return fmt.Errorf("VM %q has no SSH host port; guest shutdown is unavailable (use stop to terminate)", target.name)
	}
	sshPath, err := deps.findSSH()
	if err != nil {
		return fmt.Errorf("guest shutdown needs an SSH client: %w", err)
	}
	base := readinessSSHArgs(sshClientArgs(target.sshPort, target.sshUser, "127.0.0.1"))
	base[0] = sshPath

	command, err := guestShutdownCommand(ctx, base, deps)
	if err != nil {
		return err
	}
	_, stderr, requestErr := deps.runSSH(ctx, appendProbeCommand(base, command), shutdownRequestTimeout)
	if ctx.Err() != nil {
		return shutdownContextError(ctx)
	}
	if requestErr != nil {
		// Authentication failure proves the request did not reach the guest.
		// A transport failure before SSH could deliver the command is also an
		// undelivered request; the caller may then use the provider-owned ACPI
		// fallback. Other SSH errors may mean the connection closed as the guest
		// began poweroff, so the process-exit wait remains authoritative.
		switch classifySSHAttempt(requestErr, stderr) {
		case FailureSSHAuth, FailureSSHClientMissing:
			return fmt.Errorf("guest shutdown request: %s", boundedReadinessDetail(string(stderr)))
		}
		if shutdownTransportFailure(string(stderr)) {
			return fmt.Errorf("guest shutdown request: %s", boundedReadinessDetail(string(stderr)))
		}
	}

	if err := waitVMExit(ctx, target, deps); err != nil {
		detail := boundedShutdownDetail(string(stderr))
		if detail != "" {
			return fmt.Errorf("%s: %s", err, detail)
		}
		return err
	}
	return nil
}

// shutdownTransportFailure reports SSH failures that happened before a command
// could be delivered. These are safe to retry through the local ACPI request;
// a delivered command that later times out is not.
func shutdownTransportFailure(stderr string) bool {
	text := strings.ToLower(stderr)
	for _, marker := range []string{
		"connection closed", "connection refused", "connection reset",
		"connection aborted", "connection timed out", "no route to host",
		"network is unreachable", "broken pipe", "kex_exchange_identification",
	} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func guestShutdownCommand(ctx context.Context, base []string, deps shutdownDeps) (string, error) {
	detectCtx, cancel := context.WithTimeout(ctx, shutdownDetectTimeout)
	defer cancel()
	stdout, _, err := deps.runSSH(detectCtx, appendProbeCommand(base, shutdownPOSIXDetection), shutdownDetectTimeout)
	if ctx.Err() != nil {
		return "", shutdownContextError(ctx)
	}
	if err == nil && isPOSIXGuest(string(stdout)) {
		return shutdownPOSIXCommand, nil
	}
	return shutdownWindowsCommand, nil
}

func isPOSIXGuest(output string) bool {
	fields := strings.Fields(strings.ToLower(output))
	if len(fields) == 0 {
		return false
	}
	switch {
	case strings.Contains(fields[0], "linux"),
		strings.Contains(fields[0], "darwin"),
		strings.Contains(fields[0], "bsd"),
		strings.Contains(fields[0], "unix"):
		return true
	}
	return false
}

func waitVMExit(ctx context.Context, target guestShutdownTarget, deps shutdownDeps) error {
	ticker := time.NewTicker(shutdownPollInterval)
	defer ticker.Stop()
	for {
		if !deps.isAlive(target.name, target.pid) {
			return nil
		}
		select {
		case <-ctx.Done():
			return shutdownContextError(ctx)
		case <-ticker.C:
		}
	}
}

func shutdownContextError(ctx context.Context) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("guest did not power off before the shutdown timeout")
	}
	return fmt.Errorf("guest shutdown cancelled: %w", ctx.Err())
}

func boundedShutdownDetail(text string) string {
	text = boundedReadinessDetail(text)
	if len(text) > shutdownDetailLimit {
		text = text[:shutdownDetailLimit]
	}
	return text
}
