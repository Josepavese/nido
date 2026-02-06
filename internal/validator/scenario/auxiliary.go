package scenario

import (
	"time"

	"github.com/Josepavese/nido/internal/validator/report"
)

// Auxiliary covers help/completion/register/mcp-help/update/gui (skippable).
func Auxiliary() Scenario {
	return Scenario{
		Name: "auxiliary",
		Steps: []Step{
			helpStep,
			completionStep,
			registerStep,
			mcpJSONListStep,
			guiStep,
			updateStep,
		},
	}
}

func helpStep(ctx *Context) report.StepResult {
	args := []string{"help"}
	res := runNido(ctx, "help", args, 10*time.Second)
	addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)
	addAssertion(&res, "stdout_present", res.Stdout != "", "")
	finalize(&res)
	return res
}

func completionStep(ctx *Context) report.StepResult {
	args := []string{"completion", "bash"}
	res := runNido(ctx, "completion", args, 10*time.Second)
	addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)
	addAssertion(&res, "stdout_present", res.Stdout != "", "")
	finalize(&res)
	return res
}

func registerStep(ctx *Context) report.StepResult {
	args := []string{"register", "--json"}
	res := runNido(ctx, "register", args, 10*time.Second)
	addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)
	if payload, err := parseJSON(res.Stdout); err == nil {
		addAssertion(&res, "json_parse", true, "")
		status := payload["status"]
		addAssertion(&res, "status_ok", status == "ok", "")
	} else {
		addAssertion(&res, "json_parse", false, err.Error())
	}
	finalize(&res)
	return res
}

func mcpJSONListStep(ctx *Context) report.StepResult {
	args := []string{"mcp-help"}
	res := runNido(ctx, "mcp-help", args, 10*time.Second)
	addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)
	addAssertion(&res, "stdout_present", res.Stdout != "", "")
	finalize(&res)
	return res
}

func guiStep(ctx *Context) report.StepResult {
	if ctx.Config.SkipGUI {
		return skipResult(ctx.Config.NidoBin, []string{"gui"}, "skip-gui enabled")
	}
	// The GUI is interactive; treat timeout as acceptable failure unless exit 0.
	args := []string{"gui"}
	res := runNido(ctx, "gui", args, ctx.Config.GUITimeout)
	if res.TimedOut {
		res.Result = "SKIP"
		res.Stderr = "gui timed out (expected in headless mode)"
		return res
	}
	addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)
	finalize(&res)
	return res
}

func updateStep(ctx *Context) report.StepResult {
	if ctx.Config.SkipUpdate {
		return skipResult(ctx.Config.NidoBin, []string{"update"}, "skip-update enabled")
	}
	args := []string{"update"}
	res := runNido(ctx, "update", args, ctx.Config.DownloadTimeout)
	addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)
	finalize(&res)
	return res
}
