package scenario

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Josepavese/nido/internal/pkg/sysutil"
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
			cmdlineTest,
			stopVM,
			startVM,
			deleteVM,
			pruneVM,
		},
	}
}

func spawnVM(ctx *Context) report.StepResult {
	vmName := validatorRandomName("vm-lifecycle")
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
		status := payload["status"]
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
				if pwd, ok := vm["ssh_password"].(string); ok {
					setVar(ctx, "vm_primary_ssh_password", pwd)
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
	if ctx.Config.BaseTemplate == "none" {
		return ""
	}
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
	sshPort, _ := getVar(ctx, "vm_primary_ssh_port")
	sshUser, _ := getVar(ctx, "vm_primary_ssh_user")
	sshUser = resolveSSHUser(ctx, sshUser)
	host := "127.0.0.1"
	if ip, ok := getVar(ctx, "vm_primary_ip"); ok && ip != "" {
		host = ip
	}
	if sshPort == "" || sshUser == "" {
		return skipResult("ssh", []string{}, "ssh metadata missing (port or user)")
	}

	waitTimeout := ctx.Config.BootTimeout
	if waitTimeout <= 0 {
		waitTimeout = 3 * time.Minute
	}

	inv, execRes, attempts, err := waitForSSHCommand(ctx, sshPort, sshUser, host, "echo ok", waitTimeout, 10*time.Second, func(res runner.Result) bool {
		return res.ExitCode == 0 && !res.TimedOut && containsOK(res.Stdout)
	})
	res := buildResult("ssh", inv, execRes)
	addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)
	addAssertion(&res, "stdout_ok", res.Stdout != "" && containsOK(res.Stdout), res.Stdout)
	if err != nil {
		res.Stderr = appendDiagnostic(res.Stderr, fmt.Sprintf("SSH readiness failed after %d attempts within %s: %s", attempts, waitTimeout, err.Error()))
	}
	res.DurationMs = time.Since(start).Milliseconds()
	finalize(&res)
	if res.Result != "PASS" {
		return res
	}

	if attempts > 1 {
		addAssertion(&res, "ssh_ready_retries", true, fmt.Sprintf("attempts=%d", attempts))
	}
	if ctx.Config.CheckCloudInit {
		markerRes := runSSHCommand(ctx, sshPort, sshUser, host, "cat /tmp/nido-cli-validate-marker", 10*time.Second)
		addAssertion(&res, "cloud_init_marker", markerRes.ExitCode == 0 && containsOK(markerRes.Stdout), appendDiagnostic(markerRes.Stderr, markerRes.Stdout))
	}
	if ctx.Config.CheckForward {
		if hostPort := getVarOrDefault(ctx, "vm_primary_host_port", ""); hostPort != "" {
			runSSHCommand(ctx, sshPort, sshUser, host, "nohup python3 -m http.server 80 >/tmp/http.log 2>&1 &", 5*time.Second)
			if err := waitForPort("127.0.0.1", hostPort, 10*time.Second); err == nil {
				addAssertion(&res, "forward_dial", true, "")
			} else {
				addAssertion(&res, "forward_dial", false, err.Error())
			}
			runSSHCommand(ctx, sshPort, sshUser, host, "pkill -f http.server || true", 5*time.Second)
		}
	}
	res.DurationMs = time.Since(start).Milliseconds()
	finalize(&res)
	return res
}

func runSSHCommand(ctx *Context, port, user, host, cmd string, timeout time.Duration) runner.Result {
	inv, err := buildSSHInvocation(ctx, port, user, host, cmd, timeout)
	if err != nil {
		return runner.Result{
			Stderr:    err.Error(),
			ExitCode:  -1,
			StartTime: time.Now(),
		}
	}
	return ctx.Runner.Exec(inv)
}

func waitForSSHCommand(ctx *Context, port, user, host, cmd string, waitTimeout, attemptTimeout time.Duration, ready func(runner.Result) bool) (runner.Invocation, runner.Result, int, error) {
	if waitTimeout <= 0 {
		waitTimeout = 3 * time.Minute
	}
	if attemptTimeout <= 0 {
		attemptTimeout = 10 * time.Second
	}
	if ready == nil {
		ready = func(res runner.Result) bool {
			return res.ExitCode == 0 && !res.TimedOut
		}
	}

	deadline := time.Now().Add(waitTimeout)
	var lastInv runner.Invocation
	var lastRes runner.Result
	attempts := 0
	for {
		remaining := time.Until(deadline)
		if attempts > 0 && remaining <= 0 {
			break
		}
		timeout := attemptTimeout
		if remaining > 0 && remaining < timeout {
			timeout = remaining
		}

		inv, err := buildSSHInvocation(ctx, port, user, host, cmd, timeout)
		if err != nil {
			return inv, runner.Result{Stderr: err.Error(), ExitCode: -1, StartTime: time.Now()}, attempts, err
		}
		attempts++
		lastInv = inv
		lastRes = ctx.Runner.Exec(inv)
		if ready(lastRes) {
			return lastInv, lastRes, attempts, nil
		}

		remaining = time.Until(deadline)
		if remaining <= 0 {
			break
		}
		sleepFor := 5 * time.Second
		if remaining < sleepFor {
			sleepFor = remaining
		}
		time.Sleep(sleepFor)
	}

	if lastRes.StartTime.IsZero() {
		lastRes = runner.Result{Stderr: "SSH command was not attempted", ExitCode: -1, StartTime: time.Now()}
	}
	detail := strings.TrimSpace(lastRes.Stderr)
	if detail == "" {
		detail = strings.TrimSpace(lastRes.Stdout)
	}
	return lastInv, lastRes, attempts, fmt.Errorf("command %q did not become ready; last output: %s", cmd, detail)
}

func buildSSHInvocation(ctx *Context, port, user, host, cmd string, timeout time.Duration) (runner.Invocation, error) {
	sshUser := resolveSSHUser(ctx, user)
	if host == "" {
		host = "127.0.0.1"
	}
	if port == "" {
		return runner.Invocation{Command: "ssh", Timeout: timeout}, fmt.Errorf("ssh port is empty")
	}
	args := []string{
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=" + os.DevNull,
		"-o", "ConnectTimeout=5",
		"-o", "ConnectionAttempts=1",
		"-o", "ServerAliveInterval=5",
		"-o", "ServerAliveCountMax=1",
	}
	if runtime.GOOS != "windows" {
		args = append([]string{"-n"}, args...)
	}

	sshPwd := resolveSSHPassword(ctx)
	if sshPwd == "" {
		args = append(args,
			"-o", "BatchMode=yes",
			"-o", "NumberOfPasswordPrompts=0",
		)
	} else {
		args = append(args, "-o", "NumberOfPasswordPrompts=1")
	}
	args = append(args, validatorSSHKeyArgs()...)
	args = append(args,
		"-p", port,
		fmt.Sprintf("%s@%s", sshUser, host),
		cmd,
	)

	execCmd := "ssh"
	execArgs := args
	env := map[string]string{}

	if _, err := exec.LookPath("ssh"); err != nil {
		return runner.Invocation{Command: execCmd, Args: execArgs, Timeout: timeout}, fmt.Errorf("ssh not found in PATH: %w", err)
	}
	if sshPwd != "" {
		execCmd = "sshpass"
		execArgs = append([]string{"-e", "ssh"}, args...)
		env["SSHPASS"] = sshPwd
		if _, err := exec.LookPath("sshpass"); err != nil {
			return runner.Invocation{Command: execCmd, Args: execArgs, Timeout: timeout, Env: env}, fmt.Errorf("sshpass not found in PATH (required for password auth): %w", err)
		}
	}

	return runner.Invocation{
		Command: execCmd,
		Args:    execArgs,
		Timeout: timeout,
		Env:     env,
	}, nil
}

func resolveSSHUser(ctx *Context, user string) string {
	if user != "" {
		return user
	}
	if ctx.Config.SSHUser != "" {
		return ctx.Config.SSHUser
	}
	return "vmuser"
}

func resolveSSHPassword(ctx *Context) string {
	if ctx.Config.SSHPassword != "" {
		return ctx.Config.SSHPassword
	}
	sshPwd, _ := getVar(ctx, "vm_primary_ssh_password")
	return sshPwd
}

func appendDiagnostic(a, b string) string {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	return a + "\n" + b
}

func validatorSSHKeyArgs() []string {
	home, err := sysutil.UserHome()
	if err != nil {
		return nil
	}
	keyPath := filepath.Join(home, ".nido", "nido_ed25519")
	if _, err := os.Stat(keyPath); err != nil {
		return nil
	}
	return []string{"-i", keyPath}
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
	res := runDeleteValidatorVM(ctx, vmName, 30*time.Second)
	addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)
	if res.ExitCode == 0 {
		ctx.State.RemoveVM(vmName)
	}
	finalize(&res)
	return res
}

func pruneVM(ctx *Context) report.StepResult {
	listRes := runNido(ctx, "prune-safety-list", []string{"list", "--json"}, 15*time.Second)
	if listRes.ExitCode != 0 {
		return skipResult(ctx.Config.NidoBin, []string{"prune", "--json"}, "skipping prune: failed to inspect VM list before prune")
	}
	blocked, err := stoppedNonValidatorVMs(listRes.Stdout)
	if err != nil {
		return skipResult(ctx.Config.NidoBin, []string{"prune", "--json"}, "skipping prune: failed to parse VM list before prune: "+err.Error())
	}
	if len(blocked) > 0 {
		return skipResult(ctx.Config.NidoBin, []string{"prune", "--json"}, "skipping prune: stopped non-validator VMs exist: "+strings.Join(blocked, ","))
	}

	args := []string{"prune", "--json"}
	res := runNido(ctx, "prune", args, 15*time.Second)
	addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)
	if payload, err := parseJSON(res.Stdout); err == nil {
		addAssertion(&res, "json_parse", true, "")
		addAssertion(&res, "status_ok", payload["status"] == "ok", "")
	} else {
		addAssertion(&res, "json_parse", false, err.Error())
	}
	finalize(&res)
	return res
}

func stoppedNonValidatorVMs(raw string) ([]string, error) {
	payload, err := parseJSON(raw)
	if err != nil {
		return nil, err
	}
	var blocked []string
	data, _ := payload["data"].(map[string]interface{})
	vms, _ := data["vms"].([]interface{})
	for _, entry := range vms {
		vm, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := vm["name"].(string)
		state, _ := vm["state"].(string)
		if name != "" && state == "stopped" && !isValidatorGeneratedVMName(name) {
			blocked = append(blocked, name)
		}
	}
	return blocked, nil
}

func cmdlineTest(ctx *Context) report.StepResult {
	stepStart := time.Now()
	vmName, ok := getVar(ctx, "vm_primary")
	if !ok {
		return skipResult(ctx.Config.NidoBin, []string{"cmdline-test"}, "vm_primary not set")
	}
	sshPort, okPort := getVar(ctx, "vm_primary_ssh_port")
	sshUser, _ := getVar(ctx, "vm_primary_ssh_user")
	sshUser = resolveSSHUser(ctx, sshUser)
	host := "127.0.0.1"
	if ip, ok := getVar(ctx, "vm_primary_ip"); ok && ip != "" {
		host = ip
	}

	if !okPort {
		return skipResult("ssh", []string{}, "ssh metadata missing")
	}

	// 1. Stop the VM first
	stopArgs := []string{"stop", vmName, "--json"}
	stopRes := runNido(ctx, "stop", stopArgs, 30*time.Second)
	if stopRes.ExitCode != 0 {
		return stopRes
	}

	// 2. Start with custom --cmdline
	magicParam := "nido_val_" + util.NewRunID()[:8]
	startArgs := []string{"start", vmName, "--cmdline", "root=/dev/sda rw console=ttyS0 console=tty0 " + magicParam, "--json"}
	startRes := runNido(ctx, "start", startArgs, ctx.Config.BootTimeout)
	addAssertion(&startRes, "exit_zero", startRes.ExitCode == 0, startRes.Stderr)
	if startRes.ExitCode != 0 {
		finalize(&startRes)
		return startRes
	}

	// 3. Wait for SSH, then check /proc/cmdline only for Direct Kernel Boot guests.
	_, readyRes, attempts, err := waitForSSHCommand(ctx, sshPort, sshUser, host, "echo ok", ctx.Config.BootTimeout, 10*time.Second, func(res runner.Result) bool {
		return res.ExitCode == 0 && !res.TimedOut && containsOK(res.Stdout)
	})
	addAssertion(&startRes, "ssh_ready", err == nil, fmt.Sprintf("attempts=%d %s", attempts, appendDiagnostic(readyRes.Stderr, errDetails(err))))
	if err != nil {
		startRes.DurationMs = time.Since(stepStart).Milliseconds()
		finalize(&startRes)
		return startRes
	}

	home, _ := sysutil.UserHome()
	vmsDir := filepath.Join(home, ".nido", "vms")
	kernelPath := filepath.Join(vmsDir, vmName+".kernel")
	if _, err := os.Stat(kernelPath); err == nil {
		checkRes := runSSHCommand(ctx, sshPort, sshUser, host, "cat /proc/cmdline", 15*time.Second)
		addAssertion(&startRes, "ssh_cmdline_exit_zero", checkRes.ExitCode == 0, checkRes.Stderr)
		found := strings.Contains(checkRes.Stdout, magicParam)
		addAssertion(&startRes, "cmdline_match", found, fmt.Sprintf("Expected '%s' in /proc/cmdline, got: %s", magicParam, checkRes.Stdout))
	} else {
		addAssertion(&startRes, "cmdline_test", true, "Skipped match check (not Direct Kernel Boot)")
	}

	startRes.DurationMs = time.Since(stepStart).Milliseconds()
	finalize(&startRes)
	return startRes
}

func ensureUserData(ctx *Context) (string, error) {
	if ctx.Config.UserDataPath != "" {
		return ctx.Config.UserDataPath, nil
	}
	f, err := os.CreateTemp("", "cli-val-userdata-*.yaml")
	if err != nil {
		return "", err
	}
	content := "#!/bin/sh\nprintf 'ok\\n' > /tmp/nido-cli-validate-marker\n"
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
