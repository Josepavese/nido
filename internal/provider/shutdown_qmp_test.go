package provider

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// qmpTestSocket returns a short Unix socket path. macOS limits sun_path to 104
// bytes, so t.TempDir() (which embeds the test name) is too long there.
func qmpTestSocket(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("QMP uses a Unix socket; Windows shutdown remains SSH-only")
	}
	dir, err := os.MkdirTemp("", "nido-qmp")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return filepath.Join(dir, "qmp.sock")
}

func TestRequestQEMUPowerdownSendsCapabilitiesAndACPICommand(t *testing.T) {
	path := qmpTestSocket(t)
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()
	serverErr := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		defer conn.Close()
		decoder := json.NewDecoder(conn)
		encoder := json.NewEncoder(conn)
		if err := encoder.Encode(map[string]interface{}{"QMP": map[string]interface{}{}, "capabilities": []string{}}); err != nil {
			serverErr <- err
			return
		}
		for _, want := range []string{"qmp_capabilities", "system_powerdown"} {
			var request struct {
				Execute string `json:"execute"`
			}
			if err := decoder.Decode(&request); err != nil {
				serverErr <- err
				return
			}
			if request.Execute != want {
				serverErr <- &unexpectedQMPCommand{got: request.Execute, want: want}
				return
			}
			if err := encoder.Encode(map[string]interface{}{"return": map[string]interface{}{}}); err != nil {
				serverErr <- err
				return
			}
		}
		serverErr <- nil
	}()

	if err := requestQEMUPowerdown(context.Background(), path); err != nil {
		t.Fatalf("requestQEMUPowerdown: %v", err)
	}
	if err := <-serverErr; err != nil {
		t.Fatalf("fake QMP server: %v", err)
	}
}

func TestRequestQEMUPowerdownReportsGuestRefusal(t *testing.T) {
	path := qmpTestSocket(t)
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		decoder := json.NewDecoder(conn)
		encoder := json.NewEncoder(conn)
		_ = encoder.Encode(map[string]interface{}{"QMP": map[string]interface{}{}, "capabilities": []string{}})
		var request struct {
			Execute string `json:"execute"`
		}
		_ = decoder.Decode(&request)
		_ = encoder.Encode(map[string]interface{}{"return": map[string]interface{}{}})
		_ = decoder.Decode(&request)
		_ = encoder.Encode(map[string]interface{}{"error": map[string]interface{}{"class": "GenericError", "desc": "powerdown not supported"}})
	}()

	err = requestQEMUPowerdown(context.Background(), path)
	if err == nil || !strings.Contains(err.Error(), "powerdown not supported") {
		t.Fatalf("expected QMP refusal, got %v", err)
	}
}

func TestRequestQEMUPowerdownRequiresSocket(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fixture.qmp")
	if err := os.WriteFile(path, []byte("not a socket"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := requestQEMUPowerdown(context.Background(), path); err == nil || !strings.Contains(err.Error(), "non-socket") {
		t.Fatalf("expected non-socket rejection, got %v", err)
	}
	if err := requestQEMUPowerdown(context.Background(), ""); err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("expected unavailable endpoint, got %v", err)
	}
}

type unexpectedQMPCommand struct {
	got  string
	want string
}

func (e *unexpectedQMPCommand) Error() string {
	return "unexpected QMP command: got " + e.got + ", want " + e.want
}
