package provider

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Josepavese/nido/v4/internal/pkg/identifier"
	"github.com/Josepavese/nido/v4/internal/pkg/sysutil"
)

// Snapshot operations are disk-only internal qcow2 snapshots. They are
// intentionally limited to stopped VMs: qemu-img explicitly forbids modifying
// an image that another process has open. The VM state file remains the single
// source for runtime metadata; snapshots live inside the disk image itself.
const snapshotCommandTimeout = 2 * time.Minute

// DiskSnapshot describes one internal qcow2 snapshot. It contains no guest
// memory: after restore the guest boots from the captured disk contents.
type DiskSnapshot struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	VMSize    string `json:"vm_size,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

// qemu-img has no JSON output for snapshot metadata. The tag is the only
// variable-width text column, so parse it non-greedily between the numeric ID
// and the VM SIZE/date columns. External snapshots may use spaces in their tag.
var snapshotLine = regexp.MustCompile(`^(\d+)\s+(.+?)\s+(\d+(?:\.\d+)?\s?[KMGTPE]?i?B)\s+(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})\s+(\d{2}:\d{2}:\d{2}\.\d+)\s+(\d+)\s*$`)

func (p *QemuProvider) SnapshotCreate(ctx context.Context, name, snapshot string) (DiskSnapshot, error) {
	disk, err := p.snapshotDisk(name)
	if err != nil {
		return DiskSnapshot{}, err
	}
	if err := ValidateSnapshotName(snapshot); err != nil {
		return DiskSnapshot{}, err
	}
	current, err := p.SnapshotList(ctx, name)
	if err != nil {
		return DiskSnapshot{}, err
	}
	for _, existing := range current {
		if existing.Name == snapshot {
			return DiskSnapshot{}, fmt.Errorf("snapshot %q already exists on VM %q", snapshot, name)
		}
	}
	if _, err := p.runQemuImgSnapshot(ctx, "-c", snapshot, disk); err != nil {
		return DiskSnapshot{}, err
	}
	created, err := p.SnapshotList(ctx, name)
	if err != nil {
		return DiskSnapshot{}, err
	}
	for _, entry := range created {
		if entry.Name == snapshot {
			return entry, nil
		}
	}
	return DiskSnapshot{Name: snapshot}, nil
}

func (p *QemuProvider) SnapshotList(ctx context.Context, name string) ([]DiskSnapshot, error) {
	disk, err := p.snapshotDisk(name)
	if err != nil {
		return nil, err
	}
	output, err := p.runQemuImgSnapshot(ctx, "-l", disk)
	if err != nil {
		return nil, err
	}
	return parseSnapshotList(output)
}

func (p *QemuProvider) SnapshotRestore(ctx context.Context, name, snapshot string) error {
	disk, err := p.snapshotDisk(name)
	if err != nil {
		return err
	}
	if err := ValidateSnapshotName(snapshot); err != nil {
		return err
	}
	if err := p.requireSnapshot(ctx, name, snapshot); err != nil {
		return err
	}
	if _, err := p.runQemuImgSnapshot(ctx, "-a", snapshot, disk); err != nil {
		return err
	}
	// The next boot is a new process generation; an observation from the old
	// boot is not applicable to the restored disk.
	p.clearReadiness(name)
	return nil
}

func (p *QemuProvider) SnapshotDelete(ctx context.Context, name, snapshot string) error {
	disk, err := p.snapshotDisk(name)
	if err != nil {
		return err
	}
	if err := ValidateSnapshotName(snapshot); err != nil {
		return err
	}
	if err := p.requireSnapshot(ctx, name, snapshot); err != nil {
		return err
	}
	_, err = p.runQemuImgSnapshot(ctx, "-d", snapshot, disk)
	return err
}

// snapshotDisk is the single precondition for every snapshot operation. It
// never stops the VM implicitly: a caller must choose stop or shutdown first.
func (p *QemuProvider) snapshotDisk(name string) (string, error) {
	if err := identifier.Validate(name); err != nil {
		return "", err
	}
	if _, err := p.loadState(name); err != nil {
		return "", fmt.Errorf("VM %q: %w", name, err)
	}
	pid := p.runningPID(name)
	if pid > 0 && processAlive(pid) {
		if !p.isVMProcess(name, pid) {
			return "", fmt.Errorf("PID %d no longer belongs to VM %q", pid, name)
		}
		return "", fmt.Errorf("VM %q is running; stop or shutdown it before snapshot operations", name)
	}
	disk := filepath.Join(p.RootDir, "vms", name+".qcow2")
	if _, err := os.Stat(disk); err != nil {
		return "", fmt.Errorf("VM %q disk: %w", name, err)
	}
	return disk, nil
}

func (p *QemuProvider) requireSnapshot(ctx context.Context, name, snapshot string) error {
	current, err := p.SnapshotList(ctx, name)
	if err != nil {
		return err
	}
	for _, entry := range current {
		if entry.Name == snapshot {
			return nil
		}
	}
	return fmt.Errorf("snapshot %q does not exist on VM %q", snapshot, name)
}

func (p *QemuProvider) runQemuImgSnapshot(ctx context.Context, args ...string) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	qemuImg, err := sysutil.QemuImgBinary()
	if err != nil {
		return nil, err
	}
	runCtx, cancel := context.WithTimeout(ctx, snapshotCommandTimeout)
	defer cancel()
	commandArgs := append([]string{"snapshot"}, args...)
	unlock, err := p.lockSnapshotCommand(args)
	if err != nil {
		return nil, err
	}
	defer unlock()
	output, err := sysutil.NewCommandContext(runCtx, qemuImg, commandArgs...).CombinedOutput()
	if err != nil {
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			return output, fmt.Errorf("qemu-img snapshot timed out")
		}
		detail := snapshotFailureDetail(output, err)
		return output, fmt.Errorf("qemu-img snapshot %s: %s", snapshotOperation(args), detail)
	}
	return output, nil
}

// lockSnapshotCommand serializes Nido snapshot commands per VM across local
// processes. qemu-img also takes its own image lock; this lease gives a clear
// busy error before a concurrent command reaches the image, and is released on
// process exit if the CLI is interrupted.
func (p *QemuProvider) lockSnapshotCommand(args []string) (func(), error) {
	if len(args) == 0 {
		return func() {}, nil
	}
	disk := args[len(args)-1]
	base := filepath.Base(disk)
	if !strings.HasSuffix(base, ".qcow2") {
		return func() {}, nil
	}
	vmName := strings.TrimSuffix(base, ".qcow2")
	runDir := filepath.Join(p.RootDir, "run")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return nil, fmt.Errorf("snapshot lock directory: %w", err)
	}
	lease, err := sysutil.AcquireResource(filepath.Join(runDir, vmName+".snapshot.lock"))
	if err != nil {
		if errors.Is(err, sysutil.ErrResourceBusy) {
			return nil, fmt.Errorf("another snapshot operation is using VM %q; wait and retry", vmName)
		}
		return nil, fmt.Errorf("snapshot lock: %w", err)
	}
	return func() { _ = lease.Close() }, nil
}

func snapshotOperation(args []string) string {
	if len(args) == 0 {
		return "operation"
	}
	switch args[0] {
	case "-c":
		return "create"
	case "-l":
		return "list"
	case "-a":
		return "restore"
	case "-d":
		return "delete"
	default:
		return args[0]
	}
}

func snapshotFailureDetail(output []byte, err error) string {
	detail := boundedSnapshotDetail(string(output))
	if detail == "" {
		detail = err.Error()
	}
	lower := strings.ToLower(detail)
	switch {
	case strings.Contains(lower, "no space left on device"):
		return "no space left on device while writing snapshot data"
	case strings.Contains(lower, "already exists"):
		return "snapshot already exists"
	case strings.Contains(lower, "failed to get") && strings.Contains(lower, "lock"):
		return "disk image is locked by another process; wait for it to finish"
	}
	return detail
}

// ValidateSnapshotName applies the provider-owned tag contract shared by CLI,
// MCP and TUI callers. Tags may contain spaces, but ambiguity and path-like
// values are rejected before they reach qemu-img.
func ValidateSnapshotName(snapshot string) error {
	if snapshot == "" || strings.TrimSpace(snapshot) != snapshot {
		return fmt.Errorf("invalid snapshot name: must be non-empty without leading or trailing whitespace")
	}
	if len(snapshot) > 128 {
		return fmt.Errorf("invalid snapshot name: must be at most 128 bytes")
	}
	if strings.ContainsAny(snapshot, "/\\") {
		return fmt.Errorf("invalid snapshot name: path separators are not allowed")
	}
	for _, r := range snapshot {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("invalid snapshot name: control characters are not allowed")
		}
	}
	return nil
}

func parseSnapshotList(output []byte) ([]DiskSnapshot, error) {
	text := strings.ReplaceAll(string(output), "\r\n", "\n")
	snapshots := []DiskSnapshot{}
	started := false
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "ID") && strings.Contains(trimmed, "TAG") {
			started = true
			continue
		}
		if !started {
			// qemu-img prints "Snapshot list:" before the header; anything
			// else before the header is not snapshot data.
			continue
		}
		match := snapshotLine.FindStringSubmatch(trimmed)
		if match == nil {
			return nil, fmt.Errorf("unrecognized qemu-img snapshot row %q", trimmed)
		}
		id, err := strconv.Atoi(match[1])
		if err != nil {
			return nil, fmt.Errorf("parse snapshot id: %w", err)
		}
		snapshots = append(snapshots, DiskSnapshot{
			ID:        id,
			Name:      match[2],
			VMSize:    strings.TrimSpace(match[3]),
			CreatedAt: match[4],
		})
	}
	return snapshots, nil
}

func boundedSnapshotDetail(text string) string {
	detail := boundedReadinessDetail(text)
	if len(detail) > 512 {
		detail = detail[:512]
	}
	return detail
}
