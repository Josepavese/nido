//go:build linux

package hardware

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Josepavese/nido/v4/internal/pkg/sysutil"
)

// This PAL redirects all sysfs access into test storage. Privileged commands are
// recorded, never executed; the host PCI bus and drivers are never modified.
type fixtureIO struct {
	sysutil.NativeHostIO
	root                              string
	driver                            string
	writes                            []string
	failRecord, failVFIO, failRestore bool
	kvmErr                            error
}

func (f *fixtureIO) mapped(p string) string {
	return filepath.Join(f.root, strings.TrimPrefix(filepath.ToSlash(p), "/"))
}
func (f *fixtureIO) ReadDir(p string) ([]fs.DirEntry, error) { return os.ReadDir(f.mapped(p)) }
func (f *fixtureIO) ReadFile(p string) ([]byte, error)       { return os.ReadFile(f.mapped(p)) }
func (f *fixtureIO) Readlink(p string) (string, error) {
	if strings.HasSuffix(p, "/driver") {
		if f.driver == "" {
			return "", fs.ErrNotExist
		}
		return "/sys/bus/pci/drivers/" + f.driver, nil
	}
	return os.Readlink(f.mapped(p))
}
func (f *fixtureIO) WriteFile(p string, b []byte, m fs.FileMode) error {
	if f.failRecord {
		return fs.ErrPermission
	}
	p = f.mapped(p)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	return os.WriteFile(p, b, m)
}
func (f *fixtureIO) Remove(p string) error                        { return os.Remove(f.mapped(p)) }
func (f *fixtureIO) DeviceValue(string, uintptr) (uintptr, error) { return kvmAPIVersion, f.kvmErr }
func (f *fixtureIO) Run(ctx context.Context, _ string, _ ...string) ([]byte, error) {
	return nil, ctx.Err()
}
func (f *fixtureIO) WritePrivileged(ctx context.Context, p, value string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.writes = append(f.writes, p+"="+value)
	if strings.HasSuffix(p, "/unbind") {
		f.driver = ""
	}
	if strings.HasSuffix(p, "/bind") {
		driver := filepath.Base(filepath.Dir(p))
		if driver == "vfio-pci" && f.failVFIO || driver == "native" && f.failRestore {
			return errors.New("injected bind failure")
		}
		f.driver = driver
	}
	return nil
}
func pciFixture(t *testing.T, group, primary string) (*nativeHost, *fixtureIO) {
	t.Helper()
	f := &fixtureIO{root: t.TempDir(), driver: "native"}
	id := "0000:01:00.0"
	dev := f.mapped(filepath.Join(pciDevicesPath, id))
	if err := os.MkdirAll(dev, 0755); err != nil {
		t.Fatal(err)
	}
	for n, v := range map[string]string{"class": "0x030000", "vendor": "0x1234", "device": "0x5678", "boot_vga": primary, "driver_override": ""} {
		if err := os.WriteFile(filepath.Join(dev, n), []byte(v), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if group != "" {
		if err := os.Symlink("/sys/kernel/iommu_groups/"+group, filepath.Join(dev, "iommu_group")); err != nil {
			t.Fatal(err)
		}
	}
	return &nativeHost{io: f}, f
}
func TestPCIInventoryFailsClosed(t *testing.T) {
	for _, tc := range []struct {
		name, group, primary string
		safe                 bool
	}{{"no IOMMU", "", "0", false}, {"primary display", "7", "1", false}, {"isolated device", "7", "0", true}} {
		t.Run(tc.name, func(t *testing.T) {
			h, f := pciFixture(t, tc.group, tc.primary)
			devices, err := h.ListAccelerators()
			if err != nil || len(devices) != 1 || devices[0].IsSafe != tc.safe {
				t.Fatalf("%+v %v", devices, err)
			}
			if !tc.safe {
				if err := h.PreparePassthrough(context.Background(), devices[0].ID, "/run", "fixture"); err == nil || len(f.writes) != 0 {
					t.Fatalf("unsafe device was prepared: %v %v", err, f.writes)
				}
			}
		})
	}
	h, f := pciFixture(t, "7", "0")
	other := f.mapped(filepath.Join(pciDevicesPath, "0000:01:00.1"))
	os.MkdirAll(other, 0755)
	os.Symlink("/sys/kernel/iommu_groups/7", filepath.Join(other, "iommu_group"))
	os.WriteFile(filepath.Join(other, "class"), []byte("0x060400"), 0644)
	devices, err := h.ListAccelerators()
	if err != nil || len(devices) != 1 || devices[0].IsSafe {
		t.Fatalf("shared group accepted: %+v %v", devices, err)
	}
}
func TestPCIPrepareRequiresRecoverableDriver(t *testing.T) {
	h, f := pciFixture(t, "7", "0")
	f.failRecord = true
	err := h.PreparePassthrough(context.Background(), "01:00.0", "/run", "fixture")
	if err == nil || len(f.writes) != 0 || f.driver != "native" {
		t.Fatalf("unbound without journal: %v %v", err, f.writes)
	}
}
func TestPCIPrepareRollsBackBindingFailure(t *testing.T) {
	h, f := pciFixture(t, "7", "0")
	f.failVFIO = true
	err := h.PreparePassthrough(context.Background(), "01:00.0", "/run", "fixture")
	if err == nil || f.driver != "native" {
		t.Fatalf("failed preparation not restored: %v driver=%s writes=%v", err, f.driver, f.writes)
	}
	if _, err := f.ReadFile("/run/pci-0000:01:00.0.driver"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("completed rollback kept pending journal: %v", err)
	}
	for _, write := range f.writes {
		if strings.Contains(write, "new_id") || strings.Contains(write, "rescan") || strings.Contains(write, "/remove") {
			t.Fatalf("unsafe broad device mutation: %s", write)
		}
	}
}
func TestPCIRestoreValidatesBeforeUnbindingAndRetainsFailedJournal(t *testing.T) {
	h, f := pciFixture(t, "7", "0")
	f.driver = "vfio-pci"
	if err := h.RestorePassthrough(context.Background(), "01:00.0", "/run", "fixture"); err == nil || len(f.writes) != 0 {
		t.Fatalf("unbind without original driver: %v %v", err, f.writes)
	}
	f.WriteFile("/run/pci-0000:01:00.0.driver", []byte("../escape"), 0600)
	if err := h.RestorePassthrough(context.Background(), "01:00.0", "/run", "fixture"); err == nil || len(f.writes) != 0 {
		t.Fatalf("invalid driver used: %v %v", err, f.writes)
	}
	f.WriteFile("/run/pci-0000:01:00.0.driver", leaseBytes("fixture", "native"), 0600)
	f.failRestore = true
	if err := h.RestorePassthrough(context.Background(), "01:00.0", "/run", "fixture"); err == nil {
		t.Fatal("restore failure hidden")
	}
	if data, err := f.ReadFile("/run/pci-0000:01:00.0.driver"); err != nil || !strings.Contains(string(data), `"driver":"native"`) {
		t.Fatal("lost recovery record")
	}
	f.failRestore = false
	if err := h.RestorePassthrough(context.Background(), "01:00.0", "/run", "fixture"); err != nil || f.driver != "native" {
		t.Fatalf("retry did not restore: %v driver=%s", err, f.driver)
	}
}
func TestPCICancellationPreventsBinding(t *testing.T) {
	h, f := pciFixture(t, "7", "0")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := h.PreparePassthrough(ctx, "01:00.0", "/run", "fixture"); !errors.Is(err, context.Canceled) || len(f.writes) != 0 {
		t.Fatalf("cancellation ignored: %v %v", err, f.writes)
	}
}
func TestVirtualizationUsesAccessibleDevice(t *testing.T) {
	h, f := pciFixture(t, "7", "0")
	f.kvmErr = fs.ErrPermission
	if v := h.Virtualization(sysutil.HostArchitecture()); v.Accelerator != "tcg" || v.CPU != "qemu64" {
		t.Fatalf("claimed unavailable acceleration: %+v", v)
	}
	f.kvmErr = nil
	if v := h.Virtualization(sysutil.HostArchitecture()); v.Accelerator != "kvm" || v.CPU != "host" {
		t.Fatalf("lost real capability: %+v", v)
	}
}

func leaseBytes(owner, driver string) []byte {
	b, _ := json.Marshal(pciLease{Version: pciLeaseVersion, Owner: owner, Driver: driver})
	return b
}
func (f *fixtureIO) AcquireFile(p string, b []byte, m fs.FileMode) error {
	if f.failRecord {
		return fs.ErrPermission
	}
	target := f.mapped(p)
	os.MkdirAll(filepath.Dir(target), 0755)
	file, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, m)
	if err != nil {
		return err
	}
	_, err = file.Write(b)
	closeErr := file.Close()
	return errors.Join(err, closeErr)
}
func TestPCILeaseRejectsAnotherOwnerAndLegacyRecord(t *testing.T) {
	h, f := pciFixture(t, "7", "0")
	f.WriteFile("/run/pci-0000:01:00.0.driver", leaseBytes("other-vm", "native"), 0600)
	if err := h.PreparePassthrough(context.Background(), "01:00.0", "/run", "fixture"); err == nil || len(f.writes) != 0 {
		t.Fatalf("stole existing lease: %v %v", err, f.writes)
	}
	f.driver = "vfio-pci"
	if err := h.RestorePassthrough(context.Background(), "01:00.0", "/run", "fixture"); err == nil || len(f.writes) != 0 {
		t.Fatalf("restored another owner's device: %v %v", err, f.writes)
	}
	f.WriteFile("/run/pci-0000:01:00.0.driver", []byte("native"), 0600)
	if err := h.RestorePassthrough(context.Background(), "01:00.0", "/run", "fixture"); err == nil || len(f.writes) != 0 {
		t.Fatalf("silently reinterpreted a historical journal: %v %v", err, f.writes)
	}
	data, _ := f.ReadFile("/run/pci-0000:01:00.0.driver")
	if string(data) != "native" {
		t.Fatal("historical journal modified")
	}
}

func TestForeignGuestNeverUsesHostKVM(t *testing.T) {
	h, _ := pciFixture(t, "1", "0")
	guest := "arm64"
	if sysutil.HostArchitecture() == "arm64" {
		guest = "amd64"
	}
	v := h.Virtualization(guest)
	if v.Accelerator != "tcg" || v.Accelerated() {
		t.Fatalf("foreign guest was accelerated: %#v", v)
	}
	wantCPU := "max"
	if guest == "amd64" {
		wantCPU = "qemu64"
	}
	if v.CPU != wantCPU {
		t.Fatalf("wrong software CPU %#v", v)
	}
}
