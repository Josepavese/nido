package scenario

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Josepavese/nido/internal/validator/report"
	"github.com/Josepavese/nido/internal/validator/runner"
	"github.com/Josepavese/nido/internal/validator/util"
)

// VMSpawnResources verifies that custom memory and CPU can be set during nido spawn (CLI & MCP).
func VMSpawnResources() Scenario {
	return Scenario{
		Name: "vm-spawn-resources",
		Steps: []Step{
			spawnResourcesCLI,
			spawnResourcesCLI,
			spawnResourcesMCP,
			spawnResourcesMCP,
			spawnResourcesRawArgs,
			spawnResourcesAccelAuto,
		},
	}
}

func spawnResourcesCLI(ctx *Context) report.StepResult {
	vmName := util.RandomName("val-spawn-res-cli")

	args := []string{"spawn", vmName, "--memory", "1024", "--cpus", "2"}
	if ctx.Config.BaseImage != "" {
		args = append(args, "--image", ctx.Config.BaseImage)
	}
	args = append(args, "--json")

	res := runNido(ctx, "spawn", args, ctx.Config.BootTimeout)
	addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)

	if res.Result != "FAIL" {
		ctx.State.AddVM(vmName)

		// Verify via info
		infoArgs := []string{"info", vmName, "--json"}
		infoRes := runNido(ctx, "info", infoArgs, 5*time.Second)
		if payload, err := parseJSON(infoRes.Stdout); err == nil {
			if data, ok := payload["data"].(map[string]interface{}); ok {
				if vm, ok := data["vm"].(map[string]interface{}); ok {
					mem, _ := vm["memory_mb"].(float64)
					addAssertion(&res, "cli_mem_match", mem == 1024, fmt.Sprintf("Expected 1024, got %v", mem))
					cpus, _ := vm["vcpus"].(float64)
					addAssertion(&res, "cli_cpus_match", cpus == 2, fmt.Sprintf("Expected 2, got %v", cpus))
				}
			}
		} else {
			addAssertion(&res, "info_json_parse", false, err.Error())
		}

		// Cleanup
		runNido(ctx, "delete", []string{"delete", vmName, "--json"}, 10*time.Second)
		ctx.State.RemoveVM(vmName)
	}

	finalize(&res)
	return res
}

func spawnResourcesMCP(ctx *Context) report.StepResult {
	vmName := util.RandomName("val-spawn-res-mcp")

	// Prepare MCP request
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "vm_create",
			"arguments": map[string]interface{}{
				"name":      vmName,
				"image":     ctx.Config.BaseImage,
				"memory_mb": 512,
				"vcpus":     1,
			},
		},
		"id": 1,
	}
	reqBytes, _ := json.Marshal(req)

	start := time.Now()
	inv := runner.Invocation{
		Command: ctx.Config.NidoBin,
		Args:    []string{"mcp"},
		Stdin:   string(reqBytes),
		Timeout: ctx.Config.BootTimeout,
	}
	out := ctx.Runner.Exec(inv)

	res := report.StepResult{
		Command:    "mcp-tool:vm_create",
		Args:       nil,
		StartedAt:  start,
		Result:     "PASS",
		DurationMs: time.Since(start).Milliseconds(),
		Stdout:     out.Stdout,
		Stderr:     out.Stderr,
		ExitCode:   out.ExitCode,
	}

	if out.ExitCode != 0 {
		res.Result = "FAIL"
		addAssertion(&res, "mcp_exit_zero", false, out.Stderr)
	} else {
		// Verify MCP Response Content
		if strings.Contains(out.Stdout, "created successfully") {
			addAssertion(&res, "mcp_response_ok", true, "")
		} else {
			addAssertion(&res, "mcp_response_ok", false, "Response missing success message")
		}

		ctx.State.AddVM(vmName)

		// Verify via info
		infoArgs := []string{"info", vmName, "--json"}
		infoRes := runNido(ctx, "info", infoArgs, 5*time.Second)
		if payload, err := parseJSON(infoRes.Stdout); err == nil {
			if data, ok := payload["data"].(map[string]interface{}); ok {
				if vm, ok := data["vm"].(map[string]interface{}); ok {
					mem, _ := vm["memory_mb"].(float64)
					addAssertion(&res, "mcp_mem_match", mem == 512, fmt.Sprintf("Expected 512, got %v", mem))
					cpus, _ := vm["vcpus"].(float64)
					addAssertion(&res, "mcp_cpus_match", cpus == 1, fmt.Sprintf("Expected 1, got %v", cpus))
				}
			}
		} else {
			addAssertion(&res, "info_json_parse", false, err.Error())
		}

		// Cleanup
		runNido(ctx, "delete", []string{"delete", vmName, "--json"}, 10*time.Second)
		ctx.State.RemoveVM(vmName)
	}

	finalize(&res)
	return res
}

func spawnResourcesRawArgs(ctx *Context) report.StepResult {
	vmName := util.RandomName("val-spawn-raw")
	marker := fmt.Sprintf("guest=%s", vmName)

	args := []string{"spawn", vmName, "--qemu-arg", "-name", "--qemu-arg", marker}
	if ctx.Config.BaseImage != "" {
		args = append(args, "--image", ctx.Config.BaseImage)
	}
	args = append(args, "--json")

	res := runNido(ctx, "spawn", args, ctx.Config.BootTimeout)
	addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)

	if res.Result != "FAIL" {
		ctx.State.AddVM(vmName)

		// Verify process args using pgrep
		// We prefer pgrep -f over ps aux parsing for stability
		cmd := exec.Command("pgrep", "-f", marker)
		out, err := cmd.CombinedOutput()

		found := err == nil && len(out) > 0
		addAssertion(&res, "qemu_arg_found", found, fmt.Sprintf("pgrep output: %s", string(out)))

		// Verify via info
		infoArgs := []string{"info", vmName, "--json"}
		infoRes := runNido(ctx, "info", infoArgs, 5*time.Second)
		if payload, err := parseJSON(infoRes.Stdout); err == nil {
			if data, ok := payload["data"].(map[string]interface{}); ok {
				if vm, ok := data["vm"].(map[string]interface{}); ok {
					raw, ok := vm["raw_qemu_args"].([]interface{})
					if ok && len(raw) >= 2 {
						// We expect "-name" and "guest=..." to be in there somewhere
						// Exact match might be tricky if order is preserved (it should be)
						match := false
						for _, r := range raw {
							if s, ok := r.(string); ok && s == marker {
								match = true
								break
							}
						}
						addAssertion(&res, "info_raw_arg_match", match, "Raw arg not found in info output")
					} else {
						addAssertion(&res, "info_raw_arg_present", false, "raw_qemu_args missing or empty")
					}
				}
			}
		}

		// Cleanup
		runNido(ctx, "delete", []string{"delete", vmName, "--json"}, 10*time.Second)
		ctx.State.RemoveVM(vmName)
	}

	finalize(&res)
	return res
}

func spawnResourcesAccelAuto(ctx *Context) report.StepResult {
	vmName := util.RandomName("val-spawn-accel")

	// We test --accel auto specifically
	args := []string{"spawn", vmName, "--accel", "auto"}
	if ctx.Config.BaseImage != "" {
		args = append(args, "--image", ctx.Config.BaseImage)
	}
	args = append(args, "--json")

	// Cleanup MUST happen regardless of success/fail
	defer func() {
		// We use a separate context or just fire-and-forget delete
		// Note: we can't use the original ctx if it's cancelled, but here ctx is likely fine.
		runNido(ctx, "delete", []string{"delete", vmName, "--json"}, 10*time.Second)
		ctx.State.RemoveVM(vmName)
	}()

	res := runNido(ctx, "spawn", args, ctx.Config.BootTimeout)
	addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)

	if res.Result != "FAIL" {
		ctx.State.AddVM(vmName)

		// Verify via info
		infoArgs := []string{"info", vmName, "--json"}
		infoRes := runNido(ctx, "info", infoArgs, 5*time.Second)
		if payload, err := parseJSON(infoRes.Stdout); err == nil {
			if data, ok := payload["data"].(map[string]interface{}); ok {
				if vm, ok := data["vm"].(map[string]interface{}); ok {
					accels, ok := vm["accelerators"].([]interface{})
					if ok && len(accels) > 0 {
						addAssertion(&res, "accel_field_populated", true, fmt.Sprintf("Found %d accelerators", len(accels)))
					} else {
						// On some systems it might fail to find GPU and fallback to virtual:gpu
						// which SHOULD be in the list as "virtual:gpu"
						addAssertion(&res, "accel_field_populated", false, "accelerators list empty, expected at least virtual:gpu")
					}
				}
			}
		} else {
			addAssertion(&res, "info_json_parse", false, err.Error())
		}
	}

	finalize(&res)
	return res
}
