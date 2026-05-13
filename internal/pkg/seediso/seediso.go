package seediso

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/kdomanski/iso9660"
)

// Create writes a small ISO9660 image from sourceDir. It prefers system ISO
// tools when available, then falls back to the bundled Go writer for Windows
// and other minimal hosts.
func Create(outputPath, sourceDir, label string) error {
	if err := createWithTool(outputPath, sourceDir, label); err == nil {
		return nil
	}
	return createWithGo(outputPath, sourceDir, label)
}

func createWithTool(outputPath, sourceDir, label string) error {
	type isoTool struct {
		name string
		args []string
	}
	tools := []isoTool{
		{name: "mkisofs", args: []string{"-J", "-R", "-V", label, "-o", outputPath, sourceDir}},
		{name: "genisoimage", args: []string{"-J", "-R", "-V", label, "-o", outputPath, sourceDir}},
		{name: "xorriso", args: []string{"-as", "mkisofs", "-J", "-R", "-V", label, "-o", outputPath, sourceDir}},
	}
	for _, tool := range tools {
		if _, err := exec.LookPath(tool.name); err != nil {
			continue
		}
		cmd := exec.Command(tool.name, tool.args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to create seed ISO with %s: %v (%s)", tool.name, err, string(out))
		}
		return nil
	}
	return fmt.Errorf("no ISO creation tool found")
}

func createWithGo(outputPath, sourceDir, label string) error {
	writer, err := iso9660.NewWriter()
	if err != nil {
		return fmt.Errorf("failed to create ISO writer: %w", err)
	}
	defer writer.Cleanup()

	if err := filepath.WalkDir(sourceDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		return writer.AddLocalFile(path, filepath.ToSlash(rel))
	}); err != nil {
		return fmt.Errorf("failed to stage ISO files: %w", err)
	}

	output, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to create ISO output: %w", err)
	}
	defer output.Close()

	if err := writer.WriteTo(output, label); err != nil {
		return fmt.Errorf("failed to write ISO image: %w", err)
	}
	return nil
}
