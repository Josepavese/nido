//go:build linux

package hardware_test

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Josepavese/nido/internal/config"
	"github.com/Josepavese/nido/internal/hardware"
	"github.com/Josepavese/nido/internal/pkg/sysutil"
	"github.com/Josepavese/nido/internal/provider"
)

// TestPhysicalPCIRoundTrip is an opt-in native qualification, not a CI mock.
// Only an isolated, empty USB controller is eligible. No guest image or existing
// Nido home is opened: the provider owns a private, paused, disposable VM.
func TestPhysicalPCIRoundTrip(t *testing.T) {
	id := os.Getenv("NIDO_PCI_TEST_DEVICE")
	if id == "" {
		t.Skip("physical PCI qualification requires an explicit dedicated device and root")
	}
	if os.Geteuid() != 0 {
		t.Fatal("physical qualification requires local administrator authentication")
	}
	if err := hardware.ValidateAcceleratorID(id); err != nil || id == hardware.AutoAcceleratorID || id == hardware.VirtualGPUAcceleratorID {
		t.Fatal("an explicit PCI device is required")
	}
	base := os.Getenv("NIDO_PCI_TEST_STORAGE")
	if !filepath.IsAbs(base) {
		t.Fatal("an explicit scratch filesystem is required")
	}
	host := hardware.New()
	devices, err := host.ListAccelerators()
	if err != nil {
		t.Fatal(err)
	}
	eligible := false
	for _, device := range devices {
		if device.ID == id {
			eligible = device.IsSafe && device.IsIsolated && device.Class == hardware.DescUSB
		}
	}
	if !eligible {
		t.Fatal("device must be an isolated, non-primary USB controller")
	}
	dev := filepath.Join("/sys/bus/pci/devices", id)
	original, err := os.Readlink(filepath.Join(dev, "driver"))
	if err != nil || filepath.Base(original) != "xhci_hcd" {
		t.Fatal("qualification requires the expected xhci_hcd host driver")
	}
	usb, err := os.ReadDir("/sys/bus/usb/devices")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range usb {
		path, err := filepath.EvalSymlinks(filepath.Join("/sys/bus/usb/devices", entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(path, "/"+id+"/") && !strings.HasPrefix(entry.Name(), "usb") && !strings.HasSuffix(entry.Name(), "-0:1.0") {
			t.Fatalf("controller has an attached peripheral: %s", entry.Name())
		}
	}
	root, err := os.MkdirTemp(base, "pci-physical-")
	if err != nil {
		t.Fatal(err)
	}
	const name = "pci-physical-qualification"
	runDir := filepath.Join(root, "run")
	leasePath := filepath.Join(runDir, "pci-"+id+".driver")
	cfg := &config.Config{SSHUser: "qualification", PortRangeStart: config.DefaultPortRangeStart, PortRangeEnd: config.DefaultPortRangeEnd}
	var vm provider.VMProvider = provider.NewQemuProvider(root, cfg, host)
	// Never retire an unresolved recovery record, even when qualification fails.
	t.Cleanup(func() {
		if err := vm.Stop(name); err != nil {
			t.Errorf("restore failed; recovery state preserved at %s: %v", root, err)
			return
		}
		current, err := os.Readlink(filepath.Join(dev, "driver"))
		if err != nil || filepath.Base(current) != filepath.Base(original) {
			t.Errorf("original binding not restored; preserve %s", root)
			return
		}
		if _, err := os.Stat(leasePath); !os.IsNotExist(err) {
			t.Errorf("recovery record remains at %s", leasePath)
			return
		}
		if err := vm.Delete(name); err != nil {
			t.Error(err)
			return
		}
		if err := os.RemoveAll(root); err != nil {
			t.Error(err)
		}
	})
	for _, dir := range []string{runDir, filepath.Join(root, "vms")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	disk := filepath.Join(root, "vms", name+".qcow2")
	if out, err := sysutil.NewCommand("qemu-img", "create", "-f", "qcow2", disk, "1M").CombinedOutput(); err != nil {
		t.Fatalf("create disposable disk: %s %v", out, err)
	}
	if err := vm.Start(name, provider.VMOptions{MemoryMB: 128, VCPUs: 1, Accelerators: []string{id}, RawQemuArgs: []string{"-S", "-machine", "accel=tcg", "-cpu", "max"}}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(leasePath)
	if err != nil {
		t.Fatal(err)
	}
	var lease struct {
		Version int    `json:"version"`
		Owner   string `json:"owner"`
		Driver  string `json:"driver"`
	}
	if json.Unmarshal(data, &lease) != nil || lease.Version != 1 || lease.Owner != name || lease.Driver != "xhci_hcd" {
		t.Fatalf("invalid owned recovery lease: %s", data)
	}
	if err := host.RestorePassthrough(context.Background(), id, runDir, "another-owner"); err == nil {
		t.Fatal("foreign owner was allowed to restore an active binding")
	}
	current, err := os.Readlink(filepath.Join(dev, "driver"))
	if err != nil || filepath.Base(current) != "vfio-pci" {
		t.Fatal("VFIO binding was not retained")
	}
	conn, err := net.DialTimeout("unix", filepath.Join(runDir, name+".qmp"), 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatal(err)
	}
	decoder, encoder := json.NewDecoder(conn), json.NewEncoder(conn)
	var message map[string]json.RawMessage
	if err := decoder.Decode(&message); err != nil {
		t.Fatal(err)
	}
	request := func(command string) json.RawMessage {
		t.Helper()
		if err := encoder.Encode(map[string]string{"execute": command}); err != nil {
			t.Fatal(err)
		}
		for {
			message = nil
			if err := decoder.Decode(&message); err != nil {
				t.Fatal(err)
			}
			if value, ok := message["return"]; ok {
				return value
			}
			if value, ok := message["error"]; ok {
				t.Fatalf("QMP %s: %s", command, value)
			}
		}
	}
	request("qmp_capabilities")
	bus := request("query-pci")
	var tree any
	if err := json.Unmarshal(bus, &tree); err != nil {
		t.Fatal(err)
	}
	readID := func(field string) int {
		b, err := os.ReadFile(filepath.Join(dev, field))
		if err != nil {
			t.Fatal(err)
		}
		n, err := strconv.ParseInt(strings.TrimSpace(string(b)), 0, 32)
		if err != nil {
			t.Fatal(err)
		}
		return int(n)
	}
	vendor, device := readID("vendor"), readID("device")
	var contains func(any) bool
	contains = func(node any) bool {
		switch value := node.(type) {
		case map[string]any:
			if value["vendor"] == float64(vendor) && value["device"] == float64(device) {
				return true
			}
			for _, child := range value {
				if contains(child) {
					return true
				}
			}
		case []any:
			for _, child := range value {
				if contains(child) {
					return true
				}
			}
		}
		return false
	}
	if !contains(tree) {
		t.Fatalf("physical controller missing from guest PCI bus: %s", bus)
	}
	if err := vm.Stop(name); err != nil {
		t.Fatal(err)
	}
	t.Logf("physical PCI %s: VFIO attachment, QMP guest identity, owner refusal and host restoration verified", id)
}
