package sysutil

import "testing"

func TestIsolatedNidoHome(t *testing.T) {
	root := t.TempDir()
	t.Setenv("NIDO_HOME", root)
	if got := NidoHome(); got != root {
		t.Fatalf("root=%q", got)
	}
}
