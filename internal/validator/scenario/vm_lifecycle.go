package scenario

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Josepavese/nido/internal/validator/report"
	"github.com/Josepavese/nido/internal/validator/runner"
	"github.com/Josepavese/nido/internal/validator/util"
)

// VMLifecycle covers spawn/info/list/stop/start/delete/prune.
func VMLifecycle() Scenario {
	return Scenario{
		Name: "vm-lifecycle",
		Steps: []Step{
			spawnVM,
			infoVM,
			listVM,
			sshCheck,
			stopVM,
			startVM,
			deleteVM,
			pruneVM,
		},
	}
}

func spawnVM(ctx *Context) report.StepResult {
	vmName := util.RandomName("cli-val-vm")
	setVar(ctx, "vm_primary", vmName)

	args := []string{"spawn", vmName}
	if tpl := chooseTemplate(ctx); tpl != "" {
		args = append(args, tpl)
	} else if ctx.Config.BaseImage != "" {
		args = append(args, "--image", ctx.Config.BaseImage)
	}

	hostPort, err := reservePort(ctx, ctx.Config.FWPortBase)
	if err != nil {
		return report.StepResult{
			Command:   ctx.Config.NidoBin,
			Args:      []string{"spawn"},
			Result:    "FAIL",
			Stderr:    err.Error(),
			StartedAt: time.Now(),
		}
	}
	portFlag := fmt.Sprintf("http:80:%d/tcp", hostPort)
	args = append(args, "--port", portFlag)
	setVar(ctx, "vm_primary_host_port", fmt.Sprintf("%d", hostPort))

	if userDataPath, err := ensureUserData(ctx); err == nil && userDataPath != "" {
		args = append(args, "--user-data", userDataPath)
	} else if err != nil {
		res := skipResult(ctx.Config.NidoBin, args, "failed to prepare user-data: "+err.Error())
		return res
	}
	args = append(args, "--json")

	res := runNido(ctx, "spawn", args, ctx.Config.BootTimeout)
	addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)
	if payload, err := parseJSON(res.Stdout); err == nil {
		addAssertion(&res, "json_parse", true, "")
		status, _ := payload["status"]
		addAssertion(&res, "status_ok", status == "ok", "")
		if data, ok := payload["data"].(map[string]interface{}); ok {
			if action, ok := data["action"].(map[string]interface{}); ok {
				addAssertion(&res, "action_spawned", action["result"] == "spawned", fmt.Sprintf("%v", action["result"]))
			}
		}
	} else {
		addAssertion(&res, "json_parse", false, err.Error())
	}

	if res.Result != "FAIL" {
		ctx.State.AddVM(vmName)
	}
	finalize(&res)
	return res
}

func infoVM(ctx *Context) report.StepResult {
	vmName, ok := getVar(ctx, "vm_primary")
	if !ok {
		return skipResult(ctx.Config.NidoBin, []string{"info"}, "vm_primary not set")
	}
	args := []string{"info", vmName, "--json"}
	res := runNido(ctx, "info", args, 30*time.Second)
	addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)
	if payload, err := parseJSON(res.Stdout); err == nil {
		addAssertion(&res, "json_parse", true, "")
		if data, ok := payload["data"].(map[string]interface{}); ok {
			if vm, ok := data["vm"].(map[string]interface{}); ok {
				addAssertion(&res, "name_match", vm["name"] == vmName, "")
				addAssertion(&res, "state_present", vm["state"] != nil, "")
				if sshPort, ok := vm["ssh_port"].(float64); ok {
					setVar(ctx, "vm_primary_ssh_port", fmt.Sprintf("%.0f", sshPort))
				}
				if user, ok := vm["ssh_user"].(string); ok {
					setVar(ctx, "vm_primary_ssh_user", user)
				}
				if ip, ok := vm["ip"].(string); ok && ip != "" {
					setVar(ctx, "vm_primary_ip", ip)
				}
				expectedHost := getVarOrDefault(ctx, "vm_primary_host_port", "")
				if fwd, ok := vm["forwarding"].([]interface{}); ok && len(fwd) > 0 {
					hostMatch := expectedHost == ""
					for _, entry := range fwd {
						if m, ok := entry.(map[string]interface{}); ok {
							if host, ok := m["host_port"]; ok && fmt.Sprintf("%.0f", host.(float64)) == expectedHost {
								hostMatch = true
							}
						}
					}
					addAssertion(&res, "forwarding_present", true, "")
					addAssertion(&res, "forwarding_host_match", hostMatch, "host port mismatch")
				} else {
					addAssertion(&res, "forwarding_present", true, "forwarding not reported in info")
				}
			} else {
				addAssertion(&res, "vm_object", false, "missing vm object")
			}
		} else {
			addAssertion(&res, "data_object", false, "data not object")
		}
	} else {
		addAssertion(&res, "json_parse", false, err.Error())
	}
	finalize(&res)
	return res
}

func listVM(ctx *Context) report.StepResult {
	vmName, ok := getVar(ctx, "vm_primary")
	if !ok {
		return skipResult(ctx.Config.NidoBin, []string{"list"}, "vm_primary not set")
	}
	args := []string{"list", "--json"}
	res := runNido(ctx, "list", args, 15*time.Second)
	addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)
	if payload, err := parseJSON(res.Stdout); err == nil {
		addAssertion(&res, "json_parse", true, "")
		found := false
		if data, ok := payload["data"].(map[string]interface{}); ok {
			if vms, ok := data["vms"].([]interface{}); ok {
				for _, v := range vms {
					if m, ok := v.(map[string]interface{}); ok && m["name"] == vmName {
						found = true
						break
					}
				}
			}
		}
		addAssertion(&res, "vm_listed", found, "vm not found in list")
	} else {
		addAssertion(&res, "json_parse", false, err.Error())
	}
	finalize(&res)
	return res
}

func chooseTemplate(ctx *Context) string {
	if ctx.Config.BaseTemplate != "" {
		return ctx.Config.BaseTemplate
	}
	if auto, ok := getVar(ctx, "template_auto"); ok {
		return auto
	}
	return ""
}

func sshCheck(ctx *Context) report.StepResult {
	start := time.Now()
	sshPort, okPort := getVar(ctx, "vm_primary_ssh_port")
	sshUser, okUser := getVar(ctx, "vm_primary_ssh_user")
	host := "127.0.0.1"
	if ip, ok := getVar(ctx, "vm_primary_ip"); ok && ip != "" {
		host = ip
	}
	if !okPort || !okUser {
		return skipResult("ssh", []string{}, "ssh metadata missing")
	}

	if err := waitForPort(host, sshPort, ctx.Config.BootTimeout); err != nil {
		return report.StepResult{
			Command:    "ssh",
			Args:       []string{},
			Result:     "FAIL",
			Stderr:     err.Error(),
			DurationMs: time.Since(start).Milliseconds(),
			StartedAt:  time.Now(),
		}
	}

	args := []string{
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=10",
		"-p", sshPort,
		fmt.Sprintf("%s@%s", sshUser, host),
		"--", "echo", "ok",
	}

	var last report.StepResult
	for attempt := 0; attempt < 5; attempt++ {
		inv := runner.Invocation{
			Command: "ssh",
			Args:    args,
			Timeout: 20 * time.Second,
		}
		execRes := ctx.Runner.Exec(inv)
		res := report.StepResult{
			Command:    inv.Command,
			Args:       inv.Args,
			ExitCode:   execRes.ExitCode,
			DurationMs: execRes.Duration.Milliseconds(),
			TimedOut:   execRes.TimedOut,
			Stdout:     execRes.Stdout,
			Stderr:     execRes.Stderr,
			Result:     "FAIL",
			StartedAt:  execRes.StartTime,
		}
		addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)
		addAssertion(&res, "stdout_ok", res.Stdout != "" && containsOK(res.Stdout), res.Stdout)
		finalize(&res)
		if res.Result == "PASS" {
			// Optional cloud-init marker check
			if ctx.Config.CheckCloudInit {
				markerRes := runSSHCommand(ctx, sshPort, sshUser, host, "cat /tmp/nido-cli-validate-marker", 10*time.Second)
				addAssertion(&res, "cloud_init_marker", markerRes.ExitCode == 0, markerRes.Stderr)
			}
			// Optional forwarding connectivity check
			if ctx.Config.CheckForward {
				if hostPort := getVarOrDefault(ctx, "vm_primary_host_port", ""); hostPort != "" {
					runSSHCommand(ctx, sshPort, sshUser, host, "nohup python3 -m http.server 80 >/tmp/http.log 2>&1 &", 5*time.Second)
					if err := waitForPort("127.0.0.1", hostPort, 10*time.Second); err == nil {
						addAssertion(&res, "forward_dial", true, "")
					} else {
						addAssertion(&res, "forward_dial", false, err.Error())
					}
					// best-effort cleanup
					runSSHCommand(ctx, sshPort, sshUser, host, "pkill -f http.server || true", 5*time.Second)
				}
			}
			return res
		}
		last = res
		time.Sleep(3 * time.Second)
	}
	last.DurationMs = time.Since(start).Milliseconds()
	return last
}

func runSSHCommand(ctx *Context, port, user, host, cmd string, timeout time.Duration) runner.Result {
	args := []string{
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=10",
		"-p", port,
		fmt.Sprintf("%s@%s", user, host),
		"--", cmd,
	}
	return ctx.Runner.Exec(runner.Invocation{
		Command: "ssh",
		Args:    args,
		Timeout: timeout,
	})
}

func stopVM(ctx *Context) report.StepResult {
	vmName, ok := getVar(ctx, "vm_primary")
	if !ok {
		return skipResult(ctx.Config.NidoBin, []string{"stop"}, "vm_primary not set")
	}
	args := []string{"stop", vmName, "--json"}
	res := runNido(ctx, "stop", args, 30*time.Second)
	addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)
	finalize(&res)
	return res
}

func startVM(ctx *Context) report.StepResult {
	vmName, ok := getVar(ctx, "vm_primary")
	if !ok {
		return skipResult(ctx.Config.NidoBin, []string{"start"}, "vm_primary not set")
	}
	args := []string{"start", vmName, "--json"}
	res := runNido(ctx, "start", args, ctx.Config.BootTimeout)
	addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)
	finalize(&res)
	return res
}

func deleteVM(ctx *Context) report.StepResult {
	vmName, ok := getVar(ctx, "vm_primary")
	if !ok {
		return skipResult(ctx.Config.NidoBin, []string{"delete"}, "vm_primary not set")
	}
	args := []string{"delete", vmName, "--json"}
	res := runNido(ctx, "delete", args, 30*time.Second)
	addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)
	finalize(&res)
	return res
}

func pruneVM(ctx *Context) report.StepResult {
	args := []string{"prune", "--json"}
	res := runNido(ctx, "prune", args, 15*time.Second)
	addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)
	finalize(&res)
	return res
}

func ensureUserData(ctx *Context) (string, error) {
	if ctx.Config.UserDataPath != "" {
		return ctx.Config.UserDataPath, nil
	}
	f, err := os.CreateTemp("", "cli-val-userdata-*.yaml")
	if err != nil {
		return "", err
	}
	content := "#cloud-config\nwrite_files:\n  - path: /tmp/nido-cli-validate-marker\n    content: \"ok\"\n"
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	ctx.State.AddTempFile(f.Name())
	return f.Name(), nil
}

func containsOK(out string) bool {
	return strings.Contains(out, "ok") || strings.Contains(out, "OK")
}
