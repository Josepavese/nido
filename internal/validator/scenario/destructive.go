package scenario

import (
	"time"

	"github.com/Josepavese/nido/internal/validator/report"
)

func runDeleteValidatorVM(ctx *Context, name string, timeout time.Duration) report.StepResult {
	args := []string{"delete", name, "--json"}
	if !isValidatorGeneratedVMName(name) {
		return skipResult(ctx.Config.NidoBin, args, "refusing to delete non-validator VM: "+name)
	}
	return runNido(ctx, "delete", args, timeout)
}

func runDeleteValidatorTemplate(ctx *Context, name string, timeout time.Duration) report.StepResult {
	args := []string{"template", "delete", name, "--json"}
	if !isValidatorGeneratedTemplateName(name) {
		return skipResult(ctx.Config.NidoBin, args, "refusing to delete non-validator template: "+name)
	}
	return runNido(ctx, "template-delete", args, timeout)
}
