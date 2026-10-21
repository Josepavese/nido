//go:build !linux && !darwin && !windows

package sysutil

import "errors"

func publishFile(string, string) error { return errors.ErrUnsupported }
