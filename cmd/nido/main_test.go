package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
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

func writeTarGzWithFile(archivePath, name string, content []byte) error {
	f, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzw := gzip.NewWriter(f)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	hdr := &tar.Header{
		Name: name,
		Mode: 0755,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err = tw.Write(content)
	return err
}

func writeZipWithFile(archivePath, name string, content []byte) error {
	f, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(content)
	return err
}
