package sysutil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
)

// HostIO is the PAL used by native hardware adapters. Tests supply a private
// filesystem and command recorder; no host device access is needed.
type HostIO interface {
	AcquireFile(string, []byte, fs.FileMode) error
	ReadDir(string) ([]fs.DirEntry, error)
	ReadFile(string) ([]byte, error)
	Readlink(string) (string, error)
	Remove(string) error
	DeviceValue(string, uintptr) (uintptr, error)
	Run(context.Context, string, ...string) ([]byte, error)
	WritePrivileged(context.Context, string, string) error
}

type NativeHostIO struct{}

func (NativeHostIO) ReadDir(p string) ([]fs.DirEntry, error) { return os.ReadDir(p) }
func (NativeHostIO) ReadFile(p string) ([]byte, error)       { return os.ReadFile(p) }
func (NativeHostIO) Readlink(p string) (string, error)       { return os.Readlink(p) }
func (NativeHostIO) Remove(p string) error                   { return os.Remove(p) }
func (NativeHostIO) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("%s failed: %s (%w)", name, out, err)
	}
	return out, nil
}
func (NativeHostIO) WritePrivileged(ctx context.Context, p, value string) error {
	if isPrivileged() {
		return os.WriteFile(p, []byte(value+"\n"), 0644)
	}
	cmd := exec.CommandContext(ctx, "sudo", "-n", "tee", p)
	cmd.Stdin = bytes.NewBufferString(value + "\n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("privileged write %s: %s (%w)", p, out, err)
	}
	return nil
}

// RunPrivileged refuses interactive elevation; callers receive an explicit error.
func RunPrivileged(ctx context.Context, io HostIO, name string, args ...string) error {
	if !isPrivileged() {
		args = append([]string{"-n", name}, args...)
		name = "sudo"
	}
	_, err := io.Run(ctx, name, args...)
	return err
}

// AcquireFile reserves an absent ownership record and saves it before returning.
// A failed write retires only the newly reserved file, before any device mutation.
func (NativeHostIO) AcquireFile(path string, data []byte, mode fs.FileMode) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(data)
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		os.Remove(path)
		return errors.Join(writeErr, closeErr)
	}
	if err := FixPermissions(path); err != nil {
		os.Remove(path)
		return err
	}
	return nil
}
