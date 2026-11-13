package scenario

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Josepavese/nido/v4/internal/validator/mcpclient"
	"github.com/Josepavese/nido/v4/internal/validator/report"
)

const mcpDurableSnapshot = "before upgrade"

// MCPDurableSessions exercises the durable-session contract over a real MCP
// stdio subprocess: create, wait, guest shutdown, disk snapshot lifecycle,
// restart, running-snapshot rejection and delete.
func MCPDurableSessions() Scenario {
	return Scenario{
		Name: "mcp-durable",
		Steps: []Step{
			mcpDurableStep,
		},
	}
}

func mcpDurableStep(ctx *Context) report.StepResult {
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
		Command:   "mcp.durable",
		Result:    "PASS",
		StartedAt: time.Now(),
	}
	template := chooseTemplate(ctx)
	image := ctx.Config.BaseImage
	if template == "" && image == "" {
		return skipResult(ctx.Config.NidoBin, []string{"mcp", "durable"}, "template/image not configured")
	}

	name := validatorRandomName("mcp-durable")
	ctx.State.AddVM(name)
	deleted := false
	defer func() {
		if deleted {
			return
		}
		// Best-effort MCP cleanup for a failed step; the global sweep also sees
		// the VM through ctx.State.
		_, _ = client.CallTool("nido_vm", map[string]interface{}{"action": "stop", "name": name})
		_, _ = client.CallTool("nido_vm", map[string]interface{}{"action": "delete", "name": name})
	}()
	call := func(action string, args map[string]interface{}, timeout time.Duration) (map[string]interface{}, error) {
		args["action"] = action
		args["name"] = name
		return client.CallWithTimeout("nido_vm", args, timeout)
	}
	assertOK := func(label string, payload map[string]interface{}, err error) bool {
		if err != nil || payload["isError"] == true {
			addAssertion(&res, label, false, mcpToolFailure(payload, err))
			return false
		}
		addAssertion(&res, label, true, "")
		return true
	}

	createArgs := map[string]interface{}{}
	if template != "" {
		createArgs["template"] = template
	} else {
		createArgs["image"] = image
	}
	created, err := call("create", createArgs, 120*time.Second)
	if !assertOK("create", created, err) {
		finalize(&res)
		return res
	}
	if payload, ok := mcpToolPayload(created); ok {
		addAssertion(&res, "create_status", payload["status"] == "created", fmt.Sprintf("%v", payload["status"]))
	} else {
		addAssertion(&res, "create_payload", false, "missing MCP payload")
	}

	waitSeconds := int(ctx.Config.BootTimeout / time.Second)
	if waitSeconds <= 0 || waitSeconds > 300 {
		waitSeconds = 240
	}
	waited, err := call("wait", map[string]interface{}{"timeout_seconds": waitSeconds}, time.Duration(waitSeconds+30)*time.Second)
	if !assertOK("wait", waited, err) {
		finalize(&res)
		return res
	}
	if payload, ok := mcpToolPayload(waited); ok {
		addAssertion(&res, "wait_ready", payload["ready"] == true, fmt.Sprintf("%v", payload["ready"]))
	} else {
		addAssertion(&res, "wait_payload", false, "missing MCP payload")
	}

	shutdown, err := call("shutdown", map[string]interface{}{"timeout_seconds": 120}, 150*time.Second)
	if !assertOK("shutdown", shutdown, err) {
		finalize(&res)
		return res
	}
	if payload, ok := mcpToolPayload(shutdown); ok {
		addAssertion(&res, "shutdown_stopped", payload["status"] == "stopped", fmt.Sprintf("%v", payload["status"]))
		if detail, ok := payload["shutdown"].(map[string]interface{}); ok {
			addAssertion(&res, "shutdown_guest_mode", detail["mode"] == "guest", fmt.Sprintf("%v", detail["mode"]))
			addAssertion(&res, "shutdown_method", detail["method"] == "ssh" || detail["method"] == "acpi", fmt.Sprintf("%v", detail["method"]))
		} else {
			addAssertion(&res, "shutdown_detail", false, "missing shutdown detail")
		}
	} else {
		addAssertion(&res, "shutdown_payload", false, "missing MCP payload")
	}

	snapshotArgs := map[string]interface{}{"snapshot_name": mcpDurableSnapshot}
	snapshotCreate, err := call("snapshot_create", snapshotArgs, 30*time.Second)
	if !assertOK("snapshot_create", snapshotCreate, err) {
		finalize(&res)
		return res
	}
	snapshotList, err := call("snapshot_list", map[string]interface{}{}, 30*time.Second)
	if assertOK("snapshot_list", snapshotList, err) {
		if payload, ok := mcpToolPayload(snapshotList); ok {
			addAssertion(&res, "snapshot_listed", snapshotPayloadContains(payload, mcpDurableSnapshot), "")
		} else {
			addAssertion(&res, "snapshot_list_payload", false, "missing MCP payload")
		}
	}
	snapshotRestore, err := call("snapshot_restore", snapshotArgs, 30*time.Second)
	if assertOK("snapshot_restore", snapshotRestore, err) {
		if payload, ok := mcpToolPayload(snapshotRestore); ok {
			addAssertion(&res, "snapshot_restored", payload["status"] == "restored", fmt.Sprintf("%v", payload["status"]))
		}
	}
	snapshotDelete, err := call("snapshot_delete", snapshotArgs, 30*time.Second)
	if assertOK("snapshot_delete", snapshotDelete, err) {
		if payload, ok := mcpToolPayload(snapshotDelete); ok {
			addAssertion(&res, "snapshot_deleted", payload["status"] == "deleted", fmt.Sprintf("%v", payload["status"]))
		}
	}
	emptyList, err := call("snapshot_list", map[string]interface{}{}, 30*time.Second)
	if assertOK("snapshot_list_empty", emptyList, err) {
		if payload, ok := mcpToolPayload(emptyList); ok {
			addAssertion(&res, "snapshot_list_empty_result", !snapshotPayloadContains(payload, mcpDurableSnapshot), "")
		}
	}

	start, err := call("start", map[string]interface{}{}, 120*time.Second)
	if !assertOK("start", start, err) {
		finalize(&res)
		return res
	}
	restarted, err := call("wait", map[string]interface{}{"timeout_seconds": waitSeconds}, time.Duration(waitSeconds+30)*time.Second)
	if assertOK("wait_after_start", restarted, err) {
		if payload, ok := mcpToolPayload(restarted); ok {
			addAssertion(&res, "wait_after_start_ready", payload["ready"] == true, fmt.Sprintf("%v", payload["ready"]))
		}
	}
	runningSnapshot, err := call("snapshot_create", snapshotArgs, 30*time.Second)
	if err != nil {
		addAssertion(&res, "snapshot_running_rejected", false, err.Error())
	} else {
		addAssertion(&res, "snapshot_running_rejected", runningSnapshot["isError"] == true, "")
	}

	// Leave the VM stopped before the MCP delete action, then remove it through
	// the same transport under test.
	shutdown2, err := call("shutdown", map[string]interface{}{"timeout_seconds": 120}, 150*time.Second)
	if assertOK("shutdown_before_delete", shutdown2, err) {
		deletedResp, deleteErr := call("delete", map[string]interface{}{}, 30*time.Second)
		if assertOK("delete", deletedResp, deleteErr) {
			if payload, ok := mcpToolPayload(deletedResp); ok && payload["status"] == "deleted" {
				deleted = true
				ctx.State.RemoveVM(name)
				addAssertion(&res, "delete_status", true, "")
			} else {
				addAssertion(&res, "delete_status", false, "delete did not report deleted")
			}
		}
	}

	finalize(&res)
	return res
}

func mcpToolPayload(result map[string]interface{}) (map[string]interface{}, bool) {
	content, ok := result["content"].([]interface{})
	if !ok || len(content) == 0 {
		return nil, false
	}
	first, ok := content[0].(map[string]interface{})
	if !ok {
		return nil, false
	}
	text, ok := first["text"].(string)
	if !ok || text == "" {
		return nil, false
	}
	var payload map[string]interface{}
	if json.Unmarshal([]byte(text), &payload) != nil {
		return nil, false
	}
	return payload, true
}

func mcpToolFailure(result map[string]interface{}, err error) string {
	if err != nil {
		return err.Error()
	}
	if payload, ok := mcpToolPayload(result); ok {
		return fmt.Sprintf("%v", payload["error"])
	}
	return "MCP tool returned isError"
}

func snapshotPayloadContains(payload map[string]interface{}, snapshot string) bool {
	snapshots, ok := payload["snapshots"].([]interface{})
	if !ok {
		return false
	}
	for _, raw := range snapshots {
		entry, ok := raw.(map[string]interface{})
		if ok && entry["name"] == snapshot {
			return true
		}
	}
	return false
}
