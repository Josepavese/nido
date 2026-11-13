package scenario

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/Josepavese/nido/v4/internal/validator/report"
)

// ShutdownTimeout proves the negative shutdown contract with a real guest: a
// VM started with ACPI disabled and with SSH stopped cannot be asked to power
// off. `shutdown` must time out, report an error, leave the VM running and never
// force-terminate it; the explicit forced stop then cleans it up.
func ShutdownTimeout() Scenario {
	return Scenario{
		Name: "shutdown-timeout",
		Steps: []Step{
			shutdownTimeoutSpawn,
			waitReadyVM,
			shutdownTimeoutDisableSSH,
			shutdownTimeoutVerifySSHClosed,
			shutdownTimeoutInvoke,
			shutdownTimeoutStillRunning,
			stopVM,
			deleteVM,
		},
	}
}

func shutdownTimeoutSpawn(ctx *Context) report.StepResult {
	if runtime.GOARCH != "amd64" {
		return skipResult(ctx.Config.NidoBin, []string{"spawn"}, "shutdown-timeout requires x86 ACPI semantics")
	}
	vmName := validatorRandomName("shutdown-timeout")
	setVar(ctx, "vm_primary", vmName)
	args := []string{"spawn", vmName}
	if tpl := chooseTemplate(ctx); tpl != "" {
		args = append(args, tpl)
	} else if ctx.Config.BaseImage != "" {
		args = append(args, "--image", ctx.Config.BaseImage)
	} else {
		// Standalone runs do not execute the image-pool scenario, so choose a
		// known systemd guest instead of spawning from an empty source.
		args = append(args, "--image", "ubuntu:24.04")
	}
	args = append(args, "--qemu-arg=-no-acpi", "--json")
	res := runNido(ctx, "spawn-no-acpi", args, ctx.Config.BootTimeout)
	addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)
	if payload, err := parseJSON(res.Stdout); err == nil {
		addAssertion(&res, "json_parse", true, "")
		addAssertion(&res, "status_ok", payload["status"] == "ok", "")
		if data, ok := payload["data"].(map[string]interface{}); ok {
			if action, ok := data["action"].(map[string]interface{}); ok {
				addAssertion(&res, "action_spawned", action["result"] == "spawned", fmt.Sprintf("%v", action["result"]))
			}
		}
	} else {
		addAssertion(&res, "json_parse", false, err.Error())
	}
	if res.ExitCode == 0 {
		ctx.State.AddVM(vmName)
	}
	finalize(&res)
	return res
}

func shutdownTimeoutDisableSSH(ctx *Context) report.StepResult {
	vmName, ok := getVar(ctx, "vm_primary")
	if !ok {
		return skipResult(ctx.Config.NidoBin, []string{"ssh"}, "vm_primary not set")
	}
	// Try the common service managers with direct argv, avoiding nested shell
	// quoting on minimal guests. Verification below is authoritative: stopping
	// SSH can also close the command session with a non-zero status.
	attempts := [][]string{
		{"sudo", "-n", "systemctl", "mask", "--runtime", "--now", "ssh.socket", "ssh.service"},
		{"sudo", "-n", "/etc/init.d/S50dropbear", "stop"},
		{"sudo", "-n", "/etc/init.d/sshd", "stop"},
		{"sudo", "-n", "pkill", "-9", "-f", "sshd"},
	}
	var last report.StepResult
	timedOut := false
	for _, remote := range attempts {
		args := append([]string{"ssh", vmName, "--"}, remote...)
		last = runNido(ctx, "ssh-disable", args, 20*time.Second)
		if last.TimedOut {
			timedOut = true
			break
		}
	}
	addAssertion(&last, "command_completed", !timedOut, last.Stderr)
	time.Sleep(2 * time.Second)
	finalizeExpectedFailure(&last)
	return last
}

func shutdownTimeoutVerifySSHClosed(ctx *Context) report.StepResult {
	vmName, ok := getVar(ctx, "vm_primary")
	if !ok {
		return skipResult(ctx.Config.NidoBin, []string{"ssh"}, "vm_primary not set")
	}
	args := []string{"ssh", vmName, "--", "true"}
	res := runNido(ctx, "ssh-closed", args, 20*time.Second)
	addAssertion(&res, "ssh_unreachable", res.ExitCode != 0, "SSH still reached the guest")
	finalizeExpectedFailure(&res)
	return res
}

func shutdownTimeoutInvoke(ctx *Context) report.StepResult {
	vmName, ok := getVar(ctx, "vm_primary")
	if !ok {
		return skipResult(ctx.Config.NidoBin, []string{"shutdown"}, "vm_primary not set")
	}
	args := []string{"shutdown", vmName, "--timeout", "10", "--json"}
	res := runNido(ctx, "shutdown-timeout", args, 40*time.Second)
	addAssertion(&res, "exit_nonzero", res.ExitCode != 0, "shutdown unexpectedly succeeded")
	if payload, err := parseJSON(res.Stdout); err == nil {
		addAssertion(&res, "json_parse", true, "")
		addAssertion(&res, "status_error", payload["status"] == "error", fmt.Sprintf("%v", payload["status"]))
		if errorBody, ok := payload["error"].(map[string]interface{}); ok {
			addAssertion(&res, "error_code", errorBody["code"] == "ERR_SHUTDOWN_FAILED", fmt.Sprintf("%v", errorBody["code"]))
			detail := fmt.Sprintf("%v", errorBody["detail"])
			addAssertion(&res, "timeout_reported", strings.Contains(detail, "did not power off") || strings.Contains(detail, "powerdown requested"), detail)
		} else {
			addAssertion(&res, "error_body", false, "error object missing")
		}
	} else {
		addAssertion(&res, "json_parse", false, err.Error())
	}
	finalizeExpectedFailure(&res)
	return res
}

func shutdownTimeoutStillRunning(ctx *Context) report.StepResult {
	vmName, ok := getVar(ctx, "vm_primary")
	if !ok {
		return skipResult(ctx.Config.NidoBin, []string{"info"}, "vm_primary not set")
	}
	res := runNido(ctx, "info-after-shutdown-timeout", []string{"info", vmName, "--json"}, 30*time.Second)
	addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)
	if payload, err := parseJSON(res.Stdout); err == nil {
		data, _ := payload["data"].(map[string]interface{})
		vm, _ := data["vm"].(map[string]interface{})
		addAssertion(&res, "json_parse", true, "")
		addAssertion(&res, "state_running", vm["state"] == "running", fmt.Sprintf("%v", vm["state"]))
		// info intentionally omits PID in some runtime snapshots; running state
		// is the contract assertion here, process identity is provider-owned.
	} else {
		addAssertion(&res, "json_parse", false, err.Error())
	}
	finalize(&res)
	return res
}
