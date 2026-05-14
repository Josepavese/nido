package builder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSafeJoinRejectsPathEscape(t *testing.T) {
	root := t.TempDir()
	cases := []string{
		"../escape.qcow2",
		filepath.Join("..", "escape.qcow2"),
		filepath.Join("nested", "..", "..", "escape.qcow2"),
	}

	for _, tc := range cases {
		t.Run(tc, func(t *testing.T) {
			if _, err := safeJoin(root, tc); err == nil {
				t.Fatalf("safeJoin(%q) succeeded, want error", tc)
			}
		})
	}
}

func TestLoadBlueprintRejectsExternalScriptEscape(t *testing.T) {
	dir := t.TempDir()
	blueprint := `
name: test
description: test
version: "0.0.1"
iso_url: https://example.test/test.iso
output_image: test.qcow2
output_size: 1G
build_specs:
  cpu: 1
  memory: 512M
  timeout: 1m
scripts:
  setup.sh: "@../outside.sh"
`
	path := filepath.Join(dir, "blueprint.yaml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(blueprint)), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadBlueprint(path); err == nil {
		t.Fatal("LoadBlueprint succeeded for path-escaping external script, want error")
	}
}

func TestVerifyOptionalChecksumRejectsMismatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "asset.iso")
	if err := os.WriteFile(path, []byte("bad"), 0644); err != nil {
		t.Fatal(err)
	}

	err := verifyOptionalChecksum(path, "0000000000000000000000000000000000000000000000000000000000000000")
	if err == nil {
		t.Fatal("verifyOptionalChecksum accepted mismatched checksum")
	}
}

func TestValidateDownloadURLRejectsPlainHTTPRemote(t *testing.T) {
	if err := validateDownloadURL("http://example.com/asset.iso"); err == nil {
		t.Fatal("validateDownloadURL accepted remote plain HTTP")
	}
	if err := validateDownloadURL("http://127.0.0.1/asset.iso"); err != nil {
		t.Fatalf("validateDownloadURL rejected loopback HTTP: %v", err)
	}
}

func TestCacheAssetNameSanitizesWindowsFwlink(t *testing.T) {
	got, err := cacheAssetName("", "https://go.microsoft.com/fwlink/?linkid=2334167&clcid=0x409&culture=en-us&country=us", "installer.iso")
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(got, `<>:"/\|?*`) {
		t.Fatalf("cache asset name contains Windows-invalid characters: %q", got)
	}
	if filepath.Ext(got) != ".iso" {
		t.Fatalf("cache asset name should keep an ISO extension, got %q", got)
	}
}

func TestListBlueprintsReportsBuiltOutputAndDeduplicates(t *testing.T) {
	cwd := t.TempDir()
	nidoDir := t.TempDir()
	imageDir := filepath.Join(t.TempDir(), "images")
	if err := os.MkdirAll(filepath.Join(cwd, "registry", "blueprints"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(nidoDir, "blueprints"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		t.Fatal(err)
	}

	projectPath := filepath.Join(cwd, "registry", "blueprints", "fixture.yaml")
	userPath := filepath.Join(nidoDir, "blueprints", "fixture.yaml")
	writeTestBlueprint(t, projectPath, "fixture", "project fixture")
	writeTestBlueprint(t, userPath, "fixture", "user fixture")
	if err := os.WriteFile(filepath.Join(imageDir, "fixture.qcow2"), []byte("qcow2"), 0644); err != nil {
		t.Fatal(err)
	}

	blueprints, err := ListBlueprints(cwd, nidoDir, imageDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(blueprints) != 1 {
		t.Fatalf("ListBlueprints returned %d entries, want 1: %#v", len(blueprints), blueprints)
	}
	got := blueprints[0]
	if got.Name != "fixture" || got.Source != "project" || got.Path != projectPath {
		t.Fatalf("unexpected blueprint summary: %#v", got)
	}
	if !got.Built || got.OutputTag != "fixture" || got.OutputPath != filepath.Join(imageDir, "fixture.qcow2") {
		t.Fatalf("unexpected output state: %#v", got)
	}
}

func writeTestBlueprint(t *testing.T, path, name, description string) {
	t.Helper()
	data := `name: ` + name + `
description: ` + description + `
version: "0.1.0"
iso_url: https://example.test/fixture.iso
iso_checksum: ""
build_specs:
  cpu: 1
  memory: 512M
  timeout: 1m
output_image: ` + name + `.qcow2
output_size: 1G
scripts:
  setup.cmd: echo ok
`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadRegistryWindowsBlueprints(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	tests := []struct {
		file          string
		driverProfile string
	}{
		{file: "windows-11-eval.yaml", driverProfile: `\w11\`},
		{file: "windows-11-iot-ltsc-eval.yaml", driverProfile: `\w11\`},
		{file: "windows-server-2022-core-eval.yaml", driverProfile: `\2k22\`},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			bp, err := LoadBlueprint(filepath.Join(repoRoot, "registry", "blueprints", tt.file))
			if err != nil {
				t.Fatal(err)
			}
			if bp.SSHUser != "vmuser" || bp.SSHPassword != "nido" {
				t.Fatalf("unexpected blueprint SSH metadata: user=%q password=%q", bp.SSHUser, bp.SSHPassword)
			}
			content := bp.Scripts["autounattend.xml"]
			if content == "" {
				t.Fatal("missing autounattend.xml script")
			}
			if strings.Contains(content, "{{windows_driver_profile}}") {
				t.Fatal("windows_driver_profile placeholder was not expanded")
			}
			if strings.Contains(content, "{{windows_image_index}}") {
				t.Fatal("windows_image_index placeholder was not expanded")
			}
			if !strings.Contains(content, tt.driverProfile) {
				t.Fatalf("autounattend.xml missing driver profile %q", tt.driverProfile)
			}
			if !strings.Contains(content, "<Key>/IMAGE/INDEX</Key>") || !strings.Contains(content, "<Value>1</Value>") {
				t.Fatalf("autounattend.xml missing fixed install image index")
			}
			offlineStart := strings.Index(content, `<settings pass="offlineServicing">`)
			specializeStart := strings.Index(content, `<settings pass="specialize">`)
			if offlineStart == -1 || specializeStart == -1 || offlineStart >= specializeStart {
				t.Fatal("autounattend.xml missing expected offlineServicing block")
			}
			offlineServicing := content[offlineStart:specializeStart]
			if strings.Contains(offlineServicing, "<Order>1</Order>") {
				t.Fatal("autounattend.xml contains unsupported offlineServicing PathAndCredentials Order")
			}
			if !strings.Contains(content, `Microsoft-Windows-International-Core`) || !strings.Contains(content, "<AutoLogon>") {
				t.Fatal("autounattend.xml missing oobeSystem locale or autologon settings")
			}
			oobeStart := strings.Index(content, `<settings pass="oobeSystem">`)
			if specializeStart == -1 || oobeStart == -1 || specializeStart >= oobeStart {
				t.Fatal("autounattend.xml missing expected specialize block")
			}
			specialize := content[specializeStart:oobeStart]
			if !strings.Contains(specialize, "Microsoft-Windows-Deployment") ||
				!strings.Contains(specialize, "windows-setup-openssh.ps1") {
				t.Fatal("autounattend.xml must install and enable OpenSSH Server during specialize")
			}
			sshScript := bp.Scripts["windows-setup-openssh.ps1"]
			if !strings.Contains(sshScript, "OpenSSH.Server*") ||
				!strings.Contains(sshScript, "Win32-OpenSSH/releases/latest/download/OpenSSH-Win64.zip") ||
				!strings.Contains(sshScript, "Set-Service -Name sshd -StartupType Automatic") {
				t.Fatal("windows OpenSSH setup script missing capability, archive fallback, or service enablement")
			}
			if strings.Contains(content, "<FirstLogonCommands>") {
				t.Fatal("autounattend.xml must not depend on FirstLogonCommands for SSH provisioning")
			}
		})
	}
}

func TestRegistryWindowsAutounattendXMLIsWellFormed(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	data, err := os.ReadFile(filepath.Join(repoRoot, "registry", "blueprints", "shared", "windows-autounattend.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := validateScript("autounattend.xml", string(data)); err != nil {
		t.Fatal(err)
	}
}

func TestInstallerAccelerationArgs(t *testing.T) {
	if args, cpu, ok := installerAccelerationArgs("windows"); !ok || cpu != "host" || strings.Join(args, " ") != "-accel whpx" {
		t.Fatalf("windows acceleration mismatch: args=%v cpu=%s ok=%v", args, cpu, ok)
	}
	if args, cpu, ok := installerAccelerationArgs("darwin"); !ok || cpu != "host" || strings.Join(args, " ") != "-accel hvf" {
		t.Fatalf("darwin acceleration mismatch: args=%v cpu=%s ok=%v", args, cpu, ok)
	}
}

func TestCreateSeedISOFallsBackToBundledWriter(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	tmp := t.TempDir()
	source := filepath.Join(tmp, "source")
	if err := os.Mkdir(source, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "autounattend.xml"), []byte("<unattend/>"), 0644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(tmp, "seed.iso")
	if err := createSeedISO(out, source, "OEMDRV"); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(out); err != nil || info.Size() == 0 {
		t.Fatalf("seed ISO was not created: info=%v err=%v", info, err)
	}
}
