//go:build !windows

package sysutil

import (
	"reflect"
	"strings"
	"testing"
)

func TestShellRenderingRoundTripsWithoutExecutingArguments(t *testing.T) {
	args := []string{"path with spaces", "$(echo injected)", "a'b\"c", "", `C:\Test User\key`, "line\nbreak"}
	argv := append([]string{"printf", "%s\\0"}, args...)
	output, err := NewCommand("sh", "-c", ShellCommand(argv)).Output()
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Split(strings.TrimSuffix(string(output), "\x00"), "\x00")
	if !reflect.DeepEqual(got, args) {
		t.Fatalf("shell argument boundaries changed: %q, want %q", got, args)
	}
}
