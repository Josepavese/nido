package image

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

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
