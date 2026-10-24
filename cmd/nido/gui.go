package main

import (
	"context"

	app "github.com/Josepavese/nido/v4/internal/tui/app"
)

// cmdGUI launches the new Bubble Tea GUI for Nido.
func cmdGUI(a *appContext) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := app.Run(ctx, a.Provider, a.Config, a.Events, a.ConfigPath); err != nil {
		// Keep stdout clean; errors to stderr
		println("Failed to launch Nido GUI:", err.Error())
	}
}
