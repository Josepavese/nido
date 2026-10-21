package sysutil

import (
	"path/filepath"
	"strings"
)

func nativeWindowsPath(path string) string {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	if strings.HasPrefix(absolute, `\\?\`) {
		return absolute
	}
	if strings.HasPrefix(absolute, `\\`) {
		return `\\?\UNC\` + absolute[2:]
	}
	return `\\?\` + absolute
}
