package main

import (
	clijson "github.com/Josepavese/nido/v5/internal/cli"
	"github.com/Josepavese/nido/v5/internal/ui"
	"github.com/spf13/cobra"
)

func actionMigrate(app *appContext) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		count, err := app.Provider.MigrateArchitecture()
		if err != nil {
			exitCLIError(err)
			return
		}
		if jsonEnabled(cmd) {
			_ = clijson.PrintJSON(clijson.NewResponseOK("migrate", map[string]interface{}{"converted": count}))
			return
		}
		ui.Success("Architecture metadata adopted for %d records. Disk payloads and backing paths preserved.", count)
	}
}
