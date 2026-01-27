// Package services provides command functions for the Nido TUI.
// These functions return tea.Cmd that can be dispatched by the model.
// Extracting commands here reduces model.go complexity and improves testability.
package ops

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Josepavese/nido/internal/build"
	"github.com/Josepavese/nido/internal/config"
	"github.com/Josepavese/nido/internal/image"
	"github.com/Josepavese/nido/internal/provider"
	view "github.com/Josepavese/nido/internal/tui/kit/view"
	tea "github.com/charmbracelet/bubbletea"
)

// --- Message Types ---

// VMListMsg contains the refreshed list of VMs.
type VMListMsg struct {
	Items []VMItem
	Err   error
}

// VMItem represents a VM in the fleet list.
type VMItem struct {
	Name    string
	State   string
	PID     int
	SSHPort int
	VNCPort int
	SSHUser string
}

// VMDetailMsg contains detailed VM information.
type VMDetailMsg struct {
	Name   string
	Detail provider.VMDetail
	Err    error
}

// TemplateListMsg contains the list of existing templates.
type TemplateListMsg struct {
	Templates []string
	Err       error
}

// OpResultMsg is the result of a VM operation.
type OpResultMsg struct {
	Op   string
	Err  error
	Path string // Optional: for templates
	Data any    // Generic data (e.g., stats)
}

// UpdateCheckMsg contains version check results.
type UpdateCheckMsg struct {
	Current string
	Latest  string
	Manual  bool
	Err     error
}

// ConfigSavedMsg confirms a config value was saved.
type ConfigSavedMsg struct {
	Key   string
	Value string
}

// VMDetailRequestMsg requests details for a VM.
type VMDetailRequestMsg struct {
	Name string
}

// VM Operation Constants
const (
	OpStart  = "start"
	OpStop   = "stop"
	OpDelete = "delete"
)

// RequestOpMsg requests a VM operation.
type RequestOpMsg struct {
	Op   string
	Name string
}

// --- VM Commands ---

// RefreshFleet fetches the current VM list from the provider.
func RefreshFleet(prov provider.VMProvider) tea.Cmd {
	return func() tea.Msg {
		vms, err := prov.List()
		if err != nil {
			return VMListMsg{Err: err}
		}

		// Sort VMs alphabetically by Name
		sort.Slice(vms, func(i, j int) bool {
			return strings.ToLower(vms[i].Name) < strings.ToLower(vms[j].Name)
		})

		items := make([]VMItem, 0, len(vms))
		for _, v := range vms {
			items = append(items, VMItem{
				Name:    v.Name,
				State:   v.State,
				PID:     v.PID,
				SSHPort: v.SSHPort,
				VNCPort: v.VNCPort,
				SSHUser: v.SSHUser,
			})
		}
		return VMListMsg{Items: items}
	}
}

// FetchVMInfo retrieves detailed information about a VM.
func FetchVMInfo(prov provider.VMProvider, name string) tea.Cmd {
	return func() tea.Msg {
		detail, err := prov.Info(name)
		return VMDetailMsg{Name: name, Detail: detail, Err: err}
	}
}

// SpawnVM creates a new VM options. It automatically pulls the image if missing.
func SpawnVM(prov provider.VMProvider, name, source, userData string, gui bool, memoryMB, vcpus int, ports []provider.PortForward) tea.Cmd {
	opName := "spawn"

	return func() tea.Msg {
		// 1. Check if source is a local template (file or name)
		// Logic similar to QemuProvider.Spawn but used here to decide whether to pull.

		// If it's absolute path, assume it exists or let provider fail
		if filepath.IsAbs(source) || strings.Contains(source, "/") {
			// Just spawn, provider handles file not found or uses it
			opts := provider.VMOptions{
				DiskPath:     source,
				UserDataPath: userData,
				Gui:          gui,
				MemoryMB:     memoryMB,
				VCPUs:        vcpus,
			}
			err := prov.Spawn(name, opts)
			return OpResultMsg{Op: opName, Err: err}
		}

		// Check if it's a known template
		// Accessing provider internal config is hard here without import cycle or breaking abstraction?
		// We can list templates.
		templates, _ := prov.ListTemplates() // Ignore error for now, treat as empty
		isTemplate := false
		for _, t := range templates {
			if t == source {
				isTemplate = true
				break
			}
		}

		if isTemplate {
			opts := provider.VMOptions{
				DiskPath:     source,
				UserDataPath: userData,
				Gui:          gui,
				MemoryMB:     memoryMB,
				VCPUs:        vcpus,
				Forwarding:   ports,
			}
			err := prov.Spawn(name, opts)
			return OpResultMsg{Op: opName, Err: err}
		}

		// 2. Not a template? Check Catalog and Pull if needed
		// This uses the channel-based progress logic
		ch := make(chan ProgressMsg, 10)

		go func() {
			defer close(ch)

			// Resolve Image from Catalog
			cfg := prov.GetConfig()
			imgDir := cfg.ImageDir
			if imgDir == "" {
				home, _ := os.UserHomeDir()
				imgDir = filepath.Join(home, ".nido", "images")
			}

			catalog, err := image.LoadCatalog(imgDir, image.DefaultCacheTTL)
			if err != nil {
				// Fallback to spawn if catalog fails, maybe it's a special template not listed?
				// Or return error. Safe to return error.
				ch <- ProgressMsg{Result: &OpResultMsg{Op: opName, Err: fmt.Errorf("catalog load failed: %w", err)}}
				return
			}

			pName, pVer := source, ""
			if strings.Contains(source, ":") {
				parts := strings.Split(source, ":")
				pName, pVer = parts[0], parts[1]
			}

			img, ver, err := catalog.FindImage(pName, pVer)
			if err != nil {
				// Not found in catalog? Maybe the provider can handle it (e.g. unknown magic).
				// Try direct spawn.
				opts := provider.VMOptions{
					DiskPath:     source,
					UserDataPath: userData,
					Gui:          gui,
					Forwarding:   ports,
				}
				err := prov.Spawn(name, opts)
				ch <- ProgressMsg{Result: &OpResultMsg{Op: opName, Err: err}}
				return
			}

			// Image found! Check if we have it.
			destPath := filepath.Join(imgDir, fmt.Sprintf("%s-%s.qcow2", img.Name, ver.Version))
			if _, err := os.Stat(destPath); os.IsNotExist(err) {
				// NEED TO PULL
				ch <- ProgressMsg{
					OpName: opName,
					Status: view.StatusMsg{
						Loading:   true,
						Operation: fmt.Sprintf("Pulling %s", source),
						Progress:  0.0,
					},
				}

				downloader := image.Downloader{
					Quiet: true,
					OnProgress: func(current, total int64) {
						ratio := 0.0
						if total > 0 {
							ratio = float64(current) / float64(total)
						}
						// Limit updates?
						ch <- ProgressMsg{
							OpName: opName,
							Status: view.StatusMsg{
								Loading:   true,
								Operation: fmt.Sprintf("Pulling %s", source),
								Progress:  ratio,
							},
						}
					},
				}

				downloadPath := destPath
				isTarXz := strings.HasSuffix(ver.URL, ".tar.xz")
				isZst := strings.Contains(ver.URL, ".zst") || strings.Contains(ver.URL, ".zstandard")
				if len(ver.PartURLs) > 0 {
					isZst = strings.Contains(ver.PartURLs[0], ".zst") || strings.Contains(ver.PartURLs[0], ".zstandard")
				}

				if isTarXz {
					downloadPath = destPath + ".tar.xz"
				} else if isZst {
					downloadPath = destPath + ".zst"
				}

				if len(ver.PartURLs) > 0 {
					err = downloader.DownloadMultiPart(ver.PartURLs, downloadPath, ver.SizeBytes)
				} else {
					err = downloader.Download(ver.URL, downloadPath, ver.SizeBytes)
				}

				if err != nil {
					ch <- ProgressMsg{Result: &OpResultMsg{Op: opName, Err: fmt.Errorf("download failed: %w", err)}}
					return
				}

				// Verify & Decompress
				ch <- ProgressMsg{
					OpName: opName,
					Status: view.StatusMsg{
						Loading:   true,
						Operation: fmt.Sprintf("Verifying %s", pName),
						Progress:  1.0,
					},
				}

				if ver.Checksum != "" {
					if err := image.VerifyChecksum(downloadPath, ver.Checksum, ver.ChecksumType); err != nil {
						os.Remove(downloadPath)
						ch <- ProgressMsg{Result: &OpResultMsg{Op: opName, Err: fmt.Errorf("verification failed: %w", err)}}
						return
					}
				}

				if isTarXz || isZst {
					ch <- ProgressMsg{
						OpName: opName,
						Status: view.StatusMsg{
							Loading:   true,
							Operation: fmt.Sprintf("Decompressing %s", pName),
							Progress:  1.0,
						},
					}
					if err := downloader.Decompress(downloadPath, destPath); err != nil {
						os.Remove(downloadPath)
						ch <- ProgressMsg{Result: &OpResultMsg{Op: opName, Err: fmt.Errorf("decompression failed: %w", err)}}
						return
					}
					os.Remove(downloadPath)
				}
			}

			// 3. Spawning
			ch <- ProgressMsg{
				OpName: opName,
				Status: view.StatusMsg{
					Loading:   true,
					Operation: fmt.Sprintf("Hatching %s", name),
					Progress:  1.0,
				},
			}

			// We pass the RAW source string, trusting that QemuProvider (which we updated)
			// will resolve it to the now-existing file.
			opts := provider.VMOptions{
				DiskPath:     source,
				UserDataPath: userData,
				Gui:          gui,
				SSHUser:      img.SSHUser, // Use user from catalog if available
				MemoryMB:     memoryMB,
				VCPUs:        vcpus,
				Forwarding:   ports,
			}
			err = prov.Spawn(name, opts)
			ch <- ProgressMsg{Result: &OpResultMsg{Op: opName, Err: err}}
		}()

		return waitForProgress(ch)()
	}
}

// StartVM starts a stopped VM.
func StartVM(prov provider.VMProvider, name string) tea.Cmd {
	return func() tea.Msg {
		err := prov.Start(name, provider.VMOptions{Gui: true})
		return OpResultMsg{Op: "start", Err: err}
	}
}

// StopVM stops a running VM.
func StopVM(prov provider.VMProvider, name string) tea.Cmd {
	return func() tea.Msg {
		err := prov.Stop(name, true)
		return OpResultMsg{Op: "stop", Err: err}
	}
}

// DeleteVM removes a VM.
func DeleteVM(prov provider.VMProvider, name string) tea.Cmd {
	return func() tea.Msg {
		err := prov.Delete(name)
		return OpResultMsg{Op: "delete", Err: err}
	}
}

// CreateTemplate creates a new template from a VM.
func CreateTemplate(prov provider.VMProvider, vmName, templateName string) tea.Cmd {
	return func() tea.Msg {
		path, err := prov.CreateTemplate(vmName, templateName)
		return OpResultMsg{Op: "create-template", Err: err, Path: path}
	}
}

// DeleteTemplate removes a template.
func DeleteTemplate(prov provider.VMProvider, name string, force bool) tea.Cmd {
	return func() tea.Msg {
		err := prov.DeleteTemplate(name, force)
		return OpResultMsg{Op: "delete-template", Err: err}
	}
}

// CheckTemplateUsage checks if a template is used by any VMs.
func CheckTemplateUsage(prov provider.VMProvider, name string) tea.Cmd {
	return func() tea.Msg {
		if prov == nil {
			return TemplateUsageMsg{Err: fmt.Errorf("provider is nil")}
		}
		used, err := prov.GetUsedBackingFiles()
		if err != nil {
			return TemplateUsageMsg{Name: name, Err: err}
		}

		// Check if our template is in the used list.
		// We need to resolve paths similar to QemuProvider implementation
		// But since we can't easily resolve the template path here without config,
		// we might rely on simple string matching if possible, or we need to access config.
		// BETTER: The Provider should expose `IsTemplateInUse(name)`.
		// BUT: provider method `GetUsedBackingFiles` returns paths.
		// Let's iterate and check for suffix for now, as a heuristic.
		// The template file is usually `name + ".compact.qcow2"`.

		// Actually, QemuProvider logic was:
		// templatePath := filepath.Join(p.Config.BackupDir, name+".compact.qcow2")

		// Replicating that logic here is brittle.
		// Ideally `prov.InfoTemplate(name)` or similar would return status.
		// OR we trust the simple check: does any backing file end with `name + ".compact.qcow2"`?
		suffix := fmt.Sprintf("/%s.compact.qcow2", name)

		var usedBy []string
		for _, u := range used {
			if strings.HasSuffix(u, suffix) {
				usedBy = append(usedBy, "unknown-vm") // We don't know WHICH VM uses it from this list yet, only that it is used.
				// Unless GetUsedBackingFiles returns map? It returns slice of strings.
			}
		}

		return TemplateUsageMsg{Name: name, InUse: len(usedBy) > 0, UsedBy: usedBy}
	}
}

// FetchTemplatesList retrieves the list of existing templates.
func FetchTemplatesList(prov provider.VMProvider) tea.Cmd {
	return func() tea.Msg {
		if prov == nil {
			return TemplateListMsg{Err: fmt.Errorf("provider is nil")}
		}
		templates, err := prov.ListTemplates()
		if err != nil {
			return TemplateListMsg{Err: err}
		}
		sort.Strings(templates)
		return TemplateListMsg{Templates: templates}
	}
}

// --- Config Commands ---

// CheckUpdate checks for available updates via GitHub.
func CheckUpdate(manual bool) tea.Cmd {
	return func() tea.Msg {
		latest, err := build.GetLatestVersion()
		return UpdateCheckMsg{
			Current: build.Version,
			Latest:  latest,
			Manual:  manual,
			Err:     err,
		}
	}
}

// ApplyUpdateMsg is sent when an update operation completes.
type ApplyUpdateMsg struct {
	Err error
}

// ApplyUpdate performs the evolutionary ascent by calling the CLI update.
func ApplyUpdate() tea.Cmd {
	return func() tea.Msg {
		// We execute 'nido update' and wait for it to finish.
		// We try to find the current executable to call it explicitly.
		exe, err := os.Executable()
		if err != nil {
			exe = "nido" // Fallback
		}
		cmd := exec.Command(exe, "update")
		out, err := cmd.CombinedOutput()
		if err != nil {
			// Wrap the error with the output so the user sees WHY it failed
			return ApplyUpdateMsg{Err: fmt.Errorf("%s: %s", err, string(out))}
		}
		return ApplyUpdateMsg{Err: nil}
	}
}

// DoctorReport represents a single diagnostic check result.
type DoctorReport struct {
	Label   string `json:"label"`
	Passed  bool   `json:"passed"`
	Details string `json:"details"`
}

// DoctorResultMsg contains the parsed output of the doctor check.
type DoctorResultMsg struct {
	Reports []DoctorReport
	Err     error
}

// RunDoctor executes the system diagnostic tool.
func RunDoctor() tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("nido", "doctor", "--json").CombinedOutput()
		if err != nil {
			return DoctorResultMsg{Err: err}
		}

		var resp struct {
			Data struct {
				Reports []string `json:"reports"`
			} `json:"data"`
		}

		if err := json.Unmarshal(out, &resp); err != nil {
			return DoctorResultMsg{Err: err}
		}

		reports := make([]DoctorReport, 0, len(resp.Data.Reports))
		for _, r := range resp.Data.Reports {
			var status string
			var statusIdx int
			if idx := strings.Index(r, "[PASS]"); idx != -1 {
				status = "[PASS]"
				statusIdx = idx
			} else if idx := strings.Index(r, "[FAIL]"); idx != -1 {
				status = "[FAIL]"
				statusIdx = idx
			} else {
				continue
			}

			label := strings.TrimSpace(r[:statusIdx])
			details := strings.TrimSpace(r[statusIdx+len(status):])

			reports = append(reports, DoctorReport{
				Label:   label,
				Passed:  status == "[PASS]",
				Details: details,
			})
		}

		return DoctorResultMsg{Reports: reports}
	}
}

// Config Request Messages
type SaveConfigMsg struct{ Key, Value string }

// REMOVED DUPLICATE ConfigSavedMsg HERE

type ConfigBatchSavedMsg struct {
	Updates map[string]string
}

// SaveConfig updates a config value and reloads the config.
func SaveConfig(cfg *config.Config, key, value string) tea.Cmd {
	return func() tea.Msg {
		// Find config file path
		home, _ := os.UserHomeDir()
		path := filepath.Join(home, ".nido", "config.env")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			cwd, _ := os.Getwd()
			path = filepath.Join(cwd, "config", "config.env")
		}

		err := config.UpdateConfig(path, key, value)
		if err != nil {
			return nil
		}

		// Reload config into memory
		newCfg, _ := config.LoadConfig(path)
		*cfg = *newCfg

		return ConfigSavedMsg{Key: key, Value: value}
	}
}

// SaveConfigMany updates multiple config values and reloads the config.
func SaveConfigMany(cfg *config.Config, updates map[string]string) tea.Cmd {
	return func() tea.Msg {
		// Find config file path
		home, _ := os.UserHomeDir()
		path := filepath.Join(home, ".nido", "config.env")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			cwd, _ := os.Getwd()
			path = filepath.Join(cwd, "config", "config.env")
		}

		err := config.UpdateConfigMany(path, updates)
		if err != nil {
			return nil
		}

		// Reload config into memory
		newCfg, _ := config.LoadConfig(path)
		*cfg = *newCfg

		return ConfigBatchSavedMsg{Updates: updates}
	}
}
