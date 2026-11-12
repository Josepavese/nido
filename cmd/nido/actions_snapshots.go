package main

import (
	"context"
	"os"

	clijson "github.com/Josepavese/nido/v4/internal/cli"
	"github.com/Josepavese/nido/v4/internal/ui"
	"github.com/spf13/cobra"
)

func actionSnapshotCreate(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		jsonOut := jsonEnabled(cmd)
		result, err := app.Provider.SnapshotCreate(context.Background(), args[0], args[1])
		if err != nil {
			snapshotActionError(jsonOut, "create", err)
			return
		}
		if jsonOut {
			_ = clijson.PrintJSON(clijson.NewResponseOK("snapshot create", map[string]interface{}{
				"action": map[string]interface{}{
					"name":     args[0],
					"snapshot": args[1],
					"result":   "created",
				},
				"snapshot": result,
			}))
			return
		}
		ui.Success("Snapshot %s created for VM %s.", result.Name, args[0])
	}
}

func actionSnapshotList(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		jsonOut := jsonEnabled(cmd)
		snapshots, err := app.Provider.SnapshotList(context.Background(), args[0])
		if err != nil {
			snapshotActionError(jsonOut, "list", err)
			return
		}
		if jsonOut {
			_ = clijson.PrintJSON(clijson.NewResponseOK("snapshot list", map[string]interface{}{
				"action": map[string]interface{}{
					"name":   args[0],
					"result": "listed",
				},
				"snapshots": snapshots,
			}))
			return
		}
		if len(snapshots) == 0 {
			ui.Info("No snapshots for VM %s.", args[0])
			return
		}
		ui.Success("Snapshots for VM %s:", args[0])
		for _, snapshot := range snapshots {
			ui.Info("  #%d  %s  (%s, %s)", snapshot.ID, snapshot.Name, snapshot.VMSize, snapshot.CreatedAt)
		}
	}
}

func actionSnapshotRestore(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		jsonOut := jsonEnabled(cmd)
		if err := app.Provider.SnapshotRestore(context.Background(), args[0], args[1]); err != nil {
			snapshotActionError(jsonOut, "restore", err)
			return
		}
		if jsonOut {
			_ = clijson.PrintJSON(clijson.NewResponseOK("snapshot restore", map[string]interface{}{
				"action": map[string]interface{}{
					"name":     args[0],
					"snapshot": args[1],
					"result":   "restored",
				},
			}))
			return
		}
		ui.Success("VM %s restored to snapshot %s.", args[0], args[1])
	}
}

func actionSnapshotDelete(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		jsonOut := jsonEnabled(cmd)
		if err := app.Provider.SnapshotDelete(context.Background(), args[0], args[1]); err != nil {
			snapshotActionError(jsonOut, "delete", err)
			return
		}
		if jsonOut {
			_ = clijson.PrintJSON(clijson.NewResponseOK("snapshot delete", map[string]interface{}{
				"action": map[string]interface{}{
					"name":     args[0],
					"snapshot": args[1],
					"result":   "deleted",
				},
			}))
			return
		}
		ui.Success("Snapshot %s deleted from VM %s.", args[1], args[0])
	}
}

func snapshotActionError(jsonOut bool, action string, err error) {
	if jsonOut {
		_ = clijson.PrintJSON(clijson.NewResponseError("snapshot "+action, "ERR_SNAPSHOT_FAILED", "Snapshot operation failed", err.Error(),
			"Stop or shutdown the VM first, then verify the VM and snapshot names.", nil))
	} else {
		ui.Error("Snapshot %s failed: %v", action, err)
	}
	os.Exit(1)
}
