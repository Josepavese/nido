package lifecycle

import (
	"bytes"
	"fmt"
	registryassets "github.com/Josepavese/nido/v5/registry"
	"io/fs"
	"os"
	"path/filepath"
)

func SyncBundledRegistryFromEmbedded(nidoDir string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "nido-registry-embedded-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	extractedRegistry := filepath.Join(tmpDir, "registry")
	if err := copyEmbeddedRegistry(extractedRegistry); err != nil {
		return "", err
	}
	return syncBundledRegistryFromDir(extractedRegistry, nidoDir)
}

func copyEmbeddedRegistry(destDir string) error {
	return fs.WalkDir(registryassets.FS, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if name == "." {
			return nil
		}

		destPath := filepath.Join(destDir, filepath.FromSlash(name))
		if entry.IsDir() {
			return os.MkdirAll(destPath, 0o755)
		}
		if !entry.Type().IsRegular() {
			return nil
		}

		data, err := registryassets.FS.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read embedded registry asset %s: %w", name, err)
		}
		return writeReaderToFile(destPath, bytes.NewReader(data), 0o644)
	})
}
