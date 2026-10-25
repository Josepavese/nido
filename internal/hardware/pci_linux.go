//go:build linux

package hardware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Josepavese/nido/v5/internal/pkg/sysutil"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const pciDevicesPath = "/sys/bus/pci/devices"
const passthroughTimeout = 30 * time.Second
const pciLeaseVersion = 1

type pciLease struct {
	Version int    `json:"version"`
	Owner   string `json:"owner"`
	Driver  string `json:"driver"`
}

var driverNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-][a-zA-Z0-9_.-]*$`)

// PreparePassthrough exclusively leases an isolated, non-primary device for one
// VM. The saved original driver precedes all unbinding; overrides are per-device.
func (h *nativeHost) PreparePassthrough(ctx context.Context, pciID, stateDir, owner string) (retErr error) {
	ctx, cancel := context.WithTimeout(ctx, passthroughTimeout)
	defer cancel()
	id, err := normalizePCIID(pciID)
	if err != nil {
		return err
	}
	if owner == "" {
		return fmt.Errorf("PCI lease owner is required")
	}
	devices, err := h.ListAccelerators()
	if err != nil {
		return err
	}
	safe := false
	for _, device := range devices {
		if device.ID == id {
			safe = device.IsSafe
			break
		}
	}
	if !safe {
		return fmt.Errorf("PCI device %s is not safe for passthrough", id)
	}
	dev := filepath.Join(pciDevicesPath, id)
	link, err := h.io.Readlink(filepath.Join(dev, "driver"))
	if err != nil {
		return fmt.Errorf("read original PCI driver: %w", err)
	}
	original := filepath.Base(link)
	if original == "vfio-pci" {
		return fmt.Errorf("PCI device %s is already bound to VFIO; reconcile its owner before preparing it", id)
	}
	if !driverNamePattern.MatchString(original) {
		return fmt.Errorf("invalid original PCI driver")
	}
	override, err := h.io.ReadFile(filepath.Join(dev, "driver_override"))
	if err != nil {
		return fmt.Errorf("read PCI driver override: %w", err)
	}
	value := strings.TrimSpace(string(override))
	if value != "" && value != "(null)" {
		return fmt.Errorf("PCI device %s already has a driver override", id)
	}

	record := filepath.Join(stateDir, fmt.Sprintf("pci-%s.driver", id))
	data, err := json.Marshal(pciLease{Version: pciLeaseVersion, Owner: owner, Driver: original})
	if err != nil {
		return err
	}
	if err := h.io.AcquireFile(record, data, 0600); err != nil {
		return fmt.Errorf("reserve PCI lease for %s: %w", owner, err)
	}
	detached := false
	overrideAttempted := false
	defer func() {
		if retErr == nil {
			return
		}
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), passthroughTimeout)
		defer cancel()
		if detached {
			if err := h.RestorePassthrough(cleanup, id, stateDir, owner); err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("restore after preparation failure: %w", err))
			}
		} else {
			if overrideAttempted {
				if err := h.io.WritePrivileged(cleanup, filepath.Join(dev, "driver_override"), ""); err != nil {
					retErr = errors.Join(retErr, err)
					return
				}
			}
			if err := h.io.Remove(record); err != nil {
				retErr = errors.Join(retErr, err)
			}
		}
	}()
	if err := sysutil.RunPrivileged(ctx, h.io, "modprobe", "vfio-pci"); err != nil {
		return err
	}
	if err := sysutil.AllowMemoryLock(ctx, h.io); err != nil {
		return fmt.Errorf("allow VFIO memory locking: %w", err)
	}
	overrideAttempted = true
	if err := h.io.WritePrivileged(ctx, filepath.Join(dev, "driver_override"), "vfio-pci"); err != nil {
		return err
	}
	if err := h.io.WritePrivileged(ctx, filepath.Join(dev, "driver", "unbind"), id); err != nil {
		return fmt.Errorf("unbind original PCI driver: %w", err)
	}
	detached = true
	if err := h.io.WritePrivileged(ctx, "/sys/bus/pci/drivers/vfio-pci/bind", id); err != nil {
		return fmt.Errorf("bind VFIO device: %w", err)
	}
	link, err = h.io.Readlink(filepath.Join(dev, "driver"))
	if err != nil || filepath.Base(link) != "vfio-pci" {
		return fmt.Errorf("VFIO driver binding was not confirmed for %s", id)
	}
	return nil
}

// RestorePassthrough checks schema and VM ownership before changing hardware.
// A failed restore retains the complete lease for retry; no remove/rescan occurs.
func (h *nativeHost) RestorePassthrough(ctx context.Context, pciID, stateDir, owner string) error {
	ctx, cancel := context.WithTimeout(ctx, passthroughTimeout)
	defer cancel()
	id, err := normalizePCIID(pciID)
	if err != nil {
		return err
	}
	record := filepath.Join(stateDir, fmt.Sprintf("pci-%s.driver", id))
	data, err := h.io.ReadFile(record)
	if errors.Is(err, fs.ErrNotExist) {
		link, readErr := h.io.Readlink(filepath.Join(pciDevicesPath, id, "driver"))
		if readErr == nil && filepath.Base(link) != "vfio-pci" {
			return nil
		}
		return fmt.Errorf("PCI device %s lacks an ownership record; cannot restore an unowned binding", id)
	}
	if err != nil {
		return fmt.Errorf("read PCI lease: %w", err)
	}
	var lease pciLease
	if err := json.Unmarshal(data, &lease); err != nil || lease.Version != pciLeaseVersion {
		return fmt.Errorf("PCI lease %s requires explicit conversion to the current schema", record)
	}
	if lease.Owner != owner || owner == "" {
		return fmt.Errorf("PCI lease is owned by %q, not %q", lease.Owner, owner)
	}
	original := lease.Driver
	if !driverNamePattern.MatchString(original) || original == "vfio-pci" {
		return fmt.Errorf("invalid original PCI driver record")
	}
	dev := filepath.Join(pciDevicesPath, id)
	link, err := h.io.Readlink(filepath.Join(dev, "driver"))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	current := ""
	if err == nil {
		current = filepath.Base(link)
	}
	if current != "" && current != original && current != "vfio-pci" {
		return fmt.Errorf("PCI device %s has an unexpected driver %s", id, current)
	}
	if current == "vfio-pci" {
		if err := h.io.WritePrivileged(ctx, "/sys/bus/pci/drivers/vfio-pci/unbind", id); err != nil {
			return fmt.Errorf("unbind VFIO: %w", err)
		}
	}
	if err := h.io.WritePrivileged(ctx, filepath.Join(dev, "driver_override"), ""); err != nil {
		return err
	}
	if current != original {
		if err := h.io.WritePrivileged(ctx, filepath.Join("/sys/bus/pci/drivers", original, "bind"), id); err != nil {
			return fmt.Errorf("restore original PCI driver: %w", err)
		}
	}
	link, err = h.io.Readlink(filepath.Join(dev, "driver"))
	if err != nil || filepath.Base(link) != original {
		return fmt.Errorf("restored driver was not confirmed for %s; lease preserved", id)
	}
	if err := h.io.Remove(record); err != nil {
		return fmt.Errorf("retire restored PCI lease: %w", err)
	}
	return nil
}
