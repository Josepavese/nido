package provider

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"
)

// qmpPowerdownTimeout bounds the local control handshake and the one command
// that asks the guest firmware/OS to begin an ACPI poweroff. The later process
// exit is bounded by the caller's shutdown budget.
const qmpPowerdownTimeout = 10 * time.Second

// requestQEMUPowerdown sends QEMU's system_powerdown command over the
// provider-owned control socket. It only requests a guest poweroff: if the
// guest ignores ACPI, the caller's process-exit wait still fails without a
// forced kill.
func requestQEMUPowerdown(ctx context.Context, path string) error {
	if path == "" {
		return fmt.Errorf("QMP control endpoint is unavailable on this host")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("QMP control endpoint: %w", err)
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("refusing non-socket QMP control endpoint")
	}
	dialer := &net.Dialer{Timeout: qmpPowerdownTimeout}
	conn, err := dialer.DialContext(ctx, "unix", path)
	if err != nil {
		return fmt.Errorf("connect QMP control endpoint: %w", err)
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(qmpPowerdownTimeout)); err != nil {
		return fmt.Errorf("set QMP deadline: %w", err)
	}

	reader := bufio.NewReader(conn)
	decoder := json.NewDecoder(reader)
	encoder := json.NewEncoder(conn)
	var greeting map[string]json.RawMessage
	if err := decoder.Decode(&greeting); err != nil {
		return fmt.Errorf("read QMP greeting: %w", err)
	}
	if greeting["QMP"] == nil {
		return fmt.Errorf("QMP greeting missing")
	}
	if err := qmpCommand(encoder, decoder, "qmp_capabilities"); err != nil {
		return err
	}
	return qmpCommand(encoder, decoder, "system_powerdown")
}

func qmpCommand(encoder *json.Encoder, decoder *json.Decoder, command string) error {
	if err := encoder.Encode(map[string]string{"execute": command}); err != nil {
		return fmt.Errorf("send QMP %s: %w", command, err)
	}
	for {
		var message map[string]json.RawMessage
		if err := decoder.Decode(&message); err != nil {
			return fmt.Errorf("read QMP %s response: %w", command, err)
		}
		if raw, ok := message["error"]; ok {
			var detail struct {
				Class string `json:"class"`
				Desc  string `json:"desc"`
			}
			_ = json.Unmarshal(raw, &detail)
			if detail.Desc != "" {
				return fmt.Errorf("QMP %s: %s", command, detail.Desc)
			}
			return fmt.Errorf("QMP %s failed", command)
		}
		if _, ok := message["return"]; ok {
			return nil
		}
		// QMP may interleave asynchronous events with the command response.
	}
}
