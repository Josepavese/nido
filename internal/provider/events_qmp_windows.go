package provider

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/Josepavese/nido/v4/internal/events"
	"github.com/Josepavese/nido/v4/internal/pkg/sysutil"
)

const qemuEventSuffix = ".events.addr"
const qemuEventLoopback = "127.0.0.1"
const qemuEventEndpointMode = 0600

func prepareQEMUEvents(root, name string) ([]string, error) {
	if _, err := sysutil.PrivateEventsDirectory(root); err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp", net.JoinHostPort(qemuEventLoopback, "0"))
	if err != nil {
		return nil, err
	}
	addr := listener.Addr().String()
	listener.Close()
	if err := sysutil.WriteFile(filepath.Join(sysutil.EventsDirectory(root), name+qemuEventSuffix), []byte(addr), qemuEventEndpointMode); err != nil {
		return nil, err
	}
	// QEMU owns the actual bind; allocation collisions fail startup explicitly.
	return []string{"-qmp", "tcp:" + addr + ",server=on,wait=off"}, nil
}
func dialQEMUEvents(ctx context.Context, root, name string) (net.Conn, error) {
	data, err := os.ReadFile(filepath.Join(sysutil.EventsDirectory(root), name+qemuEventSuffix))
	if err != nil {
		return nil, err
	}
	addr := strings.TrimSpace(string(data))
	host, _, err := net.SplitHostPort(addr)
	if err != nil || host != qemuEventLoopback {
		return nil, fmt.Errorf("invalid local QMP event address")
	}
	return (&net.Dialer{Timeout: events.IOTimeout}).DialContext(ctx, "tcp", addr)
}
