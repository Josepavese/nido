package ui

import (
	"bytes"
	"os"
	"strings"
	"testing"

	clijson "github.com/Josepavese/nido/internal/cli"
)

func TestUIPrintsInHumanMode(t *testing.T) {
	clijson.SetJSONMode(false)
	restore, read := captureStdout(t)
	defer restore()

	Header("Fleet")
	Info("hello")
	Success("done")
	Warn("warn")
	Error("fail")
	Step("step")
	FancyLabel("Key", "Value")
	Rule(10)

	restore()
	out := read()
	for _, want := range []string{"NIDO", "info", "done", "warn", "fail", "step", "Key", "──────────"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output:\n%s", want, out)
		}
	}
}

func TestUISuppressedInJSONMode(t *testing.T) {
	clijson.SetJSONMode(true)
	defer clijson.SetJSONMode(false)

	restore, read := captureStdout(t)
	defer restore()

	Header("Fleet")
	Info("hello")
	Success("done")
	Warn("warn")
	Error("fail")
	Step("step")
	FancyLabel("Key", "Value")
	Rule(10)

	restore()
	if got := read(); got != "" {
		t.Fatalf("expected no UI output in JSON mode, got %q", got)
	}
}

func captureStdout(t *testing.T) (func(), func() string) {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	buf := &bytes.Buffer{}
	done := make(chan struct{})
	go func() {
		_, _ = buf.ReadFrom(r)
		close(done)
	}()
	restored := false
	restore := func() {
		if restored {
			return
		}
		restored = true
		_ = w.Close()
		os.Stdout = old
		<-done
		_ = r.Close()
	}
	return restore, func() string { return buf.String() }
}
