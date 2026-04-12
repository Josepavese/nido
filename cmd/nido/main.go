package main

import (
	"fmt"
	"os"

	clijson "github.com/Josepavese/nido/internal/cli"
)

func main() {
	if hasJSONFlag(os.Args[1:]) {
		clijson.SetJSONMode(true)
	}

	app := newAppContext()
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
