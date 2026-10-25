package main

import (
	"bytes"
	"encoding/json"
	"github.com/Josepavese/nido/v5/internal/hardware"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	clijson "github.com/Josepavese/nido/v5/internal/cli"
	"github.com/Josepavese/nido/v5/internal/config"
	"github.com/Josepavese/nido/v5/internal/provider"
)

func TestConfigRawQemuArgsCLI(t *testing.T) {
	app := configTestApp(t)
	steps := []struct {
		name   string
		flags  []string
		want   []string
		result string
	}{
		{"literal empty argument", []string{"--qemu-arg=-append", "--qemu-arg="}, []string{"-append", ""}, "updated"},
		{"replace", []string{"--qemu-arg=-fw_cfg", "--qemu-arg=name=opt/nido-test,string=a b,c"}, []string{"-fw_cfg", "name=opt/nido-test,string=a b,c"}, "updated"},
		{"omit", []string{"--cpus=3"}, []string{"-fw_cfg", "name=opt/nido-test,string=a b,c"}, "updated"},
		{"false clear", []string{"--clear-qemu-args=false"}, []string{"-fw_cfg", "name=opt/nido-test,string=a b,c"}, "noop"},
		{"clear", []string{"--clear-qemu-args"}, []string{}, "updated"},
		{"clear again", []string{"--clear-qemu-args"}, []string{}, "updated"},
	}
	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			root, err := newRootCommand(app)
			if err != nil {
				t.Fatal(err)
			}
			root.SetArgs(append([]string{"config", "fixture", "--json"}, step.flags...))
			stdout, stderr := captureProcessIO(t, func() {
				if err := root.Execute(); err != nil {
					t.Fatal(err)
				}
			})
			var resp struct {
				Status string `json:"status"`
				Data   struct {
					Result string `json:"result"`
				} `json:"data"`
			}
			if err := json.Unmarshal([]byte(stdout), &resp); err != nil || stderr != "" || resp.Status != "ok" || resp.Data.Result != step.result {
				t.Fatalf("unexpected CLI response: stdout=%s stderr=%s error=%v", stdout, stderr, err)
			}
			reopened := provider.NewQemuProvider(app.NidoDir, app.Config, hardware.New())
			info, err := reopened.Info("fixture")
			if err != nil || !slices.Equal(info.RawQemuArgs, step.want) || info.State != "stopped" || info.MemoryMB != 512 || info.SSHUser != "fixture-user" {
				t.Fatalf("persisted config = %#v, error = %v, want raw args %#v", info, err, step.want)
			}
		})
	}
}

func TestConfigClearFlagCompletion(t *testing.T) {
	root, err := newRootCommand(testAppContext(t))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"__complete", "config", "fixture", "--clear"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out.Bytes(), []byte("--clear-qemu-args")) {
		t.Fatalf("config completion did not expose the clear flag: %s", out.String())
	}
}

func configTestApp(t *testing.T) *appContext {
	t.Helper()
	app := testAppContext(t)
	// No host QEMU probing: these tests exercise config/state only.
	t.Setenv("PATH", t.TempDir())
	run := filepath.Join(app.NidoDir, "run")
	if err := os.Mkdir(run, 0o700); err != nil {
		t.Fatal(err)
	}
	data := []byte(`{"arch":"amd64","name":"fixture","pid":0,"memory_mb":512,"vcpus":1,"ssh_port":51022,"ssh_user":"fixture-user","raw_qemu_args":["-name","original-marker"]}`)
	if err := os.WriteFile(filepath.Join(run, "fixture.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	app.Provider = provider.NewQemuProvider(app.NidoDir, app.Config, hardware.New())
	return app
}

func TestConfigRejectsConflictingRawArgs(t *testing.T) {
	app := configTestApp(t)
	root, err := newRootCommand(app)
	if err != nil {
		t.Fatal(err)
	}
	root.SetArgs([]string{"config", "fixture", "--qemu-arg=-nodefaults", "--clear-qemu-args", "--json"})
	if err := root.Execute(); err == nil {
		t.Fatal("setting and clearing raw args together must fail")
	}
	info, err := app.Provider.Info("fixture")
	if err != nil || !slices.Equal(info.RawQemuArgs, []string{"-name", "original-marker"}) {
		t.Fatalf("rejected request changed state: %#v, %v", info, err)
	}
}

func TestConfigCLIUpdateFailure(t *testing.T) {
	// Run the real action in a subprocess because its existing error boundary exits.
	if os.Getenv("NIDO_CONFIG_FAILURE_HELPER") == "1" {
		rootDir := os.Getenv("NIDO_CONFIG_FAILURE_ROOT")
		app := &appContext{Provider: provider.NewQemuProvider(rootDir, &config.Config{}, hardware.New())}
		root, err := newRootCommand(app)
		if err != nil {
			t.Fatal(err)
		}
		root.SetArgs([]string{"config", "fixture", "--memory=127", "--qemu-arg=-nodefaults", "--json"})
		if err := root.Execute(); err != nil {
			exitCLIError(err)
		}
		return
	}
	app := configTestApp(t)
	statePath := filepath.Join(app.NidoDir, "run", "fixture.json")
	before, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=^TestConfigCLIUpdateFailure$")
	cmd.Env = append(os.Environ(), "NIDO_CONFIG_FAILURE_HELPER=1", "NIDO_CONFIG_FAILURE_ROOT="+app.NidoDir)
	out, err := cmd.Output()
	if err == nil {
		t.Fatal("invalid update must exit with an error")
	}
	var resp clijson.Response
	if err := json.Unmarshal(out, &resp); err != nil || resp.Status != "error" || resp.Error == nil || resp.Error.Code != "ERR_UPDATE" {
		t.Fatalf("invalid update reported success or invalid JSON: %s, error=%v", out, err)
	}
	after, err := os.ReadFile(statePath)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("rejected update changed persisted state: %s, error=%v", after, err)
	}
}
