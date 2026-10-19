package provider

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/Josepavese/nido/internal/config"
)

func TestUpdateConfigRawQemuArgs(t *testing.T) {
	oldArgs := []string{"-name", "original-marker"}
	newArgs := []string{"-fw_cfg", "name=opt/nido-test,string=a b,c", "-nodefaults"}
	emptyArgs := []string{}
	cpus := 3
	invalidMemory := 127
	tests := []struct {
		name    string
		initial []string
		updates VMConfigUpdates
		want    []string
		wantErr bool
	}{
		{name: "set", updates: VMConfigUpdates{RawQemuArgs: &newArgs}, want: newArgs},
		{name: "replace", initial: oldArgs, updates: VMConfigUpdates{RawQemuArgs: &newArgs}, want: newArgs},
		{name: "omit", initial: oldArgs, updates: VMConfigUpdates{VCPUs: &cpus}, want: oldArgs},
		{name: "clear", initial: oldArgs, updates: VMConfigUpdates{RawQemuArgs: &emptyArgs}, want: emptyArgs},
		{name: "clear empty", updates: VMConfigUpdates{RawQemuArgs: &emptyArgs}, want: emptyArgs},
		{name: "reject invalid update", initial: oldArgs, updates: VMConfigUpdates{RawQemuArgs: &newArgs, MemoryMB: &invalidMemory}, want: oldArgs, wantErr: true},
	}
	for _, pid := range []int{0, os.Getpid()} {
		stateName := "stopped"
		if pid != 0 {
			// A live test-process PID models persisted running state; no VM is started.
			stateName = "running"
		}
		for _, tt := range tests {
			t.Run(stateName+"/"+tt.name, func(t *testing.T) {
				root := t.TempDir()
				if err := os.Mkdir(filepath.Join(root, "run"), 0o700); err != nil {
					t.Fatal(err)
				}
				p := NewQemuProvider(root, &config.Config{})
				fw := []PortForward{{Label: "fixture", GuestPort: 80, HostPort: 31080, Protocol: "tcp"}}
				if err := p.saveState("fixture", pid, 51022, 0, false, "fixture-user", fw, "fixture-cmdline", 512, 1, tt.initial, []string{"virtual:gpu"}); err != nil {
					t.Fatal(err)
				}
				before, err := p.loadState("fixture")
				if err != nil {
					t.Fatal(err)
				}
				err = p.UpdateConfig("fixture", tt.updates)
				if (err != nil) != tt.wantErr {
					t.Fatalf("UpdateConfig error = %v, want error %v", err, tt.wantErr)
				}
				// Reopen through a new provider: assert durable state, not a shared slice.
				reopened := NewQemuProvider(root, &config.Config{})
				after, err := reopened.loadState("fixture")
				if err != nil {
					t.Fatal(err)
				}
				if !slices.Equal(after.RawQemuArgs, tt.want) {
					t.Fatalf("persisted raw args = %#v, want %#v", after.RawQemuArgs, tt.want)
				}
				wantState := before
				wantState.RawQemuArgs = after.RawQemuArgs // Empty lists may be omitted by the state codec.
				if tt.updates.VCPUs != nil && !tt.wantErr {
					wantState.VCPUs = *tt.updates.VCPUs
				}
				if !reflect.DeepEqual(after, wantState) {
					t.Fatalf("unrelated state changed:\ngot  %#v\nwant %#v", after, wantState)
				}
				// Exercise the argument builder used by Start, without launching QEMU.
				args := reopened.buildQemuArgs(after.Name, filepath.Join(root, "fixture.qcow2"), after.SSHPort, after.VNCPort, after.Forwarding, filepath.Join(root, "run"), after.Cmdline, after.MemoryMB, after.VCPUs, after.RawQemuArgs, after.Accelerators)
				if len(tt.want) > 0 && !slices.Equal(args[len(args)-len(tt.want):], tt.want) {
					t.Fatalf("next-start argument suffix = %#v, want %#v", args, tt.want)
				}
				if !slices.Contains(tt.want, "original-marker") && slices.Contains(args, "original-marker") {
					t.Fatal("next-start arguments still contain the removed value")
				}
			})
		}
	}
}

func TestUpdateConfigMissingStateFails(t *testing.T) {
	p := NewQemuProvider(t.TempDir(), &config.Config{})
	args := []string{"-nodefaults"}
	if err := p.UpdateConfig("missing", VMConfigUpdates{RawQemuArgs: &args}); err == nil {
		t.Fatal("updating a VM without state must fail")
	}
}
