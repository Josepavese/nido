package sysutil

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func TestCommandHelper(t *testing.T) {
	index := -1
	for i, arg := range os.Args {
		if arg == "NIDO_PAL_HELPER" {
			index = i
			break
		}
	}
	if index < 0 {
		return
	}
	args := os.Args[index+1:]
	switch args[0] {
	case "env":
		json.NewEncoder(os.Stdout).Encode(os.Getenv(args[1]))
		os.Exit(0)
	case "argv":
		json.NewEncoder(os.Stdout).Encode(args[1:])
		os.Exit(0)
	case "exit":
		code, _ := strconv.Atoi(args[1])
		os.Exit(code)
	case "wait":
		os.Stdout.Write([]byte("ready\n"))
		for {
			time.Sleep(time.Second)
		}
	}
	os.Exit(2)
}

func TestCommandEnvironmentOverrideIsConfinedToChild(t *testing.T) {
	const key = "NIDO_PAL_ENV_FIXTURE"
	t.Setenv(key, "parent")
	executable, args := helper(t, "env", key)
	data, err := NewCommandWithEnvironment(context.Background(), map[string]string{key: "child"}, executable, args...).Output()
	var value string
	if err != nil || json.Unmarshal(data, &value) != nil || value != "child" || os.Getenv(key) != "parent" {
		t.Fatalf("environment confinement: %s %v", data, err)
	}
}
func helper(t *testing.T, args ...string) (string, []string) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return executable, append([]string{"-test.run=^TestCommandHelper$", "--", "NIDO_PAL_HELPER"}, args...)
}
func TestNativeCommandPreservesArgumentBoundariesAndExitCode(t *testing.T) {
	values := []string{"path with spaces", "$(do not execute)", "a'b\"c", "", `C:\Users\Test User\key`}
	executable, args := helper(t, append([]string{"argv"}, values...)...)
	output, err := NewCommand(executable, args...).Output()
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	if err := json.Unmarshal(output, &got); err != nil || !reflect.DeepEqual(got, values) {
		t.Fatalf("argv changed: %q %v", output, err)
	}
	executable, args = helper(t, "exit", "7")
	err = NewCommand(executable, args...).Run()
	if err == nil || ExitCode(err) != 7 {
		t.Fatalf("portable exit status: %v", err)
	}
}
func TestCommandReaderFailureCancelsAndReapsProducer(t *testing.T) {
	executable, args := helper(t, "wait")
	sentinel := errors.New("consumer failed")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := ReadCommandOutput(ctx, executable, args, func(reader io.Reader) error {
		var b [1]byte
		if _, err := reader.Read(b[:]); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("consumer error lost: %v", err)
	}
	if ctx.Err() != nil {
		t.Fatal("producer was not reaped promptly after consumer failure")
	}
}
func TestTimedCommandWaitsForCancelledChild(t *testing.T) {
	executable, args := helper(t, "wait")
	_, err := RunTimedCommand(executable, args, 100*time.Millisecond, io.Discard)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline not reported: %v", err)
	}
}
