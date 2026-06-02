package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractBinaryFromTarGz(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "nido-linux-amd64.tar.gz")
	destPath := filepath.Join(tmpDir, "nido")
	want := []byte("test-binary-tar")

	if err := writeTarGzWithFile(archivePath, "nido-linux-amd64/nido", want); err != nil {
		t.Fatalf("failed to create tar.gz fixture: %v", err)
	}

	if err := extractBinaryFromTarGz(archivePath, destPath, "nido"); err != nil {
		t.Fatalf("extractBinaryFromTarGz failed: %v", err)
	}

	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read extracted binary: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("extracted content = %q, want %q", got, want)
	}
}

func TestExtractBinaryFromZip(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "nido-windows-amd64.zip")
	destPath := filepath.Join(tmpDir, "nido.exe")
	want := []byte("test-binary-zip")

	if err := writeZipWithFile(archivePath, "nido-windows-amd64/nido.exe", want); err != nil {
		t.Fatalf("failed to create zip fixture: %v", err)
	}

	if err := extractBinaryFromZip(archivePath, destPath, "nido.exe"); err != nil {
		t.Fatalf("extractBinaryFromZip failed: %v", err)
	}

	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read extracted binary: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("extracted content = %q, want %q", got, want)
	}
}

func TestExtractBinaryFromReleaseAssetRejectsUnsupportedFormat(t *testing.T) {
	err := extractBinaryFromReleaseAsset("release.bin", "ignored", "nido")
	if err == nil {
		t.Fatal("expected unsupported format error, got nil")
	}
}

func TestExtractRegistryFromTarGz(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "nido-linux-amd64.tar.gz")
	destDir := filepath.Join(tmpDir, "registry")

	if err := writeTarGzFiles(archivePath, map[string][]byte{
		"nido-linux-amd64/nido": []byte("binary"),
		"nido-linux-amd64/registry/blueprints/windows-11-iot-ltsc-eval.yaml":    []byte("fixed-blueprint"),
		"nido-linux-amd64/registry/blueprints/shared/windows-autounattend.xml":  []byte("<unattend/>"),
		"nido-linux-amd64/registry/blueprints/shared/windows-setup-openssh.ps1": []byte("script"),
	}); err != nil {
		t.Fatalf("failed to create tar.gz fixture: %v", err)
	}

	if err := extractRegistryFromReleaseAsset(archivePath, destDir); err != nil {
		t.Fatalf("extractRegistryFromReleaseAsset failed: %v", err)
	}

	assertFileContent(t, filepath.Join(destDir, "blueprints", "windows-11-iot-ltsc-eval.yaml"), "fixed-blueprint")
	assertFileContent(t, filepath.Join(destDir, "blueprints", "shared", "windows-autounattend.xml"), "<unattend/>")
	assertFileContent(t, filepath.Join(destDir, "blueprints", "shared", "windows-setup-openssh.ps1"), "script")
	if _, err := os.Stat(filepath.Join(destDir, "nido")); !os.IsNotExist(err) {
		t.Fatalf("registry extraction copied non-registry binary; stat err = %v", err)
	}
}

func TestExtractRegistryFromZip(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "nido-windows-amd64.zip")
	destDir := filepath.Join(tmpDir, "registry")

	if err := writeZipFiles(archivePath, map[string][]byte{
		"nido-windows-amd64/nido.exe":                                             []byte("binary"),
		"nido-windows-amd64/registry/blueprints/windows-11-iot-ltsc-eval.yaml":    []byte("fixed-blueprint"),
		"nido-windows-amd64/registry/blueprints/shared/windows-autounattend.xml":  []byte("<unattend/>"),
		"nido-windows-amd64/registry/blueprints/shared/windows-setup-openssh.ps1": []byte("script"),
	}); err != nil {
		t.Fatalf("failed to create zip fixture: %v", err)
	}

	if err := extractRegistryFromReleaseAsset(archivePath, destDir); err != nil {
		t.Fatalf("extractRegistryFromReleaseAsset failed: %v", err)
	}

	assertFileContent(t, filepath.Join(destDir, "blueprints", "windows-11-iot-ltsc-eval.yaml"), "fixed-blueprint")
	assertFileContent(t, filepath.Join(destDir, "blueprints", "shared", "windows-autounattend.xml"), "<unattend/>")
	assertFileContent(t, filepath.Join(destDir, "blueprints", "shared", "windows-setup-openssh.ps1"), "script")
	if _, err := os.Stat(filepath.Join(destDir, "nido.exe")); !os.IsNotExist(err) {
		t.Fatalf("registry extraction copied non-registry binary; stat err = %v", err)
	}
}

func TestSyncBundledRegistryFromReleaseAssetBacksUpAndOverlays(t *testing.T) {
	tmpDir := t.TempDir()
	nidoDir := filepath.Join(tmpDir, ".nido")
	installedBlueprint := filepath.Join(nidoDir, "registry", "blueprints", "windows-11-iot-ltsc-eval.yaml")
	customFile := filepath.Join(nidoDir, "registry", "custom.yaml")
	if err := os.MkdirAll(filepath.Dir(installedBlueprint), 0o755); err != nil {
		t.Fatalf("failed to create installed registry: %v", err)
	}
	if err := os.WriteFile(installedBlueprint, []byte("stale-blueprint"), 0o644); err != nil {
		t.Fatalf("failed to write stale blueprint: %v", err)
	}
	if err := os.WriteFile(customFile, []byte("custom"), 0o644); err != nil {
		t.Fatalf("failed to write custom registry file: %v", err)
	}

	archivePath := filepath.Join(tmpDir, "nido-linux-amd64.tar.gz")
	if err := writeTarGzFiles(archivePath, map[string][]byte{
		"nido-linux-amd64/registry/blueprints/windows-11-iot-ltsc-eval.yaml": []byte("fixed-blueprint"),
	}); err != nil {
		t.Fatalf("failed to create tar.gz fixture: %v", err)
	}

	backupPath, err := syncBundledRegistryFromReleaseAsset(archivePath, nidoDir)
	if err != nil {
		t.Fatalf("syncBundledRegistryFromReleaseAsset failed: %v", err)
	}
	if backupPath == "" {
		t.Fatal("expected registry backup path, got empty string")
	}

	assertFileContent(t, installedBlueprint, "fixed-blueprint")
	assertFileContent(t, customFile, "custom")
	assertFileContent(t, filepath.Join(backupPath, "blueprints", "windows-11-iot-ltsc-eval.yaml"), "stale-blueprint")
	assertFileContent(t, filepath.Join(backupPath, "custom.yaml"), "custom")
}

func TestSyncBundledRegistryFromEmbeddedUpdatesStaleRegistry(t *testing.T) {
	tmpDir := t.TempDir()
	nidoDir := filepath.Join(tmpDir, ".nido")
	installedBlueprint := filepath.Join(nidoDir, "registry", "blueprints", "windows-11-iot-ltsc-eval.yaml")
	installedShared := filepath.Join(nidoDir, "registry", "blueprints", "shared", "windows-autounattend.xml")
	if err := os.MkdirAll(filepath.Dir(installedShared), 0o755); err != nil {
		t.Fatalf("failed to create installed registry: %v", err)
	}
	staleBlueprint := `name: windows-11-iot-ltsc-eval
description: stale blueprint fixture
version: "0.0.0"
iso_name: "windows-11-iot-ltsc-eval.iso"
scripts:
  autounattend.xml: "@shared/windows-autounattend.xml"
`
	if err := os.WriteFile(installedBlueprint, []byte(staleBlueprint), 0o644); err != nil {
		t.Fatalf("failed to write stale blueprint: %v", err)
	}
	if err := os.WriteFile(installedShared, []byte("<unattend>stale</unattend>"), 0o644); err != nil {
		t.Fatalf("failed to write stale shared asset: %v", err)
	}

	backupPath, err := syncBundledRegistryFromEmbedded(nidoDir)
	if err != nil {
		t.Fatalf("syncBundledRegistryFromEmbedded failed: %v", err)
	}
	if backupPath == "" {
		t.Fatal("expected registry backup path, got empty string")
	}

	assertFileContains(t, installedBlueprint,
		`iso_name: "windows-11-iot-ltsc-eval-en-us.iso"`,
		`Autounattend.xml: "@shared/windows-autounattend.xml"`,
		`windows-setup-openssh.ps1: "@shared/windows-setup-openssh.ps1"`,
	)
	assertFileContains(t, filepath.Join(nidoDir, "registry", "sources.yaml"), "sources:")
	assertFileContains(t, filepath.Join(backupPath, "blueprints", "windows-11-iot-ltsc-eval.yaml"), `iso_name: "windows-11-iot-ltsc-eval.iso"`)

	secondBackupPath, err := syncBundledRegistryFromEmbedded(nidoDir)
	if err != nil {
		t.Fatalf("second syncBundledRegistryFromEmbedded failed: %v", err)
	}
	if secondBackupPath != "" {
		t.Fatalf("expected current registry to skip backup, got %s", secondBackupPath)
	}
}

func writeTarGzWithFile(archivePath, name string, content []byte) error {
	return writeTarGzFiles(archivePath, map[string][]byte{name: content})
}

func writeTarGzFiles(archivePath string, files map[string][]byte) error {
	f, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzw := gzip.NewWriter(f)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0o755,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write(content); err != nil {
			return err
		}
	}
	return nil
}

func writeZipWithFile(archivePath, name string, content []byte) error {
	return writeZipFiles(archivePath, map[string][]byte{name: content})
}

func writeZipFiles(archivePath string, files map[string][]byte) error {
	f, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		if _, err := w.Write(content); err != nil {
			return err
		}
	}
	return nil
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	if string(got) != want {
		t.Fatalf("%s = %q, want %q", path, got, want)
	}
}

func assertFileContains(t *testing.T, path string, wants ...string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	text := string(got)
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Fatalf("%s does not contain %q:\n%s", path, want, text)
		}
	}
}
