package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Josepavese/nido/internal/builder"
	clijson "github.com/Josepavese/nido/internal/cli"
	"github.com/Josepavese/nido/internal/pkg/sysutil"
	"github.com/Josepavese/nido/internal/ui"
)

func cmdBuild(nidoDir, imageDir string, args []string, jsonOut bool) {
	if len(args) < 1 {
		ui.Error("Usage: nido build <blueprint>")
		ui.Info("Example: nido build windows-11-eval")
		os.Exit(1)
	}

	blueprintName := args[0]
	// Auto-append .yaml if missing
	if !strings.HasSuffix(blueprintName, ".yaml") && !strings.HasSuffix(blueprintName, ".yml") {
		blueprintName += ".yaml"
	}

	// Locate blueprint
	// 1. Current directory
	cwd, _ := os.Getwd()
	localPath := filepath.Join(cwd, blueprintName)

	// 2. Registry directory in CWD
	// 2. Registry directory in CWD
	registryPath := filepath.Join(cwd, "registry", "blueprints", blueprintName)

	// 3. Global Registry (~/.nido/registry)
	homeDir, _ := sysutil.UserHome()
	globalRegistryPath := filepath.Join(homeDir, ".nido", "registry", "blueprints", blueprintName)

	targetPath := ""
	if _, err := os.Stat(localPath); err == nil {
		targetPath = localPath
	} else if _, err := os.Stat(registryPath); err == nil {
		targetPath = registryPath
	} else if _, err := os.Stat(globalRegistryPath); err == nil {
		targetPath = globalRegistryPath
	} else {
		ui.Error("Blueprint not found: %s", blueprintName)
		ui.Info("Searched in:\n  - %s\n  - %s\n  - %s", localPath, registryPath, globalRegistryPath)
		os.Exit(1)
	}

	bp, err := builder.LoadBlueprint(targetPath)
	if err != nil {
		ui.Error("Failed to load blueprint: %v", err)
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

	opts := []builder.EngineOption{}
	if !jsonOut {
		opts = append(opts, builder.WithReporter(cliBuildReporter{}))
	}
	eng := builder.NewEngine(cacheDir, workDir, imageDir, opts...)

	if err := eng.Build(bp); err != nil {
		if jsonOut {
			resp := clijson.NewResponseError("build", "ERR_BUILD_FAILED", "Build failed", err.Error(), "", nil)
			clijson.PrintJSON(resp)
			os.Exit(1)
		}
		ui.Error("Build failed: %v", err)
		os.Exit(1)
	}

	if jsonOut {
		resp := clijson.NewResponseOK("build", map[string]string{
			"result": "success",
			"image":  bp.OutputImage,
		})
		clijson.PrintJSON(resp)
		return
	}

	ui.Success("Image built successfully: %s", bp.OutputImage)
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
