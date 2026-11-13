package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Josepavese/nido/v4/internal/pkg/identifier"
	"github.com/Josepavese/nido/v4/internal/pkg/sysutil"
)

// Readiness defaults and bounds are owned here so every consumer uses the same
// observation interval and attempt budget.
const (
	DefaultReadyTimeout = 180 * time.Second
	MaxReadyTimeout     = 15 * time.Minute

	readinessPollInterval   = 2 * time.Second
	readinessAttemptTimeout = 8 * time.Second
	readinessDetailLimit    = 512
)

// ReadinessState is the last observed readiness outcome.
type ReadinessState string

const (
	ReadinessPending ReadinessState = "pending"
	ReadinessReady   ReadinessState = "ready"
	ReadinessFailed  ReadinessState = "failed"
)

// ReadinessPhase is the furthest lifecycle stage reached by an observation.
type ReadinessPhase string

const (
	ReadinessPhaseProcess      ReadinessPhase = "process"
	ReadinessPhaseSSH          ReadinessPhase = "ssh"
	ReadinessPhaseProvisioning ReadinessPhase = "provisioning"
	ReadinessPhaseReady        ReadinessPhase = "ready"
)

// ReadinessFailure classifies why a readiness observation did not become ready.
type ReadinessFailure string

const (
	FailureVMNotRunning     ReadinessFailure = "vm_not_running"
	FailureProcessExited    ReadinessFailure = "process_exited"
	FailureSSHClientMissing ReadinessFailure = "ssh_client_missing"
	FailureSSHUnavailable   ReadinessFailure = "ssh_unavailable"
	FailureSSHAuth          ReadinessFailure = "ssh_auth_failed"
	FailureProvisioning     ReadinessFailure = "provisioning_error"
	FailureTimeout          ReadinessFailure = "timeout"
	FailureCancelled        ReadinessFailure = "cancelled"
)

// ProvisioningState summarizes optional guest provisioning (cloud-init).
type ProvisioningState string

const (
	ProvisioningAbsent   ProvisioningState = "absent"
	ProvisioningRunning  ProvisioningState = "running"
	ProvisioningDone     ProvisioningState = "done"
	ProvisioningDegraded ProvisioningState = "degraded"
	ProvisioningError    ProvisioningState = "error"
)

// GuestReadiness is an observed, timestamped answer to "can an agent run a
// command in this guest now?". It is advisory metadata, not an event and not a
// property of the VM process: the caller must still handle a later failure.
type GuestReadiness struct {
	Ready        bool              `json:"ready"`
	State        ReadinessState    `json:"state"`
	Phase        ReadinessPhase    `json:"phase"`
	PID          int               `json:"pid,omitempty"`
	Attempts     int               `json:"attempts,omitempty"`
	Provisioning ProvisioningState `json:"provisioning,omitempty"`
	Failure      ReadinessFailure  `json:"failure,omitempty"`
	Detail       string            `json:"detail,omitempty"`
	CheckedAt    time.Time         `json:"checked_at"`
	DurationMS   int64             `json:"duration_ms"`
}

type readinessDeps struct {
	findSSH func() (string, error)
	runSSH  func(ctx context.Context, argv []string, timeout time.Duration) (stdout, stderr []byte, err error)
	isAlive func(name string, pid int) bool
	now     func() time.Time
	sleep   func(ctx context.Context, d time.Duration) error
}

func defaultReadinessDeps(p *QemuProvider) readinessDeps {
	return readinessDeps{
		findSSH: func() (string, error) { return sysutil.FindExecutable("ssh") },
		runSSH:  runReadinessSSH,
		isAlive: func(name string, pid int) bool { return p.isVMProcess(name, pid) },
		now:     time.Now,
		sleep: func(ctx context.Context, d time.Duration) error {
			timer := time.NewTimer(d)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			}
		},
	}
}

func runReadinessSSH(ctx context.Context, argv []string, timeout time.Duration) ([]byte, []byte, error) {
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := sysutil.NewCommandContext(runCtx, argv[0], argv[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.SetStdout(&stdout)
	cmd.SetStderr(&stderr)
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

// WaitReady observes guest readiness until ready, timeout, or context
// cancellation. Invalid names and unreadable VM state are errors; a reached
// terminal observation that is not ready is returned in GuestReadiness.
func (p *QemuProvider) WaitReady(ctx context.Context, name string, timeout time.Duration) (GuestReadiness, error) {
	if err := identifier.Validate(name); err != nil {
		return GuestReadiness{}, err
	}
	if timeout <= 0 {
		timeout = DefaultReadyTimeout
	}
	if timeout > MaxReadyTimeout {
		timeout = MaxReadyTimeout
	}

	state, err := p.loadState(name)
	if err != nil {
		return GuestReadiness{}, fmt.Errorf("VM %q: %w", name, err)
	}

	pid := 0
	if pidData, readErr := os.ReadFile(filepath.Join(p.RootDir, "run", name+".pid")); readErr == nil {
		fmt.Sscanf(string(pidData), "%d", &pid)
	}
	if pid == 0 {
		pid = state.PID
	}

	return p.waitReady(ctx, name, state, pid, p.isVMProcess(name, pid), timeout, defaultReadinessDeps(p)), nil
}

func (p *QemuProvider) waitReady(ctx context.Context, name string, state VMState, pid int, running bool, timeout time.Duration, deps readinessDeps) GuestReadiness {
	start := deps.now()
	observation := GuestReadiness{
		State:     ReadinessPending,
		Phase:     ReadinessPhaseProcess,
		PID:       pid,
		CheckedAt: start.UTC(),
	}
	if !running {
		observation.State = ReadinessFailed
		observation.Failure = FailureVMNotRunning
		observation.Detail = "VM process is not running"
		return p.finishReadiness(name, observation, start, 0, deps)
	}
	if state.SSHPort == 0 {
		observation.State = ReadinessFailed
		observation.Failure = FailureSSHUnavailable
		observation.Detail = "VM has no SSH host port"
		return p.finishReadiness(name, observation, start, 0, deps)
	}

	sshPath, err := deps.findSSH()
	if err != nil {
		observation.State = ReadinessFailed
		observation.Failure = FailureSSHClientMissing
		observation.Detail = boundedReadinessDetail("ssh client not found: " + err.Error())
		return p.finishReadiness(name, observation, start, 0, deps)
	}

	probeArgs := readinessSSHArgs(sshClientArgs(state.SSHPort, state.SSHUser, "127.0.0.1"))
	probeArgs[0] = sshPath
	deadline := start.Add(timeout)
	attempts := 0
	lastDetail := ""

	// Persist the pending observation so other surfaces can show that a probe
	// is in progress. Failures here never mask the readiness result.
	_ = p.saveReadiness(name, &observation)

	for {
		if ctxErr := ctx.Err(); ctxErr != nil {
			observation.State = ReadinessFailed
			observation.Failure = FailureCancelled
			observation.Detail = "wait was cancelled"
			return p.finishReadiness(name, observation, start, attempts, deps)
		}
		if !deps.now().Before(deadline) {
			observation.State = ReadinessFailed
			observation.Failure = FailureTimeout
			if lastDetail == "" {
				if observation.Phase == ReadinessPhaseProvisioning {
					lastDetail = "timed out waiting for cloud-init"
				} else {
					lastDetail = "timed out waiting for SSH"
				}
			}
			observation.Detail = boundedReadinessDetail(lastDetail)
			return p.finishReadiness(name, observation, start, attempts, deps)
		}
		if !deps.isAlive(name, pid) {
			observation.State = ReadinessFailed
			observation.Phase = ReadinessPhaseProcess
			observation.Failure = FailureProcessExited
			observation.Detail = "VM process exited before the guest became ready"
			return p.finishReadiness(name, observation, start, attempts, deps)
		}

		remaining := deadline.Sub(deps.now())
		attemptTimeout := readinessAttemptTimeout
		if remaining < attemptTimeout {
			attemptTimeout = remaining
		}

		attempts++
		_, stderr, sshErr := deps.runSSH(ctx, appendProbeCommand(probeArgs, "exit 0"), attemptTimeout)
		if sshErr == nil {
			observation.Phase = ReadinessPhaseProvisioning
			stdout, _, _ := deps.runSSH(ctx, appendProbeCommand(probeArgs, cloudInitProbeCommand), attemptTimeout)
			provisioning := parseCloudInitStatus(stdout)
			observation.Provisioning = provisioning.State
			switch provisioning.State {
			case ProvisioningError:
				observation.State = ReadinessFailed
				observation.Failure = FailureProvisioning
				observation.Detail = boundedReadinessDetail(provisioning.Detail)
				return p.finishReadiness(name, observation, start, attempts, deps)
			case ProvisioningRunning:
				lastDetail = "cloud-init is still running"
			default:
				observation.State = ReadinessReady
				observation.Phase = ReadinessPhaseReady
				observation.Ready = true
				observation.Detail = boundedReadinessDetail(provisioning.Detail)
				return p.finishReadiness(name, observation, start, attempts, deps)
			}
		} else {
			observation.Phase = ReadinessPhaseSSH
			lastDetail = boundedReadinessDetail(string(stderr))
			if failure := classifySSHAttempt(sshErr, stderr); failure != "" {
				observation.State = ReadinessFailed
				observation.Failure = failure
				observation.Detail = lastDetail
				return p.finishReadiness(name, observation, start, attempts, deps)
			}
		}

		if sleepErr := deps.sleep(ctx, readinessPollInterval); sleepErr != nil {
			if ctx.Err() != nil {
				observation.State = ReadinessFailed
				observation.Failure = FailureCancelled
				observation.Detail = "wait was cancelled"
				return p.finishReadiness(name, observation, start, attempts, deps)
			}
		}
	}
}

func (p *QemuProvider) finishReadiness(name string, observation GuestReadiness, start time.Time, attempts int, deps readinessDeps) GuestReadiness {
	checkedAt := deps.now()
	observation.CheckedAt = checkedAt.UTC()
	observation.DurationMS = checkedAt.Sub(start).Milliseconds()
	observation.Attempts = attempts
	observation.Detail = boundedReadinessDetail(observation.Detail)
	_ = p.saveReadiness(name, &observation)
	return observation
}

func (p *QemuProvider) saveReadiness(name string, readiness *GuestReadiness) error {
	state, err := p.loadState(name)
	if err != nil {
		return err
	}
	state.Readiness = readiness
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return sysutil.WriteFile(filepath.Join(p.RootDir, "run", name+".json"), data, 0600)
}

func (p *QemuProvider) clearReadiness(name string) {
	_ = p.saveReadiness(name, nil)
}

// visibleReadiness only exposes an observation that still belongs to the
// running VM process. A stale record is evidence from a previous boot.
func visibleReadiness(readiness *GuestReadiness, running bool, pid int) *GuestReadiness {
	if readiness == nil || !running {
		return nil
	}
	if readiness.PID != 0 && readiness.PID != pid {
		return nil
	}
	return readiness
}

func readinessSSHArgs(base []string) []string {
	probeOptions := []string{
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=5",
		"-o", "ConnectionAttempts=1",
		"-o", "NumberOfPasswordPrompts=0",
		"-o", "ServerAliveInterval=5",
		"-o", "ServerAliveCountMax=1",
	}
	args := make([]string, 0, len(base)+len(probeOptions))
	args = append(args, base[0])
	args = append(args, probeOptions...)
	args = append(args, base[1:]...)
	return args
}

func appendProbeCommand(base []string, command string) []string {
	args := make([]string, 0, len(base)+1)
	args = append(args, base...)
	args = append(args, command)
	return args
}

const cloudInitAbsentMarker = "__NIDO_NO_CLOUD_INIT__"

const cloudInitProbeCommand = "if ! command -v cloud-init >/dev/null 2>&1; then echo " +
	cloudInitAbsentMarker + "; exit 0; fi; cloud-init status --format=json 2>/dev/null || cloud-init status 2>/dev/null || true"

type cloudInitObservation struct {
	State  ProvisioningState
	Detail string
}

func parseCloudInitStatus(stdout []byte) cloudInitObservation {
	text := strings.TrimSpace(string(stdout))
	if text == "" {
		return cloudInitObservation{State: ProvisioningAbsent}
	}
	if strings.Contains(text, cloudInitAbsentMarker) {
		return cloudInitObservation{State: ProvisioningAbsent}
	}

	var payload struct {
		Status            string              `json:"status"`
		ExtendedStatus    string              `json:"extended_status"`
		Errors            []string            `json:"errors"`
		RecoverableErrors map[string][]string `json:"recoverable_errors"`
	}
	if json.Unmarshal([]byte(text), &payload) == nil && (payload.Status != "" || payload.ExtendedStatus != "") {
		recoverable := make([]string, 0, len(payload.RecoverableErrors))
		for _, entries := range payload.RecoverableErrors {
			recoverable = append(recoverable, entries...)
		}
		return classifyCloudInit(payload.Status, payload.ExtendedStatus, payload.Errors, recoverable)
	}

	status, extended := parseCloudInitText(text)
	if status == "" && extended == "" {
		return cloudInitObservation{State: ProvisioningAbsent}
	}
	return classifyCloudInit(status, extended, nil, nil)
}

func parseCloudInitText(text string) (status, extended string) {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "status:"):
			status = strings.TrimSpace(strings.TrimPrefix(line, "status:"))
		case strings.HasPrefix(line, "extended_status:"):
			extended = strings.TrimSpace(strings.TrimPrefix(line, "extended_status:"))
		}
	}
	return status, extended
}

func classifyCloudInit(status, extended string, errs, recoverable []string) cloudInitObservation {
	status = strings.ToLower(strings.TrimSpace(status))
	extended = strings.ToLower(strings.TrimSpace(extended))
	detail := cloudInitDetail(errs, recoverable)

	switch {
	case strings.Contains(status, "error") || strings.Contains(extended, "error"):
		return cloudInitObservation{State: ProvisioningError, Detail: detail}
	case strings.Contains(status, "recoverable"):
		return cloudInitObservation{State: ProvisioningError, Detail: detail}
	case strings.Contains(extended, "degraded"):
		return cloudInitObservation{State: ProvisioningDegraded, Detail: detail}
	case strings.Contains(status, "disabled"):
		return cloudInitObservation{State: ProvisioningAbsent}
	case strings.Contains(status, "running"), strings.Contains(status, "not started"),
		strings.Contains(extended, "running"), strings.Contains(extended, "not started"):
		return cloudInitObservation{State: ProvisioningRunning, Detail: detail}
	case status == "done", extended == "done":
		if detail != "" {
			return cloudInitObservation{State: ProvisioningDegraded, Detail: detail}
		}
		return cloudInitObservation{State: ProvisioningDone}
	default:
		return cloudInitObservation{State: ProvisioningAbsent, Detail: detail}
	}
}

func cloudInitDetail(errs, recoverable []string) string {
	all := append(append([]string{}, errs...), recoverable...)
	for _, entry := range all {
		if detail := boundedReadinessDetail(entry); detail != "" {
			return detail
		}
	}
	return ""
}

func classifySSHAttempt(err error, stderr []byte) ReadinessFailure {
	if err == nil {
		return ""
	}
	if errors.Is(err, exec.ErrNotFound) {
		return FailureSSHClientMissing
	}
	if strings.Contains(strings.ToLower(err.Error()), "executable file not found") {
		return FailureSSHClientMissing
	}
	text := strings.ToLower(string(stderr))
	switch {
	case strings.Contains(text, "permission denied"),
		strings.Contains(text, "authentication failed"),
		strings.Contains(text, "no supported authentication methods"),
		strings.Contains(text, "host key verification failed"):
		return FailureSSHAuth
	default:
		return ""
	}
}

func boundedReadinessDetail(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if idx := strings.IndexAny(text, "\r\n"); idx >= 0 {
		text = strings.TrimSpace(text[:idx])
	}
	text = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return ' '
		}
		return r
	}, text)
	text = strings.Join(strings.Fields(text), " ")
	if len(text) > readinessDetailLimit {
		text = text[:readinessDetailLimit] + "..."
	}
	return text
}
