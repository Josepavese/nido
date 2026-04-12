package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	clijson "github.com/Josepavese/nido/internal/cli"
	"github.com/Josepavese/nido/internal/image"
	"github.com/Josepavese/nido/internal/provider"
	"github.com/Josepavese/nido/internal/ui"
)

// cmdImageList identifies all species currently documented in the catalog
// and identifies which ones have already been pulle to our local nest.
func cmdImageList(imageDir string, args []string, jsonOut bool) {
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
	ui.Header("Image Catalog")
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
		fmt.Printf("%sOfficial%s\n", ui.Bold+ui.Cyan, ui.Reset)
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
		fmt.Printf("%sNido Images%s\n", ui.Bold+ui.Magenta, ui.Reset)
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
func cmdImagePull(imageDir string, args []string, jsonOut bool) {
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
			ui.Header("Image Download")
			ui.FancyLabel("Image", fmt.Sprintf("%s:%s", img.Name, ver.Version))
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
				ui.Step("Verifying image checksum...")
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
				ui.Step("Decompressing image...")
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
		ui.Info("Image %s:%s is already present in cache.", img.Name, ver.Version)
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

	ui.Success("Image downloaded and verified.")
	ui.Info("You can now spawn a VM using: nido spawn my-vm --image %s:%s", img.Name, ver.Version)
}

// cmdImageInfo probes an image for metadata. Currently a fledgling command.
func cmdImageInfo(imageDir string, args []string, jsonOut bool) {
	if len(args) < 1 {
		if jsonOut {
			_ = clijson.PrintJSON(clijson.NewResponseError("image info", "ERR_INVALID_ARGS", "Missing image reference", "Usage: nido images info <image>[:version]", "", nil))
			os.Exit(1)
		}
		ui.Error("Usage: nido images info <image>[:version]")
		os.Exit(1)
	}

	name, version := parseImageRef(args[0])
	catalog, err := loadImageCatalog(imageDir)
	if err != nil {
		if jsonOut {
			_ = clijson.PrintJSON(clijson.NewResponseError("image info", "ERR_IO", "Catalog load failed", err.Error(), "Check your network connection and try again.", nil))
			os.Exit(1)
		}
		ui.Error("Failed to load catalog: %v", err)
		os.Exit(1)
	}

	img, ver, err := catalog.FindImage(name, version)
	if err != nil {
		if jsonOut {
			_ = clijson.PrintJSON(clijson.NewResponseError("image info", "ERR_NOT_FOUND", "Image not found", err.Error(), "Run 'nido images list' to see available images.", nil))
			os.Exit(1)
		}
		ui.Error("Image not found: %s", args[0])
		os.Exit(1)
	}

	diskPath := filepath.Join(imageDir, fmt.Sprintf("%s-%s.qcow2", img.Name, ver.Version))
	kernelPath := filepath.Join(imageDir, fmt.Sprintf("%s-%s.kernel", img.Name, ver.Version))
	initrdPath := filepath.Join(imageDir, fmt.Sprintf("%s-%s.initrd", img.Name, ver.Version))
	downloaded := fileExists(diskPath)

	resolvedSSHUser := img.SSHUser
	if ver.SSHUser != "" {
		resolvedSSHUser = ver.SSHUser
	}
	hasInitialPassword := ver.SSHPassword != "" || img.SSHPassword != ""
	sizeHuman := ver.SizeHuman
	if sizeHuman == "" {
		sizeHuman = image.FormatBytes(ver.SizeBytes)
	}

	if jsonOut {
		_ = clijson.PrintJSON(clijson.NewResponseOK("image info", map[string]interface{}{
			"image": map[string]interface{}{
				"name":                 img.Name,
				"registry":             img.Registry,
				"description":          img.Description,
				"homepage":             img.Homepage,
				"ssh_user":             resolvedSSHUser,
				"has_initial_password": hasInitialPassword,
				"downloaded":           downloaded,
				"paths": map[string]interface{}{
					"disk":   diskPath,
					"kernel": kernelPath,
					"initrd": initrdPath,
				},
				"version": map[string]interface{}{
					"version":        ver.Version,
					"aliases":        ver.Aliases,
					"arch":           ver.Arch,
					"url":            ver.URL,
					"size_bytes":     ver.SizeBytes,
					"size":           sizeHuman,
					"format":         ver.Format,
					"checksum_type":  ver.ChecksumType,
					"kernel_url":     ver.KernelURL,
					"initrd_url":     ver.InitrdURL,
					"cmdline":        ver.Cmdline,
					"kernel_present": fileExists(kernelPath),
					"initrd_present": fileExists(initrdPath),
				},
			},
		}))
		return
	}

	ui.Header("Image Details")
	ui.FancyLabel("Name", fmt.Sprintf("%s:%s", img.Name, ver.Version))
	ui.FancyLabel("Registry", img.Registry)
	ui.FancyLabel("Description", img.Description)
	if img.Homepage != "" {
		ui.FancyLabel("Homepage", img.Homepage)
	}
	if resolvedSSHUser != "" {
		ui.FancyLabel("SSH User", resolvedSSHUser)
	}
	ui.FancyLabel("Initial Password", ternaryString(hasInitialPassword, "Provided by image metadata", "Not specified"))
	ui.FancyLabel("Arch", ver.Arch)
	ui.FancyLabel("Format", ver.Format)
	ui.FancyLabel("Size", sizeHuman)
	ui.FancyLabel("Downloaded", ternaryString(downloaded, "Yes", "No"))
	if len(ver.Aliases) > 0 {
		ui.FancyLabel("Aliases", strings.Join(ver.Aliases, ", "))
	}
	if ver.Cmdline != "" {
		ui.FancyLabel("Default Cmdline", ver.Cmdline)
	}
	if ver.URL != "" {
		ui.FancyLabel("Source", ver.URL)
	}
	if ver.KernelURL != "" {
		ui.FancyLabel("Kernel", ternaryString(fileExists(kernelPath), "Downloaded", "Available separately"))
	}
	if ver.InitrdURL != "" {
		ui.FancyLabel("Initrd", ternaryString(fileExists(initrdPath), "Downloaded", "Available separately"))
	}
}

func cmdImageRemove(imageDir string, prov provider.VMProvider, args []string, jsonOut bool) {
	if len(args) < 1 {
		if jsonOut {
			_ = clijson.PrintJSON(clijson.NewResponseError("image remove", "ERR_INVALID_ARGS", "Missing image reference", "Usage: nido images remove <image>[:version]", "", nil))
			os.Exit(1)
		}
		ui.Error("Usage: nido images remove <image>[:version]")
		os.Exit(1)
	}

	name, version := parseImageRef(args[0])
	if err := prov.CacheRemove(name, version); err != nil {
		if isNotFoundErr(err) {
			if jsonOut {
				_ = clijson.PrintJSON(clijson.NewResponseOK("image remove", map[string]interface{}{
					"action": map[string]interface{}{"name": name, "version": version, "result": "not_found"},
				}))
				return
			}
			ui.Info("Image %s is already gone from cache.", args[0])
			return
		}
		if jsonOut {
			_ = clijson.PrintJSON(clijson.NewResponseError("image remove", "ERR_IO", "Image remove failed", err.Error(), "Check whether the image is still in use by a VM.", nil))
			os.Exit(1)
		}
		ui.Error("Failed to remove image: %v", err)
		os.Exit(1)
	}

	if version != "" {
		_ = os.Remove(filepath.Join(imageDir, fmt.Sprintf("%s-%s.kernel", name, version)))
		_ = os.Remove(filepath.Join(imageDir, fmt.Sprintf("%s-%s.initrd", name, version)))
	}

	if jsonOut {
		_ = clijson.PrintJSON(clijson.NewResponseOK("image remove", map[string]interface{}{
			"action": map[string]interface{}{"name": name, "version": version, "result": "removed"},
		}))
		return
	}
	ui.Success("Image %s removed from cache.", args[0])
}

func cmdImageUpdate(imageDir string, args []string, jsonOut bool) {
	if !jsonOut {
		ui.Step("Refreshing catalog...")
	}
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

	ui.Success("Catalog updated.")
}

func loadImageCatalog(imageDir string) (*image.Catalog, error) {
	cwd, _ := os.Getwd()
	localRegistry := filepath.Join(cwd, "registry", "images.json")
	if _, statErr := os.Stat(localRegistry); statErr == nil {
		return image.LoadCatalogFromFile(localRegistry)
	}
	return image.LoadCatalog(imageDir, image.DefaultCacheTTL)
}

func parseImageRef(target string) (string, string) {
	name := target
	version := ""
	if strings.Contains(target, ":") {
		parts := strings.SplitN(target, ":", 2)
		name = parts[0]
		version = parts[1]
	}
	return name, version
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func ternaryString(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
