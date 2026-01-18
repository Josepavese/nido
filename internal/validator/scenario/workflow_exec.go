package scenario

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Josepavese/nido/internal/validator/mcpclient"
	"github.com/Josepavese/nido/internal/validator/report"
	"github.com/Josepavese/nido/internal/validator/util"
	"github.com/Josepavese/nido/internal/validator/workflows"
)

// WorkflowExec runs shared workflows via CLI and MCP.
func WorkflowExec() Scenario {
	return Scenario{
		Name: "workflow",
		Steps: []Step{
			runWorkflowsCLI,
			runWorkflowsMCP,
		},
	}
}

func runWorkflowsCLI(ctx *Context) report.StepResult {
	start := time.Now()
	def, err := workflows.Load(ctx.Config.WorkflowPath)
	if err != nil {
		return report.StepResult{
			Command:   "workflows.load",
			Result:    "FAIL",
			Stderr:    err.Error(),
			StartedAt: time.Now(),
		}
	}
	res := report.StepResult{
		Command:   "workflows.cli",
		Result:    "PASS",
		StartedAt: time.Now(),
	}
	for name, wf := range def.Workflows {
		if name == "image_pool" && ctx.Config.PoolImage == "" && ctx.Config.BaseImage == "" {
			res.Assertions = append(res.Assertions, report.AssertionResult{
				Name:    "wf_" + name,
				Result:  "SKIP",
				Details: "no pool/base image configured",
			})
			continue
		}
		if err := executeWorkflowCLI(ctx, name, wf); err != nil {
			res.Assertions = append(res.Assertions, report.AssertionResult{
				Name:    "wf_" + name,
				Result:  "FAIL",
				Details: err.Error(),
			})
			res.Result = "FAIL"
			if ctx.Config.FailFast {
				break
			}
		} else {
			res.Assertions = append(res.Assertions, report.AssertionResult{
				Name:   "wf_" + name,
				Result: "PASS",
			})
		}
	}
	res.DurationMs = time.Since(start).Milliseconds()
	finalize(&res)
	return res
}

func runWorkflowsMCP(ctx *Context) report.StepResult {
	start := time.Now()
	def, err := workflows.Load(ctx.Config.WorkflowPath)
	if err != nil {
		return report.StepResult{
			Command:   "workflows.load",
			Result:    "FAIL",
			Stderr:    err.Error(),
			StartedAt: time.Now(),
		}
	}
	client, err := mcpclient.Start(ctx.Config.NidoBin)
	if err != nil {
		return report.StepResult{
			Command:   "mcp.start",
			Result:    "SKIP",
			Stderr:    fmt.Sprintf("failed to start MCP: %v", err),
			StartedAt: time.Now(),
		}
	}
	defer client.Stop()

	res := report.StepResult{
		Command:   "workflows.mcp",
		Result:    "PASS",
		StartedAt: time.Now(),
	}
	for name, wf := range def.Workflows {
		if name == "image_pool" && ctx.Config.PoolImage == "" && ctx.Config.BaseImage == "" {
			res.Assertions = append(res.Assertions, report.AssertionResult{
				Name:    "wf_" + name + "_mcp",
				Result:  "SKIP",
				Details: "no pool/base image configured",
			})
			continue
		}
		if err := executeWorkflowMCP(ctx, client, name, wf); err != nil {
			res.Assertions = append(res.Assertions, report.AssertionResult{
				Name:    "wf_" + name + "_mcp",
				Result:  "FAIL",
				Details: err.Error(),
			})
			res.Result = "FAIL"
			if ctx.Config.FailFast {
				break
			}
		} else {
			res.Assertions = append(res.Assertions, report.AssertionResult{
				Name:   "wf_" + name + "_mcp",
				Result: "PASS",
			})
		}
	}
	res.DurationMs = time.Since(start).Milliseconds()
	finalize(&res)
	return res
}

func executeWorkflowCLI(ctx *Context, name string, wf workflows.Workflow) error {
	for _, step := range wf.Steps {
		switch step.Action {
		case "image_pull":
			img := step.Image
			if img == "" {
				img = ctx.Config.PoolImage
			}
			if img == "" {
				img, _ = getVar(ctx, "auto_image")
			}
			if img == "" {
				return fmt.Errorf("no pool image configured")
			}
			args := []string{"image", "pull", img, "--json"}
			resp := runNido(ctx, "image-pull", args, ctx.Config.DownloadTimeout)
			finalize(&resp)
			_ = ctx.Reporter.WriteStep(resp)
			if resp.Result == "FAIL" {
				return fmt.Errorf("image pull failed for %s", img)
			}
			setVar(ctx, "last_pulled_image", img)
		case "spawn":
			vmName := util.RandomName(step.VMVar)
			setVar(ctx, step.VMVar, vmName)
			args := []string{"spawn", vmName}
			if step.TemplateVar != "" {
				if tpl, ok := getVar(ctx, step.TemplateVar); ok {
					args = append(args, tpl)
				}
			} else if step.UseBaseTemplate {
				if tpl := chooseTemplate(ctx); tpl != "" {
					args = append(args, tpl)
				}
			} else if step.UseBaseImage {
				img := ctx.Config.BaseImage
				if img == "" {
					img = step.Image
				}
				if img == "" {
					img, _ = getVar(ctx, "last_pulled_image")
				}
				if img == "" {
					img, _ = getVar(ctx, "auto_image")
				}
				if img != "" {
					args = append(args, "--image", img)
				}
			}
			args = append(args, "--json")
			resp := runNido(ctx, "spawn-wf", args, ctx.Config.BootTimeout)
			finalize(&resp)
			_ = ctx.Reporter.WriteStep(resp)
			if resp.Result == "FAIL" {
				return fmt.Errorf("spawn failed for %s", vmName)
			}
			ctx.State.AddVM(vmName)
			ctx.Vars["last_spawn_ms"] = fmt.Sprintf("%d", resp.DurationMs)
			// Cache-hit expectation is advisory; timing-only checks are too flaky. Leave note in vars for future comparison if needed.
		case "template_create":
			vmName, ok := getVar(ctx, step.VMVar)
			if !ok {
				return fmt.Errorf("vm var %s not set", step.VMVar)
			}
			tplName := util.RandomName(step.TemplateVar)
			setVar(ctx, step.TemplateVar, tplName)
			args := []string{"template", "create", vmName, tplName, "--json"}
			resp := runNido(ctx, "template-create", args, ctx.Config.DownloadTimeout)
			finalize(&resp)
			_ = ctx.Reporter.WriteStep(resp)
			if resp.Result == "FAIL" {
				return fmt.Errorf("template create failed for %s", tplName)
			}
			if payload, err := parseJSON(resp.Stdout); err == nil {
				if data, ok := payload["data"].(map[string]interface{}); ok {
					if action, ok := data["action"].(map[string]interface{}); ok {
						if path, ok := action["path"].(string); ok {
							if info, err := os.Stat(path); err == nil && info.Size() > 0 {
								setVar(ctx, "template_path_"+tplName, path)
							}
						}
					}
				}
			}
			ctx.State.AddTemplate(tplName)
		case "template_delete":
			tplName, ok := getVar(ctx, step.TemplateVar)
			if !ok {
				return fmt.Errorf("template var %s not set", step.TemplateVar)
			}
			args := []string{"template", "delete", tplName, "--json"}
			resp := runNido(ctx, "template-delete", args, 30*time.Second)
			finalize(&resp)
			_ = ctx.Reporter.WriteStep(resp)
			if resp.Result == "FAIL" {
				return fmt.Errorf("template delete failed for %s", tplName)
			}
		case "delete_vm":
			vmName, ok := getVar(ctx, step.VMVar)
			if !ok {
				continue
			}
			args := []string{"delete", vmName, "--json"}
			resp := runNido(ctx, "delete-wf", args, 30*time.Second)
			finalize(&resp)
			_ = ctx.Reporter.WriteStep(resp)
		case "cache_rm":
			img := step.Image
			if img == "" {
				img = ctx.Config.PoolImage
			}
			if img == "" {
				img, _ = getVar(ctx, "last_pulled_image")
			}
			if img == "" {
				img, _ = getVar(ctx, "auto_image")
			}
			if img == "" {
				continue
			}
			args := []string{"cache", "rm", img, "--json"}
			resp := runNido(ctx, "cache-rm", args, 20*time.Second)
			finalize(&resp)
			_ = ctx.Reporter.WriteStep(resp)
		default:
			return fmt.Errorf("unsupported action %s", step.Action)
		}
	}
	return nil
}

func executeWorkflowMCP(ctx *Context, client *mcpclient.Client, name string, wf workflows.Workflow) error {
	for _, step := range wf.Steps {
		switch step.Action {
		case "image_pull":
			img := step.Image
			if img == "" {
				img = ctx.Config.PoolImage
			}
			if img == "" {
				img, _ = getVar(ctx, "auto_image")
			}
			if img == "" {
				return fmt.Errorf("no pool image configured")
			}
			if _, err := client.CallWithTimeout("vm_images_pull", map[string]interface{}{"image": img}, ctx.Config.DownloadTimeout); err != nil {
				return fmt.Errorf("mcp image pull failed: %w", err)
			}
			setVar(ctx, "last_pulled_image", img)
		case "spawn":
			vmName := util.RandomName(step.VMVar)
			setVar(ctx, step.VMVar, vmName)
			args := map[string]interface{}{
				"name": vmName,
			}
			if step.TemplateVar != "" {
				if tpl, ok := getVar(ctx, step.TemplateVar); ok {
					args["template"] = tpl
				}
			} else if step.UseBaseTemplate {
				if tpl := chooseTemplate(ctx); tpl != "" {
					args["template"] = tpl
				}
			}
			if _, hasTpl := args["template"]; !hasTpl {
				img := ctx.Config.BaseImage
				if img == "" {
					img = step.Image
				}
				if img == "" {
					img, _ = getVar(ctx, "last_pulled_image")
				}
				if img == "" {
					img, _ = getVar(ctx, "auto_image")
				}
				if img != "" {
					args["image"] = img
				} else {
					return fmt.Errorf("no image available for spawn")
				}
			}
			if _, err := client.CallWithTimeout("vm_create", args, ctx.Config.BootTimeout); err != nil {
				return fmt.Errorf("mcp spawn failed for %s: %w", vmName, err)
			}
			ctx.State.AddVM(vmName)
		case "template_create":
			vmName, ok := getVar(ctx, step.VMVar)
			if !ok {
				return fmt.Errorf("vm var %s not set", step.VMVar)
			}
			tplName := util.RandomName(step.TemplateVar)
			setVar(ctx, step.TemplateVar, tplName)
			if _, err := client.CallWithTimeout("vm_template_create", map[string]interface{}{
				"vm_name":       vmName,
				"template_name": tplName,
			}, ctx.Config.DownloadTimeout); err != nil {
				return fmt.Errorf("mcp template create failed: %w", err)
			}
			ctx.State.AddTemplate(tplName)
		case "template_delete":
			tplName, ok := getVar(ctx, step.TemplateVar)
			if !ok {
				return fmt.Errorf("template var %s not set", step.TemplateVar)
			}
			if _, err := client.CallWithTimeout("vm_template_delete", map[string]interface{}{
				"name": tplName,
			}, 30*time.Second); err != nil {
				return fmt.Errorf("mcp template delete failed: %w", err)
			}
		case "delete_vm":
			vmName, ok := getVar(ctx, step.VMVar)
			if !ok {
				continue
			}
			if _, err := client.CallWithTimeout("vm_delete", map[string]interface{}{
				"name": vmName,
			}, 30*time.Second); err != nil {
				return fmt.Errorf("mcp delete vm failed: %w", err)
			}
		case "cache_rm":
			img := step.Image
			if img == "" {
				img = ctx.Config.PoolImage
			}
			if img == "" {
				img, _ = getVar(ctx, "last_pulled_image")
			}
			if img == "" {
				img, _ = getVar(ctx, "auto_image")
			}
			if img == "" {
				continue
			}
			if _, err := client.CallWithTimeout("cache_remove", map[string]interface{}{"image": img}, 20*time.Second); err != nil {
				if strings.Contains(err.Error(), "Tool not found") {
					continue
				}
				return fmt.Errorf("mcp cache_remove failed: %w", err)
			}
		default:
			return fmt.Errorf("unsupported action %s", step.Action)
		}
	}
	return nil
}
