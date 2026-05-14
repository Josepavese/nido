package runner

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestExecTimeoutKillsChildProcesses(t *testing.T) {
	r := New()
	inv := Invocation{
		Command: os.Args[0],
		Args:    []string{"-test.run=TestHelperProcess", "--"},
		Timeout: 300 * time.Millisecond,
		Env: map[string]string{
			"NIDO_RUNNER_HELPER": "parent",
		},
	}

	done := make(chan Result, 1)
	go func() {
		done <- r.Exec(inv)
	}()

	select {
	case res := <-done:
		if !res.TimedOut {
			t.Fatalf("expected timeout, got exit=%d stdout=%q stderr=%q", res.ExitCode, res.Stdout, res.Stderr)
		}
		if res.ExitCode != -1 {
			t.Fatalf("expected exit -1 on timeout, got %d", res.ExitCode)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("runner did not return after timeout; child process likely kept stdio open")
	}
}

func TestHelperProcess(t *testing.T) {
	mode := os.Getenv("NIDO_RUNNER_HELPER")
	if mode == "" {
		return
	}

	switch mode {
	case "parent":
		cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess", "--")
		cmd.Env = append(os.Environ(), "NIDO_RUNNER_HELPER=child")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		time.Sleep(30 * time.Second)
		os.Exit(0)
	case "child":
		fmt.Fprintln(os.Stdout, "child started")
		time.Sleep(30 * time.Second)
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "unknown helper mode %q\n", mode)
		os.Exit(2)
	}
}
