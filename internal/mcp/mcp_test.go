package mcp

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Josepavese/nido/internal/config"
	"github.com/Josepavese/nido/internal/provider"
)

type mockProvider struct {
	cfg                  config.Config
	cachePruneUnusedOnly bool
	cachePruneCalls      int
}

func (m *mockProvider) Spawn(name string, opts provider.VMOptions) error               { return nil }
func (m *mockProvider) Start(name string, opts provider.VMOptions) error               { return nil }
func (m *mockProvider) Stop(name string, graceful bool) error                          { return nil }
func (m *mockProvider) Delete(name string) error                                       { return nil }
func (m *mockProvider) List() ([]provider.VMStatus, error)                             { return nil, nil }
func (m *mockProvider) Info(name string) (provider.VMDetail, error)                    { return provider.VMDetail{}, nil }
func (m *mockProvider) GetConfig() config.Config                                       { return m.cfg }
func (m *mockProvider) CreateDisk(name string, size string, templatePath string) error { return nil }
func (m *mockProvider) CreateTemplate(vmName string, templateName string) (string, error) {
	return "", nil
}
func (m *mockProvider) ListTemplates() ([]string, error)                  { return nil, nil }
func (m *mockProvider) ListImages() ([]string, error)                     { return nil, nil }
func (m *mockProvider) ListAccelerators() ([]provider.Accelerator, error) { return nil, nil }
func (m *mockProvider) GetUsedBackingFiles() ([]string, error)            { return nil, nil }
func (m *mockProvider) DeleteTemplate(name string, force bool) error      { return nil }
func (m *mockProvider) Prune() (int, error)                               { return 0, nil }
func (m *mockProvider) ListCachedImages() ([]provider.CachedImage, error) { return nil, nil }
func (m *mockProvider) CacheInfo() (provider.CacheInfoResult, error) {
	return provider.CacheInfoResult{}, nil
}
func (m *mockProvider) CacheRemove(name, version string) error { return nil }
func (m *mockProvider) SSHCommand(name string) (string, error) { return "", nil }
func (m *mockProvider) PortForward(name string, pf provider.PortForward) (provider.PortForward, error) {
	return pf, nil
}
func (m *mockProvider) PortUnforward(name string, guestPort int, protocol string) error  { return nil }
func (m *mockProvider) PortList(name string) ([]provider.PortForward, error)             { return nil, nil }
func (m *mockProvider) UpdateConfig(name string, updates provider.VMConfigUpdates) error { return nil }
func (m *mockProvider) Doctor() []string                                                 { return []string{"ok"} }
func (m *mockProvider) CachePrune(unusedOnly bool) (int, int64, error) {
	m.cachePruneCalls++
	m.cachePruneUnusedOnly = unusedOnly
	return 7, 1234, nil
}

func TestToolsCatalogIncludesExpectedToolsAndOmitsPassword(t *testing.T) {
	tools := ToolsCatalog()
	var foundConfigGet bool
	var foundDoctor bool

	for _, tool := range tools {
		name, _ := tool["name"].(string)
		switch name {
		case "vm_config_get":
			foundConfigGet = true
		case "vm_doctor":
			foundDoctor = true
		case "vm_config_update":
			schema, _ := tool["inputSchema"].(map[string]interface{})
			props, _ := schema["properties"].(map[string]interface{})
			if _, ok := props["ssh_password"]; ok {
				t.Fatal("vm_config_update must not expose ssh_password")
			}
		}
	}

	if !foundConfigGet {
		t.Fatal("vm_config_get not present in tool catalog")
	}
	if !foundDoctor {
		t.Fatal("vm_doctor not present in tool catalog")
	}
}

func TestServerImageDirUsesProviderConfig(t *testing.T) {
	p := &mockProvider{
		cfg: config.Config{
			ImageDir: filepath.Join(t.TempDir(), "custom-images"),
		},
	}
	s := NewServer(p)

	if got := s.imageDir(); got != p.cfg.ImageDir {
		t.Fatalf("imageDir() = %q, want %q", got, p.cfg.ImageDir)
	}
}

func TestHandleToolsCallCachePruneDelegatesToProvider(t *testing.T) {
	p := &mockProvider{}
	s := NewServer(p)

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      42,
		Method:  "tools/call",
	}

	params := CallParams{
		Name: "vm_cache_prune",
	}
	args, err := json.Marshal(map[string]bool{"unused_only": true})
	if err != nil {
		t.Fatal(err)
	}
	params.Arguments = args
	req.Params, err = json.Marshal(params)
	if err != nil {
		t.Fatal(err)
	}

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	s.handleToolsCall(req)

	_ = w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}

	if p.cachePruneCalls != 1 {
		t.Fatalf("CachePrune called %d times, want 1", p.cachePruneCalls)
	}
	if !p.cachePruneUnusedOnly {
		t.Fatal("CachePrune should receive unused_only=true")
	}
	if !strings.Contains(string(out), "Pruned 7 cached image(s)") {
		t.Fatalf("unexpected MCP response: %s", string(out))
	}
}
