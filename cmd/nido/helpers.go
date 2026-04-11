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

func jsonEnabled(cmd *cobra.Command) bool {
	flag := cmd.Flags().Lookup("json")
	if flag == nil {
		clijson.SetJSONMode(false)
		return false
	}
	enabled, _ := cmd.Flags().GetBool("json")
	clijson.SetJSONMode(enabled)
	return enabled
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

func imageCatalog(app *appContext) (*image.Catalog, error) {
	localRegistry := filepath.Join(app.Cwd, "registry", "images.json")
	if _, err := os.Stat(localRegistry); err == nil {
		return image.LoadCatalogFromFile(localRegistry)
	}
	return image.LoadCatalog(app.ImageDir(), image.DefaultCacheTTL)
}

func supportedGlobalConfigKeys() []string {
	return []string{
		"BACKUP_DIR",
		"SSH_USER",
		"IMAGE_DIR",
		"LINKED_CLONES",
		"THEME",
		"TUI_SIDEBAR_WIDTH",
		"TUI_SIDEBAR_WIDE_WIDTH",
		"TUI_INSET_CONTENT",
		"TUI_TAB_MIN_WIDTH",
		"TUI_EXIT_ZONE_WIDTH",
		"TUI_GAP_SCALE",
		"PORT_RANGE_START",
		"PORT_RANGE_END",
	}
}

func isVMName(p provider.VMProvider, name string) bool {
	vms, err := p.List()
	if err != nil {
		return false
	}
	for _, v := range vms {
		if v.Name == name {
			return true
		}
	}
	return false
}

func requireQemu(app *appContext) *provider.QemuProvider {
	if app.Qemu == nil {
		ui.Error("QEMU provider unavailable")
		os.Exit(1)
	}
	return app.Qemu
}

func forceBoolValueForKey(key string) []string {
	if key == "LINKED_CLONES" {
		return []string{"true", "false"}
	}
	return nil
}

func toShellDirective(items []string) ([]string, cobra.ShellCompDirective) {
	if len(items) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return items, cobra.ShellCompDirectiveNoFileComp
}

func printHumanCommandError(format string, args ...any) {
	ui.Error(format, args...)
	os.Exit(1)
}

func positionalValue(args []string, idx int) string {
	if idx < 0 || idx >= len(args) {
		return ""
	}
	return args[idx]
}

func commandExamples(lines ...string) string {
	return strings.Join(lines, "\n")
}

func formatVMTablePortSummary(vm provider.VMStatus) string {
	portSum := fmt.Sprintf("%d", vm.SSHPort)
	for _, f := range vm.Forwarding {
		if f.HostPort > 0 {
			portSum += fmt.Sprintf(",%d->%d", f.GuestPort, f.HostPort)
		}
	}
	if len(portSum) > 25 {
		portSum = portSum[:22] + "..."
	}
	return portSum
}
