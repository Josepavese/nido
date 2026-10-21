package sysutil

import (
	"os"
	"os/exec"
	"path/filepath"
)

func fallbackExecutable(names []string) (string, error) {

	for _, dir := range windowsFallbackExecutableDirs() {
		for _, name := range names {
			exe := name
			if filepath.Ext(exe) == "" {
				exe += ".exe"
			}
			path := filepath.Join(dir, exe)
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				return path, nil
			}
		}
	}
	return "", exec.ErrNotFound
}
func windowsFallbackExecutableDirs() []string {
	dirs := []string{}
	if programFiles := os.Getenv("ProgramFiles"); programFiles != "" {
		dirs = append(dirs, filepath.Join(programFiles, "qemu"))
	}
	if programFilesX86 := os.Getenv("ProgramFiles(x86)"); programFilesX86 != "" {
		dirs = append(dirs, filepath.Join(programFilesX86, "qemu"))
	}
	dirs = append(dirs, `C:\Program Files\qemu`, `C:\Program Files (x86)\qemu`)
	return dirs
}
