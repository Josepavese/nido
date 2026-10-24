package events

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

const testTimeout = 6 * time.Second

func eventRoot(t *testing.T) string {
	t.Helper()
	root, err := os.MkdirTemp("", "ne-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(root) })
	return root
}
func receive(t *testing.T, ch <-chan Event, kind string) Event {
	t.Helper()
	timer := time.NewTimer(testTimeout)
	defer timer.Stop()
	for {
		select {
		case e, ok := <-ch:
			if !ok {
				t.Fatalf("stream ended before %s", kind)
			}
			if e.Kind == kind {
				return e
			}
		case <-timer.C:
			t.Fatalf("timeout waiting for %s", kind)
		}
	}
}
func drain(t *testing.T, ch <-chan Event) {
	t.Helper()
	timer := time.NewTimer(testTimeout)
	defer timer.Stop()
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
		case <-timer.C:
			t.Fatal("stream did not close")
		}
	}
}
func TestSharedFanoutAndOwnerHandoff(t *testing.T) {
	root := eventRoot(t)
	var observers atomic.Int32
	observe := func(ctx context.Context, publish Publisher) { observers.Add(1); defer observers.Add(-1); <-ctx.Done() }
	first, cancelFirst := context.WithCancel(context.Background())
	defer cancelFirst()
	a := NewClient(root, observe).Watch(first)
	initial := receive(t, a, Resync)
	second, cancelSecond := context.WithCancel(context.Background())
	defer cancelSecond()
	b := NewClient(root, observe).Watch(second)
	if got := receive(t, b, Resync); got.Epoch != initial.Epoch {
		t.Fatal("two brokers elected")
	}
	if observers.Load() != 1 {
		t.Fatal("observer is not centralized")
	}
	client := NewClient(root, nil)
	client.Publish(Event{Source: Operation, Kind: Completed, Action: VMStart, OperationID: "op-1", VM: "fixture"})
	ea, eb := receive(t, a, Completed), receive(t, b, Completed)
	if ea.Sequence != eb.Sequence || ea.Epoch != eb.Epoch || ea.OperationID != "op-1" || ea.Version != Version {
		t.Fatalf("fanout mismatch: %+v %+v", ea, eb)
	}
	cancelFirst()
	drain(t, a)
	next := receive(t, b, Resync)
	if next.Epoch == initial.Epoch {
		t.Fatal("handoff reused broker identity")
	}
	if observers.Load() != 1 {
		t.Fatal("old observer survived handoff")
	}
	client.Publish(Event{Source: Operation, Kind: Failed, Action: VMStop, OperationID: "op-2", VM: "fixture"})
	receive(t, b, Failed)
	cancelSecond()
	drain(t, b)
	if observers.Load() != 0 {
		t.Fatal("observer leaked")
	}
	if _, err := os.Lstat(filepath.Join(root, "events", "bus.sock")); !os.IsNotExist(err) {
		t.Fatalf("socket survived owner exit: %v", err)
	}
}
func TestSlowSubscriberIsDisconnectedWithoutBlockingOthers(t *testing.T) {
	slow := make(chan Event, 1)
	fast := make(chan Event, QueueSize)
	h := &hub{epoch: "test", subscribers: map[chan Event]struct{}{slow: {}, fast: {}}}
	for range 3 {
		h.publish(Event{Source: Filesystem, Kind: Changed, Resources: []string{Fleet}})
	}
	if len(fast) != 3 {
		t.Fatal("slow subscriber blocked fast subscriber")
	}
	<-slow
	if _, ok := <-slow; ok {
		t.Fatal("overflow silently dropped events instead of disconnecting")
	}
}
func TestContractRejectsUnknownAndSensitivePayloadShapes(t *testing.T) {
	for _, e := range []Event{
		{Source: "guest", Kind: "log"}, {Source: Runtime, Kind: "RTC_CHANGE", VM: "vm"},
		{Source: Operation, Kind: Completed, Action: "ssh", OperationID: "id", VM: "vm"},
		{Source: Operation, Kind: Completed, Action: VMStart, OperationID: "id", VM: "../escape"},
		{Source: Filesystem, Kind: Changed, Resources: []string{"file:///secret"}},
	} {
		if Validate(e) == nil {
			t.Fatalf("accepted out-of-contract event: %+v", e)
		}
	}
	for _, uri := range []string{"nido://vm", "nido://vm/", "nido://vm/../escape", "nido://vm/%2fetc", "nido://system/doctor", "nido://vm/x?all=true", "nido://vm/%61"} {
		if Subscribable(uri) {
			t.Fatalf("accepted resource: %s", uri)
		}
	}
}

func TestLateSubscriberReceivesCurrentRuntimeObservation(t *testing.T) {
	root := eventRoot(t)
	runtimeEvent := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	observer := func(ctx context.Context, publish Publisher) {
		publish(Event{Source: Runtime, Kind: VMPaused, VM: "paused", Resources: Resources(VMStart, "paused")})
		close(runtimeEvent)
		<-ctx.Done()
	}
	stream := NewClient(root, observer).Watch(ctx)
	receive(t, stream, Resync)
	receive(t, stream, VMPaused)
	<-runtimeEvent
	lateCtx, lateCancel := context.WithCancel(context.Background())
	defer lateCancel()
	late := NewClient(root, nil).Watch(lateCtx)
	receive(t, late, Resync)
	e := receive(t, late, VMPaused)
	if !e.Snapshot {
		t.Fatal("late subscriber missed the runtime snapshot")
	}
	lateCancel()
	drain(t, late)
	cancel()
	drain(t, stream)
}

func TestUnavailableConnectionHasLocalObservationTimestamp(t *testing.T) {
	root := eventRoot(t)
	// A fixture file at the directory path prevents both dialing and election.
	if err := os.WriteFile(filepath.Join(root, "events"), []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream := NewClient(root, nil).Watch(ctx)
	event := receive(t, stream, Unavailable)
	if event.Time.IsZero() || event.Epoch != "" {
		t.Fatal("unavailable local observation has invalid identity/time", event)
	}
	cancel()
	drain(t, stream)
}
