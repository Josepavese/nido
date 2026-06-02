package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Josepavese/nido/internal/builder"
	clijson "github.com/Josepavese/nido/internal/cli"
	"github.com/Josepavese/nido/internal/image"
	"github.com/Josepavese/nido/internal/ui"
	"github.com/spf13/cobra"
)

func actionBlueprintList(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		jsonOut := jsonEnabled(cmd)
		ensureBundledRegistryCurrent("blueprint list", app.NidoDir, jsonOut)
		cmdBlueprintList(app.Cwd, app.NidoDir, app.ImageDir(), jsonOut)
	}
}

func actionBlueprintInfo(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		jsonOut := jsonEnabled(cmd)
		ensureBundledRegistryCurrent("blueprint info", app.NidoDir, jsonOut)
		cmdBlueprintInfo(app.Cwd, app.NidoDir, app.ImageDir(), args, jsonOut)
	}
}

func actionBlueprintBuild(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		jsonOut := jsonEnabled(cmd)
		ensureBundledRegistryCurrent("blueprint build", app.NidoDir, jsonOut)
		cmdBuild(app.Cwd, app.NidoDir, app.ImageDir(), args, jsonOut, "blueprint build")
	}
}

func cmdBlueprintList(cwd, nidoDir, imageDir string, jsonOut bool) {
	blueprints, err := builder.ListBlueprints(cwd, nidoDir, imageDir)
	if err != nil {
		if jsonOut {
			resp := clijson.NewResponseError("blueprint list", "ERR_IO", "Blueprint list failed", err.Error(), "Check the blueprint registry path and try again.", nil)
			clijson.PrintJSON(resp)
		} else {
			ui.Error("Failed to list blueprints: %v", err)
		}
		os.Exit(1)
	}

	if jsonOut {
		resp := clijson.NewResponseOK("blueprint list", map[string]interface{}{"blueprints": blueprints})
		clijson.PrintJSON(resp)
		return
	}

	if len(blueprints) == 0 {
		ui.Info("No blueprints found.")
		return
	}

	ui.Header("Blueprints")
	fmt.Printf("\n %s%-46s %-10s %-8s %-24s %s%s\n", ui.Bold, "BLUEPRINT", "VERSION", "STATUS", "TAG", "SOURCE", ui.Reset)
	fmt.Printf(" %s%s%s\n", ui.Dim, strings.Repeat("-", 104), ui.Reset)
	for _, bp := range blueprints {
		status := "missing"
		if bp.Built {
			status = "ready"
		}
		fmt.Printf(" %-46s %-10s %-8s %-24s %s\n", blueprintDisplayName(bp), bp.Version, status, bp.OutputTag, bp.Source)
	}
	fmt.Println("")
}

func cmdBlueprintInfo(cwd, nidoDir, imageDir string, args []string, jsonOut bool) {
	if len(args) < 1 {
		ui.Error("Usage: nido blueprint info <blueprint>")
		os.Exit(1)
	}

	_, info, err := builder.LoadBlueprintRef(cwd, nidoDir, imageDir, args[0])
	if err != nil {
		if jsonOut {
			resp := clijson.NewResponseError("blueprint info", "ERR_NOT_FOUND", "Blueprint not found", err.Error(), "Run 'nido blueprint list' to see available blueprints.", nil)
			clijson.PrintJSON(resp)
		} else {
			ui.Error("Blueprint not found: %v", err)
		}
		os.Exit(1)
	}

	if jsonOut {
		resp := clijson.NewResponseOK("blueprint info", map[string]interface{}{"blueprint": info})
		clijson.PrintJSON(resp)
		return
	}

	status := "missing"
	if info.Built {
		status = "ready"
	}
	ui.Header("Blueprint")
	ui.FancyLabel("Name", info.Name)
	if info.DisplayName != "" {
		ui.FancyLabel("Display Name", info.DisplayName)
	}
	ui.FancyLabel("Version", info.Version)
	ui.FancyLabel("Status", status)
	ui.FancyLabel("Output", info.OutputImage)
	ui.FancyLabel("Spawn Tag", info.OutputTag)
	if info.SSHUser != "" {
		ui.FancyLabel("SSH User", info.SSHUser)
	}
	ui.FancyLabel("Initial Password", ternaryString(info.HasPassword, "Provided by blueprint metadata", "Not specified"))
	ui.FancyLabel("Source", info.Source)
	ui.FancyLabel("Path", info.Path)
	ui.FancyLabel("Build", fmt.Sprintf("%d CPU, %s RAM, timeout %s", info.CPU, info.Memory, info.Timeout))
	if info.Description != "" {
		ui.Info("%s", info.Description)
	}
}

func blueprintDisplayName(info builder.BlueprintInfo) string {
	if info.DisplayName != "" {
		return info.DisplayName
	}
	return info.Name
}

func cmdBuild(cwd, nidoDir, imageDir string, args []string, jsonOut bool, command string) {
	if len(args) < 1 {
		ui.Error("Usage: nido build <blueprint>")
		ui.Info("Example: nido build windows-11-eval")
		os.Exit(1)
	}

	bp, info, err := builder.LoadBlueprintRef(cwd, nidoDir, imageDir, args[0])
	if err != nil {
		if jsonOut {
			resp := clijson.NewResponseError(command, "ERR_NOT_FOUND", "Blueprint not found", err.Error(), "Run 'nido blueprint list' to see available blueprints.", nil)
			clijson.PrintJSON(resp)
		} else {
			ui.Error("Failed to load blueprint: %v", err)
			ui.Info("Run 'nido blueprint list' to see available blueprints.")
		}
		os.Exit(1)
	}

	if !jsonOut {
		ui.Header("Blueprint Build")
		ui.FancyLabel("Name", bp.Name)
		ui.Info("%s", bp.Description)
		fmt.Println("")
	}

	// Setup Engine
	// Cache: ~/.nido/cache
	// Work: ~/.nido/tmp
	// Image: configured image cache/output directory
	cacheDir := filepath.Join(nidoDir, "cache")
	workDir := filepath.Join(nidoDir, "tmp")

	if info.Built {
		if jsonOut {
			resp := clijson.NewResponseOK(command, map[string]string{
				"result":       "ready",
				"blueprint":    bp.Name,
				"output_image": bp.OutputImage,
				"output_tag":   info.OutputTag,
				"output_path":  info.OutputPath,
			})
			clijson.PrintJSON(resp)
			return
		}
		ui.Success("Image already built: %s", bp.OutputImage)
		printBlueprintSpawnHint(bp)
		return
	}

	opts := []builder.EngineOption{}
	if !jsonOut {
		opts = append(opts, builder.WithReporter(cliBuildReporter{}))
	}
	eng := builder.NewEngine(cacheDir, workDir, imageDir, opts...)

	if err := eng.Build(bp); err != nil {
		if jsonOut {
			resp := clijson.NewResponseError(command, "ERR_BUILD_FAILED", "Build failed", err.Error(), "", nil)
			clijson.PrintJSON(resp)
			os.Exit(1)
		}
		ui.Error("Build failed: %v", err)
		os.Exit(1)
	}

	if jsonOut {
		resp := clijson.NewResponseOK(command, map[string]string{
			"result":       "built",
			"blueprint":    bp.Name,
			"output_image": bp.OutputImage,
			"output_tag":   builder.BlueprintOutputTag(bp),
			"output_path":  filepath.Join(imageDir, bp.OutputImage),
		})
		clijson.PrintJSON(resp)
		return
	}

	ui.Success("Image built successfully: %s", bp.OutputImage)
	printBlueprintSpawnHint(bp)
}

func printBlueprintSpawnHint(bp *image.Blueprint) {
	imageTag := strings.TrimSuffix(bp.OutputImage, ".qcow2")

	// Special handling for Windows images which require a "second stage" installation on first boot
	if strings.Contains(strings.ToLower(bp.OutputImage), "windows") || strings.Contains(strings.ToLower(bp.Name), "windows") {
		ui.Warn("Windows installation is not finished yet.")
		ui.Info("The image is ready, but the actual setup (OOBE/Getting Ready) happens on the first boot.")
		ui.Info("To monitor the progress, you MUST spawn with GUI enabled:")
		fmt.Println("")
		fmt.Printf("  %snido spawn my-win-vm --image %s --gui%s\n", ui.Bold, imageTag, ui.Reset)
		fmt.Println("")
		ui.Info("If you spawn headless (without --gui), it will work but you won't see the status.")
		ui.Info("SSH will be available automatically after the setup completes (approx 2-5 mins).")
		if bp.SSHUser != "" {
			ui.Info("Initial SSH user: %s", bp.SSHUser)
		}
		if bp.SSHPassword != "" {
			ui.Info("Initial SSH password: %s", bp.SSHPassword)
		}
	} else {
		ui.Info("You can now spawn a VM using: nido spawn my-vm --image %s", imageTag)
	}
}

type cliBuildReporter struct{}

func (cliBuildReporter) Header(title string)          { ui.Header(title) }
func (cliBuildReporter) Info(msg string, args ...any) { ui.Info(msg, args...) }
func (cliBuildReporter) Warn(msg string, args ...any) { ui.Warn(msg, args...) }
func (cliBuildReporter) Success(msg string, args ...any) {
	ui.Success(msg, args...)
}
