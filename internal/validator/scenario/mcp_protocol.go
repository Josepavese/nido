package scenario

import (
	"time"

	"github.com/Josepavese/nido/internal/validator/mcpclient"
	"github.com/Josepavese/nido/internal/validator/report"
)

// MCPProtocol validates MCP protocol basics: initialize, tools/list contents, tool call, negative tool.
func MCPProtocol() Scenario {
	return Scenario{
		Name: "mcp-protocol",
		Steps: []Step{
			mcpProtocolStep,
		},
	}
}

func mcpProtocolStep(ctx *Context) report.StepResult {
	client, err := mcpclient.Start(ctx.Config.NidoBin)
	if err != nil {
		return report.StepResult{
			Command:   "mcp.start",
			Result:    "FAIL",
			Stderr:    err.Error(),
			StartedAt: time.Now(),
		}
	}
	defer client.Stop()

	res := report.StepResult{
		Command:   "mcp.protocol",
		Result:    "PASS",
		StartedAt: time.Now(),
	}

	// tools/list
	toolsResp, err := client.CallMethod("tools/list", nil, 10*time.Second)
	addAssertion(&res, "tools_list", err == nil, errDetails(err))
	expected := map[string]bool{
		"vm_list":            true,
		"vm_create":          true,
		"vm_start":           true,
		"vm_stop":            true,
		"vm_delete":          true,
		"vm_info":            true,
		"vm_ssh":             true,
		"vm_prune":           true,
		"vm_template_list":   true,
		"vm_template_create": true,
		"vm_template_delete": true,
		"vm_images_list":     true,
		"vm_images_info":     true,
		"vm_images_pull":     true,
		"vm_images_remove":   true,
		"vm_images_update":   true,
		"vm_cache_list":      true,
		"vm_cache_info":      true,
		"vm_cache_remove":    true,
		"vm_cache_prune":     true,
		"vm_port_forward":    true,
		"vm_port_unforward":  true,
		"vm_port_list":       true,
	}
	if err == nil {
		if tools, ok := toolsResp["result"].(map[string]interface{})["tools"].([]interface{}); ok {
			found := map[string]bool{}
			for _, t := range tools {
				if m, ok := t.(map[string]interface{}); ok {
					if name, ok := m["name"].(string); ok {
						found[name] = true
					}
				}
			}
			missing := false
			for name := range expected {
				if !found[name] {
					missing = true
				}
			}
			addAssertion(&res, "tools_expected", !missing, "")
		} else {
			addAssertion(&res, "tools_parse", false, "tools array missing")
		}
	}

	// positive tool call vm_list
	if _, err := client.CallWithTimeout("vm_list", map[string]interface{}{}, 10*time.Second); err != nil {
		addAssertion(&res, "vm_list_call", false, err.Error())
	} else {
		addAssertion(&res, "vm_list_call", true, "")
	}

	// negative tool call
	if _, err := client.CallWithTimeout("nonexistent_tool", map[string]interface{}{}, 5*time.Second); err != nil {
		addAssertion(&res, "unknown_tool_error", true, "")
	} else {
		addAssertion(&res, "unknown_tool_error", false, "expected error for unknown tool")
	}

	finalize(&res)
	return res
}
