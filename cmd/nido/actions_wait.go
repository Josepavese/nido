package main

import (
	"context"
	"fmt"
	"os"
	"time"

	clijson "github.com/Josepavese/nido/v4/internal/cli"
	"github.com/Josepavese/nido/v4/internal/provider"
	"github.com/Josepavese/nido/v4/internal/ui"
	"github.com/spf13/cobra"
)

func actionVMWait(app *appContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		jsonOut := jsonEnabled(cmd)
		timeoutSeconds, _ := cmd.Flags().GetInt("timeout")
		maxSeconds := int(provider.MaxReadyTimeout / time.Second)
		if timeoutSeconds <= 0 || timeoutSeconds > maxSeconds {
			message := fmt.Sprintf("timeout must be between 1 and %d seconds", maxSeconds)
			if jsonOut {
				_ = clijson.PrintJSON(clijson.NewResponseError("wait", "ERR_INVALID_ARGUMENT", "Invalid timeout", message, "Use a timeout within the supported range.", nil))
			} else {
				ui.Error("%s", message)
			}
			os.Exit(1)
		}

		if !jsonOut {
			ui.Step("Waiting for guest %s (up to %ds)...", args[0], timeoutSeconds)
		}
		readiness, err := app.Provider.WaitReady(context.Background(), args[0], time.Duration(timeoutSeconds)*time.Second)
		if err != nil {
			if jsonOut {
				_ = clijson.PrintJSON(clijson.NewResponseError("wait", "ERR_WAIT_FAILED", "Wait failed", err.Error(), "Check the VM name and state, then retry.", nil))
			} else {
				ui.Error("Wait failed: %v", err)
			}
			os.Exit(1)
		}

		if !readiness.Ready {
			detail := readinessDetail(readiness)
			if jsonOut {
				_ = clijson.PrintJSON(clijson.NewResponseError("wait", "ERR_NOT_READY", "Guest not ready", detail,
					"Inspect the VM state and serial log, fix the reported cause, then retry.", readiness))
			} else {
				ui.Error("VM %s is not ready: %s", args[0], readiness.Failure)
				ui.Info("Phase: %s | provisioning: %s | attempts: %d | %s",
					readiness.Phase, readiness.Provisioning, readiness.Attempts, readinessDuration(readiness.DurationMS))
				if readiness.Detail != "" {
					ui.Info("Detail: %s", readiness.Detail)
				}
			}
			os.Exit(1)
		}

		if jsonOut {
			_ = clijson.PrintJSON(clijson.NewResponseOK("wait", map[string]interface{}{
				"action": map[string]interface{}{
					"name":   args[0],
					"result": "ready",
				},
				"readiness": readiness,
			}))
			return
		}
		ui.Success("VM %s ready in %s (provisioning: %s, attempts: %d).",
			args[0], readinessDuration(readiness.DurationMS), readiness.Provisioning, readiness.Attempts)
	}
}

func readinessDetail(readiness provider.GuestReadiness) string {
	if readiness.Detail != "" {
		return readiness.Detail
	}
	if readiness.Failure != "" {
		return fmt.Sprintf("%s while waiting in phase %s", readiness.Failure, readiness.Phase)
	}
	return fmt.Sprintf("guest not ready in phase %s", readiness.Phase)
}

func readinessDuration(durationMS int64) string {
	return (time.Duration(durationMS) * time.Millisecond).Round(time.Millisecond).String()
}

// formatGuestReadiness renders the shared observation for human surfaces.
func formatGuestReadiness(readiness *provider.GuestReadiness) string {
	if readiness == nil {
		return "unknown"
	}
	if readiness.Ready {
		if readiness.Provisioning != "" {
			return fmt.Sprintf("ready (provisioning: %s)", readiness.Provisioning)
		}
		return "ready"
	}
	label := fmt.Sprintf("%s (%s)", readiness.State, readiness.Phase)
	if readiness.Failure != "" {
		label += ", " + string(readiness.Failure)
	}
	if readiness.Detail != "" {
		label += ": " + readiness.Detail
	}
	return label
}
