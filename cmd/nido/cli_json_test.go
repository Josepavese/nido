package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Josepavese/nido/internal/config"
	"github.com/Josepavese/nido/internal/provider"
)

func TestJSONCommandsProduceSingleJSONAndNoStderr(t *testing.T) {
	app := testAppContext(t)

	cases := [][]string{
		{"ls", "--json"},
		{"info", "vm-a", "--json"},
		{"template", "list", "--json"},
		{"cache", "info", "--json"},
		{"doctor", "--json"},
		{"config", "--json"},
		{"version", "--json"},
		{"register", "--json"},
	}

	for _, args := range cases {
		t.Run(args[0], func(t *testing.T) {
			root, err := newRootCommand(app)
			if err != nil {
				t.Fatalf("newRootCommand failed: %v", err)
			}
			stdout, stderr := captureProcessIO(t, func() {
				root.SetArgs(args)
				if err := root.Execute(); err != nil {
					t.Fatalf("Execute(%v) failed: %v", args, err)
				}
			})

			if stderr != "" {
				t.Fatalf("expected empty stderr for %v, got %q", args, stderr)
			}
			if count := bytes.Count([]byte(stdout), []byte("\n")); count != 1 {
				t.Fatalf("expected exactly one JSON line for %v, got %d lines: %q", args, count, stdout)
			}

			var payload map[string]any
			if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
				t.Fatalf("stdout is not valid JSON for %v: %v\n%s", args, err, stdout)
			}
		})
	}
}

func TestHumanOutputUsesUnifiedStyle(t *testing.T) {
	app := testAppContext(t)
	root, err := newRootCommand(app)
	if err != nil {
		t.Fatalf("newRootCommand failed: %v", err)
	}

	stdout, stderr := captureProcessIO(t, func() {
		root.SetArgs([]string{"config"})
		if err := root.Execute(); err != nil {
			t.Fatalf("Execute(config) failed: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, want := range []string{"NIDO", "Backup Dir", "SSH User", "Image Dir"} {
		if !bytes.Contains([]byte(stdout), []byte(want)) {
			t.Fatalf("expected %q in styled output:\n%s", want, stdout)
		}
	}
}

func TestHumanCommandsExecuteWithUnifiedOutput(t *testing.T) {
	app := testAppContext(t)
	cases := []struct {
		args []string
		want []string
	}{
		{args: []string{"ls"}, want: []string{"NAME", "STATE", "vm-a"}},
		{args: []string{"info", "vm-a"}, want: []string{"NIDO", "VM DETAILS", "127.0.0.1"}},
		{args: []string{"template", "list"}, want: []string{"TEMPLATES", "base-template"}},
		{args: []string{"cache", "info"}, want: []string{"CACHE STATISTICS", "Total Images"}},
		{args: []string{"doctor"}, want: []string{"SYSTEM DIAGNOSTICS", "Diagnostics completed."}},
	}

	for _, tc := range cases {
		t.Run(tc.args[0], func(t *testing.T) {
			root, err := newRootCommand(app)
			if err != nil {
				t.Fatalf("newRootCommand failed: %v", err)
			}
			stdout, stderr := captureProcessIO(t, func() {
				root.SetArgs(tc.args)
				if err := root.Execute(); err != nil {
					t.Fatalf("Execute(%v) failed: %v", tc.args, err)
				}
			})
			if stderr != "" {
				t.Fatalf("expected empty stderr, got %q", stderr)
			}
			for _, want := range tc.want {
				if !bytes.Contains([]byte(stdout), []byte(want)) {
					t.Fatalf("expected %q in output for %v:\n%s", want, tc.args, stdout)
				}
			}
		})
	}
}

func TestImageInfoAndRemoveJSONStayClean(t *testing.T) {
	app := testAppContext(t)
	writeCatalogFixture(t, app.Cwd)
	diskPath := filepath.Join(app.ImageDir(), "ubuntu-24.04.qcow2")
	if err := os.WriteFile(diskPath, []byte("qcow2"), 0o644); err != nil {
		t.Fatalf("write disk fixture: %v", err)
	}

	stdout, stderr := captureProcessIO(t, func() {
		cmdImageInfo(app.ImageDir(), []string{"ubuntu:24.04"}, true)
	})
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	assertSingleJSON(t, stdout)

	stdout, stderr = captureProcessIO(t, func() {
		cmdImageRemove(app.ImageDir(), fakeProvider{}, []string{"ubuntu:24.04"}, true)
	})
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	assertSingleJSON(t, stdout)
}

func testAppContext(t *testing.T) *appContext {
	t.Helper()
	nidoDir := t.TempDir()
	cfg := &config.Config{
		BackupDir:      filepath.Join(nidoDir, "backups"),
		SSHUser:        "vmuser",
		ImageDir:       filepath.Join(nidoDir, "images"),
		LinkedClones:   true,
		PortRangeStart: 30000,
		PortRangeEnd:   32767,
	}
	if err := os.MkdirAll(cfg.BackupDir, 0o755); err != nil {
		t.Fatalf("mkdir backup dir: %v", err)
	}
	if err := os.MkdirAll(cfg.ImageDir, 0o755); err != nil {
		t.Fatalf("mkdir image dir: %v", err)
	}
	return &appContext{
		NidoDir:    nidoDir,
		Cwd:        t.TempDir(),
		ConfigPath: filepath.Join(nidoDir, "config.env"),
		Config:     cfg,
		Provider:   fakeProvider{},
	}
}

type fakeProvider struct{}

func (fakeProvider) Spawn(name string, opts provider.VMOptions) error { return nil }
func (fakeProvider) Start(name string, opts provider.VMOptions) error { return nil }
func (fakeProvider) Stop(name string, graceful bool) error            { return nil }
func (fakeProvider) Delete(name string) error                         { return nil }
func (fakeProvider) List() ([]provider.VMStatus, error) {
	return []provider.VMStatus{{Name: "vm-a", State: "running", PID: 123, SSHPort: 50022, VNCPort: 59000, SSHUser: "vmuser"}}, nil
}
func (fakeProvider) Info(name string) (provider.VMDetail, error) {
	return provider.VMDetail{
		Name: name, State: "running", IP: "127.0.0.1", SSHUser: "vmuser", SSHPort: 50022, VNCPort: 59000, MemoryMB: 2048, VCPUs: 2,
	}, nil
}
func (fakeProvider) GetConfig() config.Config { return config.Config{} }
func (fakeProvider) CreateDisk(name string, size string, templatePath string) error {
	return nil
}
func (fakeProvider) CreateTemplate(vmName string, templateName string) (string, error) {
	return "/tmp/" + templateName + ".compact.qcow2", nil
}
func (fakeProvider) ListTemplates() ([]string, error)                  { return []string{"base-template"}, nil }
func (fakeProvider) ListImages() ([]string, error)                     { return []string{"ubuntu:24.04"}, nil }
func (fakeProvider) ListAccelerators() ([]provider.Accelerator, error) { return nil, nil }
func (fakeProvider) GetUsedBackingFiles() ([]string, error)            { return nil, nil }
func (fakeProvider) DeleteTemplate(name string, force bool) error      { return nil }
func (fakeProvider) Prune() (int, error)                               { return 1, nil }
func (fakeProvider) ListCachedImages() ([]provider.CachedImage, error) {
	return []provider.CachedImage{{Name: "ubuntu", Version: "24.04", Size: "1.2 GB"}}, nil
}
func (fakeProvider) CacheInfo() (provider.CacheInfoResult, error) {
	return provider.CacheInfoResult{Count: 1, TotalSize: "1.2 GB"}, nil
}
func (fakeProvider) CachePrune(unusedOnly bool) (int, int64, error) { return 1, 1024, nil }
func (fakeProvider) CacheRemove(name, version string) error         { return nil }
func (fakeProvider) SSHCommand(name string) (string, error) {
	return "ssh -p 50022 vmuser@127.0.0.1", nil
}
func (fakeProvider) PortForward(name string, pf provider.PortForward) (provider.PortForward, error) {
	return pf, nil
}
func (fakeProvider) PortUnforward(name string, guestPort int, protocol string) error  { return nil }
func (fakeProvider) PortList(name string) ([]provider.PortForward, error)             { return nil, nil }
func (fakeProvider) UpdateConfig(name string, updates provider.VMConfigUpdates) error { return nil }
func (fakeProvider) Doctor() []string {
	return []string{"Binary: QEMU [PASS] /usr/bin/qemu-system-x86_64"}
}

func captureProcessIO(t *testing.T, fn func()) (string, string) {
	t.Helper()

	oldStdout := os.Stdout
	oldStderr := os.Stderr
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("stderr pipe: %v", err)
	}
	os.Stdout = stdoutW
	os.Stderr = stderrW

	stdoutBuf := &bytes.Buffer{}
	stderrBuf := &bytes.Buffer{}
	doneOut := make(chan struct{})
	doneErr := make(chan struct{})
	go func() {
		_, _ = stdoutBuf.ReadFrom(stdoutR)
		close(doneOut)
	}()
	go func() {
		_, _ = stderrBuf.ReadFrom(stderrR)
		close(doneErr)
	}()

	fn()

	_ = stdoutW.Close()
	_ = stderrW.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr
	<-doneOut
	<-doneErr
	_ = stdoutR.Close()
	_ = stderrR.Close()
	return stdoutBuf.String(), stderrBuf.String()
}

func assertSingleJSON(t *testing.T, stdout string) {
	t.Helper()
	if count := bytes.Count([]byte(stdout), []byte("\n")); count != 1 {
		t.Fatalf("expected exactly one JSON line, got %d lines: %q", count, stdout)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
}

func writeCatalogFixture(t *testing.T, cwd string) {
	t.Helper()
	regDir := filepath.Join(cwd, "registry")
	if err := os.MkdirAll(regDir, 0o755); err != nil {
		t.Fatalf("mkdir registry: %v", err)
	}
	data := `{
  "schema_version": "1.0",
  "updated_at": "2026-01-01T00:00:00Z",
  "images": [
    {
      "name": "ubuntu",
      "registry": "official",
      "description": "Ubuntu image",
      "ssh_user": "vmuser",
      "versions": [
        {
          "version": "24.04",
          "aliases": ["latest"],
          "arch": "amd64",
          "url": "https://example.test/ubuntu.qcow2",
          "checksum_type": "sha256",
          "checksum": "abc",
          "size_bytes": 12345,
          "format": "qcow2"
        }
      ]
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(regDir, "images.json"), []byte(data), 0o644); err != nil {
		t.Fatalf("write catalog: %v", err)
	}
}
