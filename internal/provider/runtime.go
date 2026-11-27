package provider

import (
	"context"
	"fmt"

	"github.com/Josepavese/nido/v4/internal/events"
	"github.com/Josepavese/nido/v4/internal/pkg/identifier"
)

const (
	VMRunning = "running"
	VMStopped = "stopped"
	VMPaused  = "paused"
	VMIOError = "io-error"
	VMUnknown = "unknown"
)

type qmpRunState struct {
	Running bool   `json:"running"`
	Status  string `json:"status"`
}

func (s qmpRunState) state() string {
	switch {
	case s.Status == VMIOError:
		return VMIOError
	case s.Running:
		return VMRunning
	case s.Status == VMPaused || s.Status == "prelaunch":
		return VMPaused
	default:
		return VMUnknown
	}
}

func (p *QemuProvider) runtimeState(ctx context.Context, name string, pid int) string {
	path := p.qmpControlPath(name)
	if path == "" {
		return VMRunning // Hosts without a control endpoint can report process liveness.
	}
	var state string
	err := withQMPControl(ctx, path, events.IOTimeout, func(q *qmpSession) error {
		if err := q.verifyName(name); err != nil {
			return err
		}
		status, err := q.queryStatus()
		state = status.state()
		return err
	})
	if err != nil {
		if !p.isVMProcess(name, pid) {
			return VMStopped
		}
		return VMUnknown
	}
	return state
}

func (q *qmpSession) queryStatus() (qmpRunState, error) {
	var status qmpRunState
	err := q.command("query-status", &status)
	return status, err
}

// Resume verifies the running process and the monitor, then resumes a paused
// guest in place. A successful response observes runnable vCPUs on that same
// monitor. It says nothing about guest services or the outcome of guest writes.
func (p *QemuProvider) Resume(ctx context.Context, name string) (resultErr error) {
	if err := identifier.Validate(name); err != nil {
		return err
	}
	finish := p.operation(events.VMResume, name)
	defer func() { finish(resultErr) }()
	pid := p.runningPID(name)
	if !p.isVMProcess(name, pid) {
		return fmt.Errorf("VM %q has no running owned QEMU process", name)
	}
	return withQMPControl(ctx, p.qmpControlPath(name), events.IOTimeout, func(q *qmpSession) error {
		if err := q.verifyName(name); err != nil {
			return err
		}
		status, err := q.queryStatus()
		if err != nil {
			return err
		}
		switch status.state() {
		case VMRunning:
			return nil
		case VMPaused, VMIOError:
			if p.runningPID(name) != pid || !p.isVMProcess(name, pid) {
				return fmt.Errorf("VM %q changed process before resume", name)
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := q.command("cont", nil); err != nil {
				return err
			}
		default:
			return fmt.Errorf("VM %q cannot resume from QEMU state %q", name, status.Status)
		}
		status, err = q.queryStatus()
		if err != nil {
			return err
		}
		if status.state() != VMRunning {
			return fmt.Errorf("VM %q did not resume (QEMU state %q); check host storage and try again", name, status.Status)
		}
		return nil
	})
}
