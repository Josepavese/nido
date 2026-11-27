package provider

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/Josepavese/nido/v4/internal/events"
)

// qmpPowerdownTimeout bounds the local control handshake and the one command
// that asks the guest firmware/OS to begin an ACPI poweroff. The later process
// exit is bounded by the caller's shutdown budget.
const qmpPowerdownTimeout = 10 * time.Second

// qmpControlPath is empty on hosts without a managed control endpoint.
func (p *QemuProvider) qmpControlPath(name string) string {
	return qemuQMPPath(filepath.Join(p.RootDir, "run"), name)
}

// requestQEMUPowerdown sends QEMU's system_powerdown command over the
// provider-owned control socket. It only requests a guest poweroff: if the
// guest ignores ACPI, the caller's process-exit wait still fails without a
// forced kill.
func requestQEMUPowerdown(ctx context.Context, path string) error {
	return withQMPControl(ctx, path, qmpPowerdownTimeout, func(q *qmpSession) error {
		return q.command("system_powerdown", nil)
	})
}

func withQMPControl(ctx context.Context, path string, timeout time.Duration, use func(*qmpSession) error) error {
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
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "unix", path)
	if err != nil {
		return fmt.Errorf("connect QMP control endpoint: %w", err)
	}
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	deadline, _ := ctx.Deadline()
	if err := conn.SetDeadline(deadline); err != nil {
		return fmt.Errorf("set QMP deadline: %w", err)
	}
	q := newQMPSession(conn)
	if err := q.negotiate(); err != nil {
		return err
	}
	return use(q)
}

type qmpSession struct {
	scan *bufio.Scanner
	enc  *json.Encoder
}

func newQMPSession(conn net.Conn) *qmpSession {
	scan := bufio.NewScanner(conn)
	scan.Buffer(nil, events.MaxFrameBytes)
	return &qmpSession{scan: scan, enc: json.NewEncoder(conn)}
}

func (q *qmpSession) read() (map[string]json.RawMessage, error) {
	if !q.scan.Scan() {
		if err := q.scan.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}
	var message map[string]json.RawMessage
	err := json.Unmarshal(q.scan.Bytes(), &message)
	return message, err
}

func (q *qmpSession) negotiate() error {
	greeting, err := q.read()
	if err != nil {
		return fmt.Errorf("read QMP greeting: %w", err)
	}
	if greeting["QMP"] == nil {
		return fmt.Errorf("QMP greeting missing")
	}
	return q.command("qmp_capabilities", nil)
}

func (q *qmpSession) verifyName(name string) error {
	var identity struct {
		Name string `json:"name"`
	}
	if err := q.command("query-name", &identity); err != nil {
		return err
	}
	if identity.Name != name {
		return fmt.Errorf("QMP monitor does not belong to VM %q", name)
	}
	return nil
}

func (q *qmpSession) command(command string, result any) error {
	if err := q.enc.Encode(map[string]string{"execute": command}); err != nil {
		return fmt.Errorf("send QMP %s: %w", command, err)
	}
	for {
		message, err := q.read()
		if err != nil {
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
		if raw, ok := message["return"]; ok {
			if result != nil {
				if err := json.Unmarshal(raw, result); err != nil {
					return fmt.Errorf("decode QMP %s response: %w", command, err)
				}
			}
			return nil
		}
		// QMP may interleave asynchronous events with the command response.
	}
}
