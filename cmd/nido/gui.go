package main

import (
	"context"

	"github.com/Josepavese/nido/v4/internal/config"
	"github.com/Josepavese/nido/v4/internal/provider"
	app "github.com/Josepavese/nido/v4/internal/tui/app"
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
