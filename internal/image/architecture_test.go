package image

import (
	"github.com/Josepavese/nido/v5/internal/pkg/sysutil"
	"os"
	"path/filepath"
	"testing"
)

func TestVariantsDoNotCollideOrSubstitute(t *testing.T) {
	x86 := Version{Version: "24.04", Arch: "amd64", Aliases: []string{"lts"}}
	arm := Version{Version: "24.04", Arch: "arm64", Aliases: []string{"lts"}}
	c := Catalog{Images: []Image{{Name: "ubuntu", Versions: []Version{x86, arm}}}}
	for _, arch := range []string{"amd64", "arm64"} {
		_, v, err := c.FindImage("ubuntu", "lts", arch)
		if err != nil || v.Arch != arch {
			t.Fatalf("resolution for %s = %#v, %v", arch, v, err)
		}
	}
	for _, request := range []string{"", "auto"} {
		_, v, err := c.FindImage("ubuntu", "lts", request)
		if err != nil || v.Arch != sysutil.HostArchitecture() {
			t.Fatalf("native variant = %#v %v", v, err)
		}
	}
	if x86.CacheFileName("ubuntu") == arm.CacheFileName("ubuntu") {
		t.Fatal("different ISAs collide in cache")
	}
	c.Images[0].Versions = []Version{x86}
	if _, _, err := c.FindImage("ubuntu", "lts", "arm64"); err == nil {
		t.Fatal("silently substituted x86 for ARM")
	}
}

func TestArtifactRequiresDeclaredArchitecture(t *testing.T) {
	disk := filepath.Join(t.TempDir(), "guest.qcow2")
	if _, err := ReadArchitecture(disk); !os.IsNotExist(err) {
		t.Fatalf("missing declaration: %v", err)
	}
	if err := WriteArchitecture(disk, "aarch64"); err != nil {
		t.Fatal(err)
	}
	if got, err := ReadArchitecture(disk); got != "arm64" || err != nil {
		t.Fatalf("persisted ISA %q %v", got, err)
	}
	if err := WriteArchitecture(disk, "armv7"); err == nil {
		t.Fatal("accepted unsupported architecture")
	}
	if got, _ := ReadArchitecture(disk); got != "arm64" {
		t.Fatal("invalid write damaged existing identity")
	}
}
