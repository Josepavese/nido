package sysutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMutableCopyFromReadOnlySeedPreservesOriginal(t *testing.T) {
	root := t.TempDir()
	source, target := filepath.Join(root, "seed.fd"), filepath.Join(root, "private.fd")
	before := []byte("immutable firmware seed")
	if err := os.WriteFile(source, before, 0444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(source, 0600) })
	if err := CopyWritableFile(source, target); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(target, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("private variables remain read-only: %v", err)
	}
	if _, err := file.WriteAt([]byte("changed"), 0); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(source); err != nil || string(got) != string(before) {
		t.Fatalf("shared seed changed: %q %v", got, err)
	}
	if info, err := os.Stat(source); err != nil || info.Mode().Perm()&0200 != 0 {
		t.Fatalf("seed became writable: %v %v", info, err)
	}
	// A previously created private read-only instance is repaired without
	// resetting its content to the original seed.
	if err := os.Chmod(target, 0444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(target, 0600) })
	if err := MakeWritable(target); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(target); err != nil || len(got) < 7 || string(got[:7]) != "changed" {
		t.Fatalf("private state was reset: %q %v", got, err)
	}
}

func TestWritableInstanceRefusesSharedSeedSymlink(t *testing.T) {
	root := t.TempDir()
	seed, link := filepath.Join(root, "seed.fd"), filepath.Join(root, "variables.fd")
	if err := os.WriteFile(seed, []byte("shared"), 0444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(seed, 0600) })
	if err := os.Symlink(seed, link); err != nil {
		t.Skipf("symlink unavailable on this test host: %v", err)
	}
	if err := MakeWritable(link); err == nil {
		t.Fatal("accepted variables sharing an immutable seed")
	}
	if info, err := os.Stat(seed); err != nil || info.Mode().Perm()&0200 != 0 {
		t.Fatalf("shared seed permissions changed: %v %v", info, err)
	}
	if err := os.Chmod(seed, 0644); err != nil {
		t.Fatal(err)
	}
	if err := CopyWritableFile(seed, link); err == nil {
		t.Fatal("copied a mutable instance through a shared-seed symlink")
	}
	if got, err := os.ReadFile(seed); err != nil || string(got) != "shared" {
		t.Fatalf("copy changed the shared seed: %q %v", got, err)
	}
}
