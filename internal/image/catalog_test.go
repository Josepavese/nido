package image

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadCatalogFromFile(t *testing.T) {
	// Create temp catalog
	tmpDir := t.TempDir()
	catalogPath := filepath.Join(tmpDir, CatalogCacheFile)

	catalog := &Catalog{
		SchemaVersion: "1",
		UpdatedAt:     time.Now(),
		Images: []Image{
			{
				Name:     "test",
				Registry: "official",
				Versions: []Version{
					{Version: "1.0", Arch: "amd64"},
				},
			},
		},
	}

	data, _ := json.Marshal(catalog)
	os.WriteFile(catalogPath, data, 0644)

	// Test loading
	loaded, err := loadFromFile(catalogPath)
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	if loaded.SchemaVersion != "1" {
		t.Errorf("wrong schema version: %s", loaded.SchemaVersion)
	}

	if len(loaded.Images) != 1 {
		t.Errorf("wrong image count: %d", len(loaded.Images))
	}
}

func TestFindImage(t *testing.T) {
	catalog := &Catalog{
		Images: []Image{
			{
				Name: "ubuntu",
				Versions: []Version{
					{
						Version: "24.04",
						Aliases: []string{"latest", "noble", "lts"},
					},
					{
						Version: "22.04",
						Aliases: []string{"jammy"},
					},
				},
			},
			{
				Name: "debian",
				Versions: []Version{
					{
						Version: "12",
						Aliases: []string{"latest", "bookworm"},
					},
				},
			},
		},
	}

	tests := []struct {
		name    string
		version string
		wantVer string
		wantErr bool
	}{
		// Exact version matches
		{"ubuntu", "24.04", "24.04", false},
		{"ubuntu", "22.04", "22.04", false},
		{"debian", "12", "12", false},

		// Alias matches
		{"ubuntu", "latest", "24.04", false},
		{"ubuntu", "noble", "24.04", false},
		{"ubuntu", "lts", "24.04", false},
		{"ubuntu", "jammy", "22.04", false},
		{"debian", "bookworm", "12", false},

		// Empty version defaults to "latest"
		{"ubuntu", "", "24.04", false},
		{"debian", "", "12", false},

		// Not found cases
		{"ubuntu", "99.99", "", true},
		{"alpine", "3.20", "", true},
		{"debian", "11", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name+":"+tt.version, func(t *testing.T) {
			_, ver, err := catalog.FindImage(tt.name, tt.version)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if ver.Version != tt.wantVer {
				t.Errorf("wrong version: got %s, want %s", ver.Version, tt.wantVer)
			}
		})
	}
}

func TestFindImageCaseSensitive(t *testing.T) {
	catalog := &Catalog{
		Images: []Image{
			{
				Name: "ubuntu",
				Versions: []Version{
					{Version: "24.04", Aliases: []string{"latest"}},
				},
			},
		},
	}

	// Should not find "Ubuntu" (capital U)
	_, _, err := catalog.FindImage("Ubuntu", "24.04")
	if err == nil {
		t.Error("expected error for case mismatch, got nil")
	}
}
