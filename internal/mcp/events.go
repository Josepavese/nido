package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/Josepavese/nido/v5/internal/events"
)

const notificationInterval = 500 * time.Millisecond
const recentEventLimit = 32
const subscriptionLimit = 64

type eventSubscription struct {
	Muted bool     `json:"muted"`
	Kinds []string `json:"kinds,omitempty"`
}

func eventToolCatalog() map[string]interface{} {
	return map[string]interface{}{
		"name":        "nido_events",
		"description": "Control this MCP session's opt-in Nido event subscriptions. Subscribe to a VM URI or core resource; mute/unmute keeps filters, unsubscribe removes them. No notifications by default. Read nido://system/events for stream health and recent subscribed events. Notifications contain only an invalidated URI.",
		"inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{
			"action": map[string]interface{}{"type": "string", "enum": []string{"status", "subscribe", "unsubscribe", "mute", "unmute"}},
			"uri":    map[string]interface{}{"type": "string", "description": "Core Nido resource URI; required except for status."},
			"kinds":  map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string", "enum": eventKinds()}, "description": "Optional event kinds for subscribe; empty means all core changes."},
		}, "required": []string{"action"}},
	}
}
func eventKinds() []string {
	return append([]string{events.Completed, events.Failed, events.Changed, events.VMRemoved}, events.RuntimeKinds()...)
}

func (s *Server) eventControl(action, uri string, kinds []string) error {
	if !events.Subscribable(uri) {
		return fmt.Errorf("not a subscribable core Nido resource: %q", uri)
	}
	for _, kind := range kinds {
		if !slices.Contains(eventKinds(), kind) {
			return fmt.Errorf("unknown event kind: %s", kind)
		}
	}
	s.outputMu.Lock()
	defer s.outputMu.Unlock()
	s.eventMu.Lock()
	defer s.eventMu.Unlock()
	if s.subscriptions == nil {
		s.subscriptions = map[string]eventSubscription{}
		s.pending = map[string]bool{}
	}
	switch action {
	case "subscribe":
		if _, exists := s.subscriptions[uri]; !exists && len(s.subscriptions) >= subscriptionLimit {
			return fmt.Errorf("subscription limit reached")
		}
		s.subscriptions[uri] = eventSubscription{Kinds: slices.Clone(kinds)}
	case "unsubscribe":
		delete(s.subscriptions, uri)
		delete(s.pending, uri)
	case "mute", "unmute":
		sub, ok := s.subscriptions[uri]
		if !ok {
			return fmt.Errorf("resource is not subscribed")
		}
		sub.Muted = action == "mute"
		s.subscriptions[uri] = sub
		delete(s.pending, uri)
		if action == "unmute" {
			s.pending[uri] = true
		} // Re-read changes suppressed during mute.
	default:
		return fmt.Errorf("unknown event action")
	}
	return nil
}
func (s *Server) callEventsTool(raw json.RawMessage) (interface{}, error) {
	var args struct {
		Action string   `json:"action"`
		URI    string   `json:"uri"`
		Kinds  []string `json:"kinds"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	if args.Action != "status" {
		if err := s.eventControl(args.Action, args.URI, args.Kinds); err != nil {
			return nil, err
		}
	}
	return s.eventStatus(), nil
}
func (s *Server) handleSubscription(req JSONRPCRequest) {
	var params ResourceReadParams
	if json.Unmarshal(req.Params, &params) != nil {
		s.sendError(req.ID, -32602, "Invalid params")
		return
	}
	action := "subscribe"
	if req.Method == "resources/unsubscribe" {
		action = "unsubscribe"
	}
	if err := s.eventControl(action, params.URI, nil); err != nil {
		s.sendError(req.ID, -32602, err.Error())
		return
	}
	s.sendResponse(req.ID, map[string]interface{}{})
}
func (s *Server) eventStatus() interface{} {
	s.eventMu.Lock()
	defer s.eventMu.Unlock()
	subs := make(map[string]eventSubscription, len(s.subscriptions))
	for uri, sub := range s.subscriptions {
		sub.Kinds = slices.Clone(sub.Kinds)
		subs[uri] = sub
	}
	return map[string]interface{}{"connected": s.connected, "observer_available": s.observerAvailable, "epoch": s.eventEpoch, "sequence": s.eventSequence, "resync_required": s.resyncRequired, "subscriptions": subs, "recent": slices.Clone(s.recent), "recent_limit": recentEventLimit, "notification_interval_ms": notificationInterval.Milliseconds(), "delivery": "advisory; re-read subscribed resources after resync or reconnect"}
}
func (s *Server) receiveEvent(e events.Event) {
	s.eventMu.Lock()
	defer s.eventMu.Unlock()
	if e.Kind == events.Unavailable {
		wasAvailable := s.observerAvailable
		s.connected = e.Epoch != ""
		s.observerAvailable = false
		s.resyncRequired = true
		if sub, ok := s.subscriptions[events.Stream]; wasAvailable && ok && !sub.Muted {
			s.pending[events.Stream] = true
			s.recordEventLocked(e)
		}
		return
	}
	s.connected = true
	if e.Kind == events.Resync || slices.Contains(e.Resources, events.Config) {
		s.configDirty = true
	}
	s.eventEpoch = e.Epoch
	s.eventSequence = e.Sequence
	if e.Kind == events.Resync {
		s.observerAvailable = true
		s.resyncRequired = true
	}
	if e.Kind == events.Started {
		return
	}
	interested := false
	for uri, sub := range s.subscriptions {
		if sub.Muted {
			continue
		}
		if e.Kind == events.Resync || (len(sub.Kinds) == 0 || slices.Contains(sub.Kinds, e.Kind)) && (uri == events.Stream || slices.Contains(e.Resources, uri)) {
			s.pending[uri] = true
			interested = true
		}
	}
	if interested {
		s.recordEventLocked(e)
	}
}
func (s *Server) recordEventLocked(e events.Event) {
	if len(s.recent) == recentEventLimit {
		s.recent = s.recent[1:]
	}
	s.recent = append(s.recent, e)
}
func (s *Server) runEvents(ctx context.Context, done chan<- struct{}) {
	defer close(done)
	if s.Events == nil {
		return
	}
	ctx, cancel := context.WithCancel(ctx)
	stream := s.Events.Watch(ctx)
	defer func() {
		cancel()
		for range stream {
		}
	}()
	timer := time.NewTicker(notificationInterval)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-stream:
			if !ok {
				return
			}
			s.receiveEvent(e)
		case <-timer.C:
			if err := s.flushEvents(); err != nil {
				return
			}
		}
	}
}

func (s *Server) flushEvents() error {
	s.outputMu.Lock()
	defer s.outputMu.Unlock()
	s.eventMu.Lock()
	uris := []string{}
	if s.initialized {
		for uri := range s.pending {
			sub, ok := s.subscriptions[uri]
			if ok && !sub.Muted {
				uris = append(uris, uri)
			}
			delete(s.pending, uri)
		}
	}
	s.eventMu.Unlock()
	for _, uri := range uris {
		if err := s.encodeFrame(map[string]interface{}{"jsonrpc": "2.0", "method": "notifications/resources/updated", "params": map[string]string{"uri": uri}}); err != nil {
			return err
		}
	}
	return nil
}
