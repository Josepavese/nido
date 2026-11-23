package provider

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Josepavese/nido/v4/internal/config"
	"github.com/Josepavese/nido/v4/internal/hardware"
	"github.com/Josepavese/nido/v4/internal/image"
)

func TestARMGuestPlatform(t *testing.T) {
	p := NewQemuProvider(t.TempDir(), &config.Config{}, argumentHost{Virtual: hardware.Virtualization{Accelerator: "tcg", CPU: "max"}})
	args := p.buildQemuArgs("guest", "/guest.qcow2", 51022, 0, nil, p.RootDir, "", 512, 1, nil, nil, "arm64")
	for _, value := range []string{"virt", "max", "virtio-blk-pci,drive=root"} {
		if !slices.Contains(args, value) {
			t.Fatalf("missing %q: %v", value, args)
		}
	}
	for _, value := range []string{"pc", "qemu64", "host"} {
		if slices.Contains(args, value) {
			t.Fatalf("x86/native-only value %q in ARM emulation", value)
		}
	}
}

func TestSourceArchitectureConflictFailsBeforeDiskCreation(t *testing.T) {
	root := t.TempDir()
	disk := filepath.Join(root, "template.qcow2")
	if err := os.WriteFile(disk, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := image.WriteArchitecture(disk, "arm64"); err != nil {
		t.Fatal(err)
	}
	p := NewQemuProvider(root, &config.Config{}, hardware.New())
	if err := p.Spawn("conflict", VMOptions{DiskPath: disk, Arch: "amd64"}); err == nil {
		t.Fatal("source was relabeled")
	}
	if _, err := os.Stat(filepath.Join(root, "vms", "conflict.qcow2")); !os.IsNotExist(err) {
		t.Fatal("conflicting request created a disk")
	}
}

func TestDeleteRemovesVMArchitectureMetadata(t *testing.T) {
	root := t.TempDir()
	vmDir := filepath.Join(root, "vms")
	if err := os.Mkdir(vmDir, 0700); err != nil {
		t.Fatal(err)
	}
	disk := filepath.Join(vmDir, "guest.qcow2")
	backing := filepath.Join(root, "base.qcow2")
	for _, path := range []string{disk, backing} {
		if err := os.WriteFile(path, nil, 0600); err != nil {
			t.Fatal(err)
		}
		if err := image.WriteArchitecture(path, "amd64"); err != nil {
			t.Fatal(err)
		}
	}

	p := NewQemuProvider(root, &config.Config{}, hardware.New())
	for range 2 {
		if err := p.Delete("guest"); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{disk, image.MetadataPath(disk)} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("expected %s to be removed, stat returned %v", path, err)
		}
	}
	if _, err := os.Stat(backing); err != nil {
		t.Errorf("backing disk changed: %v", err)
	}
	if arch, err := image.ReadArchitecture(backing); err != nil || arch != "amd64" {
		t.Errorf("backing architecture = %q, %v", arch, err)
	}
}

func TestPruneRejectsUsedImageDeletionPolicy(t *testing.T) {
	root := t.TempDir()
	p := NewQemuProvider(root, &config.Config{}, hardware.New())
	if _, _, err := p.CachePrune(false); err == nil {
		t.Fatal("accepted pruning without backing protection")
	}
	if entries, _ := os.ReadDir(root); len(entries) != 0 {
		t.Fatal("invalid policy had side effects")
	}
}
