package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Josepavese/nido/internal/build"
	clijson "github.com/Josepavese/nido/internal/cli"
	"github.com/Josepavese/nido/internal/config"
	"github.com/Josepavese/nido/internal/image"
	"github.com/Josepavese/nido/internal/lifecycle"
	"github.com/Josepavese/nido/internal/mcp"
	"github.com/Josepavese/nido/internal/provider"
	"github.com/Josepavese/nido/internal/ui"
)

// Version is handled by the build package

// main is the brain of the Nido CLI. It handles command line parsing,
// configuration loading, and dispatches requests to the appropriate handlers.
func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	// Home Dir for Nido State
	home, _ := os.UserHomeDir()
	nidoDir := filepath.Join(home, ".nido")

	cwd, _ := os.Getwd()

	// Load the genetic configuration for our nest
	cfgPath := filepath.Join(nidoDir, "config.env")
	// For dev, check local project config first if nidoDir doesn't have it
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		cfgPath = filepath.Join(cwd, "config", "config.env")
	}

	cfg, _ := config.LoadConfig(cfgPath)
	// Hatch a QMP-enabled QEMU provider
	prov := provider.NewQemuProvider(nidoDir, cfg)

	switch cmd {
	case "version":
		jsonOut, _ := consumeJSONFlag(args)
		cmdVersion(jsonOut)
	case "help":
		printUsage()
	case "gui":
		cmdGUI(prov, cfg)
	case "update":
		cmdUpdate(nidoDir)
	case "mcp-help":
		// List MCP tools and their schemas
		cmdMcpHelp()
	case "ls", "list":
		// Scan the nest and report all detected life forms
		jsonOut, _ := consumeJSONFlag(args)
		vms, err := prov.List()
		if jsonOut {
			if err != nil {
				resp := clijson.NewResponseError("ls", "ERR_INTERNAL", "List failed", err.Error(), "Try again or run nido doctor for diagnostics.", nil)
				_ = clijson.PrintJSON(resp)
				os.Exit(1)
			}

			type vmJSON struct {
				Name    string `json:"name"`
				State   string `json:"state"`
				PID     int    `json:"pid"`
				SSHPort int    `json:"ssh_port"`
				VNCPort int    `json:"vnc_port"`
				SSHUser string `json:"ssh_user,omitempty"`
			}

			items := make([]vmJSON, 0, len(vms))
			for _, vm := range vms {
				items = append(items, vmJSON{
					Name:    vm.Name,
					State:   vm.State,
					PID:     vm.PID,
					SSHPort: vm.SSHPort,
					VNCPort: vm.VNCPort,
					SSHUser: vm.SSHUser,
				})
			}

			resp := clijson.NewResponseOK("ls", map[string]interface{}{
				"vms": items,
			})
			_ = clijson.PrintJSON(resp)
			return
		}
		if len(vms) == 0 {
			ui.Info("The nest is quiet. No VMs are currently making noise.")
		} else {
			fmt.Printf("\n %s%-20s %-12s %-10s %s%s\n", ui.Bold, "NAME", "STATE", "PID", "PORT", ui.Reset)
			fmt.Printf(" %s%s%s\n", ui.Dim, strings.Repeat("-", 60), ui.Reset)
			for _, vm := range vms {
				stateColor := ui.Yellow
				if vm.State == "running" {
					stateColor = ui.Green
				}

				// Port Summary (Compact)
				// Implements Section 6.2 of advanced-port-forwarding.md.
				portSum := fmt.Sprintf("%d", vm.SSHPort)
				for _, f := range vm.Forwarding {
					if f.HostPort > 0 {
						portSum += fmt.Sprintf(",%d->%d", f.GuestPort, f.HostPort)
					}
				}
				if len(portSum) > 25 {
					portSum = portSum[:22] + "..."
				}

				fmt.Printf(" %-20s %s%-12s%s %-10d %s\n", vm.Name, stateColor, vm.State, ui.Reset, vm.PID, portSum)
			}
			fmt.Println("")
		}
	case "ssh":
		if len(args) < 1 {
			ui.Error("Usage: nido ssh <name>")
			os.Exit(1)
		}
		cmdSsh(prov, args[0], args[1:])
	case "doctor":
		jsonOut, _ := consumeJSONFlag(args)
		cmdDoctor(prov, jsonOut)
	case "info":
		jsonOut, rest := consumeJSONFlag(args)
		if len(rest) < 1 {
			fmt.Println("Usage: nido info <name>")
			os.Exit(1)
		}
		info, err := prov.Info(rest[0])
		if err != nil {
			if jsonOut {
				resp := clijson.NewResponseError("info", "ERR_NOT_FOUND", "VM not found", err.Error(), "Check the VM name and try again.", nil)
				_ = clijson.PrintJSON(resp)
				os.Exit(1)
			}
			ui.Error("Failed to probe %s: %v", rest[0], err)
			os.Exit(1)
		}
		if jsonOut {
			resp := clijson.NewResponseOK("info", map[string]interface{}{
				"vm": map[string]interface{}{
					"name":     info.Name,
					"state":    info.State,
					"ip":       info.IP,
					"ssh_user": info.SSHUser,
					"ssh_port": info.SSHPort,
					"vnc_port": info.VNCPort,
				},
			})
			_ = clijson.PrintJSON(resp)
			return
		}
		ui.Header("VM Profile: " + info.Name)
		ui.FancyLabel("State", info.State)
		ui.FancyLabel("IP Address", info.IP)
		ui.FancyLabel("SSH Command", fmt.Sprintf("ssh -p %d %s@%s", info.SSHPort, info.SSHUser, info.IP))
		if info.VNCPort > 0 {
			ui.FancyLabel("GUI (VNC)", fmt.Sprintf("127.0.0.1:%d", info.VNCPort))
		}

		// Forwarded Ports Table
		// Implements Section 6.1 of advanced-port-forwarding.md.
		if len(info.Forwarding) > 0 {
			fmt.Printf("\n %s%-15s %-10s %-10s %s%s\n", ui.Bold, "LABEL", "GUEST", "HOST", "LINK", ui.Reset)
			fmt.Printf(" %s%s%s\n", ui.Dim, strings.Repeat("-", 60), ui.Reset)
			for _, f := range info.Forwarding {
				label := f.Label
				if label == "" {
					label = "-"
				}
				link := "-"
				if f.Protocol == "tcp" || f.Protocol == "" {
					link = fmt.Sprintf("http://127.0.0.1:%d", f.HostPort)
				}
				fmt.Printf(" %-15s %-10d %-10d %s%s%s\n", label, f.GuestPort, f.HostPort, ui.Dim, link, ui.Reset)
			}
		}
		fmt.Println("")
	case "spawn":
		// The main hatching event. Can spawn from images, templates, or defaults.
		jsonOut, rest := consumeJSONFlag(args)
		if len(rest) < 1 {
			if jsonOut {
				resp := clijson.NewResponseError("spawn", "ERR_INVALID_ARGS", "Missing VM name", "No VM name provided.", "Usage: nido spawn <name> [--image <name:tag> | <template>] [--user-data <file>] [--gui] [--json]", nil)
				_ = clijson.PrintJSON(resp)
				os.Exit(1)
			}
			fmt.Println("Usage: nido spawn <name> [--image <name:tag> | <template>]")
			os.Exit(1)
		}
		name := rest[0]
		if strings.HasPrefix(name, "-") {
			if jsonOut {
				resp := clijson.NewResponseError("spawn", "ERR_INVALID_ARGS", "Missing VM name", "First argument must be a VM name, not a flag.", "Usage: nido spawn <name> ...", nil)
				_ = clijson.PrintJSON(resp)
				os.Exit(1)
			}
			ui.Error("Invalid VM name '%s'. It looks like a flag.", name)
			fmt.Println("Usage: nido spawn <name> [--image <name:tag> | <template>]")
			os.Exit(1)
		}
		// Strict Name Validation: no spaces, only alphanumeric, hyphen, underscore, dot.
		for _, r := range name {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.') {
				if jsonOut {
					resp := clijson.NewResponseError("spawn", "ERR_INVALID_ARGS", "Invalid VM name", "Only alphanumeric, hyphens, underscores, and dots allowed.", "Usage: nido spawn <name> ...", nil)
					_ = clijson.PrintJSON(resp)
					os.Exit(1)
				}
				ui.Error("Invalid VM name '%s'. (No spaces allowed, only alphanumeric, -, _, .)", name)
				os.Exit(1)
			}
		}
		tpl := ""
		imageTag := ""
		userDataPath := ""
		gui := false
		var forwardings []provider.PortForward

		for i := 1; i < len(rest); i++ {
			arg := rest[i]
			if (arg == "--image") && i+1 < len(rest) {
				imageTag = rest[i+1]
				i++
			} else if (arg == "--user-data") && i+1 < len(rest) {
				userDataPath = rest[i+1]
				i++
			} else if arg == "--gui" {
				gui = true
			} else if (arg == "--port" || arg == "-p") && i+1 < len(rest) {
				pf, err := parsePortFlag(rest[i+1])
				if err != nil {
					ui.Error("Invalid port mapping: %v", err)
					os.Exit(1)
				}
				forwardings = append(forwardings, pf)
				i++
			} else if arg == "--web" {
				forwardings = append(forwardings, provider.PortForward{Label: "HTTP", GuestPort: 80, Protocol: "tcp"})
				forwardings = append(forwardings, provider.PortForward{Label: "HTTPS", GuestPort: 443, Protocol: "tcp"})
			} else if arg == "--ftp" {
				forwardings = append(forwardings, provider.PortForward{Label: "FTP", GuestPort: 21, Protocol: "tcp"})
			} else if tpl == "" && !strings.HasPrefix(arg, "-") {
				tpl = arg
			}
		}

		customSshUser := ""
		if imageTag != "" {
			// Resolve image
			imgDir := filepath.Join(nidoDir, "images")

			// For development: prefer local registry/images.json if it exists in CWD
			var catalog *image.Catalog
			localRegistry := filepath.Join(cwd, "registry", "images.json")
			if _, err := os.Stat(localRegistry); err == nil {
				catalog, err = image.LoadCatalogFromFile(localRegistry)
				if err != nil {
					if jsonOut {
						resp := clijson.NewResponseError("spawn", "ERR_IO", "Registry load failed", err.Error(), "Check your local registry file and try again.", nil)
						_ = clijson.PrintJSON(resp)
						os.Exit(1)
					}
					ui.Error("Failed to load local registry: %v", err)
					os.Exit(1)
				}
			} else {
				catalog, err = image.LoadCatalog(imgDir, image.DefaultCacheTTL)
				if err != nil {
					if jsonOut {
						resp := clijson.NewResponseError("spawn", "ERR_IO", "Catalog load failed", err.Error(), "Check your network connection and try again.", nil)
						_ = clijson.PrintJSON(resp)
						os.Exit(1)
					}
					ui.Error("Failed to load catalog: %v", err)
					os.Exit(1)
				}
			}

			pName, pVer := imageTag, ""
			if strings.Contains(imageTag, ":") {
				parts := strings.Split(imageTag, ":")
				pName, pVer = parts[0], parts[1]
			}

			img, ver, err := catalog.FindImage(pName, pVer)
			if err != nil {
				if jsonOut {
					resp := clijson.NewResponseError("spawn", "ERR_NOT_FOUND", "Image not found", err.Error(), "Run 'nido image list' to see available images.", nil)
					_ = clijson.PrintJSON(resp)
					os.Exit(1)
				}
				ui.Error("Image %s not found in catalog.", imageTag)
				os.Exit(1)
			}
			customSshUser = img.SSHUser

			imgPath := filepath.Join(imgDir, fmt.Sprintf("%s-%s.qcow2", img.Name, ver.Version))

			// Auto-pull if the species is not yet in our local cache
			if _, err := os.Stat(imgPath); os.IsNotExist(err) {
				if !jsonOut {
					ui.Info("Image not found locally. Pulling %s:%s...", img.Name, ver.Version)
				}
				downloader := image.Downloader{Quiet: jsonOut}

				// Check if the URL points to a compressed file
				downloadPath := imgPath
				isCompressed := strings.HasSuffix(ver.URL, ".tar.xz")
				if isCompressed {
					downloadPath = imgPath + ".tar.xz"
				}

				var downloadErr error
				if len(ver.PartURLs) > 0 {
					downloadErr = downloader.DownloadMultiPart(ver.PartURLs, downloadPath, ver.SizeBytes)
				} else {
					downloadErr = downloader.Download(ver.URL, downloadPath, ver.SizeBytes)
				}

				if downloadErr != nil {
					if jsonOut {
						resp := clijson.NewResponseError("spawn", "ERR_IO", "Download failed", downloadErr.Error(), "Check your network connection and try again.", nil)
						_ = clijson.PrintJSON(resp)
						os.Exit(1)
					}
					ui.Error("Download failed: %v", downloadErr)
					os.Exit(1)
				}

				// Decompress first if it's a tarball
				if isCompressed {
					// We verify the archive integrity first
					if !jsonOut {
						ui.Ironic("Verifying genetic integrity (archive)...")
					}
					if err := image.VerifyChecksum(downloadPath, ver.Checksum, ver.ChecksumType); err != nil {
						if jsonOut {
							resp := clijson.NewResponseError("spawn", "ERR_IO", "Verification failed", err.Error(), "Retry the download or choose a different image.", nil)
							_ = clijson.PrintJSON(resp)
						} else {
							ui.Error("Verification failed: %v", err)
						}
						os.Remove(downloadPath)
						os.Exit(1)
					}

					if err := downloader.Decompress(downloadPath, imgPath); err != nil {
						if jsonOut {
							resp := clijson.NewResponseError("spawn", "ERR_IO", "Decompression failed", err.Error(), "Retry the download or choose a different image.", nil)
							_ = clijson.PrintJSON(resp)
						} else {
							ui.Error("Decompression failed: %v", err)
						}
						os.Remove(downloadPath)
						os.Exit(1)
					}
					// Cleanup the archive
					os.Remove(downloadPath)
					if !jsonOut {
						ui.Success("Image extracted successfully.")
					}
				} else {
					// Standard verify for direct qcow2
					if !jsonOut {
						ui.Ironic("Verifying genetic integrity...")
					}
					if err := image.VerifyChecksum(downloadPath, ver.Checksum, ver.ChecksumType); err != nil {
						if jsonOut {
							resp := clijson.NewResponseError("spawn", "ERR_IO", "Verification failed", err.Error(), "Retry the download or choose a different image.", nil)
							_ = clijson.PrintJSON(resp)
						} else {
							ui.Error("Verification failed: %v", err)
						}
						os.Remove(downloadPath)
						os.Exit(1)
					}
					if !jsonOut {
						ui.Success("Image prepared successfully.")
					}
				}
			}

			// Use absolute path as template
			tpl = imgPath
		}

		if !jsonOut {
			ui.Ironic("Initiating hypervisor handshake...")
		}
		// Dispatch the hatching command to the provider
		if err := prov.Spawn(name, provider.VMOptions{
			DiskPath:     tpl,
			UserDataPath: userDataPath,
			Gui:          gui,
			SSHUser:      customSshUser,
			Forwarding:   forwardings,
		}); err != nil {
			if jsonOut {
				code := "ERR_INTERNAL"
				if isAlreadyExistsErr(err) {
					code = "ERR_ALREADY_EXISTS"
				}
				resp := clijson.NewResponseError("spawn", code, "Spawn failed", err.Error(), "Check the template or image and try again.", nil)
				_ = clijson.PrintJSON(resp)
				os.Exit(1)
			}
			ui.Error("Hatch failure for %s: %v", name, err)
			os.Exit(1)
		}

		source := "template"
		if imageTag != "" {
			source = "image " + imageTag
		}
		if jsonOut {
			resp := clijson.NewResponseOK("spawn", map[string]interface{}{
				"action": map[string]interface{}{
					"name":      name,
					"result":    "spawned",
					"source":    source,
					"gui":       gui,
					"user_data": userDataPath,
					"image_tag": imageTag,
				},
			})
			_ = clijson.PrintJSON(resp)
			return
		}
		ui.Success("VM %s hatched successfully from %s.", name, source)

		// Cleanup if linked clones are disabled (we don't need the cache file)
		if !cfg.LinkedClones && imageTag != "" && tpl != "" {
			if err := os.Remove(tpl); err != nil {
				if !jsonOut {
					ui.Error("Failed to cleanup image cache: %v", err)
				}
			} else if !jsonOut {
				ui.Ironic("Linked Clones disabled: Downloaded image incinerated.")
			}
		}
	case "start":
		jsonOut, rest := consumeJSONFlag(args)
		if len(rest) < 1 {
			if jsonOut {
				resp := clijson.NewResponseError("start", "ERR_INVALID_ARGS", "Missing VM name", "No VM name provided.", "Usage: nido start <name> [--gui] [--json]", nil)
				_ = clijson.PrintJSON(resp)
				os.Exit(1)
			}
			fmt.Println("Usage: nido start <name> [--gui]")
			os.Exit(1)
		}
		name := rest[0]
		gui := false
		for i := 1; i < len(rest); i++ {
			if rest[i] == "--gui" {
				gui = true
			}
		}

		if !jsonOut {
			ui.Ironic("Reviving digital consciousness...")
		}
		if err := prov.Start(name, provider.VMOptions{Gui: gui}); err != nil {
			if jsonOut {
				resp := clijson.NewResponseError("start", "ERR_INTERNAL", "Start failed", err.Error(), "Check the VM state and try again.", nil)
				_ = clijson.PrintJSON(resp)
				os.Exit(1)
			}
			ui.Error("Neural jumpstart failure for %s: %v", name, err)
			os.Exit(1)
		}
		if jsonOut {
			resp := clijson.NewResponseOK("start", map[string]interface{}{
				"action": map[string]interface{}{
					"name":   name,
					"result": "started",
					"gui":    gui,
				},
			})
			_ = clijson.PrintJSON(resp)
			return
		}
		ui.Success("VM %s is now active.", name)
	case "stop":
		jsonOut, rest := consumeJSONFlag(args)
		if len(rest) < 1 {
			if jsonOut {
				resp := clijson.NewResponseError("stop", "ERR_INVALID_ARGS", "Missing VM name", "No VM name provided.", "Usage: nido stop <name> [--json]", nil)
				_ = clijson.PrintJSON(resp)
				os.Exit(1)
			}
			fmt.Println("Usage: nido stop <name>")
			os.Exit(1)
		}
		if !jsonOut {
			ui.Ironic("Sending stop signal to nerves...")
		}
		if err := prov.Stop(rest[0], true); err != nil {
			if jsonOut {
				resp := clijson.NewResponseError("stop", "ERR_INTERNAL", "Stop failed", err.Error(), "Check the VM state and try again.", nil)
				_ = clijson.PrintJSON(resp)
				os.Exit(1)
			}
			ui.Error("Deep sleep rejection for %s: %v", rest[0], err)
			os.Exit(1)
		}
		if jsonOut {
			resp := clijson.NewResponseOK("stop", map[string]interface{}{
				"action": map[string]interface{}{
					"name":   rest[0],
					"result": "stopped",
				},
			})
			_ = clijson.PrintJSON(resp)
			return
		}
		ui.Success("VM %s is now resting.", rest[0])
	case "template":
		// Management of the "Grave": where cold storage images reside.
		jsonOut, rest := consumeJSONFlag(args)
		if len(rest) == 0 || rest[0] == "list" {
			if !jsonOut {
				ui.Header("Nido Templates (Grave)")
			}
			templates, err := prov.ListTemplates()
			if err != nil {
				if jsonOut {
					resp := clijson.NewResponseError("template list", "ERR_IO", "Template list failed", err.Error(), "Check your storage path and try again.", nil)
					_ = clijson.PrintJSON(resp)
					os.Exit(1)
				}
				ui.Error("Failed to access graveyard: %v", err)
				os.Exit(1)
			}
			if jsonOut {
				type tplJSON struct {
					Name      string `json:"name"`
					SizeBytes int64  `json:"size_bytes"`
				}
				items := make([]tplJSON, 0, len(templates))
				for _, name := range templates {
					size := int64(0)
					tplPath := filepath.Join(cfg.BackupDir, name+".compact.qcow2")
					if info, err := os.Stat(tplPath); err == nil {
						size = info.Size()
					}
					items = append(items, tplJSON{
						Name:      name,
						SizeBytes: size,
					})
				}
				resp := clijson.NewResponseOK("template list", map[string]interface{}{
					"templates": items,
				})
				_ = clijson.PrintJSON(resp)
				return
			}
			if len(templates) == 0 {
				ui.Info("The graveyard is empty. No templates found.")
			} else {
				for _, name := range templates {
					size := int64(0)
					tplPath := filepath.Join(cfg.BackupDir, name+".compact.qcow2")
					if info, err := os.Stat(tplPath); err == nil {
						size = info.Size()
					}
					fmt.Printf("  %s%-20s%s %s(%s)%s\n", ui.Cyan, name, ui.Reset, ui.Dim, ui.HumanSize(size), ui.Reset)
				}
			}
			if !jsonOut {
				fmt.Println("")
			}
			return
		}
		subCmd := rest[0]
		if subCmd == "create" {
			if len(rest) < 3 {
				ui.Error("Missing parameters. Usage: nido template create <vm-name> <template-name>")
				os.Exit(1)
			}
			vmName := rest[1]
			tplName := rest[2]
			if !jsonOut {
				ui.Ironic("Encrypting genetic data into template...")
			}
			path, err := prov.CreateTemplate(vmName, tplName)
			if err != nil {
				if jsonOut {
					resp := clijson.NewResponseError("template create", "ERR_IO", "Template create failed", err.Error(), "Check VM name and storage permissions.", nil)
					_ = clijson.PrintJSON(resp)
					os.Exit(1)
				}
				ui.Error("Cloning evolution failed: %v", err)
				os.Exit(1)
			}
			if jsonOut {
				resp := clijson.NewResponseOK("template create", map[string]interface{}{
					"action": map[string]interface{}{
						"name": tplName,
						"path": path,
					},
				})
				_ = clijson.PrintJSON(resp)
				return
			}
			ui.Success("New species '"+tplName+"' archived at: %s", path)
		} else if subCmd == "delete" {
			if len(rest) < 2 {
				ui.Error("Missing parameters. Usage: nido template delete <template-name> [--force]")
				os.Exit(1)
			}
			tplName := rest[1]
			force := false
			if len(rest) > 2 && rest[2] == "--force" {
				force = true
			}

			if !jsonOut {
				if force {
					ui.Ironic("Vaporizing genetic archive (forced)...")
				} else {
					ui.Ironic("Vaporizing genetic archive...")
				}
			}
			if err := prov.DeleteTemplate(tplName, force); err != nil {
				if isNotFoundErr(err) {
					if jsonOut {
						resp := clijson.NewResponseOK("template delete", map[string]interface{}{
							"action": map[string]interface{}{
								"name":   tplName,
								"result": "not_found",
							},
						})
						_ = clijson.PrintJSON(resp)
						return
					}
					ui.Info("Template '%s' is already gone.", tplName)
					return
				}
				if jsonOut {
					resp := clijson.NewResponseError("template delete", "ERR_IO", "Template delete failed", err.Error(), "Check the template name and try again.", nil)
					_ = clijson.PrintJSON(resp)
					os.Exit(1)
				}
				ui.Error("Failed to destroy template: %v", err)
				os.Exit(1)
			}
			if jsonOut {
				resp := clijson.NewResponseOK("template delete", map[string]interface{}{
					"action": map[string]interface{}{
						"name":   tplName,
						"result": "deleted",
					},
				})
				_ = clijson.PrintJSON(resp)
				return
			}
			ui.Success("Template '%s' has been eradicated.", tplName)
		} else {
			ui.Error("Unknown template action: %s", subCmd)
		}
	case "config":
		// View or modify the genetic code of your Nido environment.
		jsonOut, rest := consumeJSONFlag(args)
		cmdConfig(cfg, cfgPath, jsonOut, rest)
	case "register":
		jsonOut, _ := consumeJSONFlag(args)
		cmdRegister(jsonOut)
	case "cache":
		cmdCache(nidoDir, args, prov)
	case "image", "images":
		cmdImage(nidoDir, args)
	case "delete", "destroy":
		jsonOut, rest := consumeJSONFlag(args)
		if len(rest) < 1 {
			if jsonOut {
				resp := clijson.NewResponseError("delete", "ERR_INVALID_ARGS", "Missing VM name", "No VM name provided.", "Usage: nido delete <name> [--json]", nil)
				_ = clijson.PrintJSON(resp)
				os.Exit(1)
			}
			fmt.Println("Usage: nido delete <name>")
			os.Exit(1)
		}
		if !jsonOut {
			ui.Ironic("Vaporizing digital footprint...")
		}
		if err := prov.Delete(rest[0]); err != nil {
			if isNotFoundErr(err) {
				if jsonOut {
					resp := clijson.NewResponseOK("delete", map[string]interface{}{
						"action": map[string]interface{}{
							"name":   rest[0],
							"result": "not_found",
						},
					})
					_ = clijson.PrintJSON(resp)
					return
				}
				ui.Info("VM %s is already gone.", rest[0])
				return
			}
			if jsonOut {
				resp := clijson.NewResponseError("delete", "ERR_INTERNAL", "Delete failed", err.Error(), "Check the VM name and try again.", nil)
				_ = clijson.PrintJSON(resp)
				os.Exit(1)
			}
			ui.Error("Eviction failed: %v", err)
			os.Exit(1)
		}
		if jsonOut {
			resp := clijson.NewResponseOK("delete", map[string]interface{}{
				"action": map[string]interface{}{
					"name":   rest[0],
					"result": "deleted",
				},
			})
			_ = clijson.PrintJSON(resp)
			return
		}
		ui.Success("VM %s has left the nest (permanently).", rest[0])
	case "prune":
		jsonOut, _ := consumeJSONFlag(args)
		if !jsonOut {
			ui.Ironic("Cleaning up the graveyard...")
		}
		count, err := prov.Prune()
		if err != nil {
			if jsonOut {
				resp := clijson.NewResponseError("prune", "ERR_INTERNAL", "Prune failed", err.Error(), "Try again or run nido doctor.", nil)
				_ = clijson.PrintJSON(resp)
				os.Exit(1)
			}
			ui.Error("Pruning failed: %v", err)
			os.Exit(1)
		}
		if jsonOut {
			resp := clijson.NewResponseOK("prune", map[string]interface{}{
				"removed_count": count,
			})
			_ = clijson.PrintJSON(resp)
			return
		}
		ui.Success("Vaporized %d stopped life forms.", count)
	case "uninstall":
		// Nuclear option: Remove everything.
		jsonOut, rest := consumeJSONFlag(args)
		force := false
		for _, arg := range rest {
			if arg == "--force" {
				force = true
			}
		}

		if !force {
			if jsonOut {
				resp := clijson.NewResponseError("uninstall", "ERR_CONFIRMATION_REQUIRED", "Confirmation required", "Use --force to skip prompt.", "Usage: nido uninstall --force", nil)
				_ = clijson.PrintJSON(resp)
				os.Exit(1)
			}
			ui.Warn("⚠️  DANGER ZONE ⚠️")
			ui.Warn("This will PERMANENTLY DELETE:")
			fmt.Printf("  - Configuration & Data: %s\n", nidoDir)
			fmt.Printf("  - Local Templates:      %s\n", filepath.Join(nidoDir, "templates"))
			fmt.Printf("  - Desktop Entries:      Launcher / Start Menu / Applications\n")
			exe, _ := os.Executable()
			fmt.Printf("  - Nido Binary:          %s\n", exe)
			fmt.Println("")
			fmt.Print("Are you sure you want to proceed? [y/N]: ")
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) != "y" {
				ui.Info("Aborted. The nest remains safe.")
				os.Exit(0)
			}
		}

		exe, err := os.Executable()
		if err != nil {
			ui.Error("Failed to locate self: %v", err)
			os.Exit(1)
		}
		// In dev/go run, executable might be in tmp, which is fine to delete or ignore.

		if !jsonOut {
			ui.Ironic("Initiating self-destruct sequence...")
		}

		if err := lifecycle.Uninstall(nidoDir, exe); err != nil {
			if jsonOut {
				resp := clijson.NewResponseError("uninstall", "ERR_INTERNAL", "Uninstall failed", err.Error(), "Check permissions and try again.", nil)
				_ = clijson.PrintJSON(resp)
				os.Exit(1)
			}
			ui.Error("Self-destruct failed: %v", err)
			os.Exit(1)
		}

		if jsonOut {
			resp := clijson.NewResponseOK("uninstall", map[string]interface{}{
				"result": "uninstalled",
			})
			_ = clijson.PrintJSON(resp)
			return
		}
		ui.Success("Nido has been uninstalled. Farewell, Pilot.")
	case "mcp":
		// Enter MCP mode: speak directly to AI agents via machine language.
		server := mcp.NewServer(prov)
		server.Serve()
	case "completion":
		// Generate shell artifacts to help humans type faster.
		if len(args) < 1 {
			fmt.Println("Usage: nido completion <bash|zsh>")
			os.Exit(1)
		}
		sub := args[0]
		if sub == "list-vms" {
			vms, _ := prov.List()
			for _, vm := range vms {
				fmt.Printf("%s ", vm.Name)
			}
			return
		}
		if sub == "list-templates" {
			templates, _ := prov.ListTemplates()
			for _, name := range templates {
				fmt.Printf("%s ", name)
			}
			return
		}
		if sub == "list-images" {
			cachePath := filepath.Join(nidoDir, "images", ".catalog.json")
			catalog, err := image.LoadCatalogFromFile(cachePath)
			if err == nil {
				for _, img := range catalog.Images {
					for _, ver := range img.Versions {
						fmt.Printf("%s:%s ", img.Name, ver.Version)
						for _, alias := range ver.Aliases {
							fmt.Printf("%s:%s ", img.Name, alias)
						}
					}
				}
			}
			return
		}
		cmdCompletion(sub)
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}

	// Note: errors are handled within each case
}

func cmdCompletion(bashOrZsh string) {
	if bashOrZsh == "bash" {
		fmt.Print(getBashCompletion())
	} else if bashOrZsh == "zsh" {
		fmt.Print(getZshCompletion())
	} else {
		ui.Error("Unsupported shell: %s. Only bash and zsh are supported.", bashOrZsh)
		os.Exit(1)
	}
}

// cmdMcpHelp emits the MCP tool catalog as JSON for clients.
func cmdMcpHelp() {
	tools := mcp.ToolsCatalog()
	resp := clijson.NewResponseOK("mcp-help", map[string]interface{}{
		"tools": tools,
	})
	_ = clijson.PrintJSON(resp)
}

func getBashCompletion() string {
	return `_nido_completions() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    opts="ls spawn start ssh info stop images cache template config register version delete doctor prune completion update uninstall help"

    case "${prev}" in
        spawn)
            # 1st arg after spawn is Name. We don't have specific completion for a new name,
            # but we suggest flags if they start typing one.
            COMPREPLY=( $(compgen -W "--image --user-data --port --web --ftp --gui --json" -- ${cur}) )
            return 0
            ;;
        start)
            COMPREPLY=( $(compgen -W "$(nido completion list-vms) --gui --json" -- ${cur}) )
            return 0
            ;;
        ssh)
            COMPREPLY=( $(compgen -W "$(nido completion list-vms)" -- ${cur}) )
            return 0
            ;;
        ls|list|version|prune|doctor|register)
            COMPREPLY=( $(compgen -W "--json" -- ${cur}) )
            return 0
            ;;
        config)
            COMPREPLY=( $(compgen -W "set --json" -- ${cur}) )
            return 0
            ;;
        set)
            if [[ ${COMP_WORDS[COMP_CWORD-2]} == "config" ]]; then
                COMPREPLY=( $(compgen -W "BACKUP_DIR TEMPLATE_DEFAULT SSH_USER IMAGE_DIR LINKED_CLONES" -- ${cur}) )
                return 0
            fi
            ;;
        LINKED_CLONES)
            COMPREPLY=( $(compgen -W "true false" -- ${cur}) )
            return 0
            ;;
        template)
            COMPREPLY=( $(compgen -W "create list delete --json" -- ${cur}) )
            return 0
            ;;
        images)
            COMPREPLY=( $(compgen -W "ls list pull update info remove --json" -- ${cur}) )
            return 0
            ;;
        pull|info|remove)
            if [[ ${COMP_WORDS[COMP_CWORD-2]} == "images" ]]; then
                COMPREPLY=( $(compgen -W "$(nido completion list-images)" -- ${cur}) )
                return 0
            fi
            ;;
        cache)
            COMPREPLY=( $(compgen -W "ls list info rm remove prune --unused --json" -- ${cur}) )
            return 0
            ;;
        --image)
            COMPREPLY=( $(compgen -W "$(nido completion list-images)" -- ${cur}) )
            return 0
            ;;
        --user-data)
            # File completion for these flags
            COMPREPLY=( $(compgen -f -- ${cur}) )
            return 0
            ;;
        create)
            if [[ ${COMP_WORDS[COMP_CWORD-2]} == "template" ]]; then
                COMPREPLY=( $(compgen -W "$(nido completion list-vms)" -- ${cur}) )
                return 0
            fi
            ;;
        *)
            # Position-based completion for 'nido spawn <name> <template>'
            if [[ ${COMP_WORDS[1]} == "spawn" ]]; then
                if [[ ${COMP_CWORD} -eq 2 ]]; then
                    # Position 1: VM Name (suggest flags if starting with -)
                    COMPREPLY=( $(compgen -W "--image --user-data --port --web --ftp --gui --json" -- ${cur}) )
                    return 0
                elif [[ ${COMP_CWORD} -eq 3 ]] && [[ ! ${prev} == -* ]]; then
                    # Position 2: Template (if previous wasn't a flag needing a value)
                    COMPREPLY=( $(compgen -W "$(nido completion list-templates) --image --user-data --port --web --ftp --gui --json" -- ${cur}) )
                    return 0
                fi
            fi

            # Existing logic for other commands
            if [[ ${COMP_CWORD} -eq 2 ]]; then
                COMPREPLY=( $(compgen -W "$(nido completion list-vms) --json" -- ${cur}) )
                return 0
            elif [[ ${COMP_CWORD} -eq 3 ]] && [[ ${COMP_WORDS[COMP_CWORD-2]} == "template" ]] && [[ ${COMP_WORDS[COMP_CWORD-1]} == "delete" ]]; then
                COMPREPLY=( $(compgen -W "$(nido completion list-templates) --force" -- ${cur}) )
                return 0
            fi
            # Fallback for flags and deeper levels
            if [[ ${COMP_WORDS[COMP_CWORD-2]} == "delete" ]] && [[ ${COMP_WORDS[COMP_CWORD-3]} == "template" ]]; then
                COMPREPLY=( $(compgen -W "--force" -- ${cur}) )
                return 0
            fi
            ;;
    esac

    if [[ ${COMP_CWORD} -eq 1 ]] ; then
        COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
        return 0
    fi
}
complete -F _nido_completions nido
`
}

func getZshCompletion() string {
	return `_nido_completion() {
  local -a commands
  commands=(
    'spawn:Create and hatch a new VM'
    'ssh:Connect to a VM via SSH bridge'
    'ls:List all life forms in the nest'
    'start:Revive VM from deep sleep'
    'stop:Put VM into deep sleep'
    'images:Manage cloud image catalog'
    'cache:Manage cached cloud images'
    'delete:Evict VM from the nest'
    'prune:Vaporize all stopped life forms'
    'template:Manage cold templates'
    'config:Modify genetic configuration'
    'register:Prepare MCP handshake'
    'version:Check evolutionary state'
    'doctor:Run a system health check'
    'completion:Generate shell completions'
    'update:Ascend to the latest evolutionary state'
    'uninstall:Nuclear option: Remove Nido and all its data'
  )

  _arguments -C \
    '1: :->command' \
    '*:: :->args'

  case $state in
    command)
      _describe -t commands 'nido command' commands
      ;;
    args)
      case $words[1] in
        spawn)
          _arguments \
            '1:name: ' \
            '2:template:$(nido completion list-templates)' \
            '--image[Cloud image to use]:image:($(nido completion list-images))' \
            '--user-data[Cloud-init user-data file]:file:_files' \
            '--port[Port mapping (LABEL:GUEST:HOST/PROTO)]:mapping' \
            '--web[Shortcut for HTTP/HTTPS]' \
            '--ftp[Shortcut for FTP]' \
            '--gui[Enable GUI (VNC)]' \
            '--json[Structured JSON output]'
          ;;
        start)
          _arguments \
            '1:vm:$(nido completion list-vms)' \
            '--gui[Enable GUI (VNC)]' \
            '--json[Structured JSON output]'
          ;;
        ssh)
          _values 'vms' $(nido completion list-vms)
          ;;
        info|stop|delete)
          _arguments \
            '1:vm:$(nido completion list-vms)' \
            '--json[Structured JSON output]'
          ;;
        ls|version|prune|doctor|register)
          _arguments \
            '--json[Structured JSON output]'
          ;;
        config)
          if (( CURRENT == 2 )); then
            _values 'actions' 'set' '--json'
          elif (( CURRENT == 3 )) && [[ $words[2] == "set" ]]; then
            _values 'keys' 'BACKUP_DIR' 'TEMPLATE_DEFAULT' 'SSH_USER' 'IMAGE_DIR' 'LINKED_CLONES'
          elif (( CURRENT == 4 )) && [[ $words[2] == "set" ]]; then
             case $words[3] in
                LINKED_CLONES) _values 'bool' 'true' 'false' ;;
                *) ;;
             esac
          fi
          ;;
        template)
          if (( CURRENT == 2 )); then
            _values 'actions' 'create' 'list' 'delete' '--json'
          elif (( CURRENT == 3 )) && [[ $words[2] == "create" ]]; then
            _values 'vms' $(nido completion list-vms)
          elif (( CURRENT == 3 )) && [[ $words[2] == "delete" ]]; then
            _values 'templates' $(nido completion list-templates)
          elif (( CURRENT == 4 )) && [[ $words[2] == "delete" ]]; then
            _values 'options' '--force'
          fi
          ;;
        images)
          if (( CURRENT == 2 )); then
            _values 'actions' 'list' 'ls' 'pull' 'update' 'info' 'remove' '--json'
          elif (( CURRENT == 3 )) && [[ $words[2] =~ ^(pull|update|info|remove)$ ]]; then
            _values 'images' $(nido completion list-images)
          fi
          ;;
        cache)
          if (( CURRENT == 2 )); then
            _values 'actions' 'ls' 'list' 'info' 'rm' 'remove' 'prune' '--json'
          fi
          ;;
      esac
      ;;
  esac
}
compdef _nido_completion nido
`
}

func cmdSsh(prov provider.VMProvider, name string, args []string) {
	cmdStr, err := prov.SSHCommand(name)
	if err != nil {
		ui.Error("Target acquisition failed: %v", err)
		os.Exit(1)
	}
	ui.Info("Bridging to %s...", name)
	if len(args) == 0 {
		ui.Ironic("Establishing secure neural link...")
	}

	parts := strings.Split(cmdStr, " ")
	// ssh -p [port] [user]@[ip] [args...]
	// Inject options to skip fingerprint check for ephemeral VMs
	sshOptions := []string{
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR", // Reduce noise
		"-o", "BatchMode=yes", // Fail instead of prompting for password
	}

	baseArgs := append(sshOptions, parts[1:]...)
	finalArgs := append(baseArgs, args...)

	cmd := exec.Command(parts[0], finalArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// Pass through exit code if possible
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		os.Exit(1)
	}
}

func cmdDoctor(prov provider.VMProvider, jsonOut bool) {
	reports := prov.Doctor()
	if jsonOut {
		failCount := 0
		for _, r := range reports {
			if strings.Contains(r, "[FAIL]") {
				failCount++
			}
		}
		resp := clijson.NewResponseOK("doctor", map[string]interface{}{
			"reports": reports,
			"summary": map[string]interface{}{
				"total":  len(reports),
				"failed": failCount,
				"passed": len(reports) - failCount,
			},
		})
		_ = clijson.PrintJSON(resp)
		return
	}

	ui.Header("Nido System Diagnostics")
	for _, r := range reports {
		// Parse report for UI coloring
		icon := ui.IconSuccess
		if strings.Contains(r, "[FAIL]") {
			icon = ui.IconError
		}
		fmt.Printf("  %s %s\n", icon, r)
	}
	fmt.Println("")
	ui.Success("Audit complete. The nest is structurally sound.")
}

func printUsage() {
	ui.Header("Nido: The Universal Nest")
	fmt.Printf("Usage: %snido %s<command>%s [args...]\n\n", ui.Bold, ui.Cyan, ui.Reset)
	fmt.Printf("%sOutput:%s add --json for structured output on supported commands (ls, info, spawn, start, stop, delete, prune, template, images, cache, version, doctor, config, register).\n\n", ui.Dim, ui.Reset)

	fmt.Printf("%sVM MANAGEMENT%s\n", ui.Bold, ui.Reset)
	fmt.Printf("  %-10s %sCreate and hatch a new VM%s\n", "spawn", ui.Dim, ui.Reset)
	fmt.Printf("    %sFlags: --image <tag>, --user-data <file>, --port <mapping>, --web, --ftp, --gui, --json%s\n", ui.Dim, ui.Reset)
	fmt.Printf("  %-10s %sRevive a VM from deep sleep%s\n", "start", ui.Dim, ui.Reset)
	fmt.Printf("    %sFlags: --gui, --json%s\n", ui.Dim, ui.Reset)
	fmt.Printf("  %-10s %sConnect to a VM via SSH bridge%s\n", "ssh", ui.Dim, ui.Reset)
	fmt.Printf("  %-10s %sList all life forms currently in the nest%s\n", "ls", ui.Dim, ui.Reset)
	fmt.Printf("  %-10s %sInspect a specific VM's neural links%s\n", "info", ui.Dim, ui.Reset)
	fmt.Printf("  %-10s %sPut a VM into deep sleep (graceful stop)%s\n", "stop", ui.Dim, ui.Reset)
	fmt.Printf("  %-10s %sEvict a VM from the nest permanently%s\n", "delete", ui.Dim, ui.Reset)
	fmt.Printf("  %-10s %sVaporize all stopped VMs%s\n", "prune", ui.Dim, ui.Reset)

	fmt.Printf("\n%sSTORAGE & GENETICS%s\n", ui.Bold, ui.Reset)
	fmt.Printf("  %-10s %sBrowse and pull cloud images%s\n", "images", ui.Dim, ui.Reset)
	fmt.Printf("  %-10s %sManage cached cloud images (ls, info, rm, prune)%s\n", "cache", ui.Dim, ui.Reset)
	fmt.Printf("  %-10s %sArchive a VM into a cold, compressed template%s\n", "template", ui.Dim, ui.Reset)

	fmt.Printf("\n%sSYSTEM OPS%s\n", ui.Bold, ui.Reset)
	fmt.Printf("  %-10s %sRun a system health check%s\n", "doctor", ui.Dim, ui.Reset)
	fmt.Printf("  %-10s %sDump current genetic configuration%s\n", "config", ui.Dim, ui.Reset)
	fmt.Printf("  %-10s %sPrepare the MCP handshake for AI agents%s\n", "register", ui.Dim, ui.Reset)
	fmt.Printf("  %-10s %sCheck the evolutionary state of Nido%s\n", "version", ui.Dim, ui.Reset)
	fmt.Printf("  %-10s %sGenerate shell completion scripts%s\n", "completion", ui.Dim, ui.Reset)
	fmt.Printf("  %-10s %sAscend to the latest evolutionary state (Update)%s\n", "update", ui.Dim, ui.Reset)
	fmt.Printf("  %-10s %sNuclear option: Remove Nido and all its data%s\n", "uninstall", ui.Dim, ui.Reset)

	fmt.Printf("\n%s\"It's not a VM, it's a lifestyle.\"%s\n\n", ui.Dim, ui.Reset)
}

func cmdConfig(cfg *config.Config, path string, jsonOut bool, args []string) {
	if len(args) > 0 {
		if args[0] == "set" {
			if len(args) < 3 {
				ui.Error("Usage: nido config set <key> <value>")
				os.Exit(1)
			}
			key := strings.ToUpper(args[1])
			val := args[2]

			// Validate keys
			validKeys := map[string]bool{
				"BACKUP_DIR": true, "TEMPLATE_DEFAULT": true, "SSH_USER": true, "IMAGE_DIR": true, "LINKED_CLONES": true,
			}
			if !validKeys[key] {
				ui.Error("Invalid config key: %s", key)
				os.Exit(1)
			}

			if err := config.UpdateConfig(path, key, val); err != nil {
				if jsonOut {
					resp := clijson.NewResponseError("config set", "ERR_IO", "Update failed", err.Error(), "Check permissions.", nil)
					_ = clijson.PrintJSON(resp)
					os.Exit(1)
				}
				ui.Error("Failed to update config: %v", err)
				os.Exit(1)
			}

			if jsonOut {
				resp := clijson.NewResponseOK("config set", map[string]interface{}{
					"action": "set",
					"key":    key,
					"value":  val,
				})
				_ = clijson.PrintJSON(resp)
				return
			}
			ui.Success("Updated %s = %s", key, val)
			return
		}
	}

	if jsonOut {
		resp := clijson.NewResponseOK("config", map[string]interface{}{
			"config_path":   path,
			"backup_dir":    cfg.BackupDir,
			"default_tpl":   cfg.TemplateDefault,
			"ssh_user":      cfg.SSHUser,
			"linked_clones": cfg.LinkedClones,
		})
		_ = clijson.PrintJSON(resp)
		return
	}

	ui.Header("Nido Genetic Configuration")
	ui.FancyLabel("Config Path", path)
	// fmt.Println("") // Removed based on user feedback
	ui.FancyLabel("Backup Dir", cfg.BackupDir)
	ui.FancyLabel("Default Tpl", cfg.TemplateDefault)
	ui.FancyLabel("SSH User", cfg.SSHUser)
	cloneStatus := "Enabled (Space Saving)"
	if !cfg.LinkedClones {
		cloneStatus = "Disabled (Full Copy)"
	}
	ui.FancyLabel("Linked Clones", cloneStatus)
}

func cmdRegister(jsonOut bool) {
	exe, _ := os.Executable()
	if jsonOut {
		resp := clijson.NewResponseOK("register", map[string]interface{}{
			"mcpServers": map[string]interface{}{
				"nido-local-vm-manager": map[string]interface{}{
					"command": exe,
					"args":    []string{"mcp"},
				},
			},
		})
		_ = clijson.PrintJSON(resp)
		return
	}

	ui.Header("MCP Handshake Protocol")
	ui.Info("To authorize your AI Agent (Claude, Antigravity) to access the Nest,")
	ui.Info("copy the following JSON blob into your agent's configuration file:")
	fmt.Println("")
	fmt.Printf("%s%s%s%s\n", ui.Dim, ui.Bold, `{
  "mcpServers": {
    "nido-local-vm-manager": {
      "command": "`+exe+`",
      "args": ["mcp"]
    }
  }
}`, ui.Reset)
	fmt.Println("")
}

func cmdMcpJsonList(prov provider.VMProvider) {
	vms, _ := prov.List()
	if vms == nil {
		vms = []provider.VMStatus{}
	}
	data, _ := json.Marshal(vms)
	fmt.Println(string(data))
}

func cmdVersion(jsonOut bool) {
	if jsonOut {
		resp := clijson.NewResponseOK("version", map[string]interface{}{
			"version":  build.Version,
			"state":    "Evolved",
			"protocol": "v3.0",
		})
		_ = clijson.PrintJSON(resp)
		return
	}

	fmt.Printf("%sNido %s%s%s (State: Evolved)\n", ui.Dim, ui.Reset, ui.Bold, build.Version)
	ui.Ironic("Hypervisor handshake protocol v3.0 stable.")

	go func() {
		latest, err := build.GetLatestVersion()
		if err == nil && latest != "" && latest != build.Version {
			fmt.Printf("\n%s✨ A new evolutionary state is available: %s%s (current: %s)\n", ui.Yellow, ui.Bold, latest, build.Version)
			ui.Info("Run 'nido update' to ascend to the next level.")
		}
	}()
	// Give a tiny bit of time for the goroutine to potentially output,
	// but don't block. For cmdVersion, we can wait 100ms.
	time.Sleep(100 * time.Millisecond)
}

func consumeJSONFlag(args []string) (bool, []string) {
	jsonOut := false
	rest := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--json" {
			jsonOut = true
			clijson.SetJSONMode(true)
			continue
		}
		rest = append(rest, arg)
	}
	return jsonOut, rest
}

func isNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	if os.IsNotExist(err) {
		return true
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "not found") || strings.Contains(lower, "no such file")
}

func isAlreadyExistsErr(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "already exists") || strings.Contains(lower, "file exists")
}

// getLatestVersion removed, moved to build package

func cmdUpdate(nidoDir string) {
	ui.Header("Nido Evolutionary Ascent")
	ui.Ironic("Scanning the horizon for newer genetic sequences...")

	latest, err := build.GetLatestVersion()
	if err != nil {
		ui.Error("Failed to reach the mother nest: %v", err)
		os.Exit(1)
	}

	if latest == build.Version {
		ui.Success("You are already at the peak of evolution (%s).", build.Version)
		return
	}

	ui.Info("Found new version: %s (current: %s)", latest, build.Version)
	ui.Ironic("Downloading new binary from the cloud...")

	// Determine binary name based on platform
	binaryName := fmt.Sprintf("nido-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	// Fetch release assets to find the URL
	resp, err := http.Get("https://api.github.com/repos/Josepavese/nido/releases/latest")
	if err != nil {
		ui.Error("Failed to fetch release details: %v", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var release struct {
		Assets []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	json.NewDecoder(resp.Body).Decode(&release)

	downloadURL := ""
	for _, asset := range release.Assets {
		if asset.Name == binaryName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		ui.Error("Binary %s not found in latest release assets.", binaryName)
		ui.Info("You might need to build it from source or wait for the release to finalize.")
		os.Exit(1)
	}

	// Get current executable path
	exePath, err := os.Executable()
	if err != nil {
		ui.Error("Failed to locate current binary: %v", err)
		os.Exit(1)
	}

	// Check if we have write permissions to the binary
	// We do a simple check by trying to open it for appending (without writing)
	// or checking file mode/ownership if we were more pedantic.
	// A practical test is to try to touch it or open RDWR.
	f, err := os.OpenFile(exePath, os.O_RDWR, 0666)
	if err != nil {
		if os.IsPermission(err) {
			ui.Error("Permission denied: Cannot write to %s", exePath)
			ui.Info("Please run the upgrade command with sudo:")
			fmt.Printf("  sudo nido upgrade\n")
			os.Exit(1)
		}
		// Logic continues if other error, though it will likely fail later
	} else {
		f.Close()
	}

	// Download to temp file
	tmpPath := exePath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		ui.Error("Failed to create temporary file: %v", err)
		os.Exit(1)
	}
	defer os.Remove(tmpPath)

	resp, err = http.Get(downloadURL)
	if err != nil {
		ui.Error("Download failed: %v", err)
		out.Close()
		os.Exit(1)
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		ui.Error("Failed to save binary: %v", err)
		os.Exit(1)
	}

	// Set permissions
	os.Chmod(tmpPath, 0755)

	// Atomic replace (on Linux/Unix)
	// On Windows, you can't replace a running binary easily, but we'll try rename dance
	bakPath := exePath + ".bak"
	os.Remove(bakPath)
	if err := os.Rename(exePath, bakPath); err != nil {
		ui.Error("Migration failed (failed to backup current binary): %v", err)
		os.Exit(1)
	}

	if err := os.Rename(tmpPath, exePath); err != nil {
		ui.Error("Migration failed (failed to install new binary): %v", err)
		// Try to restore backup
		os.Rename(bakPath, exePath)
		os.Exit(1)
	}

	ui.Success("Nido has successfully evolved to %s! 🕊️", latest)
	ui.Ironic("Updating shell completion scripts...")

	// Re-run completion generation to ensure latest aliases are present
	// We'll detect the shell from env if possible, or just generate both for the config dir
	home, _ := os.UserHomeDir()
	nidoHome := filepath.Join(home, ".nido")

	bashPath := filepath.Join(nidoHome, "bash_completion")
	zshPath := filepath.Join(nidoHome, "zsh_completion")

	// Save them as strings for convenience
	os.WriteFile(bashPath, []byte(getBashCompletion()), 0644)
	os.WriteFile(zshPath, []byte(getZshCompletion()), 0644)

	ui.Info("Shell completions updated in %s.", nidoHome)
	ui.Info("Please restart your terminal or source your config to see changes.")
}

// cmdCache handles the transparent management of downloaded cloud images.
// It ensures Smart Cache Protection is active during pruning.
func cmdCache(nidoDir string, args []string, prov provider.VMProvider) {
	jsonOut, rest := consumeJSONFlag(args)
	if len(rest) < 1 {
		ui.Error("Usage: nido cache <ls|info|rm|prune>")
		os.Exit(1)
	}

	imgDir := filepath.Join(nidoDir, "images")
	catalog, err := image.LoadCatalog(imgDir, image.DefaultCacheTTL)
	if err != nil {
		if jsonOut {
			resp := clijson.NewResponseError("cache", "ERR_IO", "Catalog load failed", err.Error(), "Check your network connection and try again.", nil)
			_ = clijson.PrintJSON(resp)
			os.Exit(1)
		}
		ui.Error("Failed to load catalog: %v", err)
		os.Exit(1)
	}

	subCmd := rest[0]
	switch subCmd {
	case "ls", "list":
		// List all cached images with sizes
		cached, err := catalog.GetCachedImages(imgDir)
		if err != nil {
			if jsonOut {
				resp := clijson.NewResponseError("cache ls", "ERR_IO", "Cache list failed", err.Error(), "Check your cache path and try again.", nil)
				_ = clijson.PrintJSON(resp)
				os.Exit(1)
			}
			ui.Error("Failed to list cache: %v", err)
			os.Exit(1)
		}

		if len(cached) == 0 {
			if jsonOut {
				resp := clijson.NewResponseOK("cache ls", map[string]interface{}{
					"cache": []interface{}{},
				})
				_ = clijson.PrintJSON(resp)
				return
			}
			ui.Info("Cache is empty. No images downloaded yet.")
			return
		}

		if jsonOut {
			type cacheJSON struct {
				Name       string `json:"name"`
				Version    string `json:"version"`
				SizeBytes  int64  `json:"size_bytes"`
				ModifiedAt string `json:"modified_at"`
			}
			items := make([]cacheJSON, 0, len(cached))
			for _, img := range cached {
				items = append(items, cacheJSON{
					Name:       img.Name,
					Version:    img.Version,
					SizeBytes:  img.Size,
					ModifiedAt: img.ModTime.UTC().Format(time.RFC3339),
				})
			}
			resp := clijson.NewResponseOK("cache ls", map[string]interface{}{
				"cache": items,
			})
			_ = clijson.PrintJSON(resp)
			return
		}

		ui.Header("Cached Images")
		fmt.Printf("\n %s%-25s %-15s %-12s %s%s\n", ui.Bold, "IMAGE", "VERSION", "SIZE", "MODIFIED", ui.Reset)
		fmt.Printf(" %s%s%s\n", ui.Dim, strings.Repeat("-", 70), ui.Reset)
		for _, img := range cached {
			sizeStr := image.FormatBytes(img.Size)
			modStr := img.ModTime.Format("2006-01-02")
			fmt.Printf(" %-25s %-15s %-12s %s\n", img.Name, img.Version, sizeStr, modStr)
		}
		fmt.Println("")

	case "info":
		// Show cache statistics
		stats, err := catalog.GetCacheStats(imgDir)
		if err != nil {
			if jsonOut {
				resp := clijson.NewResponseError("cache info", "ERR_IO", "Cache stats failed", err.Error(), "Check your cache path and try again.", nil)
				_ = clijson.PrintJSON(resp)
				os.Exit(1)
			}
			ui.Error("Failed to get cache stats: %v", err)
			os.Exit(1)
		}

		if stats.TotalImages == 0 {
			if jsonOut {
				resp := clijson.NewResponseOK("cache info", map[string]interface{}{
					"stats": map[string]interface{}{
						"total_images": 0,
						"total_size":   0,
					},
				})
				_ = clijson.PrintJSON(resp)
				return
			}
			ui.Info("Cache is empty.")
			return
		}

		if jsonOut {
			resp := clijson.NewResponseOK("cache info", map[string]interface{}{
				"stats": map[string]interface{}{
					"total_images": stats.TotalImages,
					"total_size":   stats.TotalSize,
					"oldest":       stats.OldestImage.UTC().Format(time.RFC3339),
					"newest":       stats.NewestImage.UTC().Format(time.RFC3339),
				},
			})
			_ = clijson.PrintJSON(resp)
			return
		}

		ui.Header("Cache Statistics")
		ui.FancyLabel("Total Images", fmt.Sprintf("%d", stats.TotalImages))
		ui.FancyLabel("Total Size", image.FormatBytes(stats.TotalSize))
		ui.FancyLabel("Oldest Image", stats.OldestImage.Format("2006-01-02 15:04"))
		ui.FancyLabel("Newest Image", stats.NewestImage.Format("2006-01-02 15:04"))
		fmt.Println("")

	case "rm", "remove":
		// Remove specific cached image
		if len(rest) < 2 {
			ui.Error("Usage: nido cache rm <image:version>")
			os.Exit(1)
		}

		// Parse image:version
		parts := strings.Split(rest[1], ":")
		if len(parts) != 2 {
			ui.Error("Invalid format. Use: <image>:<version>")
			os.Exit(1)
		}

		name, version := parts[0], parts[1]
		if err := catalog.RemoveCachedImage(imgDir, name, version); err != nil {
			if isNotFoundErr(err) {
				if jsonOut {
					resp := clijson.NewResponseOK("cache rm", map[string]interface{}{
						"action": map[string]interface{}{
							"name":    name,
							"version": version,
							"result":  "not_found",
						},
					})
					_ = clijson.PrintJSON(resp)
					return
				}
				ui.Info("Cache entry %s:%s is already gone.", name, version)
				return
			}
			if jsonOut {
				resp := clijson.NewResponseError("cache rm", "ERR_IO", "Cache remove failed", err.Error(), "Check the image name and try again.", nil)
				_ = clijson.PrintJSON(resp)
				os.Exit(1)
			}
			ui.Error("Failed to remove image: %v", err)
			os.Exit(1)
		}

		if jsonOut {
			resp := clijson.NewResponseOK("cache rm", map[string]interface{}{
				"action": map[string]interface{}{
					"name":    name,
					"version": version,
					"result":  "removed",
				},
			})
			_ = clijson.PrintJSON(resp)
			return
		}

		ui.Success("Removed %s:%s from cache", name, version)

	case "prune":
		// Removing unused images with Smart Protection
		// We ignore --unused flag because we ALWAYS want to protect used images now.
		// Previously --unused might have meant "not in a running VM".
		// Now we define "used" as "is a backing file for ANY VM (stopped or running)".

		cached, err := catalog.GetCachedImages(imgDir)
		if err != nil {
			ui.Error("Failed to list cache: %v", err)
			os.Exit(1)
		}

		usedFiles, err := prov.GetUsedBackingFiles()
		if err != nil {
			ui.Error("Failed to determine used backing files: %v", err)
			ui.Error("Aborting to prevent breakage.")
			os.Exit(1)
		}

		isUsed := make(map[string]bool)
		for _, f := range usedFiles {
			isUsed[f] = true
		}

		if !jsonOut {
			ui.Ironic("Cleaning the cache (safely)...")
		}

		count := 0
		reclaimed := int64(0)
		for _, img := range cached {
			// Construct absolute path for check
			// Catalog stores Name/Version. File is <imgDir>/<Name>-<Version>.qcow2
			fullPath, _ := filepath.Abs(filepath.Join(imgDir, fmt.Sprintf("%s-%s.qcow2", img.Name, img.Version)))

			if isUsed[fullPath] {
				if !jsonOut {
					ui.Info("Skipping %s:%s (backing file for existing VM)", img.Name, img.Version)
				}
				continue
			}

			if err := catalog.RemoveCachedImage(imgDir, img.Name, img.Version); err == nil {
				count++
				reclaimed += img.Size // Corrected field name
				if !jsonOut {
					ui.Success("Removed %s:%s", img.Name, img.Version)
				}
			} else {
				ui.Error("Failed to remove %s:%s: %v", img.Name, img.Version, err)
			}
		}

		if jsonOut {
			resp := clijson.NewResponseOK("cache prune", map[string]interface{}{
				"stats": map[string]interface{}{
					"count":     count,
					"reclaimed": reclaimed,
				},
			})
			_ = clijson.PrintJSON(resp)
			return
		}

		if count == 0 {
			ui.Info("Nothing to prune.")
		} else {
			ui.Success("Pruned %d images, reclaimed %s.", count, image.FormatBytes(reclaimed))
		}

	default:
		ui.Error("Unknown cache command: %s", subCmd)
		ui.Error("Available: ls, info, rm, prune")
		os.Exit(1)
	}
}

// parsePortFlag parses strings like "web:80:32080/tcp" or "80".
// Implements Section 5.1 of advanced-port-forwarding.md.
func parsePortFlag(val string) (provider.PortForward, error) {
	pf := provider.PortForward{Protocol: "tcp"}

	// Split label if present
	if strings.Contains(val, ":") {
		parts := strings.SplitN(val, ":", 2)
		// Check if first part is a number (GuestPort) or a Label
		if _, err := provider.ParseInt(parts[0]); err != nil {
			pf.Label = parts[0]
			val = parts[1]
		}
	}

	// Handle protocol
	if strings.Contains(val, "/") {
		parts := strings.SplitN(val, "/", 2)
		pf.Protocol = strings.ToLower(parts[1])
		val = parts[0]
	}

	// Handle Guest:Host
	if strings.Contains(val, ":") {
		parts := strings.SplitN(val, ":", 2)
		gp, err := provider.ParseInt(parts[0])
		if err != nil {
			return pf, fmt.Errorf("invalid guest port: %v", err)
		}
		hp, err := provider.ParseInt(parts[1])
		if err != nil {
			return pf, fmt.Errorf("invalid host port: %v", err)
		}
		pf.GuestPort = gp
		pf.HostPort = hp
	} else {
		gp, err := provider.ParseInt(val)
		if err != nil {
			return pf, fmt.Errorf("invalid port: %v", err)
		}
		pf.GuestPort = gp
	}

	return pf, nil
}
