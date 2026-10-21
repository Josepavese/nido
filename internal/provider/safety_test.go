package provider

import (
	"errors"
	"github.com/Josepavese/nido/internal/hardware"
	"os"
	"path/filepath"
	"testing"

	"github.com/Josepavese/nido/internal/config"
)

func TestIdentifierRejectsPathsBeforeEffects(t *testing.T) {
	p := NewQemuProvider(t.TempDir(), &config.Config{}, hardware.New())
	for _, name := range []string{"", ".", "..", "../images/base", "a/b", `a\b`, `C:\disk`, "/tmp/other"} {
		t.Run(name, func(t *testing.T) {
			operations := []func() error{
				func() error { return p.Spawn(name, VMOptions{}) },
				func() error { return p.Start(name, VMOptions{}) },
				func() error { return p.Stop(name) },
				func() error { return p.Delete(name) },
				func() error { return p.CreateDisk(name, "1M", "") },
				func() error { return p.UpdateConfig(name, VMConfigUpdates{}) },
				func() error { _, err := p.Info(name); return err },
				func() error { return p.DeleteTemplate(name, true) },
				func() error { _, err := p.CreateTemplate("valid", name); return err },
				func() error { return p.CacheRemove(name, "v1") },
			}
			for i, operation := range operations {
				if operation() == nil {
					t.Fatalf("operation %d accepted path %q", i, name)
				}
			}
		})
	}
	entries, err := os.ReadDir(p.RootDir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("invalid operations changed root: %v %v", entries, err)
	}
	for _, valid := range []string{"fixture", "a.b-1_2", "ubuntu-24.04"} {
		if err := validateIdentifier(valid); err != nil {
			t.Fatal(err)
		}
	}
}

func TestInfoDistinguishesMissingAndCorruptState(t *testing.T) {
	p := NewQemuProvider(t.TempDir(), &config.Config{}, hardware.New())
	if _, err := p.Info("missing"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected not-found, got %v", err)
	}
	for _, dir := range []string{"run", "vms"} {
		if err := os.Mkdir(filepath.Join(p.RootDir, dir), 0700); err != nil {
			t.Fatal(err)
		}
	}
	disk := filepath.Join(p.RootDir, "vms", "fixture.qcow2")
	if err := os.WriteFile(disk, []byte("disposable"), 0600); err != nil {
		t.Fatal(err)
	}
	info, err := p.Info("fixture")
	if err != nil || !info.StateMissing || info.DiskMissing {
		t.Fatalf("disk-only health = %#v, %v", info, err)
	}
	state := filepath.Join(p.RootDir, "run", "fixture.json")
	if err := os.WriteFile(state, []byte("{invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Info("fixture"); err == nil {
		t.Fatal("corrupt state silently accepted")
	}
	if _, err := p.List(); err == nil {
		t.Fatal("list hides corrupt state, making prune unsafe")
	}
	if err := p.Start("fixture", VMOptions{}); err == nil {
		t.Fatal("start resets corrupt state")
	}
	data, _ := os.ReadFile(state)
	if string(data) != "{invalid" {
		t.Fatal("corrupt source was overwritten")
	}
}

func TestProcessIdentityRequiresVMNameAndDisk(t *testing.T) {
	id := vmProcessIdentity{name: "fixture", disk: "/data/with spaces/fixture.qcow2"}
	good := []string{"qemu-system-x86_64", "-name", "fixture", "-drive", "file=" + id.disk + ",format=qcow2"}
	if !matchesVMCommand(good, id) {
		t.Fatal("legitimate argument vector rejected")
	}
	for _, bad := range [][]string{{"sleep", "120"}, {"qemu-system-x86_64", "-name", "other", "-drive", good[4]}, {"qemu-system-x86_64", "-name", "fixture", "-drive", "file=/other,format=qcow2"}} {
		if matchesVMCommand(bad, id) {
			t.Fatalf("foreign command accepted: %v", bad)
		}
	}
}

func TestForwardingAllocationIsTransactional(t *testing.T) {
	cfg := &config.Config{PortRangeStart: 31000, PortRangeEnd: 31000}
	p := NewQemuProvider(t.TempDir(), cfg, hardware.New())
	input := []PortForward{{GuestPort: 80}, {GuestPort: 443}}
	if _, err := p.resolveForwarding(input, 51022, 0); err == nil {
		t.Fatal("two automatic ports fit into a one-port range")
	}
	if input[0].HostPort != 0 || input[1].HostPort != 0 {
		t.Fatal("failed allocation mutated caller state")
	}
	explicit := []PortForward{{GuestPort: 80, HostPort: 31080}}
	result, err := p.resolveForwarding(explicit, 51022, 0)
	if err != nil || result[0].HostPort != 31080 {
		t.Fatalf("explicit port changed: %v %v", result, err)
	}
}
