package provider

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testReadinessProvider(t *testing.T) *QemuProvider {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "run"), 0o755); err != nil {
		t.Fatal(err)
	}
	p := &QemuProvider{RootDir: root}
	if err := p.saveState("vm1", 0, 51022, 0, false, "ubuntu", nil, "", 512, 1, nil, nil, "amd64"); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestParseCloudInitStatus(t *testing.T) {
	tests := []struct {
		name   string
		stdout string
		want   ProvisioningState
	}{
		{"done", `{"status":"done","extended_status":"done","errors":[],"recoverable_errors":{}}`, ProvisioningDone},
		{"degraded", `{"status":"done","extended_status":"degraded done","errors":[],"recoverable_errors":{"modules-final":["package mirror unavailable"]}}`, ProvisioningDegraded},
		{"error", `{"status":"error","extended_status":"error - done","errors":["module failed"],"recoverable_errors":{}}`, ProvisioningError},
		{"running", `{"status":"running","extended_status":"running","errors":[],"recoverable_errors":{}}`, ProvisioningRunning},
		{"real running", `{"boot_status_code":"enabled-by-generator","datasource":"nocloud","detail":"DataSourceNoCloud","errors":[],"extended_status":"running","init":{"errors":[],"recoverable_errors":{}},"recoverable_errors":{},"stage":"modules-final","status":"running"}`, ProvisioningRunning},
		{"not started", `{"status":"not started"}`, ProvisioningRunning},
		{"disabled", `{"status":"disabled"}`, ProvisioningAbsent},
		{"absent marker", cloudInitAbsentMarker, ProvisioningAbsent},
		{"plain done", "status: done\nextended_status: done", ProvisioningDone},
		{"plain running", "status: running", ProvisioningRunning},
		{"empty", "", ProvisioningAbsent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseCloudInitStatus([]byte(tt.stdout)); got.State != tt.want {
				t.Fatalf("parseCloudInitStatus(%q).State = %q, want %q", tt.stdout, got.State, tt.want)
			}
		})
	}

	degraded := parseCloudInitStatus([]byte(`{"status":"done","extended_status":"degraded done","errors":[],"recoverable_errors":{"modules-final":["package mirror unavailable"]}}`))
	if !strings.Contains(degraded.Detail, "package mirror unavailable") {
		t.Fatalf("degraded detail = %q, want recoverable error", degraded.Detail)
	}
	failed := parseCloudInitStatus([]byte(`{"status":"error","errors":["module failed"]}`))
	if !strings.Contains(failed.Detail, "module failed") {
		t.Fatalf("error detail = %q, want module error", failed.Detail)
	}
}

func TestClassifySSHAttempt(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		stderr string
		want   ReadinessFailure
	}{
		{"success", nil, "", ""},
		{"permission denied", errors.New("exit status 255"), "Permission denied (publickey).", FailureSSHAuth},
		{"auth failed", errors.New("exit status 255"), "Authentication failed.", FailureSSHAuth},
		{"connection refused", errors.New("exit status 255"), "ssh: connect to host 127.0.0.1 port 51022: Connection refused", ""},
		{"missing client", errors.New("exec: \"ssh\": executable file not found in $PATH"), "", FailureSSHClientMissing},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifySSHAttempt(tt.err, []byte(tt.stderr)); got != tt.want {
				t.Fatalf("classifySSHAttempt() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVisibleReadiness(t *testing.T) {
	readiness := &GuestReadiness{Ready: true, State: ReadinessReady, PID: 42}
	if got := visibleReadiness(nil, true, 42); got != nil {
		t.Fatal("nil readiness must stay nil")
	}
	if got := visibleReadiness(readiness, false, 42); got != nil {
		t.Fatal("readiness must be hidden for a stopped VM")
	}
	if got := visibleReadiness(readiness, true, 43); got != nil {
		t.Fatal("readiness from a previous process must be hidden")
	}
	if got := visibleReadiness(readiness, true, 42); got != readiness {
		t.Fatal("current readiness must be visible")
	}
}

func TestReadinessPersistenceSurvivesSaveState(t *testing.T) {
	p := testReadinessProvider(t)
	readiness := GuestReadiness{
		Ready:     true,
		State:     ReadinessReady,
		Phase:     ReadinessPhaseReady,
		PID:       42,
		CheckedAt: time.Now().UTC(),
	}
	if err := p.saveReadiness("vm1", &readiness); err != nil {
		t.Fatal(err)
	}
	if err := p.saveState("vm1", 0, 51022, 0, false, "ubuntu", nil, "", 512, 1, nil, nil, "amd64"); err != nil {
		t.Fatal(err)
	}
	state, err := p.loadState("vm1")
	if err != nil {
		t.Fatal(err)
	}
	if state.Readiness == nil || !state.Readiness.Ready || state.Readiness.PID != 42 {
		t.Fatalf("readiness was not preserved across saveState: %+v", state.Readiness)
	}

	p.clearReadiness("vm1")
	state, err = p.loadState("vm1")
	if err != nil {
		t.Fatal(err)
	}
	if state.Readiness != nil {
		t.Fatalf("clearReadiness left %+v", state.Readiness)
	}
}

func TestWaitReadyNotRunning(t *testing.T) {
	p := testReadinessProvider(t)
	observation, err := p.WaitReady(context.Background(), "vm1", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if observation.Ready || observation.Failure != FailureVMNotRunning {
		t.Fatalf("WaitReady on stopped VM = %+v, want vm_not_running", observation)
	}
}

func TestWaitReadyStagedObservation(t *testing.T) {
	p := testReadinessProvider(t)
	state := VMState{Name: "vm1", Arch: "amd64", PID: 123, SSHPort: 51022, SSHUser: "ubuntu"}
	now := time.Now()
	sshAttempts := 0
	deps := readinessDeps{
		findSSH: func() (string, error) { return "/usr/bin/ssh", nil },
		runSSH: func(_ context.Context, argv []string, _ time.Duration) ([]byte, []byte, error) {
			command := argv[len(argv)-1]
			if strings.Contains(command, "cloud-init") {
				return []byte(`{"status":"done","extended_status":"done"}`), nil, nil
			}
			sshAttempts++
			if sshAttempts < 3 {
				return nil, []byte("ssh: connect to host 127.0.0.1 port 51022: Connection refused"), errors.New("exit status 255")
			}
			return nil, nil, nil
		},
		isAlive: func(string, int) bool { return true },
		now:     func() time.Time { return now },
		sleep: func(_ context.Context, d time.Duration) error {
			now = now.Add(d)
			return nil
		},
	}

	observation := p.waitReady(context.Background(), "vm1", state, 123, true, 30*time.Second, deps)
	if !observation.Ready || observation.State != ReadinessReady || observation.Phase != ReadinessPhaseReady {
		t.Fatalf("staged observation = %+v, want ready", observation)
	}
	if observation.Attempts != 3 || observation.Provisioning != ProvisioningDone {
		t.Fatalf("staged observation = %+v, want 3 attempts and cloud-init done", observation)
	}
}

func TestWaitReadyAuthFailureIsTerminal(t *testing.T) {
	p := testReadinessProvider(t)
	state := VMState{Name: "vm1", Arch: "amd64", PID: 123, SSHPort: 51022, SSHUser: "ubuntu"}
	deps := readinessDeps{
		findSSH: func() (string, error) { return "/usr/bin/ssh", nil },
		runSSH: func(_ context.Context, _ []string, _ time.Duration) ([]byte, []byte, error) {
			return nil, []byte("Permission denied (publickey)."), errors.New("exit status 255")
		},
		isAlive: func(string, int) bool { return true },
		now:     time.Now,
		sleep:   func(context.Context, time.Duration) error { return nil },
	}

	observation := p.waitReady(context.Background(), "vm1", state, 123, true, 30*time.Second, deps)
	if observation.Ready || observation.Failure != FailureSSHAuth || observation.Attempts != 1 {
		t.Fatalf("auth observation = %+v, want terminal ssh_auth_failed", observation)
	}
}

func TestWaitReadyProvisioningError(t *testing.T) {
	p := testReadinessProvider(t)
	state := VMState{Name: "vm1", Arch: "amd64", PID: 123, SSHPort: 51022, SSHUser: "ubuntu"}
	deps := readinessDeps{
		findSSH: func() (string, error) { return "/usr/bin/ssh", nil },
		runSSH: func(_ context.Context, argv []string, _ time.Duration) ([]byte, []byte, error) {
			if strings.Contains(argv[len(argv)-1], "cloud-init") {
				return []byte(`{"status":"error","extended_status":"error - done","errors":["module failed"]}`), nil, nil
			}
			return nil, nil, nil
		},
		isAlive: func(string, int) bool { return true },
		now:     time.Now,
		sleep:   func(context.Context, time.Duration) error { return nil },
	}

	observation := p.waitReady(context.Background(), "vm1", state, 123, true, 30*time.Second, deps)
	if observation.Ready || observation.Failure != FailureProvisioning || !strings.Contains(observation.Detail, "module failed") {
		t.Fatalf("provisioning observation = %+v, want provisioning_error with detail", observation)
	}
}

func TestWaitReadyTimeout(t *testing.T) {
	p := testReadinessProvider(t)
	state := VMState{Name: "vm1", Arch: "amd64", PID: 123, SSHPort: 51022, SSHUser: "ubuntu"}
	now := time.Now()
	deps := readinessDeps{
		findSSH: func() (string, error) { return "/usr/bin/ssh", nil },
		runSSH: func(_ context.Context, _ []string, _ time.Duration) ([]byte, []byte, error) {
			return nil, []byte("ssh: connect to host 127.0.0.1 port 51022: Connection refused"), errors.New("exit status 255")
		},
		isAlive: func(string, int) bool { return true },
		now:     func() time.Time { return now },
		sleep: func(_ context.Context, d time.Duration) error {
			now = now.Add(d)
			return nil
		},
	}

	observation := p.waitReady(context.Background(), "vm1", state, 123, true, 10*time.Second, deps)
	if observation.Ready || observation.Failure != FailureTimeout {
		t.Fatalf("timeout observation = %+v, want timeout", observation)
	}
	if !strings.Contains(strings.ToLower(observation.Detail), "ssh") {
		t.Fatalf("timeout detail = %q, want SSH context", observation.Detail)
	}
}

func TestBoundedReadinessDetail(t *testing.T) {
	if got := boundedReadinessDetail("  first line\nsecond line  "); got != "first line" {
		t.Fatalf("boundedReadinessDetail() = %q", got)
	}
	long := strings.Repeat("x", readinessDetailLimit+20)
	got := boundedReadinessDetail(long)
	if len(got) <= readinessDetailLimit || !strings.HasSuffix(got, "...") {
		t.Fatalf("long detail was not bounded: len=%d", len(got))
	}
}
