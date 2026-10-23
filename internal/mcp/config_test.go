package mcp

import (
	"encoding/json"
	"github.com/Josepavese/nido/v4/internal/hardware"
	"io"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Josepavese/nido/v4/internal/config"
	"github.com/Josepavese/nido/v4/internal/provider"
)

type configRecordingProvider struct {
	provider.VMProvider
	updates []provider.VMConfigUpdates
}

func (p *configRecordingProvider) UpdateConfig(name string, update provider.VMConfigUpdates) error {
	p.updates = append(p.updates, update)
	return p.VMProvider.UpdateConfig(name, update)
}

func TestConfigRawQemuArgsMCP(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", t.TempDir()) // Config tests never probe host QEMU.
	if err := os.Mkdir(filepath.Join(root, "run"), 0o700); err != nil {
		t.Fatal(err)
	}
	state := []byte(`{"name":"fixture","pid":0,"memory_mb":512,"vcpus":1,"ssh_port":51022,"ssh_user":"fixture-user","raw_qemu_args":["-name","original-marker"]}`)
	if err := os.WriteFile(filepath.Join(root, "run", "fixture.json"), state, 0o600); err != nil {
		t.Fatal(err)
	}
	p := &configRecordingProvider{VMProvider: provider.NewQemuProvider(root, &config.Config{}, hardware.New())}
	s := NewServer(p)
	newArgs := []string{"-fw_cfg", "name=opt/nido-test,string=a b,c"}
	steps := []struct {
		name       string
		arguments  string
		want       []string
		wantAbsent bool
		wantErr    bool
	}{
		{"replace", `"raw_qemu_args":["-fw_cfg","name=opt/nido-test,string=a b,c"]`, newArgs, false, false},
		{"omit", `"vcpus":3`, newArgs, true, false},
		{"reject null", `"raw_qemu_args":null`, newArgs, false, true},
		{"reject string", `"raw_qemu_args":"-nodefaults"`, newArgs, false, true},
		{"reject element type", `"raw_qemu_args":[7]`, newArgs, false, true},
		{"reject invalid config", `"memory_mb":127,"raw_qemu_args":[]`, newArgs, false, true},
		{"clear", `"raw_qemu_args":[]`, []string{}, false, false},
		{"clear again", `"raw_qemu_args":[]`, []string{}, false, false},
		{"missing VM", `"name":"missing","raw_qemu_args":[]`, []string{}, false, true},
	}
	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			p.updates = nil
			payload := `{"action":"config_update","name":"fixture",` + step.arguments + `}`
			// Avoid a duplicate JSON key in the missing-VM case.
			if step.name == "missing VM" {
				payload = `{"action":"config_update",` + step.arguments + `}`
			}
			response := callConfigTool(t, s, json.RawMessage(payload))
			if response.IsError != step.wantErr {
				t.Fatalf("MCP isError=%v, want %v: %#v", response.IsError, step.wantErr, response)
			}
			var body struct {
				Status string `json:"status"`
				Error  string `json:"error"`
			}
			if len(response.Content) != 1 {
				t.Fatalf("unexpected MCP content: %#v", response)
			}
			if err := json.Unmarshal([]byte(response.Content[0].Text), &body); err != nil {
				t.Fatal(err)
			}
			if step.wantErr {
				if body.Error == "" || body.Status == "updated" {
					t.Fatalf("failed request returned misleading status: %#v", body)
				}
			} else {
				if body.Status != "updated" || len(p.updates) != 1 {
					t.Fatalf("request not delegated exactly once: body=%#v updates=%#v", body, p.updates)
				}
				args := p.updates[0].RawQemuArgs
				if (args == nil) != step.wantAbsent {
					t.Fatalf("field presence lost: raw args=%#v, want absent=%v", args, step.wantAbsent)
				}
				if args != nil && !slices.Equal(*args, step.want) {
					t.Fatalf("decoded arguments = %#v, want %#v", *args, step.want)
				}
			}
			info, err := provider.NewQemuProvider(root, &config.Config{}, hardware.New()).Info("fixture")
			if err != nil || !slices.Equal(info.RawQemuArgs, step.want) || info.State != "stopped" || info.MemoryMB != 512 || info.SSHUser != "fixture-user" {
				t.Fatalf("persisted config = %#v, error = %v, want raw args %#v", info, err, step.want)
			}
		})
	}
}

type configToolResponse struct {
	IsError bool `json:"isError"`
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

func callConfigTool(t *testing.T, server *Server, arguments json.RawMessage) configToolResponse {
	t.Helper()
	params, err := json.Marshal(CallParams{Name: "nido_vm", Arguments: arguments})
	if err != nil {
		t.Fatal(err)
	}
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	oldStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()
	server.handleToolsCall(JSONRPCRequest{JSONRPC: "2.0", ID: 42, Method: "tools/call", Params: params})
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	var response struct {
		Result configToolResponse `json:"result"`
	}
	if err := json.Unmarshal(out, &response); err != nil {
		t.Fatalf("invalid MCP response %s: %v", out, err)
	}
	return response.Result
}
