package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Josepavese/nido/v4/internal/image"
	"github.com/Josepavese/nido/v4/internal/registry/builder"
)

type failedUpstream struct{}

func (failedUpstream) Fetch(builder.Source, builder.Strategy) ([]image.Version, error) {
	return nil, fmt.Errorf("upstream outage")
}

func TestFailedSyncLeavesCatalogByteForByteUnchanged(t *testing.T) {
	dir := t.TempDir()
	sources := filepath.Join(dir, "sources.yaml")
	output := filepath.Join(dir, "images.json")
	before := []byte(`{"schema_version":"1","updated_at":"2026-01-01T00:00:00Z","images":[]}`)
	if err := os.WriteFile(sources, []byte("sources:\n  - name: ubuntu\n    strategies:\n      - type: ubuntu-cloud\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, before, 0600); err != nil {
		t.Fatal(err)
	}
	if err := run(sources, output, failedUpstream{}, time.Now()); err == nil {
		t.Fatal("outage reported success")
	}
	after, err := os.ReadFile(output)
	if err != nil || string(after) != string(before) {
		t.Fatalf("catalog overwritten: %s %v", after, err)
	}
	files, _ := os.ReadDir(dir)
	if len(files) != 2 {
		t.Fatal("temporary artifacts left behind")
	}
}
