package scenario

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/Josepavese/nido/internal/validator/report"
	"github.com/Josepavese/nido/internal/validator/runner"
)

func runNido(ctx *Context, name string, args []string, timeout time.Duration) report.StepResult {
	inv := runner.Invocation{
		Command: ctx.Config.NidoBin,
		Args:    args,
		Timeout: timeout,
		Workdir: ctx.Config.WorkingDir,
		Env:     buildEnv(ctx),
	}
	execRes := ctx.Runner.Exec(inv)
	res := buildResult(name, inv, execRes)
	return res
}

func buildEnv(ctx *Context) map[string]string {
	env := map[string]string{}
	if ctx.Config.UpdateURL != "" {
		env["NIDO_UPDATE_URL"] = ctx.Config.UpdateURL
	}
	if ctx.Config.UpdateReleaseAPI != "" {
		env["NIDO_RELEASE_API"] = ctx.Config.UpdateReleaseAPI
	}
	return env
}

func buildResult(name string, inv runner.Invocation, execRes runner.Result) report.StepResult {
	return report.StepResult{
		Command:    inv.Command,
		Args:       inv.Args,
		Cwd:        inv.Workdir,
		ExitCode:   execRes.ExitCode,
		DurationMs: execRes.Duration.Milliseconds(),
		TimedOut:   execRes.TimedOut,
		Stdout:     execRes.Stdout,
		Stderr:     execRes.Stderr,
		Result:     "FAIL", // default, adjust later
		StartedAt:  execRes.StartTime,
	}
}

func addAssertion(res *report.StepResult, name string, ok bool, details string) {
	ar := report.AssertionResult{
		Name: name,
	}
	if ok {
		ar.Result = "PASS"
	} else {
		ar.Result = "FAIL"
	}
	if details != "" {
		ar.Details = details
	}
	res.Assertions = append(res.Assertions, ar)
}

func skipResult(cmd string, args []string, reason string) report.StepResult {
	return report.StepResult{
		Command:    cmd,
		Args:       args,
		Result:     "SKIP",
		DurationMs: 0,
		ExitCode:   0,
		Stdout:     "",
		Stderr:     reason,
		StartedAt:  time.Now(),
	}
}

func parseJSON(body string) (map[string]interface{}, error) {
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func mustGet(m map[string]interface{}, key string) (interface{}, error) {
	v, ok := m[key]
	if !ok {
		return nil, fmt.Errorf("missing key %s", key)
	}
	return v, nil
}

func finalize(res *report.StepResult) {
	if res.Result == "SKIP" {
		return
	}
	if res.TimedOut {
		res.Result = "FAIL"
		return
	}
	hasFail := res.ExitCode != 0
	for _, ar := range res.Assertions {
		if ar.Result == "FAIL" {
			hasFail = true
			break
		}
	}
	if hasFail {
		res.Result = "FAIL"
	} else {
		res.Result = "PASS"
	}
}

func setVar(ctx *Context, key, val string) {
	if ctx.Vars == nil {
		ctx.Vars = map[string]string{}
	}
	ctx.Vars[key] = val
}

func getVar(ctx *Context, key string) (string, bool) {
	if ctx.Vars == nil {
		return "", false
	}
	val, ok := ctx.Vars[key]
	return val, ok
}

func getVarOrDefault(ctx *Context, key, def string) string {
	if v, ok := getVar(ctx, key); ok {
		return v
	}
	return def
}

func selectTemplateFallback(ctx *Context, list []interface{}) {
	if ctx.Config.BaseTemplate == "none" {
		return
	}
	if ctx.Config.BaseTemplate != "" {
		found := false
		for _, t := range list {
			if name, ok := t.(map[string]interface{})["name"]; ok && name == ctx.Config.BaseTemplate {
				found = true
				break
			}
		}
		if found {
			ctx.State.AddTemplate(ctx.Config.BaseTemplate)
		}
		return
	}
	if len(list) == 0 {
		return
	}

	// Choose the smallest template by size_bytes if present; otherwise first.
	var chosenName string
	var chosenSize float64 = -1
	for _, t := range list {
		if m, ok := t.(map[string]interface{}); ok {
			name, _ := m["name"].(string)
			size, _ := m["size_bytes"].(float64)
			if chosenName == "" || (size > 0 && (chosenSize < 0 || size < chosenSize)) {
				chosenName = name
				chosenSize = size
			}
		}
	}
	if chosenName == "" {
		if name, ok := list[0].(map[string]interface{})["name"].(string); ok {
			chosenName = name
		}
	}
	if chosenName != "" {
		setVar(ctx, "template_auto", chosenName)
		ctx.State.AddTemplate(chosenName)
	}
}

func selectImageFallback(ctx *Context, images []interface{}) {
	var chosenName string
	var chosenSize float64 = -1
	for _, it := range images {
		m, ok := it.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := m["name"].(string)
		if name == "" {
			continue
		}
		version := ""
		if v, ok := m["version"].(string); ok && v != "" {
			version = v
		} else if latest, ok := m["latest"].(map[string]interface{}); ok {
			if v, ok := latest["version"].(string); ok {
				version = v
			}
			if sz, ok := latest["size_bytes"].(float64); ok {
				m["size_bytes"] = sz
			}
		} else if vers, ok := m["versions"].([]interface{}); ok && len(vers) > 0 {
			if first, ok := vers[0].(map[string]interface{}); ok {
				if v, ok := first["version"].(string); ok {
					version = v
				}
				if sz, ok := first["size_bytes"].(float64); ok {
					m["size_bytes"] = sz
				}
			}
		}
		candidate := name
		if version != "" {
			candidate = fmt.Sprintf("%s:%s", name, version)
		}

		size := -1.0
		if sz, ok := m["size_bytes"].(float64); ok {
			size = sz
		}
		if chosenName == "" || (size > 0 && (chosenSize < 0 || size < chosenSize)) {
			chosenName = candidate
			if size > 0 {
				chosenSize = size
			}
		}
	}
	if chosenName != "" {
		setVar(ctx, "auto_image", chosenName)
		if ctx.Config.BaseImage == "" {
			ctx.Config.BaseImage = chosenName
		}
		if ctx.Config.PoolImage == "" {
			ctx.Config.PoolImage = chosenName
		}
	}
}

func reservePort(ctx *Context, start int) (int, error) {
	for port := start; port < start+50; port++ {
		l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			continue
		}
		l.Close()
		ctx.State.AddPort(port)
		return port, nil
	}
	return 0, fmt.Errorf("no free port in range starting %d", start)
}

func waitForPort(host string, port string, timeout time.Duration) error {
	addr := net.JoinHostPort(host, port)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("port %s not ready within %s", addr, timeout.String())
}
