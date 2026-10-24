package events

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/Josepavese/nido/v4/internal/pkg/sysutil"
)

const publishPath = "/publish"
const watchPath = "/watch"
const identityBytes = 16
const localURL = "http://nido.local"

func Identity() string {
	var b [identityBytes]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}

// Client is a process-owned, thread-safe local event connection. Watch owns the
// elected broker until ctx cancellation, joins all its workers, then closes out.
// There is no daemon or durable event journal. Each connection starts with a
// resync barrier; consumers must re-read their projections after reconnect.
type Client struct {
	root    string
	observe Observer
}

func NewClient(root string, observe Observer) *Client { return &Client{root: root, observe: observe} }
func (c *Client) transport() *http.Transport {
	return &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) { return sysutil.DialLocalEvents(ctx, c.root) }, DisableKeepAlives: true}
}
func (c *Client) Publish(e Event) {
	if Validate(e) != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), IOTimeout)
	defer cancel()
	transport := c.transport()
	defer transport.CloseIdleConnections()
	data, _ := json.Marshal(e)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, localURL+publishPath, bytes.NewReader(data))
	resp, err := (&http.Client{Transport: transport}).Do(req)
	if err == nil {
		io.Copy(io.Discard, io.LimitReader(resp.Body, MaxFrameBytes))
		resp.Body.Close()
	}
}

func (c *Client) Watch(ctx context.Context) <-chan Event {
	out := make(chan Event, QueueSize)
	go func() {
		defer close(out)
		var ownerCancel context.CancelFunc
		var ownerDone <-chan struct{}
		defer func() {
			if ownerCancel != nil {
				ownerCancel()
				<-ownerDone
			}
		}()
		for ctx.Err() == nil {
			transport := c.transport()
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, localURL+watchPath, nil)
			resp, err := (&http.Client{Transport: transport}).Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				scan := bufio.NewScanner(resp.Body)
				scan.Buffer(make([]byte, MaxFrameBytes), MaxFrameBytes)
				for scan.Scan() {
					var e Event
					if json.Unmarshal(scan.Bytes(), &e) != nil || e.Version != Version {
						break
					}
					select {
					case out <- e:
					case <-ctx.Done():
						resp.Body.Close()
						transport.CloseIdleConnections()
						return
					}
				}
				resp.Body.Close()
			} else if resp != nil {
				resp.Body.Close()
			}
			transport.CloseIdleConnections()
			if ctx.Err() != nil {
				return
			}
			if ownerDone != nil {
				select {
				case <-ownerDone:
					ownerCancel = nil
					ownerDone = nil
				default:
				}
			}
			if ownerCancel == nil {
				if ln, err := sysutil.ListenLocalEvents(c.root); err == nil {
					ownerCtx, cancel := context.WithCancel(ctx)
					ownerCancel = cancel
					done := make(chan struct{})
					ownerDone = done
					go func() { defer close(done); serve(ownerCtx, ln, c.observe) }()
					continue
				}
			}
			select {
			case out <- Event{Version: Version, Time: time.Now().UTC(), Source: Service, Kind: Unavailable}:
			default:
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(ReconnectDelay):
			}
		}
	}()
	return out
}

type hub struct {
	mu          sync.Mutex
	epoch       string
	sequence    uint64
	degraded    bool
	subscribers map[chan Event]struct{}
	runtime     map[string]Event
}

func (h *hub) publish(e Event) {
	if Validate(e) != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if e.Source == Service {
		h.degraded = e.Kind == Unavailable
	}
	h.sequence++
	e.Version = Version
	e.Epoch = h.epoch
	e.Sequence = h.sequence
	e.Time = time.Now().UTC()
	e.Snapshot = false
	if e.Source == Service && e.Kind == Unavailable {
		clear(h.runtime)
	}
	if e.Source == Runtime {
		if h.runtime == nil {
			h.runtime = map[string]Event{}
		}
		if e.Kind == VMStopped {
			delete(h.runtime, e.VM)
		} else {
			h.runtime[e.VM] = e
		}
	}
	if e.Kind == VMRemoved || e.Action == VMDelete && e.Kind == Completed {
		delete(h.runtime, e.VM)
	}
	h.sendLocked(e)
	if e.Source == Service && e.Kind == Resync {
		for _, snapshot := range h.runtime {
			snapshot.Snapshot = true
			snapshot.Sequence = h.sequence
			h.sendLocked(snapshot)
		}
	}
}
func (h *hub) sendLocked(e Event) {
	for ch := range h.subscribers {
		select {
		case ch <- e:
		default:
			delete(h.subscribers, ch)
			close(ch)
		}
	}
}
func serve(ctx context.Context, ln sysutil.LocalEventListener, observe Observer) {
	defer ln.Release()
	h := &hub{epoch: Identity(), subscribers: make(map[chan Event]struct{})}
	mux := http.NewServeMux()
	slots := make(chan struct{}, MaxClients)
	mux.HandleFunc(watchPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		select {
		case slots <- struct{}{}:
			defer func() { <-slots }()
		default:
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		ch := make(chan Event, QueueSize)
		h.mu.Lock()
		h.subscribers[ch] = struct{}{}
		snapshots := make([]Event, 0, len(h.runtime))
		for _, e := range h.runtime {
			e.Snapshot = true
			e.Sequence = h.sequence
			snapshots = append(snapshots, e)
		}
		ch <- Event{Version: Version, Epoch: h.epoch, Sequence: h.sequence, Time: time.Now().UTC(), Source: Service, Kind: Resync}
		if h.degraded {
			ch <- Event{Version: Version, Epoch: h.epoch, Sequence: h.sequence, Time: time.Now().UTC(), Source: Service, Kind: Unavailable}
		}
		h.mu.Unlock()
		defer func() { h.mu.Lock(); delete(h.subscribers, ch); h.mu.Unlock() }()
		w.Header().Set("Content-Type", "application/x-ndjson")
		controller := http.NewResponseController(w)
		controller.SetWriteDeadline(time.Now().Add(IOTimeout))
		w.WriteHeader(http.StatusOK)
		controller.Flush()
		enc := json.NewEncoder(w)
		// Emit the barrier before cached runtime observations, then live events.
		initial := <-ch
		if enc.Encode(initial) != nil {
			return
		}
		for _, e := range snapshots {
			controller.SetWriteDeadline(time.Now().Add(IOTimeout))
			if enc.Encode(e) != nil {
				return
			}
			if controller.Flush() != nil {
				return
			}
		}
		if controller.Flush() != nil {
			return
		}
		for {
			select {
			case <-ctx.Done():
				return
			case <-r.Context().Done():
				return
			case e, ok := <-ch:
				if !ok {
					return
				}
				controller.SetWriteDeadline(time.Now().Add(IOTimeout))
				if enc.Encode(e) != nil {
					return
				}
				if controller.Flush() != nil {
					return
				}
			}
		}
	})
	mux.HandleFunc(publishPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var e Event
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxFrameBytes))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&e) != nil || decoder.Decode(new(interface{})) != io.EOF || Validate(e) != nil || e.Source != Operation {
			http.Error(w, "invalid Nido event", http.StatusBadRequest)
			return
		}
		e.Resources = Resources(e.Action, e.VM)
		h.publish(e)
		w.WriteHeader(http.StatusNoContent)
	})
	var handlerMu sync.Mutex
	var handlers sync.WaitGroup
	closing := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerMu.Lock()
		if closing {
			handlerMu.Unlock()
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		handlers.Add(1)
		handlerMu.Unlock()
		defer handlers.Done()
		mux.ServeHTTP(w, r)
	})
	server := &http.Server{Handler: handler, ReadHeaderTimeout: IOTimeout, ReadTimeout: IOTimeout, MaxHeaderBytes: MaxFrameBytes}
	observerCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		if observe != nil {
			for observerCtx.Err() == nil {
				observe(observerCtx, h.publish)
				if observerCtx.Err() != nil {
					return
				}
				h.publish(Event{Source: Service, Kind: Unavailable})
				select {
				case <-observerCtx.Done():
					return
				case <-time.After(ReconnectDelay):
				}
			}
		}
	}()
	stopped := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			server.Close()
		case <-stopped:
		}
	}()
	server.Serve(ln)
	handlerMu.Lock()
	closing = true
	handlerMu.Unlock()
	server.Close()
	handlers.Wait()
	close(stopped)
	cancel()
	<-done
	ln.Close()
}
