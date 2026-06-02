package mcp

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	climeta "github.com/Josepavese/nido/internal/cli"
	"github.com/Josepavese/nido/internal/config"
	"github.com/Josepavese/nido/internal/provider"
)

type mockProvider struct {
	cfg                  config.Config
	cachePruneUnusedOnly bool
	cachePruneCalls      int
	spawnName            string
	spawnOpts            provider.VMOptions
}

func (m *mockProvider) Spawn(name string, opts provider.VMOptions) error {
	m.spawnName = name
	m.spawnOpts = opts
	return nil
}
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
	expected := map[string]bool{
		"nido_vm":        false,
		"nido_template":  false,
		"nido_image":     false,
		"nido_blueprint": false,
		"nido_system":    false,
	}

	for _, tool := range tools {
		name, _ := tool["name"].(string)
		if _, ok := expected[name]; ok {
			expected[name] = true
		}
		switch name {
		case "nido_vm":
			schema, _ := tool["inputSchema"].(map[string]interface{})
			props, _ := schema["properties"].(map[string]interface{})
			if _, ok := props["ssh_password"]; ok {
				t.Fatal("nido_vm must not expose ssh_password")
			}
		}
	}

	for name, found := range expected {
		if !found {
			t.Fatalf("%s not present in tool catalog", name)
		}
	}
}

func TestResourceAndPromptCatalogsExposeCompactSurface(t *testing.T) {
	if len(ResourcesCatalog()) != 10 {
		t.Fatalf("ResourcesCatalog() count = %d, want 10", len(ResourcesCatalog()))
	}
	if len(ResourceTemplatesCatalog()) != 3 {
		t.Fatalf("ResourceTemplatesCatalog() count = %d, want 3", len(ResourceTemplatesCatalog()))
	}
	if len(PromptsCatalog()) != 1 {
		t.Fatalf("PromptsCatalog() count = %d, want 1", len(PromptsCatalog()))
	}
}

func TestSystemToolCatalogCoversCLISystemActions(t *testing.T) {
	tools := ToolsCatalog()
	var actions []string
	for _, tool := range tools {
		if tool["name"] != "nido_system" {
			continue
		}
		schema, _ := tool["inputSchema"].(map[string]interface{})
		props, _ := schema["properties"].(map[string]interface{})
		action, _ := props["action"].(map[string]interface{})
		actions = stringEnum(action["enum"])
	}
	seen := map[string]bool{}
	for _, action := range actions {
		seen[action] = true
	}
	for _, want := range []string{"doctor", "version", "update_check", "update", "config_get", "config_set", "accel_list", "register", "completion", "build_image", "uninstall"} {
		if !seen[want] {
			t.Fatalf("nido_system action %q not present in tool catalog", want)
		}
	}
}

func TestMCPActionsCoverCLIManifest(t *testing.T) {
	manifest, err := climeta.LoadManifest()
	if err != nil {
		t.Fatal(err)
	}
	toolActions := toolActionsByName(ToolsCatalog())
	coverage := map[string]struct {
		tool   string
		action string
	}{
		"vm.list":                      {"nido_vm", "list"},
		"vm.info":                      {"nido_vm", "info"},
		"vm.spawn":                     {"nido_vm", "create"},
		"vm.start":                     {"nido_vm", "start"},
		"vm.stop":                      {"nido_vm", "stop"},
		"vm.ssh":                       {"nido_vm", "ssh"},
		"vm.delete":                    {"nido_vm", "delete"},
		"vm.prune":                     {"nido_vm", "prune"},
		"template.list":                {"nido_template", "list"},
		"template.create":              {"nido_template", "create"},
		"template.delete":              {"nido_template", "delete"},
		"cache.list":                   {"nido_image", "cache_list"},
		"cache.info":                   {"nido_image", "cache_info"},
		"cache.remove":                 {"nido_image", "cache_remove"},
		"cache.prune":                  {"nido_image", "cache_prune"},
		"images.list":                  {"nido_image", "list"},
		"images.pull":                  {"nido_image", "pull"},
		"images.info":                  {"nido_image", "info"},
		"images.remove":                {"nido_image", "remove"},
		"images.update":                {"nido_image", "refresh_catalog"},
		"blueprint.list":               {"nido_blueprint", "list"},
		"blueprint.info":               {"nido_blueprint", "info"},
		"blueprint.build":              {"nido_blueprint", "build"},
		"build":                        {"nido_blueprint", "build"},
		"system.doctor":                {"nido_system", "doctor"},
		"system.accel.list":            {"nido_system", "accel_list"},
		"system.config":                {"nido_system", "config_get"},
		"system.config.set":            {"nido_system", "config_set"},
		"system.register":              {"nido_system", "register"},
		"system.version":               {"nido_system", "version"},
		"system.update":                {"nido_system", "update"},
		"system.uninstall":             {"nido_system", "uninstall"},
		"system.completion.bash":       {"nido_system", "completion"},
		"system.completion.zsh":        {"nido_system", "completion"},
		"system.completion.fish":       {"nido_system", "completion"},
		"system.completion.powershell": {"nido_system", "completion"},
	}
	exceptions := map[string]string{
		"ui.gui":          "interactive TUI, not an agent MCP operation",
		"system.mcp":      "MCP transport entrypoint",
		"system.mcp_help": "MCP guide is exposed by HelpPayload",
	}

	for _, action := range manifestActions(manifest.Commands) {
		if _, ok := exceptions[action]; ok {
			continue
		}
		mapped, ok := coverage[action]
		if !ok {
			t.Fatalf("CLI action %q has no MCP coverage mapping", action)
		}
		if !toolActions[mapped.tool][mapped.action] {
			t.Fatalf("CLI action %q maps to missing MCP action %s.%s", action, mapped.tool, mapped.action)
		}
	}
}

func manifestActions(commands []climeta.CommandSpec) []string {
	var actions []string
	for _, cmd := range commands {
		if cmd.Action != "" {
			actions = append(actions, cmd.Action)
		}
		actions = append(actions, manifestActions(cmd.Commands)...)
	}
	return actions
}

func toolActionsByName(tools []map[string]interface{}) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	for _, tool := range tools {
		name, _ := tool["name"].(string)
		schema, _ := tool["inputSchema"].(map[string]interface{})
		props, _ := schema["properties"].(map[string]interface{})
		action, _ := props["action"].(map[string]interface{})
		out[name] = map[string]bool{}
		for _, value := range stringEnum(action["enum"]) {
			out[name][value] = true
		}
	}
	return out
}

func stringEnum(raw interface{}) []string {
	switch values := raw.(type) {
	case []string:
		return values
	case []interface{}:
		out := make([]string, 0, len(values))
		for _, value := range values {
			if s, ok := value.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func TestHelpPayloadIncludesGuideSections(t *testing.T) {
	payload := HelpPayload()
	for _, key := range []string{"summary", "transport", "discovery", "usage_rules", "examples", "tools", "resources", "resource_templates", "prompts"} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("HelpPayload missing key %q", key)
		}
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
		Name: "nido_image",
	}
	args, err := json.Marshal(map[string]interface{}{"action": "cache_prune", "unused_only": true})
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
	if !strings.Contains(string(out), "\\\"removed_count\\\":7") {
		t.Fatalf("unexpected MCP response: %s", string(out))
	}
}

func TestVMCreateFromBuiltBlueprintImageAppliesBlueprintMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	cwd := filepath.Join(tmpDir, "project")
	imageDir := filepath.Join(tmpDir, "images")
	if err := os.MkdirAll(filepath.Join(cwd, "registry", "blueprints"), 0o755); err != nil {
		t.Fatalf("mkdir blueprint registry: %v", err)
	}
	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		t.Fatalf("mkdir image dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(imageDir, "windows-fixture.qcow2"), []byte("qcow2"), 0o644); err != nil {
		t.Fatalf("write image fixture: %v", err)
	}
	data := []byte(`name: windows-fixture
description: Windows fixture blueprint
version: "0.1.0"
ssh_user: vmuser
ssh_password: nido
iso_url: https://example.test/windows.iso
iso_checksum: ""
build_specs:
  cpu: 1
  memory: 512M
  timeout: 1m
output_image: windows-fixture.qcow2
output_size: 1G
scripts:
  Autounattend.xml: "<unattend/>"
  windows-setup-openssh.ps1: "Write-Host ssh"
`)
	if err := os.WriteFile(filepath.Join(cwd, "registry", "blueprints", "windows-fixture.yaml"), data, 0o644); err != nil {
		t.Fatalf("write blueprint fixture: %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWD)

	p := &mockProvider{cfg: config.Config{ImageDir: imageDir}}
	s := NewServer(p)
	raw, err := json.Marshal(map[string]interface{}{
		"action": "create",
		"name":   "win-vm",
		"image":  "windows-fixture",
		"web":    true,
	})
	if err != nil {
		t.Fatal(err)
	}

	payload, err := s.callVMTool(raw)
	if err != nil {
		t.Fatalf("callVMTool(create) failed: %v", err)
	}
	got, _ := payload.(map[string]interface{})
	if got["source"] != "image windows-fixture" {
		t.Fatalf("source = %v, want image windows-fixture", got["source"])
	}
	if p.spawnName != "win-vm" {
		t.Fatalf("spawn name = %q, want win-vm", p.spawnName)
	}
	if p.spawnOpts.DiskPath != filepath.Join(imageDir, "windows-fixture.qcow2") {
		t.Fatalf("disk path = %q", p.spawnOpts.DiskPath)
	}
	if p.spawnOpts.SSHUser != "vmuser" || p.spawnOpts.SSHPassword != "nido" {
		t.Fatalf("unexpected SSH metadata: user=%q password=%q", p.spawnOpts.SSHUser, p.spawnOpts.SSHPassword)
	}
	if p.spawnOpts.SeedFiles["windows-setup-openssh.ps1"] != "Write-Host ssh" {
		t.Fatalf("missing blueprint support seed files: %#v", p.spawnOpts.SeedFiles)
	}
	if _, ok := p.spawnOpts.SeedFiles["Autounattend.xml"]; ok {
		t.Fatalf("root answer file should not be copied to spawn seed: %#v", p.spawnOpts.SeedFiles)
	}
	if len(p.spawnOpts.Forwarding) != 2 {
		t.Fatalf("web forwarding count = %d, want 2", len(p.spawnOpts.Forwarding))
	}
}
