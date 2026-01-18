package scenario

import (
	"time"

	"github.com/Josepavese/nido/internal/validator/report"
)

// ImageCacheTemplate scenario runs read-only checks on images, cache, and templates.
func ImageCacheTemplate() Scenario {
	return Scenario{
		Name: "image-cache-template",
		Steps: []Step{
			imageListStep,
			imagePullStep,
			cacheInfoStep,
			cacheListStep,
			templateListStep,
		},
	}
}

func imageListStep(ctx *Context) report.StepResult {
	args := []string{"image", "list", "--json"}
	res := runNido(ctx, "image-list", args, ctx.Config.DownloadTimeout)
	addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)
	payload, err := parseJSON(res.Stdout)
	addAssertion(&res, "json_parse", err == nil, errDetails(err))
	if err == nil {
		status, _ := mustGet(payload, "status")
		addAssertion(&res, "status_ok", status == "ok", "")
		data, _ := mustGet(payload, "data")
		if m, ok := data.(map[string]interface{}); ok {
			if images, ok := m["images"].([]interface{}); ok {
				addAssertion(&res, "images_present", true, "")
				selectImageFallback(ctx, images)
			} else {
				addAssertion(&res, "images_present", false, "missing images array")
			}
		} else {
			addAssertion(&res, "data_object", false, "data not object")
		}
	}
	finalize(&res)
	return res
}

func imagePullStep(ctx *Context) report.StepResult {
	img := ctx.Config.BaseImage
	if img == "" {
		img, _ = getVar(ctx, "auto_image")
	}
	if img == "" {
		return skipResult(ctx.Config.NidoBin, []string{"image", "pull"}, "no BaseImage configured; skipping pull")
	}
	args := []string{"image", "pull", img, "--json"}
	res := runNido(ctx, "image-pull", args, ctx.Config.DownloadTimeout)
	addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)
	payload, err := parseJSON(res.Stdout)
	addAssertion(&res, "json_parse", err == nil, errDetails(err))
	if err == nil {
		status, _ := mustGet(payload, "status")
		addAssertion(&res, "status_ok", status == "ok", "")
	}
	finalize(&res)
	return res
}

func cacheInfoStep(ctx *Context) report.StepResult {
	args := []string{"cache", "info", "--json"}
	res := runNido(ctx, "cache-info", args, 15*time.Second)
	addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)
	if payload, err := parseJSON(res.Stdout); err == nil {
		addAssertion(&res, "json_parse", true, "")
		if data, ok := payload["data"].(map[string]interface{}); ok {
			_, hasStats := data["stats"]
			addAssertion(&res, "stats_present", hasStats, "missing stats")
		} else {
			addAssertion(&res, "data_object", false, "data not object")
		}
	} else {
		addAssertion(&res, "json_parse", false, err.Error())
	}
	finalize(&res)
	return res
}

func cacheListStep(ctx *Context) report.StepResult {
	args := []string{"cache", "ls", "--json"}
	res := runNido(ctx, "cache-list", args, 15*time.Second)
	addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)
	if payload, err := parseJSON(res.Stdout); err == nil {
		addAssertion(&res, "json_parse", true, "")
		if data, ok := payload["data"].(map[string]interface{}); ok {
			_, hasCache := data["cache"]
			addAssertion(&res, "cache_present", hasCache, "missing cache array")
		} else {
			addAssertion(&res, "data_object", false, "data not object")
		}
	} else {
		addAssertion(&res, "json_parse", false, err.Error())
	}
	finalize(&res)
	return res
}

func templateListStep(ctx *Context) report.StepResult {
	args := []string{"template", "list", "--json"}
	res := runNido(ctx, "template-list", args, 15*time.Second)
	addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)
	if payload, err := parseJSON(res.Stdout); err == nil {
		addAssertion(&res, "json_parse", true, "")
		if data, ok := payload["data"].(map[string]interface{}); ok {
			list, ok := data["templates"].([]interface{})
			addAssertion(&res, "templates_present", ok, "missing templates list")
			selectTemplateFallback(ctx, list)
		} else {
			addAssertion(&res, "data_object", false, "data not object")
		}
	} else {
		addAssertion(&res, "json_parse", false, err.Error())
	}
	finalize(&res)
	return res
}
