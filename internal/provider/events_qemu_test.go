//go:build !windows

package provider

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Josepavese/nido/v4/internal/config"
	"github.com/Josepavese/nido/v4/internal/events"
	"github.com/Josepavese/nido/v4/internal/hardware"
)

type eventSoftwareHost struct{ hardware.Host }

func (eventSoftwareHost) Virtualization(guest string) hardware.Virtualization {
	return hardware.Virtualization{CPU: "qemu64"}
}

// This opt-in qualification uses actual QEMU software emulation with a blank
// private 1 MiB disk. It never boots a guest OS or touches PCI hardware.
func TestRealQEMUEventLifecycle(t *testing.T) {
	storage := os.Getenv("NIDO_EVENT_QEMU_TEST_STORAGE")
	if storage == "" {
		t.Skip("set NIDO_EVENT_QEMU_TEST_STORAGE for isolated real QEMU qualification")
	}
	root, err := os.MkdirTemp(storage, "nq-")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "config.env")
	if err := os.WriteFile(path, []byte("SSH_USER=fixture\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	host := eventSoftwareHost{hardware.New()}
	p := NewQemuProvider(root, cfg, host)
	ctx, cancel := context.WithCancel(context.Background())
	client := events.NewClient(root, ObserveEvents(root, path, host))
	stream := client.Watch(ctx)
	p.Events = client.Publish
	defer func() {
		cancel()
		for range stream {
		}
		if err := p.Stop("fixture"); err != nil {
			t.Errorf("preserved active fixture at %s: %v", root, err)
			return
		}
		os.RemoveAll(root)
	}()
	wait := func(kind string) {
		t.Helper()
		timer := time.NewTimer(8 * time.Second)
		defer timer.Stop()
		for {
			select {
			case e := <-stream:
				if e.Kind == kind {
					t.Logf("observed %s epoch=%s sequence=%d", kind, e.Epoch, e.Sequence)
					return
				}
			case <-timer.C:
				t.Fatalf("missing real QEMU event: %s", kind)
			}
		}
	}
	wait(events.Resync)
	if err := p.CreateDisk("fixture", "1M", ""); err != nil {
		t.Fatal(err)
	}
	if err := p.Start("fixture", VMOptions{Arch: "amd64", MemoryMB: 128, VCPUs: 1, RawQemuArgs: []string{"-S"}}); err != nil {
		t.Fatal(err)
	}
	wait(events.VMPaused)
	before, err := p.Info("fixture")
	if err != nil || before.State != VMPaused || before.PID == 0 {
		t.Fatalf("prelaunch status: %+v %v", before, err)
	}
	if err := p.Start("fixture", VMOptions{}); err != nil {
		t.Fatal(err)
	}
	if count, err := p.Prune(); err != nil || count != 0 {
		t.Fatalf("prune touched a paused VM: %d %v", count, err)
	}
	if err := p.Resume(ctx, "fixture"); err != nil {
		t.Fatal(err)
	}
	wait(events.VMResumed)
	after, err := p.List()
	if err != nil || len(after) != 1 || after[0].State != VMRunning || after[0].PID != before.PID {
		t.Fatalf("resume replaced the process or failed: %+v %v", after, err)
	}
	if err := p.Resume(ctx, "fixture"); err != nil {
		t.Fatalf("idempotent resume: %v", err)
	}
	if err := withQMPControl(ctx, p.qmpControlPath("fixture"), events.IOTimeout, func(q *qmpSession) error {
		return q.command("stop", nil)
	}); err != nil {
		t.Fatal(err)
	}
	wait(events.VMPaused)
	if err := p.Stop("fixture"); err != nil {
		t.Fatal(err)
	}
	wait(events.VMStopped)
	if info, err := p.Info("fixture"); err != nil || info.State != "stopped" {
		t.Fatalf("stop not confirmed: %+v %v", info, err)
	}
	if err := p.Resume(ctx, "fixture"); err == nil {
		t.Fatal("resume accepted a stopped VM")
	}
	t.Log("real pause, same-process resume and termination verified; guest readiness not tested")
}
