package validators

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVMName(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"", true},
		{"a", false},
		{"my-vm", false},
		{"my_vm", false},
		{"my-vm-123", false},
		{"MyVM", false},
		{"-invalid", true},
		{"_invalid", true},
		{"has space", true},
		{"has.dot", true},
		{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", true}, // 34 chars
		{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa12", false},  // 32 chars
	}

	for _, tc := range tests {
		err := VMName(tc.input)
		if tc.wantErr && err == nil {
			t.Errorf("VMName(%q) expected error, got nil", tc.input)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("VMName(%q) unexpected error: %v", tc.input, err)
		}
	}
}

func TestPort(t *testing.T) {
	tests := []struct {
		input   int
		wantErr bool
	}{
		{0, true},
		{1, false},
		{80, false},
		{443, false},
		{22, false},
		{65535, false},
		{65536, true},
		{-1, true},
	}

	for _, tc := range tests {
		err := Port(tc.input)
		if tc.wantErr && err == nil {
			t.Errorf("Port(%d) expected error, got nil", tc.input)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("Port(%d) unexpected error: %v", tc.input, err)
		}
	}
}

func TestFilePath(t *testing.T) {
	// Empty is OK (optional)
	if err := FilePath(""); err != nil {
		t.Errorf("FilePath('') should be nil, got %v", err)
	}

	// Non-existent file
	if err := FilePath("/nonexistent/path/file.txt"); err == nil {
		t.Error("FilePath('/nonexistent/...') should error")
	}

	// Create temp file and test
	tmp := filepath.Join(os.TempDir(), "nido_test_file")
	os.WriteFile(tmp, []byte("test"), 0644)
	defer os.Remove(tmp)

	if err := FilePath(tmp); err != nil {
		t.Errorf("FilePath(%q) unexpected error: %v", tmp, err)
	}

	// Directory should fail
	if err := FilePath(os.TempDir()); err == nil {
		t.Error("FilePath(dir) should error")
	}
}

func TestNotEmpty(t *testing.T) {
	if err := NotEmpty(""); err == nil {
		t.Error("NotEmpty('') should error")
	}
	if err := NotEmpty("   "); err == nil {
		t.Error("NotEmpty('   ') should error")
	}
	if err := NotEmpty("hello"); err != nil {
		t.Errorf("NotEmpty('hello') unexpected error: %v", err)
	}
}

func TestConfigKey(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"", true},
		{"SSH_USER", false},
		{"BACKUP_DIR", false},
		{"A", false},
		{"lowercase", true},
		{"Mixed_Case", true},
		{"HAS-DASH", true},
		{"123_NUM", true},
	}

	for _, tc := range tests {
		err := ConfigKey(tc.input)
		if tc.wantErr && err == nil {
			t.Errorf("ConfigKey(%q) expected error, got nil", tc.input)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("ConfigKey(%q) unexpected error: %v", tc.input, err)
		}
	}
}
