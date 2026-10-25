package main

import (
	"fmt"

	clijson "github.com/Josepavese/nido/v5/internal/cli"
	"github.com/Josepavese/nido/v5/internal/pkg/sysutil"
	"github.com/spf13/cobra"
)

func actionEvents(app *appContext) clijson.ActionFunc {
	return func(cmd *cobra.Command, args []string) {
		ctx, cancel := sysutil.InterruptContext(cmd.Context())
		defer cancel()
		stream := app.Events.Watch(ctx)
		defer func() {
			cancel()
			for range stream {
			}
		}()
		for e := range stream {
			var err error
			if jsonEnabled(cmd) {
				err = clijson.PrintJSON(clijson.NewResponseOK("events", e))
			} else {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s %-20s %-20s %s\n", e.Time.Format("15:04:05"), e.Kind, e.VM, e.Action)
			}
			if err != nil {
				return
			}
		}
	}
}
