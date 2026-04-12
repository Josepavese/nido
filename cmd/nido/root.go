package main

import (
	"io"
	"os"

	climeta "github.com/Josepavese/nido/internal/cli"
	"github.com/spf13/cobra"
)

func newRootCommand(app *appContext) (*cobra.Command, error) {
	manifest, err := climeta.LoadManifest()
	if err != nil {
		return nil, err
	}

	builder := &climeta.Builder{
		Manifest:              manifest,
		Actions:               buildActionRegistry(app),
		FlagCompletions:       buildCompletionRegistry(app),
		PositionalCompletions: buildCompletionRegistry(app),
	}

	root, err := builder.BuildRoot()
	if err != nil {
		return nil, err
	}

	root.SetOut(os.Stdout)
	if climeta.IsJSONMode() {
		root.SetErr(io.Discard)
	} else {
		root.SetErr(os.Stderr)
	}
	return root, nil
}
