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
)

// QemuProvider implements VMProvider using raw QEMU.
type QemuProvider struct {
	RootDir string
	Config  *config.Config
}

func NewQemuProvider(rootDir string, cfg *config.Config) *QemuProvider {
	return &QemuProvider{
		RootDir: rootDir,
		Config:  cfg,
	}
}

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

	// 3. Generate Cloud-Init Seed ISO
	// This ensures that if the image supports cloud-init (like official cloud images),
	// it will automatically configure the user and SSH keys.
	seedPath := filepath.Join(p.RootDir, "vms", name+"-seed.iso")
	sshKey := GetLocalSSHKey()

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
		User:           p.Config.SSHUser,
		SSHKey:         sshKey,
		CustomUserData: customUserData,
	}

	// Create seed ISO (warn on failure but don't block spawn)
	if err := ci.GenerateISO(seedPath); err != nil {
		fmt.Printf("⚠️  Warning: Failed to generate cloud-init seed: %v\n", err)
	}

	// 4. Save Initial State (with GUI preference)
	p.saveState(name, 0, 0, 0, opts.Gui)

	// 5. Start
	return p.Start(name, opts)
}

func (p *QemuProvider) Start(name string, opts VMOptions) error {
	// 0. Check if already running
	if status, err := p.Info(name); err == nil && status.State == "running" {
		return nil // Already running
	}

	// 1. Prepare Paths
	runDir := filepath.Join(p.RootDir, "run")
	vmsDir := filepath.Join(p.RootDir, "vms")
	os.MkdirAll(runDir, 0755)
	os.MkdirAll(vmsDir, 0755)

	diskPath := filepath.Join(vmsDir, name+".qcow2")
	if _, err := os.Stat(diskPath); os.IsNotExist(err) {
		return fmt.Errorf("disk image not found: %s", diskPath)
	}

	// 2. Port Management
	state, _ := p.loadState(name)
	updated := false
	if state.SSHPort == 0 {
		state.SSHPort = p.findAvailablePort(50022)
		updated = true
	}
	if state.Gui && state.VNCPort == 0 {
		state.VNCPort = p.findAvailablePort(59000) // VNC range usually starts at 5900, but we use high ports
		updated = true
	}

	if updated {
		p.saveState(name, 0, state.SSHPort, state.VNCPort, state.Gui)
	}

	// 3. Build Arguments (cross-platform)
	args := p.buildQemuArgs(name, diskPath, state.SSHPort, state.VNCPort, runDir)

	cmd := exec.Command("qemu-system-x86_64", args...)
	fmt.Fprintf(os.Stderr, "⚡ Debug: Running QEMU: qemu-system-x86_64 %s\n", strings.Join(args, " "))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ QEMU Error Output: %s\n", stderr.String())
		return fmt.Errorf("QEMU failed to start: %w (stderr: %s)", err, stderr.String())
	}

	// 4. Skip Bootloader (Background)
	// We send "Enter" key via QMP to skip any guest countdowns (Alpine, GRUB, etc.)
	go p.skipBootloader(name)

	return nil
}

// buildQemuArgs constructs QEMU arguments based on the host OS.
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
	)

	// Attach Cloud-Init Seed if exists
	seedPath := filepath.Join(p.RootDir, "vms", name+"-seed.iso")
	if _, err := os.Stat(seedPath); err == nil {
		args = append(args, "-cdrom", seedPath)
	}

	// QMP socket (platform-specific path handling)
	if runtime.GOOS == "windows" {
		// Windows uses named pipes for QMP
		args = append(args, "-qmp", fmt.Sprintf("tcp:127.0.0.1:0,server,nowait"))
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

func (p *QemuProvider) List() ([]VMStatus, error) {
	runDir := filepath.Join(p.RootDir, "run")
	files, err := os.ReadDir(runDir)
	if err != nil {
		return nil, err
	}

	var results []VMStatus
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".pid" {
			name := f.Name()[0 : len(f.Name())-4]
			pidData, _ := os.ReadFile(filepath.Join(runDir, f.Name()))
			pid := 0
			fmt.Sscanf(string(pidData), "%d", &pid)

			stateStr := "stopped"
			if pid > 0 {
				process, err := os.FindProcess(pid)
				if err == nil && process.Signal(syscall.Signal(0)) == nil {
					stateStr = "running"
				}
			}

			// Load port from JSON state
			vmState, _ := p.loadState(name)

			results = append(results, VMStatus{
				Name:    name,
				State:   stateStr,
				PID:     pid,
				SSHPort: vmState.SSHPort,
			})
		}
	}
	return results, nil
}

func (p *QemuProvider) Info(name string) (VMDetail, error) {
	state, err := p.loadState(name)
	if err != nil {
		return VMDetail{}, err
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

	return VMDetail{
		Name:    name,
		State:   liveness,
		IP:      "127.0.0.1",
		SSHUser: p.Config.SSHUser,
		SSHPort: state.SSHPort,
		VNCPort: state.VNCPort,
	}, nil
}

func (p *QemuProvider) Stop(name string, graceful bool) error {
	runDir := filepath.Join(p.RootDir, "run")
	pidFile := filepath.Join(runDir, name+".pid")
	pidData, _ := os.ReadFile(pidFile)
	pid := 0
	fmt.Sscanf(string(pidData), "%d", &pid)

	if pid > 0 {
		process, _ := os.FindProcess(pid)
		process.Signal(os.Interrupt)
		os.Remove(pidFile)
		os.Remove(filepath.Join(runDir, name+".qmp"))
	}
	return nil
}

func (p *QemuProvider) Delete(name string) error {
	p.Stop(name, false)
	vmsDir := filepath.Join(p.RootDir, "vms")
	diskPath := filepath.Join(vmsDir, name+".qcow2")
	os.Remove(filepath.Join(p.RootDir, "run", name+".json"))
	os.Remove(filepath.Join(vmsDir, name+"-seed.iso"))
	return os.Remove(diskPath)
}

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

func (p *QemuProvider) CreateDisk(name, size, tpl string) error {
	vmsDir := filepath.Join(p.RootDir, "vms")
	os.MkdirAll(vmsDir, 0755)
	target := filepath.Join(vmsDir, name+".qcow2")

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
	return cmd.Run()
}

// Helpers

func (p *QemuProvider) findAvailablePort(start int) int {
	for port := start; port < start+100; port++ {
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
}

func (p *QemuProvider) saveState(name string, pid int, sshPort int, vncPort int, gui bool) error {
	state := VMState{Name: name, PID: pid, SSHPort: sshPort, VNCPort: vncPort, Gui: gui}
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
