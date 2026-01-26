package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	clijson "github.com/Josepavese/nido/internal/cli"
	"github.com/Josepavese/nido/internal/image"
	"github.com/Josepavese/nido/internal/ui"
)

// cmdImage is the primary dispatcher for all image-related subcommands.
// It bridges the gap between the image registry and local cache.
func cmdImage(nidoDir string, args []string) {
	jsonOut, rest := consumeJSONFlag(args)
	if len(rest) == 0 {
		ui.Error("Usage: nido image <list|pull|info|remove|update>")
		os.Exit(1)
	}

	subcommand := rest[0]

	switch subcommand {
	case "list", "ls":
		cmdImageList(nidoDir, rest[1:], jsonOut)
	case "pull":
		cmdImagePull(nidoDir, rest[1:], jsonOut)
	case "info":
		cmdImageInfo(nidoDir, rest[1:], jsonOut)
	case "remove":
		cmdImageRemove(nidoDir, rest[1:], jsonOut)
	case "update":
		cmdImageUpdate(nidoDir, rest[1:], jsonOut)
	default:
		ui.Error("Unknown image subcommand: %s", subcommand)
		os.Exit(1)
	}
}

// cmdImageList identifies all species currently documented in the catalog
// and identifies which ones have already been pulle to our local nest.
func cmdImageList(nidoDir string, args []string, jsonOut bool) {
	// Determine image directory
	imageDir := filepath.Join(nidoDir, "images")

	// Load catalog
	var catalog *image.Catalog
	var err error
	cwd, _ := os.Getwd()
	localRegistry := filepath.Join(cwd, "registry", "images.json")
	if _, statErr := os.Stat(localRegistry); statErr == nil {
		catalog, err = image.LoadCatalogFromFile(localRegistry)
	} else {
		catalog, err = image.LoadCatalog(imageDir, image.DefaultCacheTTL)
	}

	if err != nil {
		if jsonOut {
			resp := clijson.NewResponseError("image list", "ERR_IO", "Catalog load failed", err.Error(), "Check your network connection and try again.", nil)
			_ = clijson.PrintJSON(resp)
			os.Exit(1)
		}
		ui.Error("Failed to load image catalog: %v", err)
		os.Exit(1)
	}

	if jsonOut {
		type imageJSON struct {
			Name       string   `json:"name"`
			Version    string   `json:"version"`
			Registry   string   `json:"registry"`
			SizeBytes  int64    `json:"size_bytes"`
			Aliases    []string `json:"aliases,omitempty"`
			Downloaded bool     `json:"downloaded"`
		}

		items := []imageJSON{}
		for _, img := range catalog.Images {
			for _, v := range img.Versions {
				imagePath := filepath.Join(imageDir, fmt.Sprintf("%s-%s.qcow2", img.Name, v.Version))
				downloaded := false
				if _, err := os.Stat(imagePath); err == nil {
					downloaded = true
				}
				items = append(items, imageJSON{
					Name:       img.Name,
					Version:    v.Version,
					Registry:   img.Registry,
					SizeBytes:  v.SizeBytes,
					Aliases:    v.Aliases,
					Downloaded: downloaded,
				})
			}
		}

		resp := clijson.NewResponseOK("image list", map[string]interface{}{
			"images": items,
		})
		_ = clijson.PrintJSON(resp)
		return
	}

	// Display header
	ui.Header("Available Images")
	fmt.Println("")

	// Group images by registry
	official := []image.Image{}
	nidoImages := []image.Image{}

	for _, img := range catalog.Images {
		if img.Registry == "official" {
			official = append(official, img)
		} else if img.Registry == "nido" {
			nidoImages = append(nidoImages, img)
		}
	}

	// Display official images
	if len(official) > 0 {
		fmt.Printf("%sOFFICIAL (Upstream Proxy):%s\n", ui.Bold+ui.Cyan, ui.Reset)
		for _, img := range official {
			for _, v := range img.Versions {
				aliases := ""
				if len(v.Aliases) > 0 {
					aliases = fmt.Sprintf(" %s(%s)%s", ui.Dim, v.Aliases[0], ui.Reset)
				}

				downloaded := ""
				imagePath := filepath.Join(imageDir, fmt.Sprintf("%s-%s.qcow2", img.Name, v.Version))
				if _, err := os.Stat(imagePath); err == nil {
					downloaded = fmt.Sprintf(" %s[downloaded]%s", ui.Green, ui.Reset)
				}

				fmt.Printf("  %s%-20s%s%s %s%s%s%s\n",
					ui.Cyan, fmt.Sprintf("%s:%s", img.Name, v.Version), ui.Reset,
					aliases, ui.Dim, ui.HumanSize(v.SizeBytes), ui.Reset, downloaded)
			}
		}
		fmt.Println("")
	}

	// Display nido-optimized flavours
	if len(nidoImages) > 0 {
		fmt.Printf("%sNIDO FLAVOURS (Compressed & Optimized):%s\n", ui.Bold+ui.Magenta, ui.Reset)
		for _, img := range nidoImages {
			for _, v := range img.Versions {
				downloaded := ""
				imagePath := filepath.Join(imageDir, fmt.Sprintf("%s-%s.qcow2", img.Name, v.Version))
				if _, err := os.Stat(imagePath); err == nil {
					downloaded = fmt.Sprintf(" %s[downloaded]%s", ui.Green, ui.Reset)
				}

				fmt.Printf("  %s%-20s%s %s%s%s%s %s[PRECONFIGURED]%s\n",
					ui.Magenta, fmt.Sprintf("%s:%s", img.Name, v.Version), ui.Reset,
					ui.Dim, ui.HumanSize(v.SizeBytes), ui.Reset, downloaded,
					ui.Bold+ui.Magenta, ui.Reset)
			}
		}
		fmt.Println("")
	}

	// Show catalog age
	cachePath := filepath.Join(imageDir, image.CatalogCacheFile)
	if stat, err := os.Stat(cachePath); err == nil {
		age := time.Since(stat.ModTime())
		fmt.Printf("%sCatalog updated %s ago%s\n", ui.Dim, formatDuration(age), ui.Reset)
	}

	fmt.Println("")
}

// formatDuration formats a duration in human-readable form
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	} else if d < time.Hour {
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", mins)
	} else if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	} else {
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day"
		}
		return fmt.Sprintf("%d days", days)
	}
}

// Stubs for other commands (Phase 2+)
// cmdImagePull initiates the retrieval of a specific image species.
// It handles resume logic, multi-part downloads, and verification.
func cmdImagePull(nidoDir string, args []string, jsonOut bool) {
	if len(args) < 1 {
		ui.Error("Usage: nido image pull <name>[:version]")
		os.Exit(1)
	}

	target := args[0]
	name := target
	version := ""

	// Parse version if present
	if strings.Contains(target, ":") {
		parts := strings.Split(target, ":")
		name = parts[0]
		version = parts[1]
	}

	imageDir := filepath.Join(nidoDir, "images")

	// Load catalog
	var catalog *image.Catalog
	var err error
	cwd, _ := os.Getwd()
	localRegistry := filepath.Join(cwd, "registry", "images.json")
	if _, statErr := os.Stat(localRegistry); statErr == nil {
		catalog, err = image.LoadCatalogFromFile(localRegistry)
	} else {
		catalog, err = image.LoadCatalog(imageDir, image.DefaultCacheTTL)
	}

	if err != nil {
		if jsonOut {
			resp := clijson.NewResponseError("image pull", "ERR_IO", "Catalog load failed", err.Error(), "Check your network connection and try again.", nil)
			_ = clijson.PrintJSON(resp)
			os.Exit(1)
		}
		ui.Error("Failed to load catalog: %v", err)
		os.Exit(1)
	}

	// Find image
	img, ver, err := catalog.FindImage(name, version)
	if err != nil {
		if jsonOut {
			resp := clijson.NewResponseError("image pull", "ERR_NOT_FOUND", "Image not found", err.Error(), "Run 'nido image list' to see available images.", nil)
			_ = clijson.PrintJSON(resp)
			os.Exit(1)
		}
		ui.Error("Image not found: %s", target)
		ui.Info("Try 'nido image list' to see available images.")
		os.Exit(1)
	}

	// Target file path (stored as qcow2 for now, future proofing for potential raw)
	destPath := filepath.Join(imageDir, fmt.Sprintf("%s-%s.qcow2", img.Name, ver.Version))

	// Check if already exists and verified
	// Note: We used to return early here, but now we MUST continue to check
	// for associated kernel/initrd artifacts that might be missing.
	diskExists := false
	if _, err := os.Stat(destPath); err == nil {
		diskExists = true
	}

	downloader := image.Downloader{Quiet: jsonOut}

	if !diskExists {
		// Announce download
		if !jsonOut {
			ui.Header(fmt.Sprintf("Pulling %s:%s", img.Name, ver.Version))
			ui.Info("Source: %s", ver.URL)
			ui.Info("Size:   %s", ui.HumanSize(ver.SizeBytes))
		}

		// Determine if compression is needed based on URL extension
		isCompressed := strings.Contains(ver.URL, ".zst") || strings.Contains(ver.URL, ".zstandard")
		if len(ver.PartURLs) > 0 {
			isCompressed = strings.Contains(ver.PartURLs[0], ".zst") || strings.Contains(ver.PartURLs[0], ".zstandard")
		}

		downloadPath := destPath
		if isCompressed {
			downloadPath = destPath + ".zst"
		}

		// Download
		var downloadErr error
		if len(ver.PartURLs) > 0 {
			downloadErr = downloader.DownloadMultiPart(ver.PartURLs, downloadPath, ver.SizeBytes)
		} else {
			downloadErr = downloader.Download(ver.URL, downloadPath, ver.SizeBytes)
		}

		if downloadErr != nil {
			if jsonOut {
				resp := clijson.NewResponseError("image pull", "ERR_IO", "Disk download failed", downloadErr.Error(), "Check your connection and try again.", nil)
				_ = clijson.PrintJSON(resp)
				os.Exit(1)
			}
			ui.Error("Disk download failed: %v", downloadErr)
			os.Remove(downloadPath) // Cleanup
			os.Exit(1)
		}

		// Verify Checksum (on the downloaded file)
		if ver.Checksum != "" {
			if !jsonOut {
				ui.Ironic("Verifying disk genetic integrity...")
			}
			if err := image.VerifyChecksum(downloadPath, ver.Checksum, ver.ChecksumType); err != nil {
				if jsonOut {
					resp := clijson.NewResponseError("image pull", "ERR_IO", "Disk checksum verification failed", err.Error(), "Retry the download or choose a different image.", nil)
					_ = clijson.PrintJSON(resp)
				} else {
					ui.Error("Disk checksum verification failed: %v", err)
				}
				os.Remove(downloadPath)
				os.Exit(1)
			}
		}

		// Decompress if needed
		if isCompressed {
			if !jsonOut {
				ui.Ironic("Decompressing compressed flavour...")
			}
			if err := downloader.Decompress(downloadPath, destPath); err != nil {
				if jsonOut {
					resp := clijson.NewResponseError("image pull", "ERR_IO", "Decompression failed", err.Error(), "Ensure zstd is installed on your system.", nil)
					_ = clijson.PrintJSON(resp)
				} else {
					ui.Error("Decompression failed: %v", err)
				}
				os.Remove(downloadPath)
				os.Exit(1)
			}
			// Final cleanup of the compressed part
			os.Remove(downloadPath)
		}
	} else if !jsonOut {
		ui.Success("Image disk %s:%s is already downloaded.", img.Name, ver.Version)
	}

	// --- 2. Download Kernel (if defined) ---
	if ver.KernelURL != "" {
		kernelPath := filepath.Join(imageDir, fmt.Sprintf("%s-%s.kernel", img.Name, ver.Version))
		if _, err := os.Stat(kernelPath); os.IsNotExist(err) {
			if !jsonOut {
				ui.Info("Pulling kernel from %s...", ver.KernelURL)
			}
			if err := downloader.Download(ver.KernelURL, kernelPath, 0); err != nil {
				ui.Error("Kernel download failed: %v", err)
				os.Exit(1)
			}
			if ver.KernelChecksum != "" {
				if err := image.VerifyChecksum(kernelPath, ver.KernelChecksum, ver.ChecksumType); err != nil {
					ui.Error("Kernel verification failed: %v", err)
					os.Remove(kernelPath)
					os.Exit(1)
				}
			}
		}
	}

	// --- 3. Download Initrd (if defined) ---
	if ver.InitrdURL != "" {
		initrdPath := filepath.Join(imageDir, fmt.Sprintf("%s-%s.initrd", img.Name, ver.Version))
		if _, err := os.Stat(initrdPath); os.IsNotExist(err) {
			if !jsonOut {
				ui.Info("Pulling initrd from %s...", ver.InitrdURL)
			}
			if err := downloader.Download(ver.InitrdURL, initrdPath, 0); err != nil {
				ui.Error("Initrd download failed: %v", err)
				os.Exit(1)
			}
			if ver.InitrdChecksum != "" {
				if err := image.VerifyChecksum(initrdPath, ver.InitrdChecksum, ver.ChecksumType); err != nil {
					ui.Error("Initrd verification failed: %v", err)
					os.Remove(initrdPath)
					os.Exit(1)
				}
			}
		}
	}

	if jsonOut {
		resp := clijson.NewResponseOK("image pull", map[string]interface{}{
			"action": map[string]interface{}{
				"name":    img.Name,
				"version": ver.Version,
				"result":  "downloaded",
			},
		})
		_ = clijson.PrintJSON(resp)
		return
	}

	ui.Success("Image downloaded and verified successfully! 🐣")
	ui.Info("You can now spawn a VM using: nido spawn my-vm --image %s:%s", img.Name, ver.Version)
}

// cmdImageInfo probes an image for metadata. Currently a fledgling command.
func cmdImageInfo(nidoDir string, args []string, jsonOut bool) {
	if jsonOut {
		resp := clijson.NewResponseError("image info", "ERR_NOT_IMPLEMENTED", "Not implemented", "Image info is not available yet.", "Use 'nido image list' to see available images.", nil)
		_ = clijson.PrintJSON(resp)
		return
	}
	ui.Info("Coming soon! Use 'nido image list' to see available images.")
}

func cmdImageRemove(nidoDir string, args []string, jsonOut bool) {
	if jsonOut {
		resp := clijson.NewResponseError("image remove", "ERR_NOT_IMPLEMENTED", "Not implemented", "Image remove is not available yet.", "Use 'nido cache rm' for cached images.", nil)
		_ = clijson.PrintJSON(resp)
		return
	}
	ui.Info("Coming soon! This will remove downloaded images.")
}

func cmdImageUpdate(nidoDir string, args []string, jsonOut bool) {
	if !jsonOut {
		ui.Ironic("Refreshing the catalog...")
	}
	imageDir := filepath.Join(nidoDir, "images")
	cachePath := filepath.Join(imageDir, image.CatalogCacheFile)

	// Remove cache to force refresh
	if err := os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
		if jsonOut {
			resp := clijson.NewResponseError("image update", "ERR_IO", "Cache clear failed", err.Error(), "Check file permissions and try again.", nil)
			_ = clijson.PrintJSON(resp)
			os.Exit(1)
		}
		ui.Error("Failed to clear cache: %v", err)
		os.Exit(1)
	}

	// Reload catalog (will fetch from remote)
	_, err := image.LoadCatalog(imageDir, 0) // TTL=0 forces refresh
	if err != nil {
		if jsonOut {
			resp := clijson.NewResponseError("image update", "ERR_IO", "Catalog update failed", err.Error(), "Check your network connection and try again.", nil)
			_ = clijson.PrintJSON(resp)
			os.Exit(1)
		}
		ui.Error("Failed to update catalog: %v", err)
		os.Exit(1)
	}

	if jsonOut {
		resp := clijson.NewResponseOK("image update", map[string]interface{}{
			"action": map[string]interface{}{
				"result": "updated",
			},
		})
		_ = clijson.PrintJSON(resp)
		return
	}

	ui.Success("Catalog updated successfully! 🐣")
}
