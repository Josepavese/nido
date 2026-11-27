package provider

import (
	"context"
	"encoding/json"
	"net"
	"time"

	"github.com/Josepavese/nido/v4/internal/events"
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
	q := newQMPSession(conn)
	if q.negotiate() != nil || q.verifyName(name) != nil {
		return
	}
	emitVM := func(kind string) {
		emit(events.Event{Source: events.Runtime, Kind: kind, VM: name, Resources: events.Resources(events.VMConfigure, name)})
	}
	status, err := q.queryStatus()
	if err != nil {
		return
	}
	state := status.state()
	ioPaused := state == VMIOError
	switch state {
	case VMRunning:
		emitVM(events.VMRunning)
	case VMPaused:
		emitVM(events.VMPaused)
	case VMIOError:
		emitVM(events.VMIOError)
	}
	conn.SetDeadline(time.Time{})
	for ctx.Err() == nil {
		msg, err := q.read()
		if err != nil {
			return
		}
		var event string
		if json.Unmarshal(msg["event"], &event) != nil {
			continue
		}
		kind := qmpEventKinds[event]
		switch kind {
		case events.VMIOError:
			var data struct {
				Action string `json:"action"`
			}
			if json.Unmarshal(msg["data"], &data) == nil && data.Action == "stop" {
				ioPaused = true
			}
		case events.VMPaused:
			// QEMU follows a stopping BLOCK_IO_ERROR with STOP. Keep the
			// cause in the broker's latest snapshot for new subscribers.
			if ioPaused {
				kind = events.VMIOError
			}
		case events.VMResumed, events.VMReset, events.VMWakeup, events.VMShutdown:
			ioPaused = false
		}
		if kind != "" {
			emitVM(kind)
		}
	}
}
