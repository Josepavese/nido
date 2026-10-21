//go:build !windows

package sysutil

import "os/exec"

func fallbackExecutable([]string) (string, error) { return "", exec.ErrNotFound }
