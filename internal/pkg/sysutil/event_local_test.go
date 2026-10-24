package sysutil

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestEventElectionPreservesForeignEndpointAndOwner(t *testing.T) {
	root, err := os.MkdirTemp("", "nl-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	dir, err := PrivateEventsDirectory(root)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, eventSocket)
	original := []byte("KEEP USER FILE")
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	if listener, err := ListenLocalEvents(root); err == nil {
		listener.Close()
		listener.Release()
		t.Fatal("replaced non-socket endpoint")
	}
	bytes, err := os.ReadFile(path)
	if err != nil || string(bytes) != string(original) {
		t.Fatal("foreign endpoint altered")
	}
	os.Remove(path) // Exact file owned by this test.
	listener, err := ListenLocalEvents(root)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Release()
	defer listener.Close()
	if other, err := ListenLocalEvents(root); err == nil {
		other.Close()
		other.Release()
		t.Fatal("second owner elected")
	}
	conn, err := DialLocalEvents(context.Background(), root)
	if err != nil {
		t.Fatal("second election damaged first endpoint", err)
	}
	conn.Close()
}
