package main

import (
	"context"

	"github.com/Josepavese/nido/internal/config"
	"github.com/Josepavese/nido/internal/provider"
	tui "github.com/Josepavese/nido/internal/tui/gui"
)

// cmdGUI launches the new Bubble Tea GUI for Nido.
func cmdGUI(prov provider.VMProvider, cfg *config.Config) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := tui.Run(ctx, prov, cfg); err != nil {
		// Keep stdout clean; errors to stderr
		println("Failed to launch Nido GUI:", err.Error())
	}
}
