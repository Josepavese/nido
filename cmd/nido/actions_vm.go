package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	clijson "github.com/Josepavese/nido/internal/cli"
	"github.com/Josepavese/nido/internal/image"
	"github.com/Josepavese/nido/internal/provider"
	"github.com/Josepavese/nido/internal/ui"
	"github.com/spf13/cobra"
)

func actionVMList(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		jsonOut := jsonEnabled(cmd)
		vms, err := app.Provider.List()
		if jsonOut {
			if err != nil {
				_ = clijson.PrintJSON(clijson.NewResponseError("ls", "ERR_INTERNAL", "List failed", err.Error(), "Try again or run nido doctor for diagnostics.", nil))
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

			_ = clijson.PrintJSON(clijson.NewResponseOK("ls", map[string]interface{}{"vms": items}))
			return
		}

		if err != nil {
			ui.Error("List failed: %v", err)
			os.Exit(1)
		}
		if len(vms) == 0 {
			ui.Info("No VMs found.")
			return
		}

		fmt.Printf("\n %s%-20s %-12s %-10s %s%s\n", ui.Bold, "NAME", "STATE", "PID", "PORT", ui.Reset)
		fmt.Printf(" %s%s%s\n", ui.Dim, strings.Repeat("-", 60), ui.Reset)
		for _, vm := range vms {
			stateColor := ui.Yellow
			if vm.State == "running" {
				stateColor = ui.Green
			}
			fmt.Printf(" %-20s %s%-12s%s %-10d %s\n", vm.Name, stateColor, vm.State, ui.Reset, vm.PID, formatVMTablePortSummary(vm))
		}
		fmt.Println("")
	}
}

func actionVMInfo(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		jsonOut := jsonEnabled(cmd)
		info, err := app.Provider.Info(args[0])
		if err != nil {
			if jsonOut {
				_ = clijson.PrintJSON(clijson.NewResponseError("info", "ERR_NOT_FOUND", "VM not found", err.Error(), "Check the VM name and try again.", nil))
			} else {
				ui.Error("Failed to probe %s: %v", args[0], err)
			}
			os.Exit(1)
		}

		if jsonOut {
			_ = clijson.PrintJSON(clijson.NewResponseOK("info", map[string]interface{}{
				"vm": map[string]interface{}{
					"name":          info.Name,
					"state":         info.State,
					"ip":            info.IP,
					"ssh_user":      info.SSHUser,
					"ssh_port":      info.SSHPort,
					"vnc_port":      info.VNCPort,
					"memory_mb":     info.MemoryMB,
					"vcpus":         info.VCPUs,
					"gui":           info.Gui,
					"cmdline":       info.Cmdline,
					"forwarding":    info.Forwarding,
					"raw_qemu_args": info.RawQemuArgs,
					"accelerators":  info.Accelerators,
				},
			}))
			return
		}

		ui.Header("VM Details")
		ui.FancyLabel("Name", info.Name)
		ui.FancyLabel("State", info.State)
		ui.FancyLabel("IP Address", info.IP)
		ui.FancyLabel("SSH Command", fmt.Sprintf("ssh -p %d %s@%s", info.SSHPort, info.SSHUser, info.IP))
		if info.VNCPort > 0 {
			ui.FancyLabel("GUI (VNC)", fmt.Sprintf("127.0.0.1:%d", info.VNCPort))
		}
		ui.FancyLabel("Memory", fmt.Sprintf("%d MB", info.MemoryMB))
		ui.FancyLabel("vCPUs", fmt.Sprintf("%d", info.VCPUs))
		ui.FancyLabel("GUI Enabled", fmt.Sprintf("%v", info.Gui))
		if info.Cmdline != "" {
			ui.FancyLabel("Cmdline", info.Cmdline)
		}
		if len(info.RawQemuArgs) > 0 {
			ui.FancyLabel("Raw QEMU Args", fmt.Sprintf("%v", info.RawQemuArgs))
		}
		if len(info.Accelerators) > 0 {
			ui.FancyLabel("Accelerators", fmt.Sprintf("%v", info.Accelerators))
		}
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
	}
}

func actionVMSpawn(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		jsonOut := jsonEnabled(cmd)
		name := args[0]
		for _, r := range name {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.') {
				if jsonOut {
					_ = clijson.PrintJSON(clijson.NewResponseError("spawn", "ERR_INVALID_ARGS", "Invalid VM name", "Only alphanumeric, hyphens, underscores, and dots allowed.", "Usage: nido spawn <name> ...", nil))
				} else {
					ui.Error("Invalid VM name '%s'. (No spaces allowed, only alphanumeric, -, _, .)", name)
				}
				os.Exit(1)
			}
		}

		tpl := ""
		if len(args) > 1 {
			tpl = args[1]
		}
		imageTag, _ := cmd.Flags().GetString("image")
		userDataPath, _ := cmd.Flags().GetString("user-data")
		gui, _ := cmd.Flags().GetBool("gui")
		cmdline, _ := cmd.Flags().GetString("cmdline")
		spawnMem, _ := cmd.Flags().GetInt("memory")
		spawnCPUs, _ := cmd.Flags().GetInt("cpus")
		rawArgs, _ := cmd.Flags().GetStringArray("qemu-arg")
		accelerators, _ := cmd.Flags().GetStringArray("accel")
		portMappings, _ := cmd.Flags().GetStringArray("port")
		web, _ := cmd.Flags().GetBool("web")
		ftp, _ := cmd.Flags().GetBool("ftp")

		var forwardings []provider.PortForward
		for _, mapping := range portMappings {
			pf, err := provider.ParsePortForward(mapping)
			if err != nil {
				ui.Error("Invalid port mapping: %v", err)
				os.Exit(1)
			}
			forwardings = append(forwardings, pf)
		}
		if web {
			forwardings = append(forwardings,
				provider.PortForward{Label: "HTTP", GuestPort: 80, Protocol: "tcp"},
				provider.PortForward{Label: "HTTPS", GuestPort: 443, Protocol: "tcp"},
			)
		}
		if ftp {
			forwardings = append(forwardings, provider.PortForward{Label: "FTP", GuestPort: 21, Protocol: "tcp"})
		}

		customSshUser := ""
		customSshPassword := ""
		var resolvedVersion *image.Version

		if imageTag != "" {
			localPath := filepath.Join(app.ImageDir(), imageTag+".qcow2")
			localExists := false
			if _, err := os.Stat(localPath); err == nil {
				localExists = true
			} else if _, err := os.Stat(filepath.Join(app.ImageDir(), imageTag)); err == nil {
				localPath = filepath.Join(app.ImageDir(), imageTag)
				localExists = true
			}

			if localExists {
				if !jsonOut {
					ui.Info("Found local image: %s", filepath.Base(localPath))
				}
				tpl = localPath
			} else {
				catalog, err := imageCatalog(app)
				if err != nil {
					if jsonOut {
						_ = clijson.PrintJSON(clijson.NewResponseError("spawn", "ERR_IO", "Catalog load failed", err.Error(), "Check your network connection and try again.", nil))
					} else {
						ui.Error("Failed to load catalog: %v", err)
					}
					os.Exit(1)
				}

				pName, pVer := imageTag, ""
				if strings.Contains(imageTag, ":") {
					parts := strings.Split(imageTag, ":")
					pName, pVer = parts[0], parts[1]
				}

				img, ver, err := catalog.FindImage(pName, pVer)
				if err != nil {
					if jsonOut {
						_ = clijson.PrintJSON(clijson.NewResponseError("spawn", "ERR_NOT_FOUND", "Image not found", err.Error(), "Run 'nido images list' to see available images.", nil))
					} else {
						ui.Error("Image %s not found in catalog (and not found locally in %s).", imageTag, app.ImageDir())
					}
					os.Exit(1)
				}
				resolvedVersion = ver
				customSshUser = img.SSHUser
				if ver.SSHPassword != "" {
					customSshPassword = ver.SSHPassword
				} else if img.SSHPassword != "" {
					customSshPassword = img.SSHPassword
				}

				imgPath := filepath.Join(app.ImageDir(), fmt.Sprintf("%s-%s.qcow2", img.Name, ver.Version))
				if _, err := os.Stat(imgPath); os.IsNotExist(err) {
					if !jsonOut {
						ui.Info("Image not found locally. Pulling %s:%s...", img.Name, ver.Version)
					}
					downloader := image.Downloader{Quiet: jsonOut}

					downloadPath := imgPath
					isTarXz := strings.HasSuffix(ver.URL, ".tar.xz")
					isZst := strings.Contains(ver.URL, ".zst") || strings.Contains(ver.URL, ".zstandard")
					if len(ver.PartURLs) > 0 {
						isZst = strings.Contains(ver.PartURLs[0], ".zst") || strings.Contains(ver.PartURLs[0], ".zstandard")
					}
					if isTarXz {
						downloadPath = imgPath + ".tar.xz"
					} else if isZst {
						downloadPath = imgPath + ".zst"
					}

					var downloadErr error
					if len(ver.PartURLs) > 0 {
						downloadErr = downloader.DownloadMultiPart(ver.PartURLs, downloadPath, ver.SizeBytes)
					} else {
						downloadErr = downloader.Download(ver.URL, downloadPath, ver.SizeBytes)
					}
					if downloadErr != nil {
						if jsonOut {
							_ = clijson.PrintJSON(clijson.NewResponseError("spawn", "ERR_IO", "Download failed", downloadErr.Error(), "Check your network connection and try again.", nil))
						} else {
							ui.Error("Download failed: %v", downloadErr)
						}
						os.Exit(1)
					}

					if isTarXz || isZst {
						if ver.Checksum != "" {
							if !jsonOut {
								ui.Ironic("Verifying genetic integrity (archive)...")
							}
							if err := image.VerifyChecksum(downloadPath, ver.Checksum, ver.ChecksumType); err != nil {
								if jsonOut {
									_ = clijson.PrintJSON(clijson.NewResponseError("spawn", "ERR_IO", "Verification failed", err.Error(), "Retry the download or choose a different image.", nil))
								} else {
									ui.Error("Verification failed: %v", err)
								}
								_ = os.Remove(downloadPath)
								os.Exit(1)
							}
						} else if !jsonOut {
							ui.Warn("⚠️ No checksum provided. Integrity cannot be verified.")
						}
						if err := downloader.Decompress(downloadPath, imgPath); err != nil {
							if jsonOut {
								_ = clijson.PrintJSON(clijson.NewResponseError("spawn", "ERR_IO", "Decompression failed", err.Error(), "Retry the download or choose a different image.", nil))
							} else {
								ui.Error("Decompression failed: %v", err)
							}
							_ = os.Remove(downloadPath)
							os.Exit(1)
						}
						_ = os.Remove(downloadPath)
						if !jsonOut {
							ui.Success("Image extracted successfully.")
						}
					} else {
						if ver.Checksum != "" {
							if !jsonOut {
								ui.Ironic("Verifying genetic integrity...")
							}
							if err := image.VerifyChecksum(downloadPath, ver.Checksum, ver.ChecksumType); err != nil {
								if jsonOut {
									_ = clijson.PrintJSON(clijson.NewResponseError("spawn", "ERR_IO", "Verification failed", err.Error(), "Retry the download or choose a different image.", nil))
								} else {
									ui.Error("Verification failed: %v", err)
								}
								_ = os.Remove(downloadPath)
								os.Exit(1)
							}
						} else if !jsonOut {
							ui.Warn("⚠️ No checksum provided. Integrity cannot be verified.")
						}
						if !jsonOut {
							ui.Success("Image prepared successfully.")
						}
					}
				}

				tpl = imgPath
			}
		}

		if cmdline == "" && resolvedVersion != nil {
			cmdline = resolvedVersion.Cmdline
		}

		if !jsonOut {
			ui.Step("Creating VM...")
		}

		spawnOpts := provider.VMOptions{
			DiskPath:     tpl,
			UserDataPath: userDataPath,
			Gui:          gui,
			SSHUser:      customSshUser,
			SSHPassword:  customSshPassword,
			Forwarding:   forwardings,
			Cmdline:      cmdline,
			MemoryMB:     spawnMem,
			VCPUs:        spawnCPUs,
			RawQemuArgs:  rawArgs,
			Accelerators: accelerators,
		}
		if err := app.Provider.Spawn(name, spawnOpts); err != nil {
			if jsonOut {
				code := "ERR_INTERNAL"
				if isAlreadyExistsErr(err) {
					code = "ERR_ALREADY_EXISTS"
				}
				_ = clijson.PrintJSON(clijson.NewResponseError("spawn", code, "Spawn failed", err.Error(), "Check the template or image and try again.", nil))
			} else {
				ui.Error("VM creation failed for %s: %v", name, err)
			}
			os.Exit(1)
		}

		source := "template"
		if imageTag != "" {
			source = "image " + imageTag
		}
		if jsonOut {
			_ = clijson.PrintJSON(clijson.NewResponseOK("spawn", map[string]interface{}{
				"action": map[string]interface{}{
					"name":      name,
					"result":    "spawned",
					"source":    source,
					"gui":       gui,
					"user_data": userDataPath,
					"image_tag": imageTag,
				},
			}))
			return
		}

		ui.Success("VM %s created from %s.", name, source)
		if customSshPassword != "" {
			ui.Info("Initial SSH password: %s", customSshPassword)
		}
		if !app.Config.LinkedClones && imageTag != "" && tpl != "" {
			if err := os.Remove(tpl); err != nil {
				ui.Error("Failed to cleanup image cache: %v", err)
			} else {
				ui.Step("Linked clones disabled: cleaned temporary image cache.")
			}
		}
	}
}

func actionVMStart(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		jsonOut := jsonEnabled(cmd)
		gui, _ := cmd.Flags().GetBool("gui")
		startCmdline, _ := cmd.Flags().GetString("cmdline")

		if !jsonOut {
			ui.Step("Starting VM...")
		}
		if err := app.Provider.Start(args[0], provider.VMOptions{Gui: gui, Cmdline: startCmdline}); err != nil {
			if jsonOut {
				_ = clijson.PrintJSON(clijson.NewResponseError("start", "ERR_INTERNAL", "Start failed", err.Error(), "Check the VM state and try again.", nil))
			} else {
				ui.Error("Failed to start VM %s: %v", args[0], err)
			}
			os.Exit(1)
		}
		if jsonOut {
			_ = clijson.PrintJSON(clijson.NewResponseOK("start", map[string]interface{}{
				"action": map[string]interface{}{
					"name":   args[0],
					"result": "started",
					"gui":    gui,
				},
			}))
			return
		}
		ui.Success("VM %s started.", args[0])
	}
}

func actionVMStop(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		jsonOut := jsonEnabled(cmd)
		if !jsonOut {
			ui.Step("Stopping VM...")
		}
		if err := app.Provider.Stop(args[0], true); err != nil {
			if jsonOut {
				_ = clijson.PrintJSON(clijson.NewResponseError("stop", "ERR_INTERNAL", "Stop failed", err.Error(), "Check the VM state and try again.", nil))
			} else {
				ui.Error("Failed to stop VM %s: %v", args[0], err)
			}
			os.Exit(1)
		}
		if jsonOut {
			_ = clijson.PrintJSON(clijson.NewResponseOK("stop", map[string]interface{}{
				"action": map[string]interface{}{
					"name":   args[0],
					"result": "stopped",
				},
			}))
			return
		}
		ui.Success("VM %s stopped.", args[0])
	}
}

func actionVMSSH(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		cmdSsh(app.Provider, args[0], args[1:])
	}
}

func actionVMDelete(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		jsonOut := jsonEnabled(cmd)
		if !jsonOut {
			ui.Step("Deleting VM...")
		}
		if err := app.Provider.Delete(args[0]); err != nil {
			if isNotFoundErr(err) {
				if jsonOut {
					_ = clijson.PrintJSON(clijson.NewResponseOK("delete", map[string]interface{}{
						"action": map[string]interface{}{"name": args[0], "result": "not_found"},
					}))
				} else {
					ui.Info("VM %s is already gone.", args[0])
				}
				return
			}
			if jsonOut {
				_ = clijson.PrintJSON(clijson.NewResponseError("delete", "ERR_INTERNAL", "Delete failed", err.Error(), "Check the VM name and try again.", nil))
			} else {
				ui.Error("VM deletion failed: %v", err)
			}
			os.Exit(1)
		}
		if jsonOut {
			_ = clijson.PrintJSON(clijson.NewResponseOK("delete", map[string]interface{}{
				"action": map[string]interface{}{"name": args[0], "result": "deleted"},
			}))
			return
		}
		ui.Success("VM %s deleted.", args[0])
	}
}

func actionVMPrune(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		jsonOut := jsonEnabled(cmd)
		if !jsonOut {
			ui.Step("Pruning stopped VMs...")
		}
		count, err := app.Provider.Prune()
		if err != nil {
			if jsonOut {
				_ = clijson.PrintJSON(clijson.NewResponseError("prune", "ERR_INTERNAL", "Prune failed", err.Error(), "Try again or run nido doctor.", nil))
			} else {
				ui.Error("VM prune failed: %v", err)
			}
			os.Exit(1)
		}
		if jsonOut {
			_ = clijson.PrintJSON(clijson.NewResponseOK("prune", map[string]interface{}{"removed_count": count}))
			return
		}
		ui.Success("Removed %d stopped VMs.", count)
	}
}
