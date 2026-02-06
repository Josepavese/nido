package scenario

import (
	"strings"
	"time"

	"github.com/Josepavese/nido/internal/validator/report"
)

// PreClean removes stale prefixed VMs before running the suite.
func PreClean() Scenario {
	return Scenario{
		Name: "preclean",
		Steps: []Step{
			precleanStep,
			precleanTemplates,
		},
	}
}

// Sweep performs a best-effort global cleanup of all test artifacts.
// It is designed to be safe to run from signal handlers or cleanup hooks.
func Sweep(ctx *Context) {
	// 1. List all VMs
	args := []string{"list", "--json"}
	// Use a short timeout for listing
	res := runNido(ctx, "sweep-list", args, 5*time.Second)
	if res.ExitCode != 0 {
		return // Cannot list, cannot clean
	}

	payload, err := parseJSON(res.Stdout)
	if err != nil {
		return
	}

	vms := []string{}
	if data, ok := payload["data"].(map[string]interface{}); ok {
		if arr, ok := data["vms"].([]interface{}); ok {
			for _, v := range arr {
				if m, ok := v.(map[string]interface{}); ok {
					if name, ok := m["name"].(string); ok && isPrefixedTestVM(name) {
						vms = append(vms, name)
					}
				}
			}
		}
	}

	// 2. Destroy them all
	for _, name := range vms {
		// Fire and forget delete
		runNido(ctx, "sweep-delete", []string{"delete", name, "--force"}, 10*time.Second)
	}
}

func precleanStep(ctx *Context) report.StepResult {
	// Re-use Sweep logic implicitly?
	// For now, keep the rigorous assertions of precleanStep but maybe share detection logic?
	// Actually, precleanStep has specific assertions we want to keep.
	// But let's at least update the prefix check.

	args := []string{"list", "--json"}
	res := runNido(ctx, "preclean-list", args, 15*time.Second)
	addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)
	payload, err := parseJSON(res.Stdout)
	if err != nil {
		addAssertion(&res, "json_parse", false, err.Error())
		finalize(&res)
		return res
	}
	vms := []string{}
	if data, ok := payload["data"].(map[string]interface{}); ok {
		if arr, ok := data["vms"].([]interface{}); ok {
			for _, v := range arr {
				if m, ok := v.(map[string]interface{}); ok {
					if name, ok := m["name"].(string); ok && isPrefixedTestVM(name) {
						vms = append(vms, name)
					}
				}
			}
		}
	}
	for _, name := range vms {
		del := runNido(ctx, "preclean-delete", []string{"delete", name, "--json"}, 30*time.Second)
		finalize(&del)
		_ = ctx.Reporter.WriteStep(del)
		if del.Result == "PASS" {
			ctx.State.AddVM(name)
		}
	}
	addAssertion(&res, "deleted_prefixed", true, strings.Join(vms, ","))
	finalize(&res)
	return res
}

func isPrefixedTestVM(name string) bool {
	return strings.HasPrefix(name, "cli-val-") ||
		strings.HasPrefix(name, "val-") || // Added for new test pattern
		strings.HasPrefix(name, "vm_template_src") ||
		strings.HasPrefix(name, "vm_from_template")
}

func precleanTemplates(ctx *Context) report.StepResult {
	args := []string{"template", "list", "--json"}
	res := runNido(ctx, "preclean-templates", args, 15*time.Second)
	addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)
	payload, err := parseJSON(res.Stdout)
	if err != nil {
		addAssertion(&res, "json_parse", false, err.Error())
		finalize(&res)
		return res
	}
	toDelete := []string{}
	if data, ok := payload["data"].(map[string]interface{}); ok {
		if arr, ok := data["templates"].([]interface{}); ok {
			for _, t := range arr {
				if m, ok := t.(map[string]interface{}); ok {
					if name, ok := m["name"].(string); ok && strings.HasPrefix(name, "tpl_primary") {
						toDelete = append(toDelete, name)
					}
				}
			}
		}
	}
	for _, tpl := range toDelete {
		del := runNido(ctx, "preclean-template-delete", []string{"template", "delete", tpl, "--force"}, 30*time.Second)
		finalize(&del)
		_ = ctx.Reporter.WriteStep(del)
		if del.Result == "PASS" {
			ctx.State.AddTemplate(tpl)
		}
	}
	addAssertion(&res, "deleted_templates", true, strings.Join(toDelete, ","))
	finalize(&res)
	return res
}
