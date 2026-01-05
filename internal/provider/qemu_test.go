package provider

import (
	"runtime"
	"strings"
	"testing"

	"github.com/Josepavese/nido/internal/config"
)

// TestBuildQemuArgs_CrossPlatform tests QEMU argument generation for all platforms
func TestBuildQemuArgs_CrossPlatform(t *testing.T) {
	p := &QemuProvider{
		RootDir: "/tmp/nido-test",
		Config:  &config.Config{},
	}

	tests := []struct {
		name          string
		goos          string
		expectedAccel []string
		expectedCPU   string
		shouldContain []string
	}{
		{
			name:          "Linux with KVM",
			goos:          "linux",
			expectedAccel: []string{"-enable-kvm"},
			expectedCPU:   "host",
			shouldContain: []string{"-enable-kvm", "-cpu", "host"},
		},
		{
			name:          "macOS with HVF",
			goos:          "darwin",
			expectedAccel: []string{"-accel", "hvf"},
			expectedCPU:   "host",
			shouldContain: []string{"-accel", "hvf", "-cpu", "host"},
		},
		{
			name:          "Windows with WHPX",
			goos:          "windows",
			expectedAccel: []string{"-accel", "whpx"},
			expectedCPU:   "host",
			shouldContain: []string{"-accel", "whpx", "-cpu", "host"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: We can't actually mock runtime.GOOS in tests,
			// but we can verify the logic by checking the current platform
			args := p.buildQemuArgs("test-vm", "/tmp/test.qcow2", 50022, "/tmp/run")

			// Verify common arguments are present
			if !contains(args, "-name") {
				t.Error("Missing -name argument")
			}
			if !contains(args, "test-vm") {
				t.Error("Missing VM name")
			}
			if !contains(args, "-m") {
				t.Error("Missing memory argument")
			}
			if !contains(args, "2048") {
				t.Error("Missing memory value")
			}

			// Verify platform-specific acceleration (only for current platform)
			if runtime.GOOS == tt.goos {
				for _, expected := range tt.shouldContain {
					if !contains(args, expected) {
						t.Errorf("Expected argument %q not found in: %v", expected, args)
					}
				}
			}

			// Verify network forwarding
			hasNetdev := false
			for _, arg := range args {
				if strings.Contains(arg, "hostfwd=tcp::50022-:22") {
					hasNetdev = true
					break
				}
			}
			if !hasNetdev {
				t.Error("Missing SSH port forwarding configuration")
			}

			// Verify QMP socket configuration
			hasQMP := false
			for i, arg := range args {
				if arg == "-qmp" && i+1 < len(args) {
					qmpArg := args[i+1]
					if runtime.GOOS == "windows" {
						if !strings.Contains(qmpArg, "tcp:") {
							t.Error("Windows should use TCP for QMP")
						}
					} else {
						if !strings.Contains(qmpArg, "unix:") {
							t.Error("Unix-like systems should use Unix sockets for QMP")
						}
					}
					hasQMP = true
					break
				}
			}
			if !hasQMP {
				t.Error("Missing QMP configuration")
			}
		})
	}
}

// TestBuildQemuArgs_CommonArguments verifies arguments common to all platforms
func TestBuildQemuArgs_CommonArguments(t *testing.T) {
	p := &QemuProvider{
		RootDir: "/tmp/nido-test",
		Config:  &config.Config{},
	}

	args := p.buildQemuArgs("test-vm", "/tmp/test.qcow2", 50022, "/tmp/run")

	requiredArgs := map[string]bool{
		"-name":      false,
		"-m":         false,
		"-drive":     false,
		"-daemonize": false,
		"-pidfile":   false,
		"-netdev":    false,
		"-device":    false,
		"-qmp":       false,
	}

	for _, arg := range args {
		if _, exists := requiredArgs[arg]; exists {
			requiredArgs[arg] = true
		}
	}

	for arg, found := range requiredArgs {
		if !found {
			t.Errorf("Required argument %q not found in QEMU args", arg)
		}
	}
}

// TestBuildQemuArgs_DiskPath verifies disk path is correctly formatted
func TestBuildQemuArgs_DiskPath(t *testing.T) {
	p := &QemuProvider{
		RootDir: "/tmp/nido-test",
		Config:  &config.Config{},
	}

	diskPath := "/path/to/vm.qcow2"
	args := p.buildQemuArgs("test-vm", diskPath, 50022, "/tmp/run")

	found := false
	for _, arg := range args {
		if strings.Contains(arg, diskPath) && strings.Contains(arg, "format=qcow2") {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Disk path %q not properly formatted in args: %v", diskPath, args)
	}
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
