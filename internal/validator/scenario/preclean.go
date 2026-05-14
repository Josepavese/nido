package scenario

import (
	"strings"
	"time"

	"github.com/Josepavese/nido/internal/validator/report"
)

// PreClean removes stale validator-owned resources before running the suite.
func PreClean() Scenario {
	return Scenario{
		Name: "preclean",
		Steps: []Step{
			precleanStep,
			precleanTemplates,
		},
	}
}

// Sweep performs a best-effort cleanup of validator-owned artifacts.
// It is designed to be safe to run from signal handlers or cleanup hooks.
func Sweep(ctx *Context) {
	snapshot := ctx.State.Snapshot()

	// 1. List all VMs
	args := []string{"list", "--json"}
	// Use a short timeout for listing
	res := runNido(ctx, "sweep-list", args, 5*time.Second)
	vmSet := map[string]bool{}
	for _, name := range snapshot.VMs {
		if isValidatorGeneratedVMName(name) {
			vmSet[name] = true
		}
	}
	if res.ExitCode == 0 {
		payload, err := parseJSON(res.Stdout)
		if err == nil {
			if data, ok := payload["data"].(map[string]interface{}); ok {
				if arr, ok := data["vms"].([]interface{}); ok {
					for _, v := range arr {
						if m, ok := v.(map[string]interface{}); ok {
							if name, ok := m["name"].(string); ok && isValidatorGeneratedVMName(name) {
								vmSet[name] = true
							}
						}
					}
				}
			}
		}
	}

	// 2. Destroy them all
	for name := range vmSet {
		runDeleteValidatorVM(ctx, name, 10*time.Second)
	}

	templateSet := map[string]bool{}
	for _, name := range snapshot.Templates {
		if isValidatorGeneratedTemplateName(name) {
			templateSet[name] = true
		}
	}

	tplRes := runNido(ctx, "sweep-template-list", []string{"template", "list", "--json"}, 5*time.Second)
	if tplRes.ExitCode == 0 {
		payload, err := parseJSON(tplRes.Stdout)
		if err == nil {
			if data, ok := payload["data"].(map[string]interface{}); ok {
				if arr, ok := data["templates"].([]interface{}); ok {
					for _, t := range arr {
						name := templateName(t)
						if isValidatorGeneratedTemplateName(name) {
							templateSet[name] = true
						}
					}
				}
			}
		}
	}

	for name := range templateSet {
		runDeleteValidatorTemplate(ctx, name, 10*time.Second)
	}
}

func precleanStep(ctx *Context) report.StepResult {
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
					if name, ok := m["name"].(string); ok && isValidatorGeneratedVMName(name) {
						vms = append(vms, name)
					}
				}
			}
		}
	}
	for _, name := range vms {
		del := runDeleteValidatorVM(ctx, name, 30*time.Second)
		finalize(&del)
		_ = ctx.Reporter.WriteStep(del)
	}
	addAssertion(&res, "deleted_validator_vms", true, strings.Join(vms, ","))
	finalize(&res)
	return res
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
				name := templateName(t)
				if isValidatorGeneratedTemplateName(name) {
					toDelete = append(toDelete, name)
				}
			}
		}
	}
	for _, tpl := range toDelete {
		del := runDeleteValidatorTemplate(ctx, tpl, 30*time.Second)
		finalize(&del)
		_ = ctx.Reporter.WriteStep(del)
		if del.Result == "PASS" {
			ctx.State.AddTemplate(tpl)
		}
	}
	addAssertion(&res, "deleted_validator_templates", true, strings.Join(toDelete, ","))
	finalize(&res)
	return res
}
