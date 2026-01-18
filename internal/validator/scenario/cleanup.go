package scenario

import (
	"os"
	"time"

	"github.com/Josepavese/nido/internal/validator/report"
)

// Cleanup removes tracked resources unless keep-artifacts is set.
func Cleanup() Scenario {
	return Scenario{
		Name: "cleanup",
		Steps: []Step{
			cleanupVMs,
			cleanupTempFiles,
		},
	}
}

func cleanupVMs(ctx *Context) report.StepResult {
	snapshot := ctx.State.Snapshot()
	if ctx.Config.KeepArtifacts {
		return skipResult(ctx.Config.NidoBin, []string{"delete"}, "keep-artifacts enabled; skipping VM cleanup")
	}
	if len(snapshot.VMs) == 0 {
		return skipResult(ctx.Config.NidoBin, []string{"delete"}, "no tracked VMs")
	}
	res := report.StepResult{
		Command:   ctx.Config.NidoBin,
		Args:      []string{"delete"},
		Result:    "PASS",
		StartedAt: time.Now(),
	}
	for _, name := range snapshot.VMs {
		// Best-effort cleanup; do not stop on errors.
		del := runNido(ctx, "delete", []string{"delete", name, "--json"}, 20*time.Second)
		res.Assertions = append(res.Assertions, report.AssertionResult{
			Name:    "delete_" + name,
			Result:  map[bool]string{true: "PASS", false: "FAIL"}[del.ExitCode == 0],
			Details: del.Stderr,
		})
	}
	finalize(&res)
	return res
}

func cleanupTempFiles(ctx *Context) report.StepResult {
	snapshot := ctx.State.Snapshot()
	if ctx.Config.KeepArtifacts {
		return skipResult("rm", []string{}, "keep-artifacts enabled; skipping temp cleanup")
	}
	if len(snapshot.TempFiles) == 0 {
		return skipResult("rm", []string{}, "no temp files")
	}
	res := report.StepResult{
		Command:   "rm",
		Args:      []string{},
		StartedAt: time.Now(),
		Result:    "PASS",
	}
	allOK := true
	for _, path := range snapshot.TempFiles {
		if err := os.Remove(path); err != nil {
			allOK = false
			res.Assertions = append(res.Assertions, report.AssertionResult{
				Name:    "rm_" + path,
				Result:  "FAIL",
				Details: err.Error(),
			})
		} else {
			res.Assertions = append(res.Assertions, report.AssertionResult{
				Name:   "rm_" + path,
				Result: "PASS",
			})
		}
	}
	if !allOK {
		res.Result = "FAIL"
	}
	return res
}
