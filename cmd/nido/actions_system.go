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
	"github.com/Josepavese/nido/internal/lifecycle"
	"github.com/Josepavese/nido/internal/mcp"
	"github.com/Josepavese/nido/internal/provider"
	"github.com/Josepavese/nido/internal/ui"
	"github.com/spf13/cobra"
)

func actionDoctor(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		jsonOut := jsonEnabled(cmd)
		reports := app.Provider.Doctor()
		if jsonOut {
			failCount := 0
			for _, r := range reports {
				if strings.Contains(r, "[FAIL]") {
					failCount++
				}
			}
			_ = clijson.PrintJSON(clijson.NewResponseOK("doctor", map[string]interface{}{
				"reports": reports,
				"summary": map[string]interface{}{
					"total":  len(reports),
					"failed": failCount,
					"passed": len(reports) - failCount,
				},
			}))
			return
		}

		ui.Header("Nido System Diagnostics")
		for _, r := range reports {
			icon := ui.IconSuccess
			if strings.Contains(r, "[FAIL]") {
				icon = ui.IconError
			}
			fmt.Printf("  %s %s\n", icon, r)
		}
		fmt.Println("")
		ui.Success("Diagnostics completed.")
	}
}

func actionAccelList(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		jsonOut := jsonEnabled(cmd)
		prov := requireQemu(app)
		devs, err := prov.ListAccelerators()
		if err != nil {
			if jsonOut {
				_ = clijson.PrintJSON(clijson.NewResponseError("accel", "ERR_SCAN", "Failed to scan PCI bus", err.Error(), "", nil))
			} else {
				ui.Error("Error scanning PCI bus: %v", err)
			}
			os.Exit(1)
		}

		if jsonOut {
			_ = clijson.PrintJSON(clijson.NewResponseOK("accel", map[string]interface{}{"devices": devs}))
			return
		}

		ui.Header("Accelerators")
		fmt.Printf("%-15s %-12s %-30s %-10s %s\n", "PCI-ID", "IOMMU", "DEVICE", "ISOLATED", "STATUS")
		fmt.Println(strings.Repeat("─", 84))
		for _, d := range devs {
			status := ui.Green + "Safe" + ui.Reset
			isolated := ui.Green + "Yes" + ui.Reset
			if !d.IsIsolated {
				isolated = ui.Red + "No" + ui.Reset
			}
			if !d.IsSafe {
				status = ui.Yellow + "Warning" + ui.Reset
				if d.Warning != "" {
					status = ui.Red + "UNSAFE" + ui.Reset
				}
			}

			name := d.Class
			if d.Vendor != "" {
				name = fmt.Sprintf("[%s:%s] %s", d.Vendor, d.Device, d.Class)
			}
			if len(name) > 30 {
				name = name[:27] + "..."
			}

			fmt.Printf("%-15s %-12s %-30s %-10s %s\n", d.ID, "Grp "+d.IOMMUGroup, name, isolated, status)
			if d.Warning != "" {
				fmt.Printf("  -> %s%s%s\n", ui.Dim, d.Warning, ui.Reset)
			}
		}
		fmt.Println(strings.Repeat("─", 84))
		ui.Info("Use: nido spawn ... --accel <PCI-ID>")
	}
}

func actionConfig(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		jsonOut := jsonEnabled(cmd)
		if len(args) == 0 {
			showGlobalConfig(app.Config, app.ConfigPath, jsonOut)
			return
		}
		updateVMConfig(cmd, app.Provider, args[0], jsonOut)
	}
}

func actionConfigSet(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		jsonOut := jsonEnabled(cmd)
		key := strings.ToUpper(args[0])
		val := args[1]

		valid := map[string]bool{}
		for _, k := range supportedGlobalConfigKeys() {
			valid[k] = true
		}
		if !valid[key] {
			if jsonOut {
				_ = clijson.PrintJSON(clijson.NewResponseError("config set", "ERR_INVALID_ARGS", "Invalid config key", key, "Use 'nido config --help' or shell completion to inspect supported keys.", nil))
			} else {
				ui.Error("Invalid config key: %s", key)
			}
			os.Exit(1)
		}

		if err := config.UpdateConfig(app.ConfigPath, key, val); err != nil {
			if jsonOut {
				_ = clijson.PrintJSON(clijson.NewResponseError("config set", "ERR_IO", "Update failed", err.Error(), "Check permissions.", nil))
			} else {
				ui.Error("Failed to update config: %v", err)
			}
			os.Exit(1)
		}

		if cfg, err := config.LoadConfig(app.ConfigPath); err == nil {
			app.Config = cfg
		}

		if jsonOut {
			_ = clijson.PrintJSON(clijson.NewResponseOK("config set", map[string]interface{}{
				"action": "set",
				"key":    key,
				"value":  val,
			}))
			return
		}
		ui.Success("Updated %s = %s", key, val)
	}
}

func actionRegister(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		jsonOut := jsonEnabled(cmd)
		exe, _ := os.Executable()
		payload := map[string]interface{}{
			"mcpServers": map[string]interface{}{
				"nido-local-vm-manager": map[string]interface{}{
					"command": exe,
					"args":    []string{"mcp"},
				},
			},
		}

		if jsonOut {
			_ = clijson.PrintJSON(clijson.NewResponseOK("register", payload))
			return
		}

		ui.Header("MCP Registration")
		ui.Info("Copy the following JSON block into your agent configuration:")
		fmt.Println("")
		raw, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Printf("%s%s%s\n\n", ui.Dim, string(raw), ui.Reset)
	}
}

func actionVersion(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		jsonOut := jsonEnabled(cmd)
		if jsonOut {
			_ = clijson.PrintJSON(clijson.NewResponseOK("version", map[string]interface{}{
				"version":  build.Version,
				"state":    "Evolved",
				"protocol": "v3.0",
			}))
			return
		}

		ui.Header("Version")
		ui.FancyLabel("Version", build.Version)
		ui.FancyLabel("State", "Evolved")
		ui.Info("Protocol: v3.0")

		go func() {
			latest, err := build.GetLatestVersion()
			if err == nil && latest != "" && latest != build.Version {
				fmt.Printf("\n%sA new evolutionary state is available: %s%s%s (current: %s)\n", ui.Yellow, ui.Bold, latest, ui.Reset, build.Version)
				ui.Info("Run 'nido update' to ascend to the next level.")
			}
		}()
		time.Sleep(100 * time.Millisecond)
	}
}

func actionUpdate(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		ui.Header("Nido Update")
		ui.Step("Checking for updates...")

		latest, err := build.GetLatestVersion()
		if err != nil {
			ui.Error("Failed to reach the mother nest: %v", err)
			os.Exit(1)
		}
		if latest == build.Version {
			ui.Success("Already up to date (%s).", build.Version)
			return
		}

		ui.Info("Found new version: %s (current: %s)", latest, build.Version)
		ui.Step("Downloading release asset...")

		assetName := fmt.Sprintf("nido-%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH)
		internalBinaryName := "nido"
		if runtime.GOOS == "windows" {
			assetName = fmt.Sprintf("nido-%s-%s.zip", runtime.GOOS, runtime.GOARCH)
			internalBinaryName = "nido.exe"
		}

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Get("https://api.github.com/repos/Josepavese/nido/releases/latest")
		if err != nil {
			ui.Error("Failed to fetch release details: %v", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			ui.Error("Failed to fetch release details: GitHub returned HTTP %d", resp.StatusCode)
			os.Exit(1)
		}

		var release struct {
			Assets []struct {
				Name               string `json:"name"`
				BrowserDownloadURL string `json:"browser_download_url"`
			} `json:"assets"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
			ui.Error("Failed to decode release details: %v", err)
			os.Exit(1)
		}

		downloadURL := ""
		for _, asset := range release.Assets {
			if asset.Name == assetName {
				downloadURL = asset.BrowserDownloadURL
				break
			}
		}
		if downloadURL == "" {
			ui.Error("Package %s not found in latest release assets.", assetName)
			ui.Info("You might need to build it from source or wait for the release to finalize.")
			os.Exit(1)
		}

		exePath, err := os.Executable()
		if err != nil {
			ui.Error("Failed to locate current binary: %v", err)
			os.Exit(1)
		}
		if f, err := os.OpenFile(exePath, os.O_RDWR, 0); err != nil {
			if os.IsPermission(err) {
				ui.Error("Permission denied: cannot write to %s", exePath)
				ui.Info("Run the upgrade command with sudo if the binary is system-installed.")
				os.Exit(1)
			}
		} else {
			_ = f.Close()
		}

		tmpDir, err := os.MkdirTemp("", "nido-update-*")
		if err != nil {
			ui.Error("Failed to create temporary directory: %v", err)
			os.Exit(1)
		}
		defer os.RemoveAll(tmpDir)

		archivePath := filepath.Join(tmpDir, assetName)
		if err := downloadFile(client, downloadURL, archivePath); err != nil {
			ui.Error("Download failed: %v", err)
			os.Exit(1)
		}

		tmpPath := filepath.Join(tmpDir, internalBinaryName)
		if err := extractBinaryFromReleaseAsset(archivePath, tmpPath, internalBinaryName); err != nil {
			ui.Error("Failed to unpack release asset: %v", err)
			os.Exit(1)
		}
		if runtime.GOOS != "windows" {
			if err := os.Chmod(tmpPath, 0755); err != nil {
				ui.Error("Failed to set permissions on updated binary: %v", err)
				os.Exit(1)
			}
		}

		bakPath := exePath + ".bak"
		_ = os.Remove(bakPath)
		if err := os.Rename(exePath, bakPath); err != nil {
			ui.Error("Update failed while backing up current binary: %v", err)
			os.Exit(1)
		}
		if err := os.Rename(tmpPath, exePath); err != nil {
			ui.Error("Update failed while installing new binary: %v", err)
			_ = os.Rename(bakPath, exePath)
			os.Exit(1)
		}

		ui.Success("Updated to %s.", latest)
		writeInstalledShellCompletions(exePath, app.NidoDir)
	}
}

func actionUninstall(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		jsonOut := jsonEnabled(cmd)
		force, _ := cmd.Flags().GetBool("force")

		if !force {
			if jsonOut {
				_ = clijson.PrintJSON(clijson.NewResponseError("uninstall", "ERR_CONFIRMATION_REQUIRED", "Confirmation required", "Use --force to skip prompt.", "Usage: nido uninstall --force", nil))
				os.Exit(1)
			}

			ui.Warn("DANGER ZONE")
			ui.Warn("This will permanently delete:")
			fmt.Printf("  - Configuration & Data: %s\n", app.NidoDir)
			fmt.Printf("  - Local Templates:      %s\n", filepath.Join(app.NidoDir, "templates"))
			fmt.Printf("  - Desktop Entries:      Launcher / Start Menu / Applications\n")
			exe, _ := os.Executable()
			fmt.Printf("  - Nido Binary:          %s\n", exe)
			fmt.Println("")
			fmt.Print("Are you sure you want to proceed? [y/N]: ")
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) != "y" {
				ui.Info("Aborted. The nest remains safe.")
				return
			}
		}

		exe, err := os.Executable()
		if err != nil {
			ui.Error("Failed to locate self: %v", err)
			os.Exit(1)
		}
		if !jsonOut {
			ui.Step("Removing Nido installation...")
		}

		if err := lifecycle.Uninstall(app.NidoDir, exe); err != nil {
			if jsonOut {
				_ = clijson.PrintJSON(clijson.NewResponseError("uninstall", "ERR_INTERNAL", "Uninstall failed", err.Error(), "Check permissions and try again.", nil))
			} else {
				ui.Error("Self-destruct failed: %v", err)
			}
			os.Exit(1)
		}

		if jsonOut {
			_ = clijson.PrintJSON(clijson.NewResponseOK("uninstall", map[string]interface{}{"result": "uninstalled"}))
			return
		}
		ui.Success("Nido uninstalled.")
	}
}

func actionShellCompletion(shell string) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		if err := generateShellCompletion(cmd.Root(), shell, os.Stdout); err != nil {
			ui.Error("Failed to generate %s completion: %v", shell, err)
			os.Exit(1)
		}
	}
}

func actionMCPHelp(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		_ = clijson.PrintJSON(clijson.NewResponseOK("mcp-help", map[string]interface{}{
			"tools": mcp.ToolsCatalog(),
		}))
	}
}

func actionMCP(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		mcp.NewServer(app.Provider).Serve()
	}
}

func showGlobalConfig(cfg *config.Config, path string, jsonOut bool) {
	if jsonOut {
		_ = clijson.PrintJSON(clijson.NewResponseOK("config", map[string]interface{}{
			"config_path": path,
			"config": map[string]interface{}{
				"backup_dir":       cfg.BackupDir,
				"ssh_user":         cfg.SSHUser,
				"image_dir":        cfg.ImageDir,
				"linked_clones":    cfg.LinkedClones,
				"theme":            cfg.Theme,
				"port_range_start": cfg.PortRangeStart,
				"port_range_end":   cfg.PortRangeEnd,
				"tui": map[string]interface{}{
					"sidebar_width":      cfg.TUI.SidebarWidth,
					"sidebar_wide_width": cfg.TUI.SidebarWideWidth,
					"inset_content":      cfg.TUI.InsetContent,
					"tab_min_width":      cfg.TUI.TabMinWidth,
					"exit_zone_width":    cfg.TUI.ExitZoneWidth,
					"gap_scale":          cfg.TUI.GapScale,
				},
			},
		}))
		return
	}

	ui.Header("Nido Genetic Configuration")
	ui.FancyLabel("Config Path", path)
	ui.FancyLabel("Backup Dir", cfg.BackupDir)
	ui.FancyLabel("SSH User", cfg.SSHUser)
	ui.FancyLabel("Image Dir", cfg.ImageDir)
	ui.FancyLabel("Theme", ternaryString(cfg.Theme != "", cfg.Theme, "auto"))
	cloneStatus := "Enabled (Space Saving)"
	if !cfg.LinkedClones {
		cloneStatus = "Disabled (Full Copy)"
	}
	ui.FancyLabel("Linked Clones", cloneStatus)
	ui.FancyLabel("Port Range", fmt.Sprintf("%d-%d", cfg.PortRangeStart, cfg.PortRangeEnd))
}

func updateVMConfig(cmd *cobra.Command, prov provider.VMProvider, name string, jsonOut bool) {
	updates := provider.VMConfigUpdates{}
	hasUpdates := false

	if cmd.Flags().Changed("memory") {
		val, _ := cmd.Flags().GetInt("memory")
		updates.MemoryMB = &val
		hasUpdates = true
	}
	if cmd.Flags().Changed("cpus") {
		val, _ := cmd.Flags().GetInt("cpus")
		updates.VCPUs = &val
		hasUpdates = true
	}
	if cmd.Flags().Changed("ssh-port") {
		val, _ := cmd.Flags().GetInt("ssh-port")
		updates.SSHPort = &val
		hasUpdates = true
	}
	if cmd.Flags().Changed("vnc-port") {
		val, _ := cmd.Flags().GetInt("vnc-port")
		updates.VNCPort = &val
		hasUpdates = true
	}
	if cmd.Flags().Changed("gui") {
		val, _ := cmd.Flags().GetBool("gui")
		updates.Gui = &val
		hasUpdates = true
	}
	if cmd.Flags().Changed("ssh-user") {
		val, _ := cmd.Flags().GetString("ssh-user")
		updates.SSHUser = &val
		hasUpdates = true
	}
	if cmd.Flags().Changed("cmdline") {
		val, _ := cmd.Flags().GetString("cmdline")
		updates.Cmdline = &val
		hasUpdates = true
	}
	if cmd.Flags().Changed("port") {
		mappings, _ := cmd.Flags().GetStringArray("port")
		forwarding := make([]provider.PortForward, 0, len(mappings))
		for _, mapping := range mappings {
			pf, err := provider.ParsePortForward(mapping)
			if err != nil {
				if jsonOut {
					_ = clijson.PrintJSON(clijson.NewResponseError("config", "ERR_INVALID_ARGS", "Invalid port mapping", err.Error(), "", nil))
				} else {
					ui.Error("Invalid port mapping: %v", err)
				}
				os.Exit(1)
			}
			forwarding = append(forwarding, pf)
		}
		updates.Forwarding = &forwarding
		hasUpdates = true
	}
	if cmd.Flags().Changed("qemu-arg") {
		val, _ := cmd.Flags().GetStringArray("qemu-arg")
		updates.RawQemuArgs = &val
		hasUpdates = true
	}
	if cmd.Flags().Changed("accel") {
		val, _ := cmd.Flags().GetStringArray("accel")
		updates.Accelerators = &val
		hasUpdates = true
	}

	if _, err := prov.Info(name); err != nil {
		if jsonOut {
			_ = clijson.PrintJSON(clijson.NewResponseError("config", "ERR_NOT_FOUND", "VM not found", err.Error(), "Check the VM name and try again.", nil))
		} else {
			ui.Error("VM not found: %v", err)
		}
		os.Exit(1)
	}

	if !hasUpdates {
		if jsonOut {
			_ = clijson.PrintJSON(clijson.NewResponseOK("config", map[string]interface{}{
				"name":   name,
				"result": "noop",
			}))
			return
		}
		ui.Info("No configuration changes requested.")
		return
	}

	if !jsonOut {
		ui.Ironic(fmt.Sprintf("Rewriting genetic sequence for %s...", name))
	}
	if err := prov.UpdateConfig(name, updates); err != nil {
		if jsonOut {
			_ = clijson.PrintJSON(clijson.NewResponseError("config", "ERR_UPDATE", "Config update failed", err.Error(), "", nil))
		} else {
			ui.Error("Mutation failed: %v", err)
		}
		os.Exit(1)
	}

	if jsonOut {
		_ = clijson.PrintJSON(clijson.NewResponseOK("config", map[string]interface{}{
			"name":   name,
			"result": "updated",
		}))
		return
	}
	ui.Success("Configuration updated. Restart VM to apply changes.")
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
	extraOptions := []string{
		"-o", "LogLevel=ERROR",
		"-o", "ConnectTimeout=5",
	}
	finalArgs := append([]string{}, extraOptions...)
	finalArgs = append(finalArgs, parts[1:]...)
	finalArgs = append(finalArgs, args...)

	command := exec.Command(parts[0], finalArgs...)
	command.Stdout = os.Stdout
	command.Stdin = os.Stdin
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		os.Exit(1)
	}
}

func generateShellCompletion(root *cobra.Command, shell string, w io.Writer) error {
	switch shell {
	case "bash":
		return root.GenBashCompletionV2(w, true)
	case "zsh":
		return root.GenZshCompletion(w)
	case "fish":
		return root.GenFishCompletion(w, true)
	case "powershell":
		return root.GenPowerShellCompletionWithDesc(w)
	default:
		return fmt.Errorf("unsupported shell %q", shell)
	}
}

func writeInstalledShellCompletions(exePath, nidoDir string) {
	ui.Step("Refreshing shell completion scripts...")

	type shellTarget struct {
		shell string
		path  string
	}
	targets := []shellTarget{
		{shell: "bash", path: filepath.Join(nidoDir, "bash_completion")},
		{shell: "zsh", path: filepath.Join(nidoDir, "zsh_completion")},
	}

	for _, target := range targets {
		out, err := exec.Command(exePath, "completion", target.shell).Output()
		if err != nil {
			ui.Warn("Failed to refresh %s completion: %v", target.shell, err)
			continue
		}
		if err := os.WriteFile(target.path, out, 0644); err != nil {
			ui.Warn("Failed to write %s completion: %v", target.shell, err)
		}
	}
	ui.Info("Shell completions updated in %s.", nidoDir)
}

func downloadFile(client *http.Client, url, destPath string) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
