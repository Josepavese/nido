package provider

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"time"

	"github.com/Josepavese/nido/v5/internal/events"
)

var qmpEventKinds = map[string]string{
	"STOP": events.VMPaused, "RESUME": events.VMResumed, "RESET": events.VMReset,
	"SHUTDOWN": events.VMShutdown, "GUEST_PANICKED": events.VMPanic,
	"BLOCK_IO_ERROR": events.VMIOError, "WATCHDOG": events.VMWatchdog,
	"SUSPEND": events.VMSuspended, "SUSPEND_DISK": events.VMSuspended, "WAKEUP": events.VMWakeup,
}

func (p *QemuProvider) observeQMP(ctx context.Context, name string, emit events.Publisher) {
	for ctx.Err() == nil {
		conn, err := dialQEMUEvents(ctx, p.RootDir, name)
		if err == nil {
			stop := context.AfterFunc(ctx, func() { conn.Close() })
			p.readQMPEvents(ctx, conn, name, emit)
			stop()
			conn.Close()
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(events.ReconcileInterval):
		}
	}
}
func (p *QemuProvider) readQMPEvents(ctx context.Context, conn net.Conn, name string, emit events.Publisher) {
	conn.SetDeadline(time.Now().Add(events.IOTimeout))
	scan := bufio.NewScanner(conn)
	scan.Buffer(make([]byte, events.MaxFrameBytes), events.MaxFrameBytes)
	read := func() (map[string]json.RawMessage, bool) {
		if !scan.Scan() {
			return nil, false
		}
		var msg map[string]json.RawMessage
		err := json.Unmarshal(scan.Bytes(), &msg)
		return msg, err == nil
	}
	msg, ok := read()
	if !ok || msg["QMP"] == nil {
		return
	}
	enc := json.NewEncoder(conn)
	call := func(command string) (json.RawMessage, bool) {
		if enc.Encode(map[string]string{"execute": command}) != nil {
			return nil, false
		}
		for {
			msg, ok := read()
			if !ok || msg["error"] != nil {
				return nil, false
			}
			if result, ok := msg["return"]; ok {
				return result, true
			}
		}
	}
	if _, ok := call("qmp_capabilities"); !ok {
		return
	}
	result, ok := call("query-name")
	if !ok {
		return
	}
	var identity struct {
		Name string `json:"name"`
	}
	if json.Unmarshal(result, &identity) != nil || identity.Name != name {
		return
	}
	emitVM := func(kind string) {
		emit(events.Event{Source: events.Runtime, Kind: kind, VM: name, Resources: events.Resources(events.VMConfigure, name)})
	}
	result, ok = call("query-status")
	if !ok {
		return
	}
	var status struct {
		Running bool   `json:"running"`
		Status  string `json:"status"`
	}
	if json.Unmarshal(result, &status) != nil {
		return
	}
	if status.Running {
		emitVM(events.VMRunning)
	} else if status.Status == "paused" || status.Status == "prelaunch" {
		emitVM(events.VMPaused)
	}
	conn.SetDeadline(time.Time{})
	for ctx.Err() == nil {
		msg, ok := read()
		if !ok {
			return
		}
		var event string
		if json.Unmarshal(msg["event"], &event) == nil {
			if kind := qmpEventKinds[event]; kind != "" {
				emitVM(kind)
			}
		}
	}
}
