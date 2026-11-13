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
		"2         after-upgrade          0 B 2026-11-13 08:02:03 00:00:00.000          4\n" +
		"3         external before upgrade          1.5 GiB 2026-11-13 09:01:02 00:00:02.500          9\n"
	snapshots, err := parseSnapshotList([]byte(output))
	if err != nil {
		t.Fatalf("parseSnapshotList: %v", err)
	}
	if len(snapshots) != 3 {
		t.Fatalf("snapshots = %#v", snapshots)
	}
	if snapshots[0].ID != 1 || snapshots[0].Name != "before-test" || snapshots[0].VMSize != "16 MiB" || snapshots[0].CreatedAt != "2026-11-12 23:16:14" {
		t.Fatalf("first snapshot = %#v", snapshots[0])
	}
	if snapshots[1].Name != "after-upgrade" || snapshots[1].VMSize != "0 B" {
		t.Fatalf("second snapshot = %#v", snapshots[1])
	}
	if snapshots[2].Name != "external before upgrade" || snapshots[2].VMSize != "1.5 GiB" {
		t.Fatalf("external snapshot = %#v", snapshots[2])
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
	spaced, err := provider.SnapshotCreate(ctx, name, "before upgrade")
	if err != nil {
		t.Fatalf("SnapshotCreate with spaces: %v", err)
	}
	if spaced.Name != "before upgrade" || spaced.ID == 0 {
		t.Fatalf("spaced snapshot = %#v", spaced)
	}
	listed, err = provider.SnapshotList(ctx, name)
	if err != nil || len(listed) != 2 {
		t.Fatalf("list with spaced snapshot = %#v, %v", listed, err)
	}
	if err := provider.SnapshotRestore(ctx, name, "before upgrade"); err != nil {
		t.Fatalf("SnapshotRestore with spaces: %v", err)
	}
	if err := provider.SnapshotDelete(ctx, name, "before upgrade"); err != nil {
		t.Fatalf("SnapshotDelete with spaces: %v", err)
	}
	// A third-party qemu-img snapshot can use a space in its tag; Nido must
	// still list and manage it without falling back to a path-derived identifier.
	if err := sysutil.NewCommand(qemuImg, "snapshot", "-c", "external before upgrade", disk).Run(); err != nil {
		t.Fatalf("external qemu-img snapshot: %v", err)
	}
	listed, err = provider.SnapshotList(ctx, name)
	if err != nil || len(listed) != 2 {
		t.Fatalf("external list = %#v, %v", listed, err)
	}
	if err := provider.SnapshotDelete(ctx, name, "external before upgrade"); err != nil {
		t.Fatalf("external delete: %v", err)
	}
	if err := provider.SnapshotDelete(ctx, name, "before-test"); err != nil {
		t.Fatalf("SnapshotDelete: %v", err)
	}
	listed, err = provider.SnapshotList(ctx, name)
	if err != nil || len(listed) != 0 {
		t.Fatalf("after delete = %#v, %v", listed, err)
	}
	if _, err := os.Stat(filepath.Join(root, "run", name+".snapshot.lock")); !os.IsNotExist(err) {
		t.Fatalf("snapshot lease was not retired: %v", err)
	}
}

func TestSnapshotLockRejectsConcurrentMutation(t *testing.T) {
	if _, err := sysutil.QemuImgBinary(); err != nil {
		t.Skip("qemu-img unavailable")
	}
	root := t.TempDir()
	name := "snapshot-fixture"
	if err := os.MkdirAll(filepath.Join(root, "run"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "vms"), 0o755); err != nil {
		t.Fatal(err)
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
	lease, err := sysutil.AcquireResource(filepath.Join(root, "run", name+".snapshot.lock"))
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	if _, err := provider.SnapshotList(context.Background(), name); err == nil || !strings.Contains(err.Error(), "another snapshot operation") {
		t.Fatalf("concurrent snapshot error = %v", err)
	}
}

func TestSnapshotFailureDetail(t *testing.T) {
	if got := snapshotFailureDetail([]byte("qemu-img: Could not create snapshot: No space left on device"), nil); !strings.Contains(got, "no space left") {
		t.Fatalf("space error = %q", got)
	}
	if got := snapshotFailureDetail([]byte("Failed to get write lock: Is another process using the image?"), nil); !strings.Contains(got, "locked by another process") {
		t.Fatalf("lock error = %q", got)
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
	if err := ValidateSnapshotName(" before"); err == nil {
		t.Fatal("expected leading whitespace rejection")
	}
	if err := ValidateSnapshotName("before/after"); err == nil {
		t.Fatal("expected path separator rejection")
	}
}
