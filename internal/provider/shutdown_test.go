package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func testShutdownDeps(run func(ctx context.Context, argv []string, timeout time.Duration) ([]byte, []byte, error), alive func() bool) shutdownDeps {
	return shutdownDeps{
		findSSH: func() (string, error) { return "/usr/bin/ssh", nil },
		runSSH:  run,
		isAlive: func(string, int) bool { return alive() },
		now:     time.Now,
		sleep: func(ctx context.Context, d time.Duration) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(d):
				return nil
			}
		},
	}
}

func shutdownTarget() guestShutdownTarget {
	return guestShutdownTarget{name: "fixture", pid: 4242, sshPort: 2222, sshUser: "vmuser"}
}

func TestRequestGuestShutdownUsesGuestPoweroffAndWaitsForExit(t *testing.T) {
	alive := true
	var commands []string
	deps := testShutdownDeps(func(_ context.Context, argv []string, _ time.Duration) ([]byte, []byte, error) {
		command := argv[len(argv)-1]
		commands = append(commands, command)
		switch command {
		case shutdownPOSIXDetection:
			return []byte("Linux\n"), nil, nil
		case shutdownPOSIXCommand:
			alive = false
			return nil, nil, nil
		default:
			return nil, nil, fmt.Errorf("unexpected command %q", command)
		}
	}, func() bool { return alive })

	if err := requestGuestShutdown(context.Background(), shutdownTarget(), deps); err != nil {
		t.Fatalf("requestGuestShutdown: %v", err)
	}
	if len(commands) != 2 {
		t.Fatalf("expected detection and shutdown commands, got %v", commands)
	}
	if !strings.Contains(commands[1], "sudo -n shutdown -h now") {
		t.Fatalf("expected a guest poweroff request, got %q", commands[1])
	}
}

func TestRequestGuestShutdownFallsBackToWindowsCommand(t *testing.T) {
	alive := true
	var commands []string
	deps := testShutdownDeps(func(_ context.Context, argv []string, _ time.Duration) ([]byte, []byte, error) {
		command := argv[len(argv)-1]
		commands = append(commands, command)
		if command == shutdownPOSIXDetection {
			return nil, []byte("'uname' is not recognized"), errors.New("exit status 1")
		}
		if command == shutdownWindowsCommand {
			alive = false
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("unexpected command %q", command)
	}, func() bool { return alive })

	if err := requestGuestShutdown(context.Background(), shutdownTarget(), deps); err != nil {
		t.Fatalf("requestGuestShutdown: %v", err)
	}
	if len(commands) != 2 || commands[1] != shutdownWindowsCommand {
		t.Fatalf("expected Windows shutdown command, got %v", commands)
	}
}

func TestRequestGuestShutdownNeverForcesWhenGuestStaysAlive(t *testing.T) {
	deps := testShutdownDeps(func(_ context.Context, argv []string, _ time.Duration) ([]byte, []byte, error) {
		if argv[len(argv)-1] == shutdownPOSIXDetection {
			return []byte("Linux\n"), nil, nil
		}
		return nil, nil, nil
	}, func() bool { return true })

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	err := requestGuestShutdown(ctx, shutdownTarget(), deps)
	if err == nil {
		t.Fatal("expected a timeout instead of declaring a forced stop successful")
	}
	if !strings.Contains(err.Error(), "did not power off") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRequestGuestShutdownReportsAuthenticationFailure(t *testing.T) {
	var commands int
	deps := testShutdownDeps(func(_ context.Context, _ []string, _ time.Duration) ([]byte, []byte, error) {
		commands++
		return nil, []byte("Permission denied (publickey)"), errors.New("exit status 255")
	}, func() bool { return true })

	err := requestGuestShutdown(context.Background(), shutdownTarget(), deps)
	if err == nil || !strings.Contains(err.Error(), "Permission denied") {
		t.Fatalf("expected an authentication error, got %v", err)
	}
	if commands != 2 {
		t.Fatalf("expected detection and request attempts, got %d", commands)
	}
}

func TestRequestGuestShutdownRequiresSSHClient(t *testing.T) {
	deps := testShutdownDeps(nil, func() bool { return true })
	deps.findSSH = func() (string, error) { return "", errors.New("not installed") }

	err := requestGuestShutdown(context.Background(), shutdownTarget(), deps)
	if err == nil || !strings.Contains(err.Error(), "SSH client") {
		t.Fatalf("expected missing SSH client error, got %v", err)
	}
}

func TestNormalizeShutdownTimeout(t *testing.T) {
	if got, err := normalizeShutdownTimeout(0); err != nil || got != DefaultShutdownTimeout {
		t.Fatalf("default timeout = %v, %v", got, err)
	}
	if _, err := normalizeShutdownTimeout(MaxShutdownTimeout + time.Second); err == nil {
		t.Fatal("expected bounded timeout error")
	}
}

func TestIsPOSIXGuest(t *testing.T) {
	for _, output := range []string{"Linux", "Darwin\n", "FreeBSD", "OpenBSD"} {
		if !isPOSIXGuest(output) {
			t.Errorf("expected %q to be treated as POSIX", output)
		}
	}
	for _, output := range []string{"", "Microsoft Windows [Version 10.0]", "cmd.exe"} {
		if isPOSIXGuest(output) {
			t.Errorf("expected %q not to be treated as POSIX", output)
		}
	}
}
