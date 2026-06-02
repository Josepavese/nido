package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
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

func syncBundledRegistryFromReleaseAsset(archivePath, nidoDir string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "nido-registry-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	extractedRegistry := filepath.Join(tmpDir, "registry")
	if err := extractRegistryFromReleaseAsset(archivePath, extractedRegistry); err != nil {
		return "", err
	}

	return syncBundledRegistryFromDir(extractedRegistry, nidoDir)
}

func syncBundledRegistryFromDir(srcRegistry, nidoDir string) (string, error) {
	installedRegistry := filepath.Join(nidoDir, "registry")
	needsSync, err := registryNeedsSync(srcRegistry, installedRegistry)
	if err != nil {
		return "", err
	}
	if !needsSync {
		return "", nil
	}

	backupPath := ""
	if info, err := os.Stat(installedRegistry); err == nil {
		if !info.IsDir() {
			return "", fmt.Errorf("%s exists but is not a directory", installedRegistry)
		}
		backupPath = uniqueRegistryBackupPath(nidoDir)
		if err := copyTree(installedRegistry, backupPath); err != nil {
			return "", fmt.Errorf("backup registry: %w", err)
		}
	} else if os.IsNotExist(err) {
		if err := os.MkdirAll(nidoDir, 0o755); err != nil {
			return "", err
		}
	} else {
		return "", err
	}

	if err := copyTree(srcRegistry, installedRegistry); err != nil {
		return backupPath, fmt.Errorf("sync bundled registry: %w", err)
	}
	return backupPath, nil
}

func registryNeedsSync(srcRegistry, installedRegistry string) (bool, error) {
	info, err := os.Stat(srcRegistry)
	if err != nil {
		return false, err
	}
	if !info.IsDir() {
		return false, fmt.Errorf("%s is not a directory", srcRegistry)
	}

	if info, err := os.Stat(installedRegistry); err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	} else if !info.IsDir() {
		return false, fmt.Errorf("%s exists but is not a directory", installedRegistry)
	}

	needsSync := false
	err = filepath.Walk(srcRegistry, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil || needsSync {
			return err
		}
		if info.IsDir() || !info.Mode().IsRegular() {
			return nil
		}

		rel, err := filepath.Rel(srcRegistry, srcPath)
		if err != nil {
			return err
		}
		equal, err := filesEqual(srcPath, filepath.Join(installedRegistry, rel))
		if err != nil {
			return err
		}
		needsSync = !equal
		return nil
	})
	return needsSync, err
}

func filesEqual(pathA, pathB string) (bool, error) {
	a, err := os.ReadFile(pathA)
	if err != nil {
		return false, err
	}
	b, err := os.ReadFile(pathB)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return bytes.Equal(a, b), nil
}

func extractRegistryFromReleaseAsset(archivePath, destDir string) error {
	if strings.HasSuffix(archivePath, ".tar.gz") {
		return extractRegistryFromTarGz(archivePath, destDir)
	}
	if strings.HasSuffix(archivePath, ".zip") {
		return extractRegistryFromZip(archivePath, destDir)
	}
	return fmt.Errorf("unsupported release archive format: %s", archivePath)
}

func extractRegistryFromTarGz(archivePath, destDir string) error {
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

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	found := false
	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		rel, ok := registryRelativePath(hdr.Name)
		if !ok {
			continue
		}
		target, err := safeArchiveTarget(destDir, rel)
		if err != nil {
			return err
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, hdr.FileInfo().Mode().Perm()); err != nil {
				return err
			}
		case tar.TypeReg:
			found = true
			if err := writeReaderToFile(target, tr, hdr.FileInfo().Mode().Perm()); err != nil {
				return err
			}
		}
	}

	if !found {
		return fmt.Errorf("registry directory not found in archive")
	}
	return nil
}

func extractRegistryFromZip(archivePath, destDir string) error {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer zr.Close()

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	found := false
	for _, f := range zr.File {
		rel, ok := registryRelativePath(f.Name)
		if !ok {
			continue
		}
		target, err := safeArchiveTarget(destDir, rel)
		if err != nil {
			return err
		}

		info := f.FileInfo()
		if info.IsDir() {
			if err := os.MkdirAll(target, info.Mode().Perm()); err != nil {
				return err
			}
			continue
		}
		if !info.Mode().IsRegular() {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}
		if err := writeReaderToFile(target, rc, info.Mode().Perm()); err != nil {
			_ = rc.Close()
			return err
		}
		_ = rc.Close()
		found = true
	}

	if !found {
		return fmt.Errorf("registry directory not found in archive")
	}
	return nil
}

func registryRelativePath(name string) (string, bool) {
	slashName := strings.ReplaceAll(filepath.ToSlash(name), "\\", "/")
	clean := strings.TrimPrefix(path.Clean(slashName), "./")
	if clean == "." {
		return "", false
	}

	parts := strings.Split(clean, "/")
	for i, part := range parts {
		if part != "registry" || i+1 >= len(parts) {
			continue
		}
		rel := strings.Join(parts[i+1:], "/")
		if rel == "" || rel == "." {
			return "", false
		}
		return rel, true
	}
	return "", false
}

func safeArchiveTarget(destDir, rel string) (string, error) {
	clean := path.Clean(rel)
	if clean == "." || clean == ".." || path.IsAbs(clean) || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("unsafe archive path: %s", rel)
	}

	target := filepath.Join(destDir, filepath.FromSlash(clean))
	absDest, err := filepath.Abs(destDir)
	if err != nil {
		return "", err
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	relToDest, err := filepath.Rel(absDest, absTarget)
	if err != nil {
		return "", err
	}
	if relToDest == ".." || strings.HasPrefix(relToDest, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("unsafe archive path: %s", rel)
	}
	return target, nil
}

func writeReaderToFile(path string, r io.Reader, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if mode == 0 {
		mode = 0o644
	}
	out, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, r); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func uniqueRegistryBackupPath(nidoDir string) string {
	base := filepath.Join(nidoDir, "registry-backup-"+time.Now().UTC().Format("20060102T150405Z"))
	path := base
	for i := 1; ; i++ {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return path
		}
		path = fmt.Sprintf("%s-%d", base, i)
	}
}

func copyTree(srcDir, destDir string) error {
	info, err := os.Stat(srcDir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", srcDir)
	}

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destDir, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		return copyFile(path, target, info.Mode().Perm())
	})
}

func copyFile(srcPath, destPath string, mode os.FileMode) error {
	in, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer in.Close()

	return writeReaderToFile(destPath, in, mode)
}
