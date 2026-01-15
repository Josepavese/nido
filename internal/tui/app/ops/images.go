// Package ops provides command functions for the Nido TUI.
package ops

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Josepavese/nido/internal/image"
	"github.com/Josepavese/nido/internal/provider"
	view "github.com/Josepavese/nido/internal/tui/kit/view"
	tea "github.com/charmbracelet/bubbletea"
)

// ProgressMsg carries progress updates to the UI loop
type ProgressMsg struct {
	OpName string         // Stable operation key (e.g. "pull ubuntu:22.04")
	Status view.StatusMsg // Display status (Operation becomes the card title)
	Next   tea.Cmd
	Result *OpResultMsg
}

// --- Image/Source Message Types ---

// SourcesLoadedMsg contains the list of available sources (templates/images/VMs).
type SourcesLoadedMsg struct {
	Sources []string
	Err     error
}

// SourceAction defines what type of sources to fetch.
type SourceAction int

const (
	// SourceActionSpawn fetches templates and cloud images for spawning VMs.
	SourceActionSpawn SourceAction = 0
	// SourceActionTemplate fetches VMs for creating templates.
	SourceActionTemplate SourceAction = 1
)

// --- Image Commands ---

// FetchSources retrieves available sources based on the action type.
// cachedOnly: if true, avoids network calls (uses cache or fails for remote items).
// forceRemote: if true, forces a network refresh of the catalog.
func FetchSources(prov provider.VMProvider, action SourceAction, cachedOnly, forceRemote bool) tea.Cmd {
	return func() tea.Msg {
		// DEBUG logging
		f, _ := os.OpenFile("/tmp/nido-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if f != nil {
			fmt.Fprintf(f, "[%s] FetchSources Start: Action=%v, CachedOnly=%v, ForceRemote=%v\n", time.Now().Format(time.RFC3339), action, cachedOnly, forceRemote)
			defer f.Close()
		}

		var srcList []string

		if action == SourceActionSpawn {
			// 1. Local Templates (from BackupDir)
			if prov == nil {
				return SourcesLoadedMsg{Err: fmt.Errorf("internal error: provider is nil")}
			}

			templates, err := prov.ListTemplates()
			if err != nil {
				return SourcesLoadedMsg{Err: err}
			}
			sort.Strings(templates)

			// 2. Cloud Images from Catalog (with registry distinction)
			cfg := prov.GetConfig()
			catalogDir := cfg.ImageDir
			if catalogDir == "" {
				home, _ := os.UserHomeDir()
				catalogDir = filepath.Join(home, ".nido", "images")
			}

			var catalog *image.Catalog
			var catErr error

			if cachedOnly {
				catalog, catErr = image.LoadCatalogFromFile(filepath.Join(catalogDir, image.CatalogCacheFile))
			} else if forceRemote {
				catalog, catErr = image.LoadCatalog(catalogDir, 0)
			} else {
				catalog, catErr = image.LoadCatalog(catalogDir, image.DefaultCacheTTL)
			}

			if catErr != nil || catalog == nil {
				// Catalog failed, but we can still show templates
				for _, tpl := range templates {
					srcList = append(srcList, fmt.Sprintf("[TEMPLATE] %s", tpl))
				}
				if len(srcList) == 0 {
					return SourcesLoadedMsg{Err: fmt.Errorf("no templates found and catalog unavailable: %v", catErr)}
				}
				// Return just templates if catalog failed
				return SourcesLoadedMsg{Sources: srcList}
			}

			// Build source list with proper labels
			// Templates first
			for _, tpl := range templates {
				srcList = append(srcList, fmt.Sprintf("[TEMPLATE] %s", tpl))
			}

			// Then Flavours (registry=nido), then Cloud (registry=official)
			var flavours, cloud []string
			for _, img := range catalog.Images {
				for _, ver := range img.Versions {
					entry := fmt.Sprintf("%s:%s", img.Name, ver.Version)
					if img.Registry == "nido" {
						flavours = append(flavours, entry)
					} else {
						cloud = append(cloud, entry)
					}
				}
			}

			sort.Strings(flavours)
			sort.Strings(cloud)

			for _, f := range flavours {
				srcList = append(srcList, fmt.Sprintf("[FLAVOUR] %s", f))
			}
			for _, c := range cloud {
				srcList = append(srcList, fmt.Sprintf("[CLOUD] %s", c))
			}

			if len(srcList) == 0 {
				return SourcesLoadedMsg{Err: fmt.Errorf("no images or templates found")}
			}

			return SourcesLoadedMsg{Sources: srcList}
		}

		// SourceActionTemplate: List VMs for creating templates
		vms, err := prov.List()
		if err != nil {
			return SourcesLoadedMsg{Err: err}
		}
		sort.Slice(vms, func(i, j int) bool {
			return vms[i].Name < vms[j].Name
		})
		for _, vm := range vms {
			srcList = append(srcList, fmt.Sprintf("[VM] %s", vm.Name))
		}

		if len(srcList) == 0 {
			return SourcesLoadedMsg{Err: fmt.Errorf("no images or templates found")}
		}

		return SourcesLoadedMsg{Sources: srcList}
	}
}

// FetchRegistryImages retrieves available remote images as structured data.
func FetchRegistryImages(prov provider.VMProvider, forceRemote bool) tea.Cmd {
	return func() tea.Msg {
		if prov == nil {
			return RegistryListMsg{Err: fmt.Errorf("provider is nil")}
		}

		cfg := prov.GetConfig()
		catalogDir := cfg.ImageDir
		if catalogDir == "" {
			home, _ := os.UserHomeDir()
			catalogDir = filepath.Join(home, ".nido", "images")
		}

		ttl := image.DefaultCacheTTL
		if forceRemote {
			ttl = 0
		}
		catalog, err := image.LoadCatalog(catalogDir, ttl)
		if err != nil {
			return RegistryListMsg{Err: err}
		}

		var results []RegistryImage
		for _, img := range catalog.Images {
			for _, ver := range img.Versions {
				results = append(results, RegistryImage{
					Name:        img.Name,
					Version:     ver.Version,
					Registry:    img.Registry,
					Description: img.Description,
				})
			}
		}

		sort.Slice(results, func(i, j int) bool {
			if results[i].Registry != results[j].Registry {
				return results[i].Registry == "nido"
			}
			if results[i].Name != results[j].Name {
				return results[i].Name < results[j].Name
			}
			return results[i].Version > results[j].Version
		})

		return RegistryListMsg{Images: results}
	}
}

// PullImage initiates a download operation with progress tracking.
func PullImage(prov provider.VMProvider, imageRef string) tea.Cmd {
	opName := fmt.Sprintf("pull %s", imageRef)

	return func() tea.Msg {
		// Channel to receive progress updates from the goroutine
		// Buffer it slightly to avoid blocking the downloader too much
		ch := make(chan ProgressMsg, 10)

		// Start the work in a goroutine
		go func() {
			defer close(ch)

			if prov == nil {
				ch <- ProgressMsg{Result: &OpResultMsg{Op: opName, Err: fmt.Errorf("provider is nil")}}
				return
			}

			// Resolve Image
			cfg := prov.GetConfig()
			imgDir := cfg.ImageDir
			if imgDir == "" {
				home, _ := os.UserHomeDir()
				imgDir = filepath.Join(home, ".nido", "images")
			}

			catalog, err := image.LoadCatalog(imgDir, image.DefaultCacheTTL)
			if err != nil {
				ch <- ProgressMsg{Result: &OpResultMsg{Op: opName, Err: fmt.Errorf("catalog load failed: %w", err)}}
				return
			}

			pName, pVer := imageRef, ""
			if strings.Contains(imageRef, ":") {
				parts := strings.Split(imageRef, ":")
				pName, pVer = parts[0], parts[1]
			}

			img, ver, err := catalog.FindImage(pName, pVer)
			if err != nil {
				ch <- ProgressMsg{Result: &OpResultMsg{Op: opName, Err: fmt.Errorf("image not found: %w", err)}}
				return
			}

			destPath := filepath.Join(imgDir, fmt.Sprintf("%s-%s.qcow2", img.Name, ver.Version))
			if _, err := os.Stat(destPath); err == nil {
				// Already exists
				ch <- ProgressMsg{Result: &OpResultMsg{Op: opName, Err: nil}}
				return
			}

			// Configure Downloader with Progress Callback
			downloader := image.Downloader{
				Quiet: true,
				OnProgress: func(current, total int64) {
					ratio := 0.0
					if total > 0 {
						ratio = float64(current) / float64(total)
					}
					// Send status update
					// Use non-blocking send logic or just blocking?
					// Use blocking but the buffer helps.
					ch <- ProgressMsg{
						OpName: opName,
						Status: view.StatusMsg{
							Loading:   true,
							Operation: fmt.Sprintf("Pulling %s", imageRef),
							Progress:  ratio,
						},
					}
				},
			}

			downloadPath := destPath
			isCompressed := strings.HasSuffix(ver.URL, ".tar.xz")
			if isCompressed {
				downloadPath = destPath + ".tar.xz"
			}

			if len(ver.PartURLs) > 0 {
				err = downloader.DownloadMultiPart(ver.PartURLs, downloadPath, ver.SizeBytes)
			} else {
				err = downloader.Download(ver.URL, downloadPath, ver.SizeBytes)
			}

			if err != nil {
				ch <- ProgressMsg{Result: &OpResultMsg{Op: opName, Err: err}}
				return
			}

			// Verify & Decompress (Indeterminate progress)
			ch <- ProgressMsg{
				OpName: opName,
				Status: view.StatusMsg{
					Loading:   true,
					Operation: fmt.Sprintf("Verifying %s", pName),
					Progress:  1.0,
				},
			}

			if err := image.VerifyChecksum(downloadPath, ver.Checksum, ver.ChecksumType); err != nil {
				os.Remove(downloadPath)
				ch <- ProgressMsg{Result: &OpResultMsg{Op: opName, Err: fmt.Errorf("verification failed: %w", err)}}
				return
			}

			if isCompressed {
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

			ch <- ProgressMsg{Result: &OpResultMsg{Op: opName, Err: nil}}
		}()

		return waitForProgress(ch)()
	}
}

func waitForProgress(ch <-chan ProgressMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		// If it has a result, we stop chaining (or we chain for last update?)
		// Logic: if result is set, we return it as is (NidoApp handles it).
		// But wait, NidoApp expects ProgressMsg OR OpResultMsg?
		// NidoApp needs to handle ProgressMsg.

		// We attach Next to all messages to keep the loop valid until channel close?
		// If OpResult is set, we don't need Next because loop ends.
		if msg.Result != nil {
			// Ensure we tell UI we are done (Loading=false)?
			// NidoApp logic handles OpResultMsg by finishing action.
			return msg
		}

		// Otherwise carry on
		msg.Next = waitForProgress(ch)
		return msg
	}
}
