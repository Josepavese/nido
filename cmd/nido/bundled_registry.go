package main

import (
	"os"

	clijson "github.com/Josepavese/nido/internal/cli"
	"github.com/Josepavese/nido/internal/lifecycle"
	"github.com/Josepavese/nido/internal/ui"
)

func ensureBundledRegistryCurrent(command, nidoDir string, jsonOut bool) {
	if _, err := lifecycle.SyncBundledRegistryFromEmbedded(nidoDir); err != nil {
		if jsonOut {
			_ = clijson.PrintJSON(clijson.NewResponseError(
				command,
				"ERR_REGISTRY_SYNC",
				"Registry sync failed",
				err.Error(),
				"Check permissions for the Nido home registry and retry.",
				nil,
			))
			os.Exit(1)
		}
		ui.Error("Failed to sync bundled registry: %v", err)
		ui.Info("Check permissions for the Nido home registry and retry.")
		os.Exit(1)
	}
}
