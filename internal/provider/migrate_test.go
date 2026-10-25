package provider

import (
	"bytes"
	"github.com/Josepavese/nido/v5/internal/config"
	"github.com/Josepavese/nido/v5/internal/hardware"
	"github.com/Josepavese/nido/v5/internal/image"
	"os"
	"path/filepath"
	"testing"
)

func TestArchitectureMigrationPreservesDataAndMixedGuests(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"run", "vms", "images", "backups"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0700); err != nil {
			t.Fatal(err)
		}
	}
	original := []byte(`{"name":"old","pid":0,"ssh_user":"existing","secret_field":"preserve","forwarding":[{"host_port":31001,"guest_port":80}]}`)
	state := filepath.Join(root, "run", "old.json")
	_ = os.WriteFile(state, original, 0600)
	_ = os.WriteFile(filepath.Join(root, "run", "arm.json"), []byte(`{"name":"arm","arch":"arm64","pid":0}`), 0600)
	paths := []string{"vms/old.qcow2", "vms/arm.qcow2", "images/ubuntu-24.04.qcow2", "backups/valuable.compact.qcow2"}
	for _, path := range paths {
		if err := os.WriteFile(filepath.Join(root, path), []byte("unchanged payload and backing reference"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	p := NewQemuProvider(root, &config.Config{}, hardware.New())
	count, err := p.MigrateArchitecture()
	if err != nil || count != 5 {
		t.Fatalf("conversion %d %v", count, err)
	}
	for _, path := range paths {
		data, _ := os.ReadFile(filepath.Join(root, path))
		if !bytes.Equal(data, []byte("unchanged payload and backing reference")) {
			t.Fatal("disk changed")
		}
	}
	data, _ := os.ReadFile(state)
	if !bytes.Contains(data, []byte("secret_field")) {
		t.Fatal("lost unknown metadata")
	}
	backup, _ := os.ReadFile(state + ".pre-architecture")
	if !bytes.Equal(backup, original) {
		t.Fatal("original state not preserved")
	}
	if arch, err := image.ReadArchitecture(filepath.Join(root, "vms", "arm.qcow2")); arch != "arm64" || err != nil {
		t.Fatalf("ARM relabeled: %s %v", arch, err)
	}
	if count, err := p.MigrateArchitecture(); count != 0 || err != nil {
		t.Fatalf("not idempotent: %d %v", count, err)
	}
	if _, err := p.loadState("old"); err != nil {
		t.Fatal(err)
	}
}
