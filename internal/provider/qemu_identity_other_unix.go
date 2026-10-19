//go:build !linux && !windows

package provider

import (
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const qmpControlTimeout = vmStopTimeout

func inspectVMProcess(pid int, identity vmProcessIdentity) error {
	exe, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "comm=").Output()
	if err != nil {
		return err
	}
	args, err := exec.Command("ps", "-ww", "-p", strconv.Itoa(pid), "-o", "args=").Output()
	if err != nil {
		return err
	}
	command := " " + strings.TrimSpace(string(args)) + " "
	if !qemuExecutable(strings.TrimSpace(string(exe))) || !strings.Contains(command, " -name "+identity.name+" ") || !strings.Contains(command, "file="+identity.disk+",") {
		return fmt.Errorf("PID %d does not belong to VM %q", pid, identity.name)
	}
	return nil
}

func stopManagedProcess(pid int, identity vmProcessIdentity, _ bool) error {
	if err := inspectVMProcess(pid, identity); err != nil {
		return err
	}
	// A QMP connection is tied to the VM instance, so PID reuse can never
	// redirect the stop command to an unrelated process. No numeric kill fallback.
	path := strings.TrimSuffix(identity.pidfile, filepath.Ext(identity.pidfile)) + ".qmp"
	conn, err := net.DialTimeout("unix", path, qmpControlTimeout)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(qmpControlTimeout)); err != nil {
		return err
	}
	decoder, encoder := json.NewDecoder(conn), json.NewEncoder(conn)
	var greeting map[string]json.RawMessage
	if err := decoder.Decode(&greeting); err != nil {
		return err
	}
	call := func(command string) (json.RawMessage, error) {
		if err := encoder.Encode(map[string]string{"execute": command}); err != nil {
			return nil, err
		}
		for {
			var response map[string]json.RawMessage
			if err := decoder.Decode(&response); err != nil {
				return nil, err
			}
			if response["event"] != nil {
				continue
			}
			if response["error"] != nil {
				return nil, fmt.Errorf("QMP %s: %s", command, response["error"])
			}
			return response["return"], nil
		}
	}
	if _, err := call("qmp_capabilities"); err != nil {
		return err
	}
	value, err := call("query-name")
	if err != nil {
		return err
	}
	var name struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(value, &name); err != nil {
		return err
	}
	if name.Name != identity.name {
		return fmt.Errorf("QMP instance mismatch")
	}
	if _, err := call("quit"); err != nil {
		return err
	}
	// Disk operations must wait for QEMU to exit, rather than only its reply.
	deadline := time.Now().Add(qmpControlTimeout)
	for time.Now().Before(deadline) {
		if inspectVMProcess(pid, identity) != nil {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("VM %q did not exit", identity.name)
}
