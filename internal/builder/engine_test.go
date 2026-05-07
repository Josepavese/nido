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
