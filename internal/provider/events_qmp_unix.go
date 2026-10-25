//go:build !windows

package provider

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/Josepavese/nido/v4/internal/events"
	"github.com/Josepavese/nido/v4/internal/pkg/sysutil"
)

const qemuEventSuffix = ".events.qmp"

func prepareQEMUEvents(root, name string) ([]string, error) {
	if _, err := sysutil.PrivateEventsDirectory(root); err != nil {
		return nil, err
	}
	path := filepath.Join(sysutil.EventsDirectory(root), name+qemuEventSuffix)
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSocket == 0 {
			return nil, fmt.Errorf("refusing non-socket QMP event endpoint")
		}
		if err := os.Remove(path); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return []string{"-qmp", "unix:" + path + ",server=on,wait=off"}, nil
}
func dialQEMUEvents(ctx context.Context, root, name string) (net.Conn, error) {
	return (&net.Dialer{Timeout: events.IOTimeout}).DialContext(ctx, "unix", filepath.Join(sysutil.EventsDirectory(root), name+qemuEventSuffix))
}
