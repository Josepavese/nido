package main

import (
	"context"
	"fmt"
	"os"

	clijson "github.com/Josepavese/nido/v4/internal/cli"
	"github.com/Josepavese/nido/v4/internal/lifecycle"
)

func main() {
	if hasJSONFlag(os.Args[1:]) {
		clijson.SetJSONMode(true)
	}

	app := newAppContext()
	ctx, cancel := context.WithTimeout(context.Background(), lifecycle.StatePreparationTimeout)
	err := app.Provider.PrepareState(ctx)
	cancel()
	if err != nil {
		exitCLIError(fmt.Errorf("automatic state preparation: %w", err))
	}
	root, err := newRootCommand(app)
	if err != nil {
		exitCLIError(fmt.Errorf("failed to initialize CLI: %w", err))
	}

	if err := root.Execute(); err != nil {
		exitCLIError(err)
	}
}

func hasJSONFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--json" {
			return true
		}
	}
	return false
}

func exitCLIError(err error) {
	if clijson.IsJSONMode() {
		_ = clijson.PrintJSON(clijson.NewResponseError("cli", "ERR_CLI", "Command failed", err.Error(), "", nil))
	} else {
		_, _ = fmt.Fprintln(os.Stderr, err)
	}
	os.Exit(1)
}
