//go:build !windows

package provider

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Josepavese/nido/v5/internal/config"
	"github.com/Josepavese/nido/v5/internal/events"
	"github.com/Josepavese/nido/v5/internal/hardware"
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
	if err := p.Start("fixture", VMOptions{MemoryMB: 128, VCPUs: 1, RawQemuArgs: []string{"-S"}}); err != nil {
		t.Fatal(err)
	}
	wait(events.VMPaused)
	// Use the separate control monitor; event observation remains connected.
	conn, err := net.DialTimeout("unix", qemuQMPPath(filepath.Join(root, "run"), "fixture"), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	conn.SetDeadline(time.Now().Add(2 * time.Second))
	defer conn.Close()
	dec, enc := json.NewDecoder(conn), json.NewEncoder(conn)
	var message map[string]interface{}
	if err := dec.Decode(&message); err != nil {
		t.Fatal(err)
	}
	command := func(name string) {
		t.Helper()
		if err := enc.Encode(map[string]string{"execute": name}); err != nil {
			t.Fatal(err)
		}
		for {
			if err := dec.Decode(&message); err != nil {
				t.Fatal(err)
			}
			if _, ok := message["return"]; ok {
				return
			}
			if _, ok := message["error"]; ok {
				t.Fatal(message)
			}
		}
	}
	command("qmp_capabilities")
	command("cont")
	wait(events.VMResumed)
	command("stop")
	wait(events.VMPaused)
	conn.Close()
	if err := p.Stop("fixture"); err != nil {
		t.Fatal(err)
	}
	wait(events.VMStopped)
	if info, err := p.Info("fixture"); err != nil || info.State != "stopped" {
		t.Fatalf("stop not confirmed: %+v %v", info, err)
	}
	t.Log("real process termination verified; guest readiness not tested")
}
