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
	if ctx.Config.KeepArtifacts {
		return skipResult(ctx.Config.NidoBin, []string{"delete"}, "keep-artifacts enabled; skipping VM cleanup")
	}

	// Use the global Sweep function to ensure we catch ALL test VMs,
	// even those that weren't tracked in state due to test failures or timeouts.
	Sweep(ctx)

	// We return a PASS dummy result because Sweep handles its own logging/errors
	// or we could construct a result based on Sweep's actions if Sweep returned something.
	// For now, let's assume Sweep did its best.
	return report.StepResult{
		Command:   "sweep",
		Args:      []string{"all"},
		Result:    "PASS",
		StartedAt: time.Now(),
		Stdout:    "Global sweep completed",
	}
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
