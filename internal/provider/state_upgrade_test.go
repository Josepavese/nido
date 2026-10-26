package provider

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/Josepavese/nido/v4/internal/config"
	"github.com/Josepavese/nido/v4/internal/hardware"
	"github.com/Josepavese/nido/v4/internal/image"
)

func legacyStateFixture(t *testing.T) (*QemuProvider, string, []byte) {
	t.Helper()
	root := t.TempDir()
	for _, d := range []string{"run", "vms", "images", "backups"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0700); err != nil {
			t.Fatal(err)
		}
	}
	state := filepath.Join(root, "run", "legacy.json")
	original := []byte(`{"name":"legacy","pid":0,"ssh_user":"preserved","password":"fixture-only","unknown":{"keep":true},"forwarding":[{"host_port":31001,"guest_port":80}]}`)
	if err := os.WriteFile(state, original, 0600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"vms/legacy.qcow2", "images/old-base.qcow2", "backups/template.compact.qcow2"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("valuable payload"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return NewQemuProvider(root, &config.Config{}, hardware.New()), state, original
}

func TestAutomaticStatePreparationIsConcurrentAndOneTime(t *testing.T) {
	p, state, original := legacyStateFixture(t)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := p.PrepareState(context.Background()); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	data, err := os.ReadFile(state + ".pre-architecture")
	if err != nil || !bytes.Equal(data, original) {
		t.Fatalf("original backup: %s %v", data, err)
	}
	if _, err := p.loadState("legacy"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"vms/legacy.qcow2", "images/old-base.qcow2", "backups/template.compact.qcow2"} {
		data, err := os.ReadFile(filepath.Join(p.RootDir, name))
		if err != nil || string(data) != "valuable payload" {
			t.Fatalf("payload changed: %s %v", data, err)
		}
	}
	marker := filepath.Join(p.RootDir, stateSchemaFile)
	before, _ := os.Stat(marker)
	if err := p.PrepareState(context.Background()); err != nil {
		t.Fatal(err)
	}
	after, _ := os.Stat(marker)
	if !os.SameFile(before, after) || before.ModTime() != after.ModTime() {
		t.Fatal("completed bootstrap repeated")
	}
	if _, err := os.Stat(marker + ".lock"); !os.IsNotExist(err) {
		t.Fatalf("lease retained: %v", err)
	}
}

func TestAutomaticStatePreparationResumesAfterPartialFailure(t *testing.T) {
	p, state, original := legacyStateFixture(t)
	blocker := state + ".pre-architecture"
	if err := os.Mkdir(blocker, 0700); err != nil {
		t.Fatal(err)
	}
	if err := p.PrepareState(context.Background()); err == nil {
		t.Fatal("invalid backup accepted")
	}
	if _, err := os.Stat(filepath.Join(p.RootDir, stateSchemaFile)); !os.IsNotExist(err) {
		t.Fatal("failed preparation marked complete")
	}
	data, _ := os.ReadFile(state)
	if !bytes.Equal(data, original) {
		t.Fatal("state changed without usable backup")
	}
	if _, err := image.ReadArchitecture(filepath.Join(p.RootDir, "images", "old-base.qcow2")); err != nil {
		t.Fatalf("additive artifact phase did not complete: %v", err)
	}
	if err := os.Remove(blocker); err != nil {
		t.Fatal(err)
	}
	if err := p.PrepareState(context.Background()); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(blocker)
	if !bytes.Equal(data, original) {
		t.Fatal("resumed conversion lost original")
	}
}

func TestAutomaticStatePreparationRefusesUnknownSchemaAndPreservesRawImports(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "images"), 0700); err != nil {
		t.Fatal(err)
	}
	raw := filepath.Join(root, "images", "foreign.qcow2")
	os.WriteFile(raw, []byte("unidentified ARM import"), 0600)
	p := NewQemuProvider(root, &config.Config{}, hardware.New())
	if err := p.PrepareState(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(image.MetadataPath(raw)); !os.IsNotExist(err) {
		t.Fatal("raw import silently relabeled")
	}
	marker := filepath.Join(root, stateSchemaFile)
	os.WriteFile(marker, []byte("future-schema\n"), 0600)
	if err := p.PrepareState(context.Background()); err == nil {
		t.Fatal("future schema silently downgraded")
	}
	data, _ := os.ReadFile(marker)
	if string(data) != "future-schema\n" {
		t.Fatal("future marker changed")
	}
}
