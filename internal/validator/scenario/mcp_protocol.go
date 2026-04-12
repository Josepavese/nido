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
		"nido_vm":       true,
		"nido_template": true,
		"nido_image":    true,
		"nido_system":   true,
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

	// resources/list
	resourcesResp, err := client.CallMethod("resources/list", nil, 10*time.Second)
	addAssertion(&res, "resources_list", err == nil, errDetails(err))
	if err == nil {
		if resources, ok := resourcesResp["result"].(map[string]interface{})["resources"].([]interface{}); ok {
			addAssertion(&res, "resources_nonempty", len(resources) >= 6, "")
		} else {
			addAssertion(&res, "resources_parse", false, "resources array missing")
		}
	}

	// prompts/list
	promptsResp, err := client.CallMethod("prompts/list", nil, 10*time.Second)
	addAssertion(&res, "prompts_list", err == nil, errDetails(err))
	if err == nil {
		if prompts, ok := promptsResp["result"].(map[string]interface{})["prompts"].([]interface{}); ok {
			addAssertion(&res, "prompts_nonempty", len(prompts) >= 1, "")
		} else {
			addAssertion(&res, "prompts_parse", false, "prompts array missing")
		}
	}

	// positive tool call nido_vm list
	if _, err := client.CallWithTimeout("nido_vm", map[string]interface{}{"action": "list"}, 10*time.Second); err != nil {
		addAssertion(&res, "nido_vm_list_call", false, err.Error())
	} else {
		addAssertion(&res, "nido_vm_list_call", true, "")
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
