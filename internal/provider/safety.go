package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Josepavese/nido/v5/internal/config"
	nidonet "github.com/Josepavese/nido/v5/internal/net"
	"github.com/Josepavese/nido/v5/internal/pkg/sysutil"
)

const vmStopTimeout = 5 * time.Second

// Like lifecycle/config mutations, reload is confined to the provider owner
// thread. Existing consumers share the canonical config pointer.
func (p *QemuProvider) ReloadConfig(path string) error {
	cfg, err := config.LoadConfig(path)
	if err != nil {
		return err
	}
	p.configMu.Lock()
	p.Config = cfg
	p.configMu.Unlock()
	return nil
}

func (p *QemuProvider) resolveForwarding(input []PortForward, sshPort, vncPort int) ([]PortForward, error) {
	cfg := p.configSnapshot()
	result := append([]PortForward(nil), input...)
	reserved := p.getReservedPorts()
	reserved[sshPort], reserved[vncPort] = true, true
	for _, pf := range result {
		if err := ValidatePortForward(pf); err != nil {
			return nil, err
		}
		if pf.HostPort > 0 {
			reserved[pf.HostPort] = true
		}
	}
	start, end := config.DefaultPortRangeStart, config.DefaultPortRangeEnd
	if cfg != nil {
		if cfg.PortRangeStart != 0 {
			start = cfg.PortRangeStart
		}
		if cfg.PortRangeEnd != 0 {
			end = cfg.PortRangeEnd
		}
	}
	for i := range result {
		if result[i].HostPort == 0 {
			port, err := nidonet.FindAvailablePort(start, end, reserved)
			if err != nil {
				return nil, fmt.Errorf("allocate forwarding port: %w", err)
			}
			result[i].HostPort = port
			reserved[port] = true
		}
	}
	return result, nil
}

// usedBackingFiles resolves the full graph before any destructive action.
// An unreadable/corrupt chain is an error, never proof that an image is unused.
func (p *QemuProvider) usedBackingFiles() ([]string, error) {
	cfg := p.configSnapshot()
	used := map[string]bool{}
	dirs := []string{filepath.Join(p.RootDir, "vms")}
	if cfg != nil && cfg.BackupDir != "" {
		dirs = append(dirs, cfg.BackupDir)
	}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".qcow2") {
				continue
			}
			root, err := filepath.Abs(filepath.Join(dir, entry.Name()))
			if err != nil {
				return nil, err
			}
			binary, err := sysutil.QemuImgBinary()
			if err != nil {
				return nil, err
			}
			out, err := sysutil.NewCommand(binary, "info", "-U", "--backing-chain", "--output=json", root).CombinedOutput()
			if err != nil {
				return nil, fmt.Errorf("inspect backing chain %s: %w (%s)", root, err, out)
			}
			var chain []struct {
				Filename    string `json:"filename"`
				Backing     string `json:"backing-filename"`
				FullBacking string `json:"full-backing-filename"`
			}
			if err := json.Unmarshal(out, &chain); err != nil {
				return nil, fmt.Errorf("inspect backing chain %s: %w", root, err)
			}
			if len(chain) == 0 {
				return nil, fmt.Errorf("empty backing chain for %s", root)
			}
			current := root
			for _, node := range chain {
				backing := node.FullBacking
				if backing == "" {
					backing = node.Backing
				}
				if backing == "" {
					continue
				}
				if !filepath.IsAbs(backing) {
					backing = filepath.Join(filepath.Dir(current), backing)
				}
				backing = filepath.Clean(backing)
				// Normalize symlink aliases so the same image cannot evade usage checks.
				resolved, err := filepath.EvalSymlinks(backing)
				if err != nil {
					return nil, fmt.Errorf("resolve backing %s: %w", backing, err)
				}
				used[resolved] = true
				used[backing] = true
				current = backing
			}
		}
	}
	result := make([]string, 0, len(used))
	for path := range used {
		result = append(result, path)
	}
	sort.Strings(result)
	return result, nil
}

type vmProcessIdentity struct{ name, disk, pidfile string }

func (p *QemuProvider) processIdentity(name string) vmProcessIdentity {
	return vmProcessIdentity{name, filepath.Join(p.RootDir, "vms", name+".qcow2"), filepath.Join(p.RootDir, "run", name+".pid")}
}

func matchesVMCommand(args []string, identity vmProcessIdentity) bool {
	named, disk := false, false
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "-name" && args[i+1] == identity.name {
			named = true
		}
		if args[i] == "-drive" && strings.HasPrefix(args[i+1], "file="+identity.disk+",") {
			disk = true
		}
	}
	return named && disk
}

func qemuExecutable(path string) bool {
	name := strings.TrimSuffix(filepath.Base(path), " (deleted)")
	return strings.HasPrefix(name, "qemu-system-") || name == "qemu-system" || name == "qemu-kvm"
}

func (p *QemuProvider) isVMProcess(name string, pid int) bool {
	return processAlive(pid) && inspectVMProcess(pid, p.processIdentity(name)) == nil
}

func (p *QemuProvider) stopVMProcess(name string, pid int) error {
	return stopManagedProcess(pid, p.processIdentity(name))
}
