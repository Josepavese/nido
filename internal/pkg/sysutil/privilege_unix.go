//go:build !windows

package sysutil

import "os"

func isPrivileged() bool { return os.Getuid() == 0 }
