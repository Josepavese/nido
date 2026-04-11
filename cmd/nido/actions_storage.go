package main

import (
	"fmt"
	"os"

	clijson "github.com/Josepavese/nido/internal/cli"
	"github.com/Josepavese/nido/internal/image"
	"github.com/Josepavese/nido/internal/ui"
	"github.com/spf13/cobra"
)

func actionTemplateList(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		jsonOut := jsonEnabled(cmd)
		templates, err := app.Provider.ListTemplates()
		if err != nil {
			if jsonOut {
				_ = clijson.PrintJSON(clijson.NewResponseError("template list", "ERR_IO", "Template list failed", err.Error(), "Check your storage path and try again.", nil))
			} else {
				ui.Error("Failed to list templates: %v", err)
			}
			os.Exit(1)
		}

		if jsonOut {
			_ = clijson.PrintJSON(clijson.NewResponseOK("template list", map[string]interface{}{"templates": templates}))
			return
		}
		if len(templates) == 0 {
			ui.Info("No templates archived yet.")
			return
		}

		ui.Header("Templates")
		fmt.Println("")
		for _, name := range templates {
			fmt.Printf("  %s%s%s\n", ui.Cyan, name, ui.Reset)
		}
		fmt.Println("")
	}
}

func actionTemplateCreate(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		jsonOut := jsonEnabled(cmd)
		if !jsonOut {
			ui.Step("Creating template...")
		}
		path, err := app.Provider.CreateTemplate(args[0], args[1])
		if err != nil {
			if jsonOut {
				_ = clijson.PrintJSON(clijson.NewResponseError("template create", "ERR_IO", "Template create failed", err.Error(), "Check VM name and storage permissions.", nil))
			} else {
				ui.Error("Failed to create template: %v", err)
			}
			os.Exit(1)
		}

		if jsonOut {
			_ = clijson.PrintJSON(clijson.NewResponseOK("template create", map[string]interface{}{
				"action": map[string]interface{}{
					"vm":       args[0],
					"template": args[1],
					"path":     path,
					"result":   "created",
				},
			}))
			return
		}
		ui.Success("Template %s created.", args[1])
		ui.Info("Path: %s", path)
	}
}

func actionTemplateDelete(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		jsonOut := jsonEnabled(cmd)
		force, _ := cmd.Flags().GetBool("force")
		if !jsonOut {
			if force {
				ui.Step("Deleting template (forced)...")
			} else {
				ui.Step("Deleting template...")
			}
		}

		if err := app.Provider.DeleteTemplate(args[0], force); err != nil {
			if isNotFoundErr(err) {
				if jsonOut {
					_ = clijson.PrintJSON(clijson.NewResponseOK("template delete", map[string]interface{}{
						"action": map[string]interface{}{"name": args[0], "result": "not_found"},
					}))
				} else {
					ui.Info("Template '%s' is already gone.", args[0])
				}
				return
			}

			if jsonOut {
				_ = clijson.PrintJSON(clijson.NewResponseError("template delete", "ERR_IO", "Template delete failed", err.Error(), "Check the template name and try again.", nil))
			} else {
				ui.Error("Failed to delete template: %v", err)
			}
			os.Exit(1)
		}

		if jsonOut {
			_ = clijson.PrintJSON(clijson.NewResponseOK("template delete", map[string]interface{}{
				"action": map[string]interface{}{"name": args[0], "result": "deleted"},
			}))
			return
		}
		ui.Success("Template %s deleted.", args[0])
	}
}

func actionCacheList(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		jsonOut := jsonEnabled(cmd)
		cached, err := app.Provider.ListCachedImages()
		if err != nil {
			if jsonOut {
				_ = clijson.PrintJSON(clijson.NewResponseError("cache list", "ERR_IO", "Cache list failed", err.Error(), "Check your cache path and try again.", nil))
			} else {
				ui.Error("Failed to list cache: %v", err)
			}
			os.Exit(1)
		}

		if jsonOut {
			_ = clijson.PrintJSON(clijson.NewResponseOK("cache list", map[string]interface{}{"cache": cached}))
			return
		}
		if len(cached) == 0 {
			ui.Info("Cache is empty. No images downloaded yet.")
			return
		}

		ui.Header("Cached Images")
		fmt.Printf("\n %s%-25s %-15s %s%s\n", ui.Bold, "IMAGE", "VERSION", "SIZE", ui.Reset)
		fmt.Printf(" %s%s%s\n", ui.Dim, stringsRepeat("-", 58), ui.Reset)
		for _, img := range cached {
			version := img.Version
			if version == "" {
				version = "-"
			}
			fmt.Printf(" %-25s %-15s %s\n", img.Name, version, img.Size)
		}
		fmt.Println("")
	}
}

func actionCacheInfo(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		jsonOut := jsonEnabled(cmd)
		stats, err := app.Provider.CacheInfo()
		if err != nil {
			if jsonOut {
				_ = clijson.PrintJSON(clijson.NewResponseError("cache info", "ERR_IO", "Cache stats failed", err.Error(), "Check your cache path and try again.", nil))
			} else {
				ui.Error("Failed to get cache stats: %v", err)
			}
			os.Exit(1)
		}

		if jsonOut {
			_ = clijson.PrintJSON(clijson.NewResponseOK("cache info", map[string]interface{}{
				"stats": map[string]interface{}{
					"total_images": stats.Count,
					"total_size":   stats.TotalSize,
				},
			}))
			return
		}

		ui.Header("Cache Statistics")
		ui.FancyLabel("Total Images", fmt.Sprintf("%d", stats.Count))
		ui.FancyLabel("Total Size", stats.TotalSize)
	}
}

func actionCacheRemove(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		jsonOut := jsonEnabled(cmd)
		name, version := parseImageRef(args[0])
		if err := app.Provider.CacheRemove(name, version); err != nil {
			if isNotFoundErr(err) {
				if jsonOut {
					_ = clijson.PrintJSON(clijson.NewResponseOK("cache remove", map[string]interface{}{
						"action": map[string]interface{}{"name": name, "version": version, "result": "not_found"},
					}))
				} else {
					ui.Info("Cache entry %s is already gone.", args[0])
				}
				return
			}

			if jsonOut {
				_ = clijson.PrintJSON(clijson.NewResponseError("cache remove", "ERR_IO", "Cache remove failed", err.Error(), "Check the image name and whether it is still in use.", nil))
			} else {
				ui.Error("Failed to remove cached image: %v", err)
			}
			os.Exit(1)
		}

		if jsonOut {
			_ = clijson.PrintJSON(clijson.NewResponseOK("cache remove", map[string]interface{}{
				"action": map[string]interface{}{"name": name, "version": version, "result": "removed"},
			}))
			return
		}
		ui.Success("Removed cached image %s.", args[0])
	}
}

func actionCachePrune(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		jsonOut := jsonEnabled(cmd)
		if !jsonOut {
			ui.Step("Pruning unused cached images...")
		}

		count, reclaimed, err := app.Provider.CachePrune(true)
		if err != nil {
			if jsonOut {
				_ = clijson.PrintJSON(clijson.NewResponseError("cache prune", "ERR_INTERNAL", "Prune failed", err.Error(), "Try again.", nil))
			} else {
				ui.Error("Cache prune failed: %v", err)
			}
			os.Exit(1)
		}

		if jsonOut {
			_ = clijson.PrintJSON(clijson.NewResponseOK("cache prune", map[string]interface{}{
				"removed_count":   count,
				"reclaimed_bytes": reclaimed,
				"reclaimed_human": image.FormatBytes(reclaimed),
				"unused_only":     true,
			}))
			return
		}
		ui.Success("Removed %d cached images and reclaimed %s.", count, image.FormatBytes(reclaimed))
	}
}

func actionImagesList(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		cmdImageList(app.ImageDir(), args, jsonEnabled(cmd))
	}
}

func actionImagesPull(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		cmdImagePull(app.ImageDir(), args, jsonEnabled(cmd))
	}
}

func actionImagesInfo(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		cmdImageInfo(app.ImageDir(), args, jsonEnabled(cmd))
	}
}

func actionImagesRemove(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		cmdImageRemove(app.ImageDir(), app.Provider, args, jsonEnabled(cmd))
	}
}

func actionImagesUpdate(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		cmdImageUpdate(app.ImageDir(), args, jsonEnabled(cmd))
	}
}

func actionBuild(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		cmdBuild(app.NidoDir, app.ImageDir(), args, jsonEnabled(cmd))
	}
}

func stringsRepeat(s string, count int) string {
	out := ""
	for i := 0; i < count; i++ {
		out += s
	}
	return out
}
