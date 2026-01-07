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

// cmdImage handles all image subcommands
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

// cmdImageList displays all available images from the catalog
func cmdImageList(nidoDir string, args []string, jsonOut bool) {
	// Determine image directory
	imageDir := filepath.Join(nidoDir, "images")

	// Load catalog
	catalog, err := image.LoadCatalog(imageDir, image.DefaultCacheTTL)
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
	catalog, err := image.LoadCatalog(imageDir, image.DefaultCacheTTL)
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
	// Note: Ideally we'd verify checksum here too, but that's slow.
	// We assume if file exists at final path, it's good.
	if _, err := os.Stat(destPath); err == nil {
		if jsonOut {
			resp := clijson.NewResponseOK("image pull", map[string]interface{}{
				"action": map[string]interface{}{
					"name":    img.Name,
					"version": ver.Version,
					"result":  "already_present",
				},
			})
			_ = clijson.PrintJSON(resp)
			return
		}
		ui.Success("Image %s:%s is already downloaded.", img.Name, ver.Version)
		return
	}

	// Announce download
	if !jsonOut {
		ui.Header(fmt.Sprintf("Pulling %s:%s", img.Name, ver.Version))
		ui.Info("Source: %s", ver.URL)
		ui.Info("Size:   %s", ui.HumanSize(ver.SizeBytes))
	}

	// Download
	downloader := image.Downloader{Quiet: jsonOut}
	var downloadErr error
	if len(ver.PartURLs) > 0 {
		downloadErr = downloader.DownloadMultiPart(ver.PartURLs, destPath, ver.SizeBytes)
	} else {
		downloadErr = downloader.Download(ver.URL, destPath, ver.SizeBytes)
	}

	if downloadErr != nil {
		if jsonOut {
			resp := clijson.NewResponseError("image pull", "ERR_IO", "Download failed", downloadErr.Error(), "Check your connection and try again.", nil)
			_ = clijson.PrintJSON(resp)
			os.Exit(1)
		}
		ui.Error("Download failed: %v", downloadErr)
		os.Exit(1)
	}

	// Verify Checksum
	if !jsonOut {
		ui.Ironic("Verifying genetic integrity...")
	}
	if err := image.VerifyChecksum(destPath, ver.Checksum, ver.ChecksumType); err != nil {
		if jsonOut {
			resp := clijson.NewResponseError("image pull", "ERR_IO", "Checksum verification failed", err.Error(), "Retry the download or choose a different image.", nil)
			_ = clijson.PrintJSON(resp)
		} else {
			ui.Error("Checksum verification failed: %v", err)
			ui.Warn("The downloaded file may be corrupted or tampered with.")
			ui.Warn("Deleting corrupted file...")
		}
		os.Remove(destPath)
		os.Exit(1)
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
