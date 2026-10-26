package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Josepavese/nido/v4/internal/image"
	"github.com/Josepavese/nido/v4/internal/machine"
	"github.com/Josepavese/nido/v4/internal/pkg/sysutil"
)

// MigrateArchitecture performs the explicit, resumable conversion from Nido's
// AMD64-only model. It never moves or opens disk payloads, rewrites backing
// chains, changes identities or relabels an already declared architecture.
// Stop other Nido clients first. Re-running after interruption is safe.
func (p *QemuProvider) MigrateArchitecture() (int, error) {
	return p.migrateArchitecture(true)
}

func (p *QemuProvider) migrateArchitecture(adoptUnidentified bool) (int, error) {
	cfg := p.configSnapshot()
	roots := []string{filepath.Join(p.RootDir, "vms"), filepath.Join(p.RootDir, "images"), filepath.Join(p.RootDir, "backups")}
	if cfg != nil {
		if cfg.ImageDir != "" {
			roots[1] = cfg.ImageDir
		}
		if cfg.BackupDir != "" {
			roots[2] = cfg.BackupDir
		}
	}
	type change struct {
		path string
		data []byte
		mode os.FileMode
	}
	var changes []change
	legacyState := false
	known := map[string]string{}
	files, err := filepath.Glob(filepath.Join(p.RootDir, "run", "*.json"))
	if err != nil {
		return 0, err
	}
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			return 0, err
		}
		var state map[string]json.RawMessage
		if err := json.Unmarshal(data, &state); err != nil {
			return 0, fmt.Errorf("%s: %w", path, err)
		}
		name := strings.TrimSuffix(filepath.Base(path), ".json")
		var identity string
		if err := json.Unmarshal(state["name"], &identity); err != nil || identity != name {
			return 0, fmt.Errorf("invalid VM identity in %s", path)
		}
		disk := filepath.Join(roots[0], name+".qcow2")
		known[disk] = machine.AMD64
		if raw, ok := state["arch"]; ok {
			var arch string
			if err := json.Unmarshal(raw, &arch); err != nil {
				return 0, err
			}
			if _, err := machine.Normalize(arch); err != nil {
				return 0, err
			}
			known[disk] = arch
			continue
		}
		var pid int
		_ = json.Unmarshal(state["pid"], &pid)
		if p.isVMProcess(name, pid) {
			return 0, fmt.Errorf("stop VM %s before architecture migration", name)
		}
		arch, err := image.ReadArchitecture(disk)
		if os.IsNotExist(err) {
			legacyState = true
			arch = machine.AMD64
		} else if err != nil {
			return 0, err
		}
		known[disk] = arch
		state["arch"], _ = json.Marshal(arch)
		next, err := json.MarshalIndent(state, "", "  ")
		if err != nil {
			return 0, err
		}
		info, err := os.Stat(path)
		if err != nil {
			return 0, err
		}
		changes = append(changes, change{path, next, info.Mode().Perm()})
	}
	// Validate all existing declarations before publishing any conversion.
	seen := map[string]bool{}
	type artifact struct{ path, arch string }
	var disks []artifact
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return 0, err
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".qcow2") {
				continue
			}
			path := filepath.Join(root, entry.Name())
			if seen[path] {
				continue
			}
			seen[path] = true
			if _, err := image.ReadArchitecture(path); os.IsNotExist(err) {
				arch := known[path]
				// A new/current installation can contain raw foreign imports.
				// Only explicit adoption or an actual legacy fleet establishes
				// the old AMD64-only contract for otherwise unidentified files.
				if arch == "" && !legacyState && !adoptUnidentified {
					continue
				}
				if arch == "" {
					arch = machine.AMD64
					for _, candidate := range machine.Architectures() {
						if strings.HasSuffix(strings.TrimSuffix(path, ".qcow2"), "-"+candidate) {
							arch = candidate
						}
					}
				}
				disks = append(disks, artifact{path, arch})
			} else if err != nil {
				return 0, fmt.Errorf("%s: %w", path, err)
			}
		}
	}
	count := 0
	// Publish additive artifact metadata first. A legacy VM record remains
	// discoverable until all artifacts are ready, so automatic retry needs no
	// second journal or heuristic after an interrupted state conversion.
	for _, path := range disks {
		if err := image.WriteArchitecture(path.path, path.arch); err != nil {
			return count, err
		}
		count++
	}
	for _, c := range changes {
		// Preserve the original small state, including credentials and unknown keys.
		backup := c.path + ".pre-architecture"
		if info, err := os.Stat(backup); os.IsNotExist(err) {
			data, err := os.ReadFile(c.path)
			if err != nil {
				return count, err
			}
			if err := sysutil.WriteFile(backup, data, c.mode); err != nil {
				return count, err
			}
		} else if err != nil {
			return count, err
		} else if !info.Mode().IsRegular() {
			return count, fmt.Errorf("original metadata backup is not a regular file: %s", backup)
		}
		if err := sysutil.WriteFile(c.path, c.data, c.mode); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}
