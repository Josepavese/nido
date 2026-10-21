package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Josepavese/nido/internal/pkg/sysutil"

	"github.com/Josepavese/nido/internal/config"
	"github.com/Josepavese/nido/internal/hardware"
	"github.com/Josepavese/nido/internal/image"
)

// QemuProvider implements VMProvider using raw QEMU. Lifecycle and configuration
// mutations are confined to the caller/owner thread (MCP dispatch is serial).
type QemuProvider struct {
	// RootDir is the base path for VM state and disks (usually ~/.nido)
	RootDir string
	// Config holds the genetic makeup of our nest
	Config   *config.Config
	Hardware hardware.Host
}

// NewQemuProvider hatches a new provider, ready to manage lifecycle events.
func NewQemuProvider(rootDir string, cfg *config.Config, host hardware.Host) *QemuProvider {
	return &QemuProvider{
		RootDir:  rootDir,
		Config:   cfg,
		Hardware: host,
	}
}

// Spawn brings a new VM to life. It handles template resolution, disk creation,
// and saves the initial state before handing over to Start.
func (p *QemuProvider) Spawn(name string, opts VMOptions) error {
	if err := validateIdentifier(name); err != nil {
		return err
	}
	// 0. Validate input
	if err := hardware.ValidateAccelerators(opts.Accelerators); err != nil {
		return err
	}
	for _, pf := range opts.Forwarding {
		if err := ValidatePortForward(pf); err != nil {
			return err
		}
	}

	// 0.5 Auto-Discovery for Accelerators
	resolvedAccels, err := p.resolveAutoAccelerators(opts.Accelerators)
	if err != nil {
		return fmt.Errorf("accelerator discovery: %w", err)
	}
	opts.Accelerators = resolvedAccels

	// 1. Template/Image Resolution
	tpl := opts.DiskPath

	// If it's not an absolute path and doesn't contain a slash, it's either a Template or an Image Tag
	if !filepath.IsAbs(tpl) && !strings.Contains(tpl, "/") && tpl != "" {
		// Defaults fallback if Config is missing (safeguard)
		var backupsDir, imgDir string
		if p.Config != nil {
			backupsDir = p.Config.BackupDir
			imgDir = p.Config.ImageDir
		} else {
			home, _ := sysutil.UserHome()
			backupsDir = filepath.Join(home, ".nido", "backups")
			imgDir = filepath.Join(home, ".nido", "images")
		}

		templatePath := filepath.Join(backupsDir, tpl+".compact.qcow2")
		if _, err := os.Stat(templatePath); err == nil {
			tpl = templatePath
		} else {
			// Resolve as Image Tag (e.g., "ubuntu:24.04")
			catalog, err := image.LoadCatalogFromFile(filepath.Join(imgDir, image.CatalogCacheFile))
			if err == nil {
				pName, pVer := tpl, ""
				if strings.Contains(tpl, ":") {
					parts := strings.Split(tpl, ":")
					pName, pVer = parts[0], parts[1]
				}
				img, ver, err := catalog.FindImage(pName, pVer)
				if err == nil {
					tpl = filepath.Join(imgDir, fmt.Sprintf("%s-%s.qcow2", img.Name, ver.Version))
					if opts.SSHUser == "" && img.SSHUser != "" {
						opts.SSHUser = img.SSHUser
					}
					if opts.SSHPassword == "" {
						if ver.SSHPassword != "" {
							opts.SSHPassword = ver.SSHPassword
						} else if img.SSHPassword != "" {
							opts.SSHPassword = img.SSHPassword
						}
					}
				}
			}
		}
	}

	// 2. Create Disk
	diskSize := sysutil.DefaultDiskSize
	if tpl != "" {
		qemuImg, qemuImgErr := sysutil.QemuImgBinary()
		if qemuImgErr == nil {
			if out, err := sysutil.NewCommand(qemuImg, "info", "--output=json", tpl).Output(); err == nil {
				var info struct {
					VirtualSize int64 `json:"virtual-size"`
				}
				if json.Unmarshal(out, &info) == nil && info.VirtualSize > 0 {
					diskSize = fmt.Sprintf("%d", info.VirtualSize)
					if info.VirtualSize < sysutil.DefaultDiskBytes {
						diskSize = sysutil.DefaultDiskSize
					}
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
	if _, err := sysutil.EnsureDir(runDir); err != nil {
		return err
	}
	if _, err := sysutil.EnsureDir(vmsDir); err != nil {
		return err
	}

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
		}
	}

	ci := CloudInit{
		Hostname:       name,
		User:           sshUser,
		SSHKey:         sshKey,
		CustomUserData: customUserData,
		ExtraFiles:     opts.SeedFiles,
	}

	// Create seed ISO (warn on failure but don't block spawn)
	// Create seed ISO (warn on failure but don't block spawn)
	// Atomic Provisioning via sysutil
	if err := sysutil.ProvisionFile(seedPath, func() error {
		return ci.GenerateISO(seedPath)
	}); err != nil {
		// Seed ISO is best-effort and must not block VM creation.
	}

	// 5. Port Assignment (at spawn time, not start time)
	reserved := p.getReservedPorts()
	sshPort := p.findAvailablePort(50022, reserved)
	vncPort := 0
	if opts.Gui {
		reserved[sshPort] = true // Mark SSH port as reserved before finding VNC
		vncPort = p.findAvailablePort(59000, reserved)
	}

	var resolveErr error
	opts.Forwarding, resolveErr = p.resolveForwarding(opts.Forwarding, sshPort, vncPort)
	if resolveErr != nil {
		return resolveErr
	}

	// 6. Save Initial State (with GUI preference, SSH user, assigned ports, and forwardings)
	// Track kernel/initrd/cmdline if they were part of the template resolution
	// (currently handled implicitly by Start checking for files, but let's be explicit if we can)

	// Default resources if not set (Dynamic resolution at spawn time via sysutil SSOT)
	mem := opts.MemoryMB
	if mem == 0 {
		mem = sysutil.DefaultMemory()
	}
	cpu := opts.VCPUs
	if cpu == 0 {
		cpu = sysutil.DefaultVCPUs()
	}

	if err := p.saveState(name, 0, sshPort, vncPort, opts.Gui, sshUser, opts.Forwarding, opts.Cmdline, mem, cpu, opts.RawQemuArgs, opts.Accelerators); err != nil {
		return fmt.Errorf("failed to save initial state: %w", err)
	}

	// 6.1. Copy Kernel/Initrd if they exist for the template
	if tpl != "" {
		// If tpl is a cached image (ends in .qcow2), check for .kernel/.initrd
		if strings.HasSuffix(tpl, ".qcow2") {
			base := strings.TrimSuffix(tpl, ".qcow2")
			kernelSrc := base + ".kernel"
			initrdSrc := base + ".initrd"

			if _, err := os.Stat(kernelSrc); err == nil {
				dst := filepath.Join(vmsDir, name+".kernel")
				sysutil.CopyFile(kernelSrc, dst)
			}
			if _, err := os.Stat(initrdSrc); err == nil {
				dst := filepath.Join(vmsDir, name+".initrd")
				sysutil.CopyFile(initrdSrc, dst)
			}
		}
	}

	// 7. Start
	return p.Start(name, opts)
}

// Start revives a VM from its deep sleep. It handles port allocation,
// builds platform-specific QEMU arguments, and launches the process.
func (p *QemuProvider) Start(name string, opts VMOptions) error {
	if err := validateIdentifier(name); err != nil {
		return err
	}
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
		if !os.IsNotExist(err) {
			return fmt.Errorf("read VM state: %w", err)
		}
		// Recover an existing disk that has no persistent runtime state.
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

	// Assign ports for disk-only recovery.
	if state.SSHPort == 0 {
		reserved := p.getReservedPorts()
		state.SSHPort = p.findAvailablePort(50022, reserved)
	}
	if state.Gui && state.VNCPort == 0 {
		reserved := p.getReservedPorts()
		state.VNCPort = p.findAvailablePort(59000, reserved)
	}
	// If opts.Cmdline is provided, override the state's cmdline
	if opts.Cmdline != "" {
		state.Cmdline = opts.Cmdline
	}
	if opts.MemoryMB > 0 {
		state.MemoryMB = opts.MemoryMB
	}
	if opts.VCPUs > 0 {
		state.VCPUs = opts.VCPUs
	}

	if len(opts.RawQemuArgs) > 0 {
		state.RawQemuArgs = opts.RawQemuArgs
	}

	if len(opts.Accelerators) > 0 {
		if err := hardware.ValidateAccelerators(opts.Accelerators); err != nil {
			return err
		}
		state.Accelerators = opts.Accelerators
	}

	// Resolve absent resources through the canonical host defaults.
	if state.MemoryMB == 0 {
		state.MemoryMB = sysutil.DefaultMemory()
	}
	if state.VCPUs == 0 {
		state.VCPUs = sysutil.DefaultVCPUs()
	}

	resolved, err := p.resolveForwarding(state.Forwarding, state.SSHPort, state.VNCPort)
	if err != nil {
		return err
	}
	state.Forwarding = resolved
	if err := p.saveState(name, 0, state.SSHPort, state.VNCPort, state.Gui, state.SSHUser, state.Forwarding, state.Cmdline, state.MemoryMB, state.VCPUs, state.RawQemuArgs, state.Accelerators); err != nil {
		return err
	}

	// 2.5 Prepare Accelerators (Zero-Config)
	// This requires root privileges. If not root, we might fail or prompt.
	// We do this BEFORE building args.
	prepared := []string{}
	rollback := func(cause error) error {
		for i := len(prepared) - 1; i >= 0; i-- {
			if err := p.restorePassthrough(name, prepared[i]); err != nil {
				cause = errors.Join(cause, fmt.Errorf("restore accelerator %s: %w", prepared[i], err))
			}
		}
		return cause
	}
	for _, pciID := range state.Accelerators {
		if err := p.preparePassthrough(name, pciID); err != nil {
			return rollback(fmt.Errorf("failed to prepare accelerator %s: %w", pciID, err))
		}
		if pciID != hardware.VirtualGPUAcceleratorID {
			prepared = append(prepared, pciID)
		}
	}

	// 3. Build Arguments (cross-platform)
	args := p.buildQemuArgs(name, diskPath, state.SSHPort, state.VNCPort, state.Forwarding, runDir, state.Cmdline, state.MemoryMB, state.VCPUs, state.RawQemuArgs, state.Accelerators)

	launchedPID, err := p.launchQEMU(args)
	if err != nil {
		return rollback(err)
	}

	// 4. Skip Bootloader (Background)
	go p.skipBootloader(name)

	// 5. Read daemon PID from QEMU pidfile on Unix; Windows has no -daemonize.
	pid := readLaunchedQemuPID(runDir, name, launchedPID)

	// 6. Update State with PID (0 if unknown)
	p.saveState(name, pid, state.SSHPort, state.VNCPort, state.Gui, state.SSHUser, state.Forwarding, state.Cmdline, state.MemoryMB, state.VCPUs, state.RawQemuArgs, state.Accelerators)

	return nil
}

func windowsTCGFallbackArgs(args []string) []string {
	out := append([]string(nil), args...)
	for i := 0; i < len(out)-1; i++ {
		switch out[i] {
		case "-accel":
			if out[i+1] == "whpx" {
				out[i+1] = "tcg"
			}
		case "-cpu":
			if out[i+1] == "host" {
				out[i+1] = "qemu64"
			}
		}
	}
	return out
}

// buildQemuArgs constructs the heavy-duty command line arguments for QEMU.
func (p *QemuProvider) buildQemuArgs(name, diskPath string, sshPort int, vncPort int, fw []PortForward, runDir string, cmdline string, memoryMB, vcpus int, rawArgs []string, accelerators []string) []string {
	vmsDir := filepath.Join(p.RootDir, "vms")

	// Safe minimums if 0 (for robustness, should be handled by Spawn)
	if memoryMB == 0 {
		memoryMB = 128
	}
	if vcpus == 0 {
		vcpus = 1
	}

	args := []string{
		"-name", name,
		"-m", fmt.Sprintf("%d", memoryMB),
		"-machine", "pc",
	}

	// Inject Raw Args early (so they can override defaults if needed, though QEMU parsing order varies)
	// We append them at the end usually, but let's put them before device definitions to be safe?
	// Actually, usually users want to append devices. So let's append them at the very end.

	// ... (rest of standard args)

	capabilities := p.Hardware.Virtualization()
	cpuArg := capabilities.CPU
	args = append(args, capabilities.QEMUArgs()...)

	// Override CPU if user requested specific count or default
	if vcpus > 0 {
		args = append(args, "-smp", fmt.Sprintf("%d", vcpus))
	}

	args = append(args, "-cpu", cpuArg)
	// Direct Kernel Boot Support
	kernelPath := filepath.Join(vmsDir, name+".kernel")
	initrdPath := filepath.Join(vmsDir, name+".initrd")
	if _, err := os.Stat(kernelPath); err == nil {
		args = append(args, "-kernel", kernelPath)
		if _, err := os.Stat(initrdPath); err == nil {
			args = append(args, "-initrd", initrdPath)
		}
		finalCmdline := cmdline
		// If cmdline is explicitly updated to empty string by user, this logic effectively prevents clearing it.
		// However, in our system, empty usually means "use default".
		if finalCmdline == "" {
			// Default cmdline for cloud images (vga console + serial + root)
			// For now we use a sensible default that matches nido-init's needs.
			finalCmdline = "root=/dev/sda rw console=ttyS0 console=tty0"
		}
		args = append(args, "-append", finalCmdline)
	}

	// Common arguments
	args = append(args,
		"-drive", fmt.Sprintf("file=%s,format=qcow2,if=virtio", diskPath),
	)
	args = append(args, qemuDaemonArgs(runDir, name)...)

	args = append(args,
		"-netdev", p.BuildNetDevArgs(sshPort, fw),
		"-device", "virtio-net-pci,netdev=net0",
		"-boot", "menu=off,strict=on,splash-time=0", // Fast boot: skip menu, no splash timeout
		"-serial", "file:"+filepath.Join(runDir, name+".serial.log"),
		"-device", "virtio-rng-pci", // Passthrough entropy from host to avoid boot hangs
	)

	// Attach Cloud-Init Seed if exists
	seedPath := filepath.Join(p.RootDir, "vms", name+"-seed.iso")
	if _, err := os.Stat(seedPath); err == nil {
		args = append(args, "-cdrom", seedPath)
	}

	args = append(args, "-qmp", qemuQMPEndpoint(runDir, name))

	// VNC Support
	if vncPort > 0 {
		// QEMU uses display numbers (port - 5900)
		display := vncPort - 5900
		args = append(args, "-vnc", fmt.Sprintf("127.0.0.1:%d", display))
	} else {
		args = append(args, "-display", "none")
	}

	// Inject Accelerators (VFIO or Virtual)
	// Linux Only for VFIO, Cross-Platform for Virtual
	for _, acc := range accelerators {
		if acc == hardware.VirtualGPUAcceleratorID {
			args = append(args, "-device", virtualGPUDevice())
			continue
		}

		// Physical IDs only reach argument construction after the HAL has
		// successfully prepared them; unsupported hosts fail before launch.
		args = append(args, "-device", fmt.Sprintf("vfio-pci,host=%s", acc))

	}

	// Inject Raw Arguments at the very end to allow overriding or adding devices
	if len(rawArgs) > 0 {
		args = append(args, rawArgs...)
	}

	return args
}

// List scans the nest and identifies all existing VMs and their current state.
func (p *QemuProvider) List() ([]VMStatus, error) {
	vmsDir := filepath.Join(p.RootDir, "vms")
	files, err := os.ReadDir(vmsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []VMStatus{}, nil
		}
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
			vmState, stateErr := p.loadState(name)
			if stateErr != nil && !os.IsNotExist(stateErr) {
				return nil, fmt.Errorf("read VM %q state: %w", name, stateErr)
			}
			if pid == 0 {
				pid = vmState.PID
			}

			stateStr := "stopped"
			if p.isVMProcess(name, pid) {
				stateStr = "running"
			}

			results = append(results, VMStatus{
				Name:       name,
				State:      stateStr,
				PID:        pid,
				SSHPort:    vmState.SSHPort,
				SSHUser:    vmState.SSHUser,
				VNCPort:    vmState.VNCPort,
				Forwarding: vmState.Forwarding,
			})
		}
	}
	return results, nil
}

// Info dives deep into a VM's neural links, returning detailed state,
// networking information, and disk health (including backing file integrity).
func (p *QemuProvider) Info(name string) (VMDetail, error) {
	if err := validateIdentifier(name); err != nil {
		return VMDetail{}, err
	}
	state, err := p.loadState(name)
	if err != nil {
		if !os.IsNotExist(err) {
			return VMDetail{}, fmt.Errorf("read VM %q state: %w", name, err)
		}
		diskPath := filepath.Join(p.RootDir, "vms", name+".qcow2")
		if _, err := os.Stat(diskPath); err != nil {
			return VMDetail{}, fmt.Errorf("VM %q: %w", name, err)
		}
		backingPath, backingMissing := backingInfo(diskPath)
		return VMDetail{Name: name, State: "stopped", IP: "127.0.0.1", DiskPath: diskPath, StateMissing: true, BackingPath: backingPath, BackingMissing: backingMissing}, nil
	}

	// Check liveness for state string
	liveness := "stopped"
	pidData, _ := os.ReadFile(filepath.Join(p.RootDir, "run", name+".pid"))
	pid := 0
	fmt.Sscanf(string(pidData), "%d", &pid)
	if pid == 0 {
		pid = state.PID
	}
	if p.isVMProcess(name, pid) {
		liveness = "running"
	}

	diskPath := filepath.Join(p.RootDir, "vms", name+".qcow2")
	_, statErr := os.Stat(diskPath)
	backingPath, backingMissing := backingInfo(diskPath)

	detail := VMDetail{
		Name:           name,
		State:          liveness,
		PID:            pid,
		IP:             "127.0.0.1",
		SSHUser:        state.SSHUser,
		SSHPort:        state.SSHPort,
		VNCPort:        state.VNCPort,
		MemoryMB:       state.MemoryMB,
		VCPUs:          state.VCPUs,
		Gui:            state.Gui,
		Cmdline:        state.Cmdline,
		Forwarding:     state.Forwarding,
		RawQemuArgs:    state.RawQemuArgs,
		Accelerators:   state.Accelerators,
		DiskPath:       diskPath,
		DiskMissing:    statErr != nil,
		BackingPath:    backingPath,
		BackingMissing: backingMissing,
	}

	return detail, nil
}

// safeRemove removes a file but ignores "not found" errors.
func safeRemove(path string) error {
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// backingInfo returns backing filename and whether it's missing.
func backingInfo(diskPath string) (string, bool) {
	if diskPath == "" {
		return "", false
	}
	type info struct {
		Backing     string `json:"backing-filename"`
		FullBacking string `json:"full-backing-filename"`
	}
	qemuImg, err := sysutil.QemuImgBinary()
	if err != nil {
		return "", false
	}
	out, err := sysutil.NewCommand(qemuImg, "info", "-U", "--output=json", diskPath).Output()
	if err != nil {
		return "", false
	}
	var meta info
	if json.Unmarshal(out, &meta) != nil || meta.Backing == "" {
		return "", false
	}
	if meta.FullBacking != "" {
		meta.Backing = meta.FullBacking
	}
	if !filepath.IsAbs(meta.Backing) {
		meta.Backing = filepath.Join(filepath.Dir(diskPath), meta.Backing)
	}
	if _, err := os.Stat(meta.Backing); err != nil {
		return meta.Backing, true
	}
	return meta.Backing, false
}

// Stop terminates the owned QEMU process and confirms exit before releasing resources.
// We clean up QMP and PID artifacts to keep the run directory tidy.
func (p *QemuProvider) Stop(name string) error {
	if err := validateIdentifier(name); err != nil {
		return err
	}
	runDir := filepath.Join(p.RootDir, "run")
	pidFile := filepath.Join(runDir, name+".pid")
	pidData, _ := os.ReadFile(pidFile)
	pid := 0
	fmt.Sscanf(string(pidData), "%d", &pid)
	if pid == 0 {
		if state, err := p.loadState(name); err == nil {
			pid = state.PID
		}
	}

	if pid > 0 && processAlive(pid) {
		if err := p.stopVMProcess(name, pid); err != nil {
			return err
		}
	}

	os.Remove(pidFile)
	os.Remove(filepath.Join(runDir, name+".qmp"))

	if state, err := p.loadState(name); err == nil {
		_ = p.saveState(name, 0, state.SSHPort, state.VNCPort, state.Gui, state.SSHUser, state.Forwarding, state.Cmdline, state.MemoryMB, state.VCPUs, state.RawQemuArgs, state.Accelerators)
	}

	// RESTORE ACCELERATORS (Start of Double Fix)
	// We must release any VFIO devices back to the host
	if state, err := p.loadState(name); err == nil {
		for _, acc := range state.Accelerators {
			if !strings.Contains(acc, ":") || strings.Contains(acc, "virtual") {
				continue
			}
			// It's a PCI ID (e.g. 0000:01:00.0)
			if err := p.restorePassthrough(name, acc); err != nil {
				return fmt.Errorf("restore accelerator %s: %w", acc, err)
			}
		}
	}
	return nil
}

func (p *QemuProvider) Delete(name string) error {
	if err := validateIdentifier(name); err != nil {
		return err
	}
	if err := p.Stop(name); err != nil {
		return err
	}
	vmsDir := filepath.Join(p.RootDir, "vms")
	diskPath := filepath.Join(vmsDir, name+".qcow2")

	// We use safeRemove for all files to be idempotent
	_ = safeRemove(filepath.Join(p.RootDir, "run", name+".json"))
	_ = safeRemove(filepath.Join(vmsDir, name+"-seed.iso"))
	_ = safeRemove(filepath.Join(vmsDir, name+".kernel"))
	_ = safeRemove(filepath.Join(vmsDir, name+".initrd"))
	return safeRemove(diskPath)
}

// CreateTemplate archives a VM into "cold storage" (a compressed qcow2).
// This is how we preserve perfected environments for future hatchlings.
func (p *QemuProvider) CreateTemplate(vmName string, templateName string) (string, error) {
	if err := validateIdentifier(vmName); err != nil {
		return "", err
	}
	if err := validateIdentifier(templateName); err != nil {
		return "", err
	}
	// 1. Ensure VM is stopped
	if err := p.Stop(vmName); err != nil {
		return "", err
	}

	vmsDir := filepath.Join(p.RootDir, "vms")
	backupsDir := p.Config.BackupDir
	os.MkdirAll(backupsDir, 0755)

	srcDisk := filepath.Join(vmsDir, vmName+".qcow2")
	targetTemplate := filepath.Join(backupsDir, templateName+".compact.qcow2")

	if _, err := os.Stat(srcDisk); os.IsNotExist(err) {
		return "", fmt.Errorf("source disk not found: %s", srcDisk)
	}

	// qemu-img convert -O qcow2 -c <src> <dest>
	qemuImg, err := sysutil.QemuImgBinary()
	if err != nil {
		return "", err
	}
	cmd := sysutil.NewCommand(qemuImg, "convert", "-O", "qcow2", "-c", srcDisk, targetTemplate)
	if err := cmd.Run(); err != nil {
		return "", err
	}

	return targetTemplate, nil
}

func (p *QemuProvider) DeleteTemplate(name string, force bool) error {
	if err := validateIdentifier(name); err != nil {
		return err
	}
	templatePath := filepath.Join(p.Config.BackupDir, name+".compact.qcow2")

	// Safety Check: Is it in use?
	if !force {
		// We only check usage if the template actually exists
		if _, err := os.Stat(templatePath); err == nil {
			used, err := p.GetUsedBackingFiles()
			if err != nil {
				return fmt.Errorf("failed to check template usage: %v", err)
			}
			absTemplate, err := filepath.EvalSymlinks(templatePath)
			if err != nil {
				return err
			}
			absTemplate, err = filepath.Abs(absTemplate)
			if err != nil {
				return err
			}
			for _, u := range used {
				if u == absTemplate || u == templatePath {
					return fmt.Errorf("template '%s' is in use by one or more VMs (use --force to override)", name)
				}
			}
		}
	}

	return safeRemove(templatePath)
}

// CreateDisk prepares the execution surface (the qcow2 file).
// It supports both standalone "Full Copies" and space-saving "Linked Clones".
func (p *QemuProvider) CreateDisk(name, size, tpl string) error {
	if err := validateIdentifier(name); err != nil {
		return err
	}
	vmsDir := filepath.Join(p.RootDir, "vms")
	os.MkdirAll(vmsDir, 0755)
	target := filepath.Join(vmsDir, name+".qcow2")
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("disk already exists: %s", target)
	}

	// 1. Full Copy Mode (No Cache)
	// If LinkedClones is disabled and we have a template, create a standalone copy.
	// Safety check: if Config is nil, default to Linked Clones behavior (safer)
	useLinked := true
	if p.Config != nil {
		useLinked = p.Config.LinkedClones
	}

	if !useLinked && tpl != "" {
		return sysutil.ProvisionFile(target, func() error {
			// Convert (Full Copy)
			qemuImg, err := sysutil.QemuImgBinary()
			if err != nil {
				return err
			}
			if out, err := sysutil.NewCommand(qemuImg, "convert", "-O", "qcow2", tpl, target).CombinedOutput(); err != nil {
				return fmt.Errorf("create(convert) failed: %v (%s)", err, string(out))
			}
			// Resize to target size
			if out, err := sysutil.NewCommand(qemuImg, "resize", target, size).CombinedOutput(); err != nil {
				return fmt.Errorf("create(resize) failed: %v (%s)", err, string(out))
			}
			return nil
		})
	}

	// 2. Linked Clone Mode (Default)
	args := []string{"create", "-f", "qcow2"}
	if tpl != "" {
		// Autodetect template format
		format := "qcow2"
		qemuImg, err := sysutil.QemuImgBinary()
		if err != nil {
			return err
		}
		out, err := sysutil.NewCommand(qemuImg, "info", "--output=json", tpl).Output()
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

	qemuImg, err := sysutil.QemuImgBinary()
	if err != nil {
		return err
	}
	cmd := sysutil.NewCommand(qemuImg, args...)
	return sysutil.ProvisionFile(target, func() error {
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("qemu-img create failed: %v (%s)", err, string(out))
		}
		return nil
	})
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
			for _, fw := range state.Forwarding {
				if fw.HostPort > 0 {
					reserved[fw.HostPort] = true
				}
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
	Name         string        `json:"name"`
	PID          int           `json:"pid"`
	SSHPort      int           `json:"ssh_port"`
	VNCPort      int           `json:"vnc_port,omitempty"`
	Gui          bool          `json:"gui,omitempty"`
	SSHUser      string        `json:"ssh_user,omitempty"`
	Forwarding   []PortForward `json:"forwarding,omitempty"`
	Cmdline      string        `json:"cmdline,omitempty"`
	MemoryMB     int           `json:"memory_mb,omitempty"`
	VCPUs        int           `json:"vcpus,omitempty"`
	RawQemuArgs  []string      `json:"raw_qemu_args,omitempty"`
	Accelerators []string      `json:"accelerators,omitempty"`
}

func (p *QemuProvider) saveState(name string, pid int, sshPort int, vncPort int, gui bool, sshUser string, fw []PortForward, cmdline string, memoryMB, vcpus int, rawArgs []string, accelerators []string) error {
	state := VMState{
		Name:         name,
		PID:          pid,
		SSHPort:      sshPort,
		VNCPort:      vncPort,
		Gui:          gui,
		SSHUser:      sshUser,
		Forwarding:   fw,
		Cmdline:      cmdline,
		MemoryMB:     memoryMB,
		VCPUs:        vcpus,
		RawQemuArgs:  rawArgs,
		Accelerators: accelerators,
	}
	data, _ := json.MarshalIndent(state, "", "  ")
	path := filepath.Join(p.RootDir, "run", name+".json")
	return sysutil.WriteFile(path, data, 0600)
}

// UpdateConfig safely modifies the persistent VMState using a read-modify-write cycle.
func (p *QemuProvider) UpdateConfig(name string, updates VMConfigUpdates) error {
	if err := validateIdentifier(name); err != nil {
		return err
	}
	// 1. Load existing state
	state, err := p.loadState(name)
	if err != nil {
		return fmt.Errorf("failed to load state for VM '%s': %w", name, err)
	}

	// 2. Apply updates
	if updates.MemoryMB != nil {
		if *updates.MemoryMB < 128 {
			return fmt.Errorf("memory must be at least 128MB")
		}
		state.MemoryMB = *updates.MemoryMB
	}
	if updates.VCPUs != nil {
		if *updates.VCPUs < 1 {
			return fmt.Errorf("vcpus must be at least 1")
		}
		state.VCPUs = *updates.VCPUs
	}
	if updates.Gui != nil {
		state.Gui = *updates.Gui
	}
	if updates.Cmdline != nil {
		state.Cmdline = *updates.Cmdline
	}
	if updates.SSHPort != nil {
		if err := validatePort(*updates.SSHPort, false, "ssh port"); err != nil {
			return err
		}
		state.SSHPort = *updates.SSHPort
	}
	if updates.VNCPort != nil {
		if err := validatePort(*updates.VNCPort, true, "vnc port"); err != nil {
			return err
		}
		state.VNCPort = *updates.VNCPort
	}
	if updates.SSHUser != nil {
		state.SSHUser = *updates.SSHUser
	}
	if updates.Forwarding != nil {
		for _, pf := range *updates.Forwarding {
			if err := ValidatePortForward(pf); err != nil {
				return err
			}
		}
		state.Forwarding = *updates.Forwarding
	}
	if updates.RawQemuArgs != nil {
		state.RawQemuArgs = *updates.RawQemuArgs
	}
	if updates.Accelerators != nil {
		if err := hardware.ValidateAccelerators(*updates.Accelerators); err != nil {
			return err
		}
		state.Accelerators = *updates.Accelerators
	}

	// 3. Persist
	return p.saveState(state.Name, state.PID, state.SSHPort, state.VNCPort, state.Gui, state.SSHUser, state.Forwarding, state.Cmdline, state.MemoryMB, state.VCPUs, state.RawQemuArgs, state.Accelerators)
}

func (p *QemuProvider) loadState(name string) (VMState, error) {
	if err := validateIdentifier(name); err != nil {
		return VMState{}, err
	}
	var state VMState
	data, err := os.ReadFile(filepath.Join(p.RootDir, "run", name+".json"))
	if err != nil {
		return state, err
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return state, err
	}
	if state.Name != name {
		return state, fmt.Errorf("VM state identity %q does not match %q", state.Name, name)
	}
	return state, nil
}

// BuildNetDevArgs translates state to QEMU runtime arguments.
// Implements Section 5.4.C of advanced-port-forwarding.md.
func (p *QemuProvider) BuildNetDevArgs(sshPort int, fw []PortForward) string {
	fwd := fmt.Sprintf("tcp:127.0.0.1:%d-:22", sshPort)
	for _, f := range fw {
		proto := strings.ToLower(strings.TrimSpace(f.Protocol))
		if proto == "" {
			proto = "tcp"
		}
		// If HostPort is 0, it means it's not yet allocated or failed.
		// Usually we allocate before calling this.
		if f.HostPort > 0 {
			fwd += fmt.Sprintf(",hostfwd=%s:127.0.0.1:%d-:%d", proto, f.HostPort, f.GuestPort)
		}
	}
	return fmt.Sprintf("user,id=net0,hostfwd=%s", fwd)
}

func (p *QemuProvider) SSHArgs(name string) ([]string, error) {
	info, err := p.Info(name)
	if err != nil {
		return nil, err
	}
	if info.SSHPort == 0 || info.State != "running" {
		return nil, fmt.Errorf("VM '%s' is not running or has no network access", name)
	}
	// Inject options to skip fingerprint check for ephemeral VMs.
	opts := []string{"ssh", "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=" + sysutil.NullDevice()}
	if keyPath := localNidoSSHKeyPath(); keyPath != "" {
		opts = append(opts, "-i", keyPath)
	}
	opts = append(opts, "-p", fmt.Sprintf("%d", info.SSHPort), "-l", info.SSHUser, "--", info.IP)
	return opts, nil
}

func (p *QemuProvider) ListTemplates() ([]string, error) {
	files, err := os.ReadDir(p.Config.BackupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	templates := []string{}
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
	if p.Config == nil {
		return config.Config{}
	}
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
	qemu, err := sysutil.QemuSystemBinary()

	qemuHint, qimgHint := qemuInstallHints()
	if err == nil {
		qemuHint = ""
	}
	add("Binary: QEMU", err == nil, qemu+" "+qemuHint)

	qimg, err := sysutil.QemuImgBinary()
	if err == nil {
		qimgHint = ""
	}
	add("Binary: qemu-img", err == nil, qimg+" "+qimgHint)

	// 2.1 ISO Creation Tools (for cloud-init)
	isoTools := []string{"cloud-localds", "genisoimage", "mkisofs", "xorriso"}
	foundTool := ""
	for _, tool := range isoTools {
		if path, err := sysutil.FindExecutable(tool); err == nil {
			foundTool = path
			break
		}
	}

	isoHint := foundTool
	if foundTool == "" {
		isoHint = "built-in Go ISO writer"
	}
	add("Tools: ISO Creator", true, isoHint)

	acceleration := p.Hardware.Virtualization()
	add("Acceleration", acceleration.Accelerator != "", fmt.Sprintf("accelerator=%s cpu=%s", acceleration.Accelerator, acceleration.CPU))

	return reports
}

func (p *QemuProvider) GetUsedBackingFiles() ([]string, error) { return p.usedBackingFiles() }

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

		imageName := name
		version := ""

		// Smarter splitting: Try to match against known image names in catalog
		catPath := filepath.Join(imagesDir, image.CatalogCacheFile)
		if catalog, err := image.LoadCatalogFromFile(catPath); err == nil {
			found := false
			// Sort images by name length descending to match longest name first (e.g. ubuntu-24.04-apache-wordpress vs ubuntu)
			imgs := catalog.Images
			sort.Slice(imgs, func(i, j int) bool {
				return len(imgs[i].Name) > len(imgs[j].Name)
			})

			for _, img := range imgs {
				if name == img.Name {
					imageName = img.Name
					version = ""
					found = true
					break
				}
				prefix := img.Name + "-"
				if strings.HasPrefix(name, prefix) {
					imageName = img.Name
					version = strings.TrimPrefix(name, prefix)
					found = true
					break
				}
			}

			if !found {
				// Fallback to simple split if not in catalog
				parts := strings.Split(name, "-")
				imageName = parts[0]
				if len(parts) > 1 {
					version = strings.Join(parts[1:], "-")
				}
			}
		} else {
			// Fallback: simple split
			parts := strings.Split(name, "-")
			imageName = parts[0]
			if len(parts) > 1 {
				version = strings.Join(parts[1:], "-")
			}
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
func (p *QemuProvider) CachePrune(unusedOnly bool) (int, int64, error) {
	imagesDir := p.Config.ImageDir
	if imagesDir == "" {
		imagesDir = filepath.Join(p.RootDir, "images")
	}

	usedBacking, err := p.GetUsedBackingFiles()
	if err != nil {
		return 0, 0, err
	}
	usedMap := make(map[string]bool)
	for _, path := range usedBacking {
		if abs, err := filepath.Abs(path); err == nil {
			usedMap[abs] = true
		} else {
			usedMap[path] = true
		}
	}

	files, err := os.ReadDir(imagesDir)
	if err != nil {
		return 0, 0, err
	}

	count := 0
	reclaimed := int64(0)

	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".qcow2") {
			continue
		}
		fullPath := filepath.Join(imagesDir, f.Name())
		absPath, err := filepath.EvalSymlinks(fullPath)
		if err != nil {
			return 0, 0, err
		}
		absPath, err = filepath.Abs(absPath)
		if err != nil {
			return 0, 0, err
		}
		if unusedOnly && usedMap[absPath] {
			continue // Skip images in use
		}

		info, err := f.Info()
		if err == nil {
			reclaimed += info.Size()
		}

		if err := os.Remove(fullPath); err == nil {
			count++
		}
	}
	return count, reclaimed, nil
}

// CacheRemove removes a specific cached image.
func (p *QemuProvider) CacheRemove(name, version string) error {
	if err := validateIdentifier(name); err != nil {
		return err
	}
	if version != "" {
		if err := validateIdentifier(version); err != nil {
			return err
		}
	}
	imagesDir := p.Config.ImageDir
	if imagesDir == "" {
		imagesDir = filepath.Join(p.RootDir, "images")
	}

	var filename string
	if version == "" {
		filename = fmt.Sprintf("%s.qcow2", name)
	} else {
		filename = fmt.Sprintf("%s-%s.qcow2", name, version)
	}
	fullPath := filepath.Join(imagesDir, filename)

	// Check usage
	usedBacking, err := p.GetUsedBackingFiles()
	if err != nil {
		return err
	}
	canonical, err := filepath.EvalSymlinks(fullPath)
	if err != nil {
		return err
	}
	canonical, err = filepath.Abs(canonical)
	if err != nil {
		return err
	}
	{
		for _, path := range usedBacking {
			absPath, _ := filepath.Abs(path)
			absFullPath := canonical
			if absPath == absFullPath {
				return fmt.Errorf("cannot remove image %s: in use by a VM", filename)
			}
		}
	}

	return os.Remove(fullPath)
}

func (p *QemuProvider) PortForward(name string, pf PortForward) (PortForward, error) {
	if err := validateIdentifier(name); err != nil {
		return pf, err
	}
	state, err := p.loadState(name)
	if err != nil {
		return pf, err
	}
	if err := ValidatePortForward(pf); err != nil {
		return pf, err
	}

	resolved, err := p.resolveForwarding([]PortForward{pf}, state.SSHPort, state.VNCPort)
	if err != nil {
		return pf, err
	}
	pf = resolved[0]
	if pf.Protocol == "" {
		pf.Protocol = "tcp"
	}

	// Check if GuestPort already forwarded for this protocol
	for i, f := range state.Forwarding {
		if f.GuestPort == pf.GuestPort && f.Protocol == pf.Protocol {
			// Update existing rule
			state.Forwarding[i] = pf
			err := p.saveState(name, state.PID, state.SSHPort, state.VNCPort, state.Gui, state.SSHUser, state.Forwarding, state.Cmdline, state.MemoryMB, state.VCPUs, state.RawQemuArgs, state.Accelerators)
			return pf, err
		}
	}

	// Add new rule
	state.Forwarding = append(state.Forwarding, pf)
	err = p.saveState(name, state.PID, state.SSHPort, state.VNCPort, state.Gui, state.SSHUser, state.Forwarding, state.Cmdline, state.MemoryMB, state.VCPUs, state.RawQemuArgs, state.Accelerators)
	return pf, err
}

func (p *QemuProvider) PortUnforward(name string, guestPort int, protocol string) error {
	if err := validateIdentifier(name); err != nil {
		return err
	}
	if err := validatePort(guestPort, false, "guest port"); err != nil {
		return err
	}
	protocol = strings.ToLower(strings.TrimSpace(protocol))
	if protocol != "" && protocol != "tcp" && protocol != "udp" {
		return fmt.Errorf("protocol must be tcp or udp")
	}
	state, err := p.loadState(name)
	if err != nil {
		return err
	}

	for i, f := range state.Forwarding {
		stateProtocol := strings.ToLower(strings.TrimSpace(f.Protocol))
		if stateProtocol == "" {
			stateProtocol = "tcp"
		}
		if f.GuestPort == guestPort && (stateProtocol == protocol || protocol == "") {
			state.Forwarding = append(state.Forwarding[:i], state.Forwarding[i+1:]...)
			return p.saveState(name, state.PID, state.SSHPort, state.VNCPort, state.Gui, state.SSHUser, state.Forwarding, state.Cmdline, state.MemoryMB, state.VCPUs, state.RawQemuArgs, state.Accelerators)
		}
	}

	return fmt.Errorf("port mapping not found: %d/%s", guestPort, protocol)
}

func (p *QemuProvider) PortList(name string) ([]PortForward, error) {
	state, err := p.loadState(name)
	if err != nil {
		return nil, err
	}
	return state.Forwarding, nil
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

// preparePassthrough handles the "Zero-Config" magic.
// It checks IOMMU, unbinds the device from host, and binds to vfio-pci.
// Requires root privileges (uses sudo if needed).

// preparePassthrough prepares a PCI device for VFIO passthrough.
// It delegates to the platform-specific sysutil implementation.
func (p *QemuProvider) preparePassthrough(owner, id string) error {
	// 0. Skip virtual accelerators
	if id == hardware.VirtualGPUAcceleratorID {
		return nil
	}

	// The HAL owns discovery, binding and restoration.
	if err := p.Hardware.PreparePassthrough(context.Background(), id, filepath.Join(p.RootDir, "run"), owner); err != nil {
		return err
	}

	return nil
}

// restorePassthrough reverses the preparePassthrough operation.
// It delegates to the platform-specific sysutil implementation.
func (p *QemuProvider) restorePassthrough(owner, pciID string) error {
	if pciID == hardware.VirtualGPUAcceleratorID {
		return nil
	}
	// Logic moved to sysutil (Platform Abstraction Layer)
	return p.Hardware.RestorePassthrough(context.Background(), pciID, filepath.Join(p.RootDir, "run"), owner)
}

// resolveAutoAccelerators handles the "auto" keyword in the accelerator list.
// It performs hardware discovery to find the best candidate (Primary vs Safe).
func (p *QemuProvider) resolveAutoAccelerators(input []string) ([]string, error) {
	autoRequested := false
	finalList := make([]string, 0, len(input))

	for _, acc := range input {
		if acc == hardware.AutoAcceleratorID {
			autoRequested = true
		} else {
			finalList = append(finalList, acc)
		}
	}

	if !autoRequested {
		return input, nil
	}

	candidates, err := p.ListAccelerators()
	if err != nil {
		if errors.Is(err, hardware.ErrUnsupported) {
			return append(finalList, hardware.VirtualGPUAcceleratorID), nil
		}
		return finalList, err
	}

	var bestCandidate *hardware.Accelerator
	// Strategy matches CLI logic:
	// 1. Look for Display Controllers (03xx)
	// 2. Pick the first one (usually 00:02.0 or 01:00.0)
	// 3. Prefer SAFE if available, but take Unsafe/Primary if it's the only option.

	for i := range candidates {
		c := candidates[i]
		if c.Class == hardware.DescDisplay {
			// Found a GPU!
			if c.IsSafe {
				bestCandidate = &c
				break // Found a safe GPU, winner!
			}
			// STRICT SAFETY: We DO NOT fallback to unsafe GPUs automatically anymore.
			// Using the primary GPU crashes the session (Confirmed by test 2026-02-06).
		}
	}

	if bestCandidate != nil {
		finalList = append(finalList, bestCandidate.ID)
	} else {
		finalList = append(finalList, hardware.VirtualGPUAcceleratorID)
	}

	return finalList, nil
}
