package main

import (
	"context"

	"github.com/Josepavese/nido/internal/config"
	"github.com/Josepavese/nido/internal/provider"
	app "github.com/Josepavese/nido/internal/tui/app"
)

// cmdGUI launches the new Bubble Tea GUI for Nido.
func cmdGUI(prov provider.VMProvider, cfg *config.Config) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := app.Run(ctx, prov, cfg); err != nil {
		// Keep stdout clean; errors to stderr
		println("Failed to launch Nido GUI:", err.Error())
	}
}
