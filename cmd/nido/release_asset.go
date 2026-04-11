package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func extractBinaryFromReleaseAsset(archivePath, destPath, binaryName string) error {
	if strings.HasSuffix(archivePath, ".tar.gz") {
		return extractBinaryFromTarGz(archivePath, destPath, binaryName)
	}
	if strings.HasSuffix(archivePath, ".zip") {
		return extractBinaryFromZip(archivePath, destPath, binaryName)
	}
	return fmt.Errorf("unsupported release archive format: %s", archivePath)
}

func extractBinaryFromTarGz(archivePath, destPath, binaryName string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag != tar.TypeReg || filepath.Base(hdr.Name) != binaryName {
			continue
		}

		out, err := os.Create(destPath)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tr); err != nil {
			_ = out.Close()
			return err
		}
		return out.Close()
	}

	return fmt.Errorf("binary %s not found in archive", binaryName)
}

func extractBinaryFromZip(archivePath, destPath, binaryName string) error {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer zr.Close()

	for _, f := range zr.File {
		if filepath.Base(f.Name) != binaryName {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(destPath)
		if err != nil {
			_ = rc.Close()
			return err
		}
		if _, err := io.Copy(out, rc); err != nil {
			_ = rc.Close()
			_ = out.Close()
			return err
		}
		_ = rc.Close()
		return out.Close()
	}

	return fmt.Errorf("binary %s not found in archive", binaryName)
}
