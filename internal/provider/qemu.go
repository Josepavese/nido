package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/Josepavese/nido/internal/config"
	"github.com/Josepavese/nido/internal/image"
)

// QemuProvider implements VMProvider using raw QEMU.
type QemuProvider struct {
	// RootDir is the base path for VM state and disks (usually ~/.nido)
	RootDir string
	// Config holds the genetic makeup of our nest
	Config *config.Config
}

// NewQemuProvider hatches a new provider, ready to manage lifecycle events.
func NewQemuProvider(rootDir string, cfg *config.Config) *QemuProvider {
	return &QemuProvider{
		RootDir: rootDir,
		Config:  cfg,
	}
}

// Spawn brings a new VM to life. It handles template resolution, disk creation,
// and saves the initial state before handing over to Start.
func (p *QemuProvider) Spawn(name string, opts VMOptions) error {
	// 1. Template Resolution
	tpl := opts.DiskPath
	if tpl == "" {
		tpl = p.Config.TemplateDefault
	}

	if !filepath.IsAbs(tpl) && !strings.Contains(tpl, "/") {
		tpl = filepath.Join(p.Config.BackupDir, tpl+".compact.qcow2")
	}

	// 2. Create Disk
	// Default to 20G, but expand if template is larger
	diskSize := "20G"
	if tpl != "" {
		out, err := exec.Command("qemu-img", "info", "--output=json", tpl).Output()
		if err == nil {
			var info struct {
				VirtualSize int64 `json:"virtual-size"`
			}
			if json.Unmarshal(out, &info) == nil && info.VirtualSize > 0 {
				// Convert virtual size to GB and add 1G buffer (or just use virtual size)
				// qemu-img create handles bytes if we provide a number without suffix
				diskSize = fmt.Sprintf("%d", info.VirtualSize)

				// Ensure at least 20G
				if info.VirtualSize < 20*1024*1024*1024 {
					diskSize = "20G"
				}
			}
		}
	}

	if err := p.CreateDisk(name, diskSize, tpl); err != nil {
		return err
	}

	// 3. Prepare Paths
	runDir := filepath.Join(p.RootDir, "run")
	vmsDir := filepath.Join(p.RootDir, "vms")
	os.MkdirAll(runDir, 0755)
	os.MkdirAll(vmsDir, 0755)

	// 4. Generate Cloud-Init Seed ISO
	seedPath := filepath.Join(vmsDir, name+"-seed.iso")
	sshKey := GetLocalSSHKey()

	// Resolve SSH user first so it can be used for Cloud-Init and state
	sshUser := opts.SSHUser
	if sshUser == "" {
		sshUser = p.Config.SSHUser
	}

	customUserData := ""
	if opts.UserDataPath != "" {
		if data, err := os.ReadFile(opts.UserDataPath); err == nil {
			customUserData = string(data)
			// Handle placeholder if present
			customUserData = strings.ReplaceAll(customUserData, "${SSH_KEY}", sshKey)
		} else {
			fmt.Printf("⚠️  Warning: Failed to read custom user-data file: %v\n", err)
		}
	}

	ci := CloudInit{
		Hostname:       name,
		User:           sshUser,
		SSHKey:         sshKey,
		CustomUserData: customUserData,
	}

	// Create seed ISO (warn on failure but don't block spawn)
	if err := ci.GenerateISO(seedPath); err != nil {
		fmt.Printf("⚠️  Warning: Failed to generate cloud-init seed: %v\n", err)
	}

	// 5. Port Assignment (at spawn time, not start time)
	reserved := p.getReservedPorts()
	sshPort := p.findAvailablePort(50022, reserved)
	vncPort := 0
	if opts.Gui {
		reserved[sshPort] = true // Mark SSH port as reserved before finding VNC
		vncPort = p.findAvailablePort(59000, reserved)
	}

	// 6. Save Initial State (with GUI preference, SSH user, and assigned ports)
	if err := p.saveState(name, 0, sshPort, vncPort, opts.Gui, sshUser); err != nil {
		return fmt.Errorf("failed to save initial state: %w", err)
	}

	// 6. Start
	return p.Start(name, opts)
}

// Start revives a VM from its deep sleep. It handles port allocation,
// builds platform-specific QEMU arguments, and launches the process.
func (p *QemuProvider) Start(name string, opts VMOptions) error {
	// 0. Check if already running
	if status, err := p.Info(name); err == nil && status.State == "running" {
		return nil // Already running
	}

	// 1. Prepare Paths
	runDir := filepath.Join(p.RootDir, "run")
	vmsDir := filepath.Join(p.RootDir, "vms")
	// Already created by Spawn or should be there
	os.MkdirAll(runDir, 0755)
	os.MkdirAll(vmsDir, 0755)

	diskPath := filepath.Join(vmsDir, name+".qcow2")
	if _, err := os.Stat(diskPath); os.IsNotExist(err) {
		return fmt.Errorf("disk image not found: %s", diskPath)
	}

	// 2. Port Management
	state, err := p.loadState(name)
	if err != nil {
		// Fallback for direct 'start' without 'spawn' (e.g. legacy)
		state = VMState{
			Name:    name,
			Gui:     opts.Gui,
			SSHUser: p.Config.SSHUser,
		}
	}

	// We honor the requested GUI flag even if the previously saved state
	// had it disabled. Evolution in action.
	if opts.Gui && !state.Gui {
		state.Gui = true
	}

	// Legacy fallback: assign ports if missing (for VMs created before this fix)
	updated := false
	if state.SSHPort == 0 {
		reserved := p.getReservedPorts()
		state.SSHPort = p.findAvailablePort(50022, reserved)
		updated = true
	}
	if state.Gui && state.VNCPort == 0 {
		reserved := p.getReservedPorts()
		state.VNCPort = p.findAvailablePort(59000, reserved)
		updated = true
	}
	if updated {
		p.saveState(name, 0, state.SSHPort, state.VNCPort, state.Gui, state.SSHUser)
	}

	// 3. Build Arguments (cross-platform)
	args := p.buildQemuArgs(name, diskPath, state.SSHPort, state.VNCPort, runDir)

	cmd := exec.Command("qemu-system-x86_64", args...)
	// In TUI mode, we should NOT print to stderr/stdout as it corrupts the UI.
	// We rely on returning errors to be logged by the caller.

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("QEMU failed to start: %w (stderr: %s)", err, stderr.String())
	}

	// 4. Skip Bootloader (Background)
	// We send "Enter" key via QMP to skip any guest countdowns (Alpine, GRUB, etc.)
	// because agents don't have time to wait for timers.
	go p.skipBootloader(name)

	// 5. Read daemon PID from QEMU pidfile
	// QEMU daemonizes itself, so we wait for it to write its PID to disk
	// so we can keep track of our hatchlings.
	pidFile := filepath.Join(runDir, name+".pid")
	pid := 0
	for i := 0; i < 10; i++ {
		pidData, err := os.ReadFile(pidFile)
		if err == nil {
			fmt.Sscanf(string(pidData), "%d", &pid)
			if pid > 0 {
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	// 6. Update State with PID (0 if unknown)
	p.saveState(name, pid, state.SSHPort, state.VNCPort, state.Gui, state.SSHUser)

	return nil
}

// buildQemuArgs constructs the heavy-duty command line arguments for QEMU.
// It detects the host OS to enable hardware acceleration: KVM for Linux,
// HVF for macOS, and WHPX for Windows.
func (p *QemuProvider) buildQemuArgs(name, diskPath string, sshPort int, vncPort int, runDir string) []string {
	args := []string{
		"-name", name,
		"-m", "2048",
		// Using 'pc' (i440fx) by default for maximum compatibility with legacy images (CirrOS, etc.)
		"-machine", "pc",
	}

	// Platform-specific acceleration
	switch runtime.GOOS {
	case "linux":
		if _, err := os.Stat("/dev/kvm"); err == nil {
			args = append(args, "-enable-kvm", "-cpu", "host")
		} else {
			// Fallback to TCG (no acceleration) for CI/CD environments without KVM
			args = append(args, "-cpu", "qemu64")
		}
	case "darwin": // macOS
		args = append(args, "-accel", "hvf", "-cpu", "host")
	case "windows":
		args = append(args, "-accel", "whpx", "-cpu", "host")
	default:
		args = append(args, "-cpu", "qemu64")
	}

	// Common arguments
	args = append(args,
		"-drive", fmt.Sprintf("file=%s,format=qcow2,if=virtio", diskPath),
		"-daemonize",
		"-pidfile", filepath.Join(runDir, name+".pid"),
		"-netdev", fmt.Sprintf("user,id=net0,hostfwd=tcp::%d-:22", sshPort),
		"-device", "virtio-net-pci,netdev=net0",
		"-boot", "menu=off,strict=on,splash-time=0", // Fast boot: skip menu, no splash timeout
		"-serial", "file:"+filepath.Join(runDir, name+".serial.log"),
	)

	// Attach Cloud-Init Seed if exists
	seedPath := filepath.Join(p.RootDir, "vms", name+"-seed.iso")
	if _, err := os.Stat(seedPath); err == nil {
		args = append(args, "-cdrom", seedPath)
	}

	// QMP socket (platform-specific path handling)
	if runtime.GOOS == "windows" {
		// Windows uses named pipes for QMP
		args = append(args, "-qmp", "tcp:127.0.0.1:0,server,nowait")
	} else {
		// Unix-like systems use Unix sockets
		args = append(args, "-qmp", "unix:"+filepath.Join(runDir, name+".qmp")+",server,nowait")
	}

	// VNC Support
	if vncPort > 0 {
		// QEMU uses display numbers (port - 5900)
		display := vncPort - 5900
		args = append(args, "-vnc", fmt.Sprintf("127.0.0.1:%d", display))
	} else {
		args = append(args, "-display", "none")
	}

	return args
}

// List scans the nest and identifies all existing VMs and their current state.
func (p *QemuProvider) List() ([]VMStatus, error) {
	vmsDir := filepath.Join(p.RootDir, "vms")
	files, err := os.ReadDir(vmsDir)
	if err != nil {
		return nil, err
	}

	var results []VMStatus
	for _, f := range files {
		// We only care about base qcow2 images (excluding templates/isos if naming convention strictly follows vmname.qcow2)
		if filepath.Ext(f.Name()) == ".qcow2" && !strings.HasSuffix(f.Name(), ".compact.qcow2") {
			name := strings.TrimSuffix(f.Name(), ".qcow2")

			// Check runtime state
			pidFile := filepath.Join(p.RootDir, "run", name+".pid")
			pidData, _ := os.ReadFile(pidFile)
			pid := 0
			fmt.Sscanf(string(pidData), "%d", &pid)

			stateStr := "stopped"
			if pid > 0 {
				process, err := os.FindProcess(pid)
				if err == nil && process.Signal(syscall.Signal(0)) == nil {
					stateStr = "running"
				}
			}

			// Load state if exists (for ports etc)
			// If stopped, state json might persist or not?
			// Stop() removes pid/qmp but currently leaves json? No, Stop() doesn't remove json in qemu.go currently.
			// Let's check Stop implementation. It removes pid, qmp. It does NOT remove json.
			vmState, _ := p.loadState(name)

			results = append(results, VMStatus{
				Name:    name,
				State:   stateStr,
				PID:     pid,
				SSHPort: vmState.SSHPort,
				SSHUser: vmState.SSHUser,
				VNCPort: vmState.VNCPort, // Added VNCPort to struct if available
			})
		}
	}
	return results, nil
}

// Info dives deep into a VM's neural links, returning detailed state,
// networking information, and disk health (including backing file integrity).
func (p *QemuProvider) Info(name string) (VMDetail, error) {
	state, err := p.loadState(name)
	if err != nil {
		// If no state file, assume VM is stopped and has no active ports
		diskPath := filepath.Join(p.RootDir, "vms", name+".qcow2")
		_, statErr := os.Stat(diskPath)
		backingPath, backingMissing := backingInfo(diskPath)
		return VMDetail{
			Name:           name,
			State:          "stopped",
			IP:             "127.0.0.1",
			DiskPath:       diskPath,
			DiskMissing:    statErr != nil,
			BackingPath:    backingPath,
			BackingMissing: backingMissing,
		}, nil
	}

	// Check liveness for state string
	liveness := "stopped"
	pidData, _ := os.ReadFile(filepath.Join(p.RootDir, "run", name+".pid"))
	pid := 0
	fmt.Sscanf(string(pidData), "%d", &pid)
	if pid > 0 {
		process, err := os.FindProcess(pid)
		if err == nil && process.Signal(syscall.Signal(0)) == nil {
			liveness = "running"
		}
	}

	diskPath := filepath.Join(p.RootDir, "vms", name+".qcow2")
	_, statErr := os.Stat(diskPath)
	backingPath, backingMissing := backingInfo(diskPath)

	return VMDetail{
		Name:           name,
		State:          liveness,
		PID:            pid,
		IP:             "127.0.0.1",
		SSHUser:        state.SSHUser,
		SSHPort:        state.SSHPort,
		VNCPort:        state.VNCPort,
		DiskPath:       diskPath,
		DiskMissing:    statErr != nil,
		BackingPath:    backingPath,
		BackingMissing: backingMissing,
	}, nil
}

// backingInfo returns backing filename and whether it's missing.
func backingInfo(diskPath string) (string, bool) {
	if diskPath == "" {
		return "", false
	}
	type info struct {
		Backing string `json:"backing-filename"`
	}
	out, err := exec.Command("qemu-img", "info", "-U", "--output=json", diskPath).Output()
	if err != nil {
		return "", false
	}
	var meta info
	if json.Unmarshal(out, &meta) != nil || meta.Backing == "" {
		return "", false
	}
	if _, err := os.Stat(meta.Backing); err != nil {
		return meta.Backing, true
	}
	return meta.Backing, false
}

// Stop gracefully asks the VM to go into deep sleep using an interrupt signal.
// We clean up QMP and PID artifacts to keep the run directory tidy.
func (p *QemuProvider) Stop(name string, graceful bool) error {
	runDir := filepath.Join(p.RootDir, "run")
	pidFile := filepath.Join(runDir, name+".pid")
	pidData, _ := os.ReadFile(pidFile)
	pid := 0
	fmt.Sscanf(string(pidData), "%d", &pid)

	if pid > 0 {
		process, _ := os.FindProcess(pid)
		process.Signal(os.Interrupt)
		for i := 0; i < 50; i++ {
			if process.Signal(syscall.Signal(0)) != nil {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	os.Remove(pidFile)
	os.Remove(filepath.Join(runDir, name+".qmp"))
	return nil
}

// Delete evicts a VM from the nest permanently. No going back.
func (p *QemuProvider) Delete(name string) error {
	p.Stop(name, false)
	vmsDir := filepath.Join(p.RootDir, "vms")
	diskPath := filepath.Join(vmsDir, name+".qcow2")
	os.Remove(filepath.Join(p.RootDir, "run", name+".json"))
	os.Remove(filepath.Join(vmsDir, name+"-seed.iso"))
	return os.Remove(diskPath)
}

// CreateTemplate archives a VM into "cold storage" (a compressed qcow2).
// This is how we preserve perfected environments for future hatchlings.
func (p *QemuProvider) CreateTemplate(vmName string, templateName string) (string, error) {
	// 1. Ensure VM is stopped
	p.Stop(vmName, true)

	vmsDir := filepath.Join(p.RootDir, "vms")
	backupsDir := p.Config.BackupDir
	os.MkdirAll(backupsDir, 0755)

	srcDisk := filepath.Join(vmsDir, vmName+".qcow2")
	targetTemplate := filepath.Join(backupsDir, templateName+".compact.qcow2")

	if _, err := os.Stat(srcDisk); os.IsNotExist(err) {
		return "", fmt.Errorf("source disk not found: %s", srcDisk)
	}

	// qemu-img convert -O qcow2 -c <src> <dest>
	cmd := exec.Command("qemu-img", "convert", "-O", "qcow2", "-c", srcDisk, targetTemplate)
	if err := cmd.Run(); err != nil {
		return "", err
	}

	return targetTemplate, nil
}

func (p *QemuProvider) DeleteTemplate(name string) error {
	templatePath := filepath.Join(p.Config.BackupDir, name+".compact.qcow2")
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return fmt.Errorf("template not found: %s", name)
	}
	return os.Remove(templatePath)
}

// CreateDisk prepares the execution surface (the qcow2 file).
// It supports both standalone "Full Copies" and space-saving "Linked Clones".
func (p *QemuProvider) CreateDisk(name, size, tpl string) error {
	vmsDir := filepath.Join(p.RootDir, "vms")
	os.MkdirAll(vmsDir, 0755)
	target := filepath.Join(vmsDir, name+".qcow2")
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("disk already exists: %s", target)
	}

	// 1. Full Copy Mode (No Cache)
	// If LinkedClones is disabled and we have a template, create a standalone copy.
	if !p.Config.LinkedClones && tpl != "" {
		// Convert (Full Copy)
		if out, err := exec.Command("qemu-img", "convert", "-O", "qcow2", tpl, target).CombinedOutput(); err != nil {
			return fmt.Errorf("create(convert) failed: %v (%s)", err, string(out))
		}
		// Resize to target size
		if out, err := exec.Command("qemu-img", "resize", target, size).CombinedOutput(); err != nil {
			return fmt.Errorf("create(resize) failed: %v (%s)", err, string(out))
		}
		return nil
	}

	// 2. Linked Clone Mode (Default)
	args := []string{"create", "-f", "qcow2"}
	if tpl != "" {
		// Autodetect template format
		format := "qcow2"
		out, err := exec.Command("qemu-img", "info", "--output=json", tpl).Output()
		if err == nil {
			var info struct {
				Format string `json:"format"`
			}
			if json.Unmarshal(out, &info) == nil && info.Format != "" {
				format = info.Format
			}
		}
		args = append(args, "-b", tpl, "-F", format)
	}
	args = append(args, target, size)

	cmd := exec.Command("qemu-img", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("qemu-img create failed: %v (%s)", err, string(out))
	}
	return nil
}

// Helpers

func (p *QemuProvider) getReservedPorts() map[int]bool {
	reserved := make(map[int]bool)
	runDir := filepath.Join(p.RootDir, "run")
	files, err := os.ReadDir(runDir)
	if err != nil {
		return reserved
	}
	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(runDir, f.Name()))
		if err != nil {
			continue
		}
		var state VMState
		if json.Unmarshal(data, &state) == nil {
			if state.SSHPort > 0 {
				reserved[state.SSHPort] = true
			}
			if state.VNCPort > 0 {
				reserved[state.VNCPort] = true
			}
		}
	}
	return reserved
}

func (p *QemuProvider) findAvailablePort(start int, reserved map[int]bool) int {
	for port := start; port < start+100; port++ {
		if reserved[port] {
			continue // Skip ports reserved by other VMs
		}
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			ln.Close()
			return port
		}
	}
	return 0
}

type VMState struct {
	Name    string `json:"name"`
	PID     int    `json:"pid"`
	SSHPort int    `json:"ssh_port"`
	VNCPort int    `json:"vnc_port,omitempty"`
	Gui     bool   `json:"gui,omitempty"`
	SSHUser string `json:"ssh_user,omitempty"`
}

func (p *QemuProvider) saveState(name string, pid int, sshPort int, vncPort int, gui bool, sshUser string) error {
	state := VMState{Name: name, PID: pid, SSHPort: sshPort, VNCPort: vncPort, Gui: gui, SSHUser: sshUser}
	data, _ := json.MarshalIndent(state, "", "  ")
	return os.WriteFile(filepath.Join(p.RootDir, "run", name+".json"), data, 0644)
}

func (p *QemuProvider) loadState(name string) (VMState, error) {
	var state VMState
	data, err := os.ReadFile(filepath.Join(p.RootDir, "run", name+".json"))
	if err != nil {
		return state, err
	}
	err = json.Unmarshal(data, &state)
	return state, err
}

func (p *QemuProvider) SSHCommand(name string) (string, error) {
	info, err := p.Info(name)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("ssh -p %d %s@%s", info.SSHPort, info.SSHUser, info.IP), nil
}

func (p *QemuProvider) ListTemplates() ([]string, error) {
	files, err := os.ReadDir(p.Config.BackupDir)
	if err != nil {
		return nil, err
	}
	var templates []string
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".compact.qcow2") {
			name := strings.TrimSuffix(f.Name(), ".compact.qcow2")
			templates = append(templates, name)
		}
	}
	return templates, nil
}

func (p *QemuProvider) Prune() (int, error) {
	vms, err := p.List()
	if err != nil {
		return 0, err
	}
	count := 0
	for _, vm := range vms {
		if vm.State == "stopped" {
			if err := p.Delete(vm.Name); err == nil {
				count++
			}
		}
	}
	return count, nil
}

func (p *QemuProvider) GetConfig() config.Config {
	return *p.Config
}

func (p *QemuProvider) Doctor() []string {
	var reports []string
	add := func(label string, passed bool, details string) {
		status := "[PASS]"
		if !passed {
			status = "[FAIL]"
		}
		reports = append(reports, fmt.Sprintf("%-20s %s %s", label, status, details))
	}

	// 1. Directories
	dirs := []string{p.RootDir, filepath.Join(p.RootDir, "bin"), filepath.Join(p.RootDir, "vms"), filepath.Join(p.RootDir, "run")}
	if p.Config != nil {
		dirs = append(dirs, p.Config.BackupDir)
	}
	for _, d := range dirs {
		_, err := os.Stat(d)
		add("Dir: "+filepath.Base(d), err == nil, d)
	}

	// 2. Binaries
	qemu, err := exec.LookPath("qemu-system-x86_64")
	add("Binary: QEMU", err == nil, qemu)

	qimg, err := exec.LookPath("qemu-img")
	add("Binary: qemu-img", err == nil, qimg)

	// 3. KVM (Linux only)
	if runtime.GOOS == "linux" {
		_, err := os.Stat("/dev/kvm")
		add("Accel: KVM", err == nil, "/dev/kvm accessibility")
	}

	return reports
}

func (p *QemuProvider) GetUsedBackingFiles() ([]string, error) {
	vmsDir := filepath.Join(p.RootDir, "vms")
	files, err := os.ReadDir(vmsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	used := make(map[string]bool)
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".qcow2") {
			diskPath := filepath.Join(vmsDir, f.Name())
			backing, exists := backingInfo(diskPath)
			if !exists && backing != "" {
				// Backing file is set but might be missing, still record it
				// backingInfo returns true if missing, false if present.
				// Actually backingInfo returns (path, missing bool).
				// We want the path regardless.
			}
			if backing != "" {
				abs, err := filepath.Abs(backing)
				if err == nil {
					used[abs] = true
				} else {
					used[backing] = true
				}
			}
		}
	}

	var result []string
	for path := range used {
		result = append(result, path)
	}
	return result, nil
}

// skipBootloader is our secret weapon for speed. It connects via QMP
// and mashes the "Enter" key while the VM is starting up to bypass
// guest bootloader menus.
func (p *QemuProvider) skipBootloader(name string) {
	qmpPath := filepath.Join(p.RootDir, "run", name+".qmp")
	if runtime.GOOS == "windows" {
		return
	}

	// 1. Initial wait: Give BIOS/UEFI time to finish and reach bootloader (3s)
	time.Sleep(3 * time.Second)

	// 2. Connect to QMP
	conn, err := net.DialTimeout("unix", qmpPath, 500*time.Millisecond)
	if err != nil {
		return
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	// 3. Handshake
	decoder := json.NewDecoder(conn)
	var greeting map[string]interface{}
	_ = decoder.Decode(&greeting)
	fmt.Fprintf(conn, `{"execute":"qmp_capabilities"}`+"\n")
	_ = decoder.Decode(&greeting)

	// 4. Send "Return" exactly 3 times with 1s gap
	// This covers potential UI lag or early bootloader states
	for i := 0; i < 3; i++ {
		cmd, _ := json.Marshal(map[string]interface{}{
			"execute": "send-key",
			"arguments": map[string]interface{}{
				"keys": []map[string]interface{}{
					{"type": "qcode", "data": "ret"},
				},
			},
		})
		fmt.Fprintf(conn, "%s\n", string(cmd))

		var res map[string]interface{}
		_ = decoder.Decode(&res)

		time.Sleep(1 * time.Second)
	}
}

func (p *QemuProvider) execQMP(qmpPath string, command map[string]interface{}) error {
	// Keep existing execQMP for general use if needed, but skipBootloader uses its own persistent conn
	conn, err := net.DialTimeout("unix", qmpPath, 100*time.Millisecond)
	if err != nil {
		return err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(100 * time.Millisecond))

	decoder := json.NewDecoder(conn)
	var greeting map[string]interface{}
	decoder.Decode(&greeting)
	fmt.Fprintf(conn, `{"execute":"qmp_capabilities"}`+"\n")
	decoder.Decode(&greeting)

	data, _ := json.Marshal(command)
	fmt.Fprintf(conn, "%s\n", data)
	return decoder.Decode(&greeting)
}

// ListImages scans the image directory for cached images.
func (p *QemuProvider) ListImages() ([]string, error) {
	imagesDir := p.Config.ImageDir
	if imagesDir == "" {
		imagesDir = filepath.Join(p.RootDir, "images")
	}

	// Load Catalog (handles local cache if network is down)
	catalog, err := image.LoadCatalog(imagesDir, image.DefaultCacheTTL)
	if err != nil {
		// If catalog fails, we can't show much. Return error.
		return nil, err
	}

	var images []string
	for _, img := range catalog.Images {
		for _, ver := range img.Versions {
			images = append(images, fmt.Sprintf("%s:%s", img.Name, ver.Version))
		}
	}
	return images, nil
}

// ListCachedImages returns all locally cached cloud images.
func (p *QemuProvider) ListCachedImages() ([]CachedImage, error) {
	imagesDir := p.Config.ImageDir
	if imagesDir == "" {
		imagesDir = filepath.Join(p.RootDir, "images")
	}

	files, err := os.ReadDir(imagesDir)
	if err != nil {
		return nil, err
	}

	var items []CachedImage
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".qcow2") {
			continue
		}
		info, err := f.Info()
		if err != nil {
			continue
		}
		// Parse name and version from filename (e.g., "ubuntu-24.04.qcow2")
		name := strings.TrimSuffix(f.Name(), ".qcow2")
		parts := strings.Split(name, "-")
		imageName := parts[0]
		version := ""
		if len(parts) > 1 {
			version = strings.Join(parts[1:], "-")
		}
		items = append(items, CachedImage{
			Name:    imageName,
			Version: version,
			Size:    formatBytes(info.Size()),
		})
	}
	return items, nil
}

// CacheInfo returns statistics about the image cache.
func (p *QemuProvider) CacheInfo() (CacheInfoResult, error) {
	imagesDir := p.Config.ImageDir
	if imagesDir == "" {
		imagesDir = filepath.Join(p.RootDir, "images")
	}

	files, err := os.ReadDir(imagesDir)
	if err != nil {
		return CacheInfoResult{}, err
	}

	var totalSize int64
	count := 0
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".qcow2") {
			continue
		}
		info, err := f.Info()
		if err != nil {
			continue
		}
		totalSize += info.Size()
		count++
	}
	return CacheInfoResult{
		Count:     count,
		TotalSize: formatBytes(totalSize),
	}, nil
}

// CachePrune removes cached images.
func (p *QemuProvider) CachePrune(unusedOnly bool) error {
	imagesDir := p.Config.ImageDir
	if imagesDir == "" {
		imagesDir = filepath.Join(p.RootDir, "images")
	}

	usedBacking, err := p.GetUsedBackingFiles()
	if err != nil {
		return err
	}
	usedMap := make(map[string]bool)
	for _, path := range usedBacking {
		usedMap[path] = true
	}

	files, err := os.ReadDir(imagesDir)
	if err != nil {
		return err
	}

	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".qcow2") {
			continue
		}
		fullPath := filepath.Join(imagesDir, f.Name())
		if unusedOnly && usedMap[fullPath] {
			continue // Skip images in use
		}
		os.Remove(fullPath)
	}
	return nil
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
