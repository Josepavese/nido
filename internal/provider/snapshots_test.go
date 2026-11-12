package provider

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Josepavese/nido/v4/internal/pkg/sysutil"
)

func TestParseSnapshotList(t *testing.T) {
	output := "Snapshot list:\n" +
		"ID        TAG               VM SIZE                DATE     VM CLOCK     ICOUNT\n" +
		"1         before-test          16 MiB 2026-11-12 23:16:14 00:00:00.000          0\n" +
		"2         after-upgrade          0 B 2026-11-13 08:02:03 00:00:00.000          4\n"
	snapshots, err := parseSnapshotList([]byte(output))
	if err != nil {
		t.Fatalf("parseSnapshotList: %v", err)
	}
	if len(snapshots) != 2 {
		t.Fatalf("snapshots = %#v", snapshots)
	}
	if snapshots[0].ID != 1 || snapshots[0].Name != "before-test" || snapshots[0].VMSize != "16 MiB" || snapshots[0].CreatedAt != "2026-11-12 23:16:14" {
		t.Fatalf("first snapshot = %#v", snapshots[0])
	}
	if snapshots[1].Name != "after-upgrade" || snapshots[1].VMSize != "0 B" {
		t.Fatalf("second snapshot = %#v", snapshots[1])
	}
}

func TestParseSnapshotListEmptyAndMalformed(t *testing.T) {
	empty, err := parseSnapshotList([]byte("Snapshot list:\nID TAG VM SIZE DATE VM CLOCK ICOUNT\n"))
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty list = %#v, %v", empty, err)
	}
	empty, err = parseSnapshotList(nil)
	if err != nil || len(empty) != 0 {
		t.Fatalf("nil list = %#v, %v", empty, err)
	}
	if _, err := parseSnapshotList([]byte("ID TAG\n1 broken\n")); err == nil {
		t.Fatal("expected malformed row error")
	}
}

func TestSnapshotLifecycleOnStoppedDisk(t *testing.T) {
	if _, err := sysutil.QemuImgBinary(); err != nil {
		t.Skip("qemu-img unavailable")
	}
	root := t.TempDir()
	name := "snapshot-fixture"
	for _, dir := range []string{filepath.Join(root, "run"), filepath.Join(root, "vms")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	state := `{"arch":"amd64","name":"snapshot-fixture","pid":0,"ssh_port":50022}`
	if err := os.WriteFile(filepath.Join(root, "run", name+".json"), []byte(state), 0o600); err != nil {
		t.Fatal(err)
	}
	disk := filepath.Join(root, "vms", name+".qcow2")
	qemuImg, _ := sysutil.QemuImgBinary()
	if err := sysutil.NewCommand(qemuImg, "create", "-q", "-f", "qcow2", disk, "8M").Run(); err != nil {
		t.Fatalf("create qcow2: %v", err)
	}
	provider := NewQemuProvider(root, nil, nil)
	ctx := context.Background()

	created, err := provider.SnapshotCreate(ctx, name, "before-test")
	if err != nil {
		t.Fatalf("SnapshotCreate: %v", err)
	}
	if created.Name != "before-test" || created.ID == 0 {
		t.Fatalf("created snapshot = %#v", created)
	}
	listed, err := provider.SnapshotList(ctx, name)
	if err != nil || len(listed) != 1 || listed[0].Name != "before-test" {
		t.Fatalf("SnapshotList = %#v, %v", listed, err)
	}
	if _, err := provider.SnapshotCreate(ctx, name, "before-test"); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("duplicate create error = %v", err)
	}
	if err := provider.SnapshotRestore(ctx, name, "missing"); err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("missing restore error = %v", err)
	}
	if err := provider.SnapshotRestore(ctx, name, "before-test"); err != nil {
		t.Fatalf("SnapshotRestore: %v", err)
	}
	if err := provider.SnapshotDelete(ctx, name, "before-test"); err != nil {
		t.Fatalf("SnapshotDelete: %v", err)
	}
	listed, err = provider.SnapshotList(ctx, name)
	if err != nil || len(listed) != 0 {
		t.Fatalf("after delete = %#v, %v", listed, err)
	}
}

func TestSnapshotRequiresStoppedKnownVM(t *testing.T) {
	provider := NewQemuProvider(t.TempDir(), nil, nil)
	if _, err := provider.SnapshotList(context.Background(), "missing"); err == nil {
		t.Fatal("expected missing VM error")
	}
}

func TestSnapshotRejectsInvalidSnapshotName(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "run"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "vms"), 0o755); err != nil {
		t.Fatal(err)
	}
	state := `{"arch":"amd64","name":"snapshot-fixture","pid":0}`
	if err := os.WriteFile(filepath.Join(root, "run", "snapshot-fixture.json"), []byte(state), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "vms", "snapshot-fixture.qcow2"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	provider := NewQemuProvider(root, nil, nil)
	if _, err := provider.SnapshotCreate(context.Background(), "snapshot-fixture", "../bad"); err == nil {
		t.Fatal("expected invalid snapshot identity error")
	}
}
