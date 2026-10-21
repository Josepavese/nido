package sysutil

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResourceReservationLifecycle(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "download.lock")
	first, err := AcquireResource(path)
	if err != nil {
		t.Fatal(err)
	}
	if other, err := AcquireResource(path); !errors.Is(err, ErrResourceBusy) {
		if other != nil {
			other.Close()
		}
		t.Fatalf("concurrent owner: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	second, err := AcquireResource(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(root)
	if len(entries) != 0 {
		t.Fatalf("owned residue: %v", entries)
	}
}
func TestResourcePreservesUnrecognizedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	os.WriteFile(path, []byte("USER DATA"), 0600)
	if lease, err := AcquireResource(path); err == nil {
		lease.Close()
		t.Fatal("accepted user file")
	}
	data, _ := os.ReadFile(path)
	if string(data) != "USER DATA" {
		t.Fatalf("lost data: %s", data)
	}
}
func TestResourceReusesCompleteCrashMarker(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	os.WriteFile(path, []byte(resourceMarker), 0600)
	lease, err := AcquireResource(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
}
