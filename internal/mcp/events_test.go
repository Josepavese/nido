package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/Josepavese/nido/v4/internal/events"
)

func TestSubscriptionsAreOptInScopedFilteredMutedAndCoalesced(t *testing.T) {
	s := NewServer(&mockProvider{})
	output := &bytes.Buffer{}
	s.output = output
	event := events.Event{Source: events.Runtime, Kind: events.VMStopped, VM: "a", Resources: []string{events.Fleet, events.VMResource("a")}}
	s.receiveEvent(event)
	s.flushEvents()
	if output.Len() != 0 {
		t.Fatal("unsolicited notification")
	}
	if err := s.eventControl("subscribe", events.VMResource("a"), []string{events.VMStopped}); err != nil {
		t.Fatal(err)
	}
	s.receiveEvent(event)
	s.flushEvents()
	if output.Len() != 0 {
		t.Fatal("notification before initialized")
	}
	s.initialized = true
	for range 100 {
		s.receiveEvent(event)
	}
	s.flushEvents()
	if got := bytes.Count(output.Bytes(), []byte("notifications/resources/updated")); got != 1 {
		t.Fatalf("burst produced %d notifications", got)
	}
	if len(s.recent) != recentEventLimit {
		t.Fatal("recent events are not bounded")
	}
	output.Reset()
	event.Kind = events.VMPaused
	s.receiveEvent(event)
	s.flushEvents()
	if output.Len() != 0 {
		t.Fatal("kind filter ignored")
	}
	event.Kind = events.VMStopped
	event.Resources = []string{events.Fleet, events.VMResource("b")}
	s.receiveEvent(event)
	s.flushEvents()
	if output.Len() != 0 {
		t.Fatal("VM filter ignored")
	}
	event.Resources = []string{events.Fleet, events.VMResource("a")}
	if err := s.eventControl("mute", events.VMResource("a"), nil); err != nil {
		t.Fatal(err)
	}
	s.receiveEvent(event)
	s.flushEvents()
	if output.Len() != 0 {
		t.Fatal("muted subscription notified")
	}
	if err := s.eventControl("unmute", events.VMResource("a"), nil); err != nil {
		t.Fatal(err)
	}
	s.flushEvents()
	if output.Len() == 0 {
		t.Fatal("unmute did not invalidate suppressed state")
	}
	output.Reset()
	if err := s.eventControl("unsubscribe", events.VMResource("a"), nil); err != nil {
		t.Fatal(err)
	}
	s.receiveEvent(event)
	s.flushEvents()
	if output.Len() != 0 {
		t.Fatal("unsubscribe ignored")
	}
	if err := s.eventControl("subscribe", "file:///etc/passwd", nil); err == nil {
		t.Fatal("non-core subscription accepted")
	}
	if err := s.eventControl("subscribe", events.Fleet, []string{"guest.log"}); err == nil {
		t.Fatal("unknown kind accepted")
	}
}
func TestMCPOutputFramesAreAtomicUnderConcurrentNotifications(t *testing.T) {
	s := NewServer(&mockProvider{})
	output := &bytes.Buffer{}
	s.output = output
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for range 50 {
				s.sendResponse("request", map[string]bool{"ok": true})
			}
		})
	}
	wg.Wait()
	decoder := json.NewDecoder(output)
	count := 0
	for {
		var frame map[string]interface{}
		err := decoder.Decode(&frame)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		count++
	}
	if count != 200 {
		t.Fatal(count)
	}
}

type testEventSource struct {
	stream  chan events.Event
	stopped chan struct{}
}

func (s *testEventSource) Watch(ctx context.Context) <-chan events.Event {
	go func() { <-ctx.Done(); close(s.stream); close(s.stopped) }()
	return s.stream
}
func TestServeEOFCancelsAndJoinsEventSource(t *testing.T) {
	source := &testEventSource{stream: make(chan events.Event), stopped: make(chan struct{})}
	server := NewServer(&mockProvider{})
	server.Events = source
	server.ServeIO(context.Background(), bytes.NewBufferString(""), io.Discard)
	select {
	case <-source.stopped:
	case <-time.After(time.Second):
		t.Fatal("event source survived EOF")
	}
}

func TestOptedInStreamHealthNotifiesOnceOnObservationLoss(t *testing.T) {
	s := NewServer(&mockProvider{})
	out := &bytes.Buffer{}
	s.output = out
	s.initialized = true
	if err := s.eventControl("subscribe", events.Stream, nil); err != nil {
		t.Fatal(err)
	}
	s.receiveEvent(events.Event{Source: events.Service, Kind: events.Resync, Epoch: "fixture"})
	s.flushEvents()
	out.Reset()
	for range 10 {
		s.receiveEvent(events.Event{Source: events.Service, Kind: events.Unavailable, Epoch: "fixture"})
		s.flushEvents()
	}
	if got := bytes.Count(out.Bytes(), []byte("notifications/resources/updated")); got != 1 {
		t.Fatalf("unavailable observer generated %d notifications", got)
	}
	if !s.connected || s.observerAvailable {
		t.Fatal("wire and observation health were conflated")
	}
}
