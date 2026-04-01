package image

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFetchChecksum(t *testing.T) {
	// Mock server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "5c3ddb00f60bc455dac0862fabe9d8bacec46c33ac1751143c5c3683404b110d *ubuntu-24.04-server-cloudimg-amd64.img")
		fmt.Fprintln(w, "ea85b16f81b3f6aa53a1260912d3f991fc33e0e0fc1d73f0b8c9c96247e42fdb *ubuntu-22.04-server-cloudimg-amd64.img")
	}))
	defer ts.Close()

	tests := []struct {
		name    string
		url     string
		regex   string
		want    string
		wantErr bool
	}{
		{
			name:    "Match Ubuntu 24.04",
			url:     ts.URL,
			regex:   "ubuntu-24\\.04-server-cloudimg-amd64\\.img",
			want:    "5c3ddb00f60bc455dac0862fabe9d8bacec46c33ac1751143c5c3683404b110d",
			wantErr: false,
		},
		{
			name:    "Match Ubuntu 22.04",
			url:     ts.URL,
			regex:   "ubuntu-22\\.04-server-cloudimg-amd64\\.img",
			want:    "ea85b16f81b3f6aa53a1260912d3f991fc33e0e0fc1d73f0b8c9c96247e42fdb",
			wantErr: false,
		},
		{
			name:    "No match",
			url:     ts.URL,
			regex:   "nonexistent-image\\.img",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FetchChecksum(tt.url, tt.regex, "sha256")
			if (err != nil) != tt.wantErr {
				t.Errorf("FetchChecksum() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("FetchChecksum() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVerifyChecksum(t *testing.T) {
	// Create a temp file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "testfile.txt")
	content := []byte("hello world")
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	// Calculate expected SHA256
	h := sha256.New()
	h.Write(content)
	expectedSum := fmt.Sprintf("%x", h.Sum(nil))

	tests := []struct {
		name     string
		expected string
		algo     string
		wantErr  bool
	}{
		{"Valid SHA256", expectedSum, "sha256", false},
		{"Invalid SHA256", "wrongsum", "sha256", true},
		{"Unsupported Algo", expectedSum, "md5", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyChecksum(filePath, tt.expected, tt.algo)
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyChecksum() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
