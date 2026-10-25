package provider

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Josepavese/nido/v4/internal/config"
	"github.com/Josepavese/nido/v4/internal/events"
	"github.com/Josepavese/nido/v4/internal/hardware"
	"github.com/Josepavese/nido/v4/internal/pkg/sysutil"
)

func TestOperationOutcomeCorrelationAndRedaction(t *testing.T) {
	root := t.TempDir()
	p := NewQemuProvider(root, &config.Config{}, hardware.New())
	got := []events.Event{}
	p.Events = func(e events.Event) { got = append(got, e) }
	if err := p.Start("missing", VMOptions{SSHPassword: "never-expose-this"}); err == nil {
		t.Fatal("missing VM started")
	}
	if len(got) != 2 || got[0].Kind != events.Started || got[1].Kind != events.Failed || got[0].OperationID != got[1].OperationID {
		t.Fatalf("invalid operation pair: %+v", got)
	}
	bytes, _ := json.Marshal(got)
	if strings.Contains(string(bytes), "never-expose-this") {
		t.Fatal("secret leaked")
	}
	if len(got[1].Resources) != 2 {
		t.Fatal("VM invalidation missing")
	}
}
func TestFilesystemClassifierIgnoresDiskContentAndStaging(t *testing.T) {
	root := t.TempDir()
	cfg := &config.Config{ImageDir: filepath.Join(root, "images"), BackupDir: filepath.Join(root, "backups")}
	p := NewQemuProvider(root, cfg, hardware.New())
	for _, change := range []sysutil.FileChange{{Path: filepath.Join(root, "vms", "vm.qcow2"), Content: true}, {Path: filepath.Join(cfg.ImageDir, "download.part")}, {Path: filepath.Join(root, "run", "vm.events.qmp")}} {
		if len(p.changedResources(change, filepath.Join(root, "config.env"))) != 0 {
			t.Fatal("noisy path exposed", change)
		}
	}
	if got := p.changedResources(sysutil.FileChange{Path: filepath.Join(root, "run", "vm.json")}, filepath.Join(root, "config.env")); !slices.Contains(got, events.VMResource("vm")) {
		t.Fatal("metadata publication missed")
	}
}
func TestQMPEndpointIdentityWhitelistAndCancellation(t *testing.T) {
	for _, identity := range []string{"fixture", "foreign"} {
		t.Run(identity, func(t *testing.T) {
			server, client := net.Pipe()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			wireDone := make(chan struct{})
			go func() {
				defer close(wireDone)
				defer server.Close()
				enc := json.NewEncoder(server)
				scan := bufio.NewScanner(server)
				enc.Encode(map[string]interface{}{"QMP": map[string]interface{}{}})
				for scan.Scan() {
					var command struct {
						Execute string `json:"execute"`
					}
					json.Unmarshal(scan.Bytes(), &command)
					switch command.Execute {
					case "qmp_capabilities":
						enc.Encode(map[string]interface{}{"return": map[string]interface{}{}})
					case "query-name":
						enc.Encode(map[string]interface{}{"return": map[string]string{"name": identity}})
					case "query-status":
						enc.Encode(map[string]interface{}{"return": map[string]interface{}{"running": true, "status": "running"}})
						for _, name := range []string{"RTC_CHANGE", "STOP", "SHUTDOWN", "GUEST_PANICKED"} {
							enc.Encode(map[string]string{"event": name})
						}
						return
					}
				}
			}()
			got := []events.Event{}
			(&QemuProvider{}).readQMPEvents(ctx, client, "fixture", func(e events.Event) { got = append(got, e) })
			client.Close()
			<-wireDone
			if identity == "foreign" {
				if len(got) != 0 {
					t.Fatal("foreign monitor accepted")
				}
				return
			}
			kinds := []string{}
			for _, e := range got {
				kinds = append(kinds, e.Kind)
			}
			if !slices.Equal(kinds, []string{events.VMRunning, events.VMPaused, events.VMShutdown, events.VMPanic}) {
				t.Fatal(kinds)
			}
			if slices.Contains(kinds, events.VMStopped) {
				t.Fatal("QMP shutdown claimed process termination")
			}
		})
	}
}
func TestNativeFilesystemObserverExternalPublicationAndCancellation(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "run"), 0700)
	os.MkdirAll(filepath.Join(root, "vms"), 0700)
	path := filepath.Join(root, "config.env")
	os.WriteFile(path, []byte("SSH_USER=fixture\n"), 0600)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	observed := make(chan events.Event, events.QueueSize)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ObserveEvents(root, path, hardware.New())(ctx, func(e events.Event) {
			select {
			case observed <- e:
			case <-ctx.Done():
			}
		})
	}()
	// Publishing repeatedly changes the same fixture until the native watch is armed.
	// The bounded ticker here is a test observer handshake, never production polling.
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	found := false
	for !found {
		select {
		case <-ticker.C:
			os.WriteFile(path, []byte("SSH_USER=updated\n"), 0600)
		case e := <-observed:
			found = slices.Contains(e.Resources, events.Config)
		case <-deadline.C:
			t.Fatal("native config change was not observed")
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("observer workers leaked")
	}
}
