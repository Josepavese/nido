package main

import (
	"os"
	"path/filepath"
	"strings"

	climeta "github.com/Josepavese/nido/internal/cli"
	"github.com/spf13/cobra"
)

func buildCompletionRegistry(app *appContext) map[string]climeta.CompletionFunc {
	return map[string]climeta.CompletionFunc{
		"vms":        completeVMs(app),
		"templates":  completeTemplates(app),
		"images":     completeImages(app),
		"blueprints": completeBlueprints(app),
		"config":     completeConfig(app),
		"config_set": completeConfigSet(),
		"spawn":      completeSpawn(app),
		"ssh":        completeSSH(app),
		"files": func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveDefault
		},
	}
}

func completeVMs(app *appContext) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		vms, err := app.Provider.List()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		items := make([]string, 0, len(vms))
		for _, vm := range vms {
			items = append(items, vm.Name)
		}
		return toShellDirective(items)
	}
}

func completeTemplates(app *appContext) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		templates, err := app.Provider.ListTemplates()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return toShellDirective(templates)
	}
}

func completeImages(app *appContext) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		catalog, err := imageCatalog(app)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		var items []string
		for _, img := range catalog.Images {
			for _, ver := range img.Versions {
				items = append(items, img.Name+":"+ver.Version)
				for _, alias := range ver.Aliases {
					items = append(items, img.Name+":"+alias)
				}
			}
		}
		return toShellDirective(items)
	}
}

func completeBlueprints(app *appContext) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		paths := []string{
			filepath.Join(app.Cwd, "registry", "blueprints"),
			filepath.Join(app.NidoDir, "blueprints"),
			filepath.Join(app.NidoDir, "registry", "blueprints"),
		}
		seen := map[string]bool{}
		var items []string
		for _, dir := range paths {
			entries, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				name := entry.Name()
				if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
					continue
				}
				base := strings.TrimSuffix(strings.TrimSuffix(name, ".yaml"), ".yml")
				if seen[base] {
					continue
				}
				seen[base] = true
				items = append(items, base)
			}
		}
		return toShellDirective(items)
	}
}

func completeConfig(app *appContext) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			items := []string{"set"}
			vms, err := app.Provider.List()
			if err == nil {
				for _, vm := range vms {
					items = append(items, vm.Name)
				}
			}
			return toShellDirective(items)
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}

func completeConfigSet() func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		switch len(args) {
		case 0:
			return toShellDirective(supportedGlobalConfigKeys())
		case 1:
			return toShellDirective(forceBoolValueForKey(strings.ToUpper(args[0])))
		default:
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
	}
}

func completeSpawn(app *appContext) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		if len(args) == 1 {
			imageFlag, _ := cmd.Flags().GetString("image")
			if imageFlag != "" {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return completeTemplates(app)(cmd, args, toComplete)
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}

func completeSSH(app *appContext) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return completeVMs(app)(cmd, args, toComplete)
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}
