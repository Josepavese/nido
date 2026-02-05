package scenario

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Josepavese/nido/internal/validator/report"
	"github.com/Josepavese/nido/internal/validator/runner"
	"github.com/Josepavese/nido/internal/validator/util"
)

// VMMutableConfig covers the 'nido config' command and 'vm_config_update' MCP tool.
func VMMutableConfig() Scenario {
	return Scenario{
		Name: "vm-mutable-config",
		Steps: []Step{
			spawnConfigVM,
			configCLI,
			configMCP,
			deleteConfigVM,
		},
	}
}

func spawnConfigVM(ctx *Context) report.StepResult {
	vmName := util.RandomName("val-cfg-vm")
	setVar(ctx, "vm_config", vmName)

	args := []string{"spawn", vmName}
	if ctx.Config.BaseImage != "" {
		args = append(args, "--image", ctx.Config.BaseImage)
	}
	// We start with default config
	args = append(args, "--json")

	res := runNido(ctx, "spawn", args, ctx.Config.BootTimeout)
	addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)

	if res.Result != "FAIL" {
		ctx.State.AddVM(vmName)
	}
	finalize(&res)
	return res
}

func configCLI(ctx *Context) report.StepResult {
	vmName, ok := getVar(ctx, "vm_config")
	if !ok {
		return skipResult(ctx.Config.NidoBin, []string{"config"}, "vm_config not set")
	}

	// Change RAM to 1024, CPUs to 2, Valid Ports
	args := []string{"config", vmName, "--memory", "1024", "--cpus", "2", "--ssh-port", "60022", "--port", "8080:80", "--json"}
	res := runNido(ctx, "config", args, 10*time.Second)
	addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)

	// Verify Output
	if payload, err := parseJSON(res.Stdout); err == nil {
		status, _ := payload["status"]
		addAssertion(&res, "status_ok", status == "ok", "Response status not OK")
		if data, ok := payload["data"].(map[string]interface{}); ok {
			addAssertion(&res, "result_updated", data["result"] == "updated", fmt.Sprintf("Result: %v", data["result"]))
			addAssertion(&res, "vm_name_match", data["name"] == vmName, "VM Name mismatch")
		} else {
			addAssertion(&res, "data_present", false, "Response data missing")
		}
	} else {
		addAssertion(&res, "json_parse", false, err.Error())
	}

	// Verify State Persistence by running 'info'
	infoArgs := []string{"info", vmName, "--json"}
	infoRes := runNido(ctx, "info", infoArgs, 5*time.Second)
	if payload, err := parseJSON(infoRes.Stdout); err == nil {
		if data, ok := payload["data"].(map[string]interface{}); ok {
			if vm, ok := data["vm"].(map[string]interface{}); ok {
				// Check memory
				mem, _ := vm["memory_mb"].(float64)
				addAssertion(&res, "mem_persisted", mem == 1024, fmt.Sprintf("Expected 1024, got %v", mem))
				// Check cpus
				cpus, _ := vm["vcpus"].(float64)
				addAssertion(&res, "cpus_persisted", cpus == 2, fmt.Sprintf("Expected 2, got %v", cpus))
				// Check ssh port
				sshPort, _ := vm["ssh_port"].(float64)
				addAssertion(&res, "ssh_port_persisted", sshPort == 60022, fmt.Sprintf("Expected 60022, got %v", sshPort))

				// Check Forwarding
				if fw, ok := vm["forwarding"].([]interface{}); ok {
					found := false
					for _, f := range fw {
						if fm, ok := f.(map[string]interface{}); ok {
							gp, _ := fm["guest_port"].(float64)
							if gp == 80 {
								hp, _ := fm["host_port"].(float64)
								if hp == 8080 {
									found = true
								}
							}
						}
					}
					addAssertion(&res, "port_forward_persisted", found, "Expected 8080->80 mapping")
				}
			}
		}
	}

	finalize(&res)
	return res
}

func configMCP(ctx *Context) report.StepResult {
	vmName, ok := getVar(ctx, "vm_config")
	if !ok {
		return skipResult(ctx.Config.NidoBin, []string{"mcp"}, "vm_config not set")
	}

	// Prepare JSON-RPC request to change Memory to 512 via MCP
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "vm_config_update",
			"arguments": map[string]interface{}{
				"name":      vmName,
				"memory_mb": 512,
				"gui":       true,
			},
		},
		"id": 1,
	}
	reqBytes, _ := json.Marshal(req)
	reqStr := string(reqBytes)

	// Run MCP
	start := time.Now()
	inv := runner.Invocation{
		Command: ctx.Config.NidoBin,
		Args:    []string{"mcp"},
		Stdin:   reqStr,
		Timeout: 5 * time.Second,
	}
	out := ctx.Runner.Exec(inv)

	res := report.StepResult{
		Command:    "mcp-tool:vm_config_update",
		Args:       nil,
		StartedAt:  start,
		Result:     "PASS", // Optimistic
		DurationMs: time.Since(start).Milliseconds(),
		Stdout:     out.Stdout,
		Stderr:     out.Stderr,
		ExitCode:   out.ExitCode,
	}

	if out.ExitCode != 0 {
		res.Result = "FAIL"
		addAssertion(&res, "mcp_exit_zero", false, out.Stderr)
		return res
	}

	// Check response
	// Expect: {"jsonrpc":"2.0","id":1,"result":{...}}
	if strings.Contains(out.Stdout, "updated") {
		addAssertion(&res, "mcp_response_ok", true, "")
	} else {
		addAssertion(&res, "mcp_response_ok", false, "Response did not contain success message")
	}

	// Verify State Persistence again via 'info'
	infoArgs := []string{"info", vmName, "--json"}
	infoRes := runNido(ctx, "info", infoArgs, 5*time.Second)
	if payload, err := parseJSON(infoRes.Stdout); err == nil {
		if data, ok := payload["data"].(map[string]interface{}); ok {
			if vm, ok := data["vm"].(map[string]interface{}); ok {
				// Check memory (should be 512 now)
				mem, _ := vm["memory_mb"].(float64)
				addAssertion(&res, "mcp_mem_persisted", mem == 512, fmt.Sprintf("Expected 512, got %v", mem))
				// Check gui (should be true)
				gui, _ := vm["gui"].(bool)
				addAssertion(&res, "mcp_gui_persisted", gui == true, fmt.Sprintf("Expected true, got %v", gui))
			}
		}
	}

	finalize(&res)
	return res
}

func deleteConfigVM(ctx *Context) report.StepResult {
	vmName, ok := getVar(ctx, "vm_config")
	if !ok {
		return skipResult(ctx.Config.NidoBin, []string{"delete"}, "vm_config not set")
	}

	args := []string{"delete", vmName, "--json"}
	res := runNido(ctx, "delete", args, 10*time.Second)
	addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)

	if res.Result == "PASS" {
		ctx.State.RemoveVM(vmName)
	}

	finalize(&res)
	return res
}
