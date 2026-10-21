package sysutil

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestMetadataWriteFailurePreservesBytesAndRetiresStage(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "state.json")
	before := []byte(`{"owner":"keep","pending":true}`)
	if err := os.WriteFile(target, before, 0600); err != nil {
		t.Fatal(err)
	}
	failure := errors.New("simulated storage exhaustion")
	err := writeFileAtomically(target, 0600, func(writer io.Writer) error { writer.Write([]byte("partial")); return failure })
	if !errors.Is(err, failure) {
		t.Fatalf("write failure hidden: %v", err)
	}
	data, _ := os.ReadFile(target)
	entries, _ := os.ReadDir(root)
	if string(data) != string(before) || len(entries) != 1 {
		t.Fatalf("lost metadata or retained scratch: %s %v", data, entries)
	}
}
func TestMetadataWritePreservesModeAndCanonicalSymlink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "state.json")
	os.WriteFile(target, []byte("before"), 0600)
	beforeInfo, _ := os.Stat(target)
	beforeMode := beforeInfo.Mode().Perm()
	link := filepath.Join(root, "alias.json")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("native file symlink unavailable: %v", err)
	}
	if err := WriteFile(link, []byte("after"), 0644); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Lstat(link)
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("replaced explicit metadata symlink")
	}
	data, _ := os.ReadFile(target)
	if string(data) != "after" {
		t.Fatal("canonical metadata target was not updated")
	}
	info, _ = os.Stat(target)
	if info.Mode().Perm() != beforeMode {
		t.Fatalf("changed private metadata mode: %v", info.Mode())
	}
}
