package provider

import (
	"context"
	"github.com/Josepavese/nido/v5/internal/pkg/identifier"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/Josepavese/nido/v5/internal/config"
	"github.com/Josepavese/nido/v5/internal/events"
	"github.com/Josepavese/nido/v5/internal/hardware"
	"github.com/Josepavese/nido/v5/internal/pkg/sysutil"
)

const observerWakeQueueSize = 1

func (p *QemuProvider) EventPublisher() events.Publisher { return p.Events }
func (p *QemuProvider) operation(action, name string) func(error) {
	return events.Begin(p.Events, action, name)
}

// ObserveEvents owns an independent read-only provider, filesystem watcher and
// QMP sessions. It never shares the mutable configuration of a TUI/MCP provider.
// The elected broker alone invokes it. Process identity is reconciled centrally
// because QMP SHUTDOWN/EOF are not proof of host process exit.
func ObserveEvents(root, configPath string, host hardware.Host) events.Observer {
	return func(ctx context.Context, emit events.Publisher) {
		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			emit(events.Event{Source: events.Service, Kind: events.Unavailable})
			return
		}
		p := NewQemuProvider(root, cfg, host)
		watcher, err := sysutil.NewDirectoryWatcher()
		if err != nil {
			emit(events.Event{Source: events.Service, Kind: events.Unavailable})
			return
		}
		defer watcher.Close()
		var workers sync.WaitGroup
		changes := make(chan sysutil.FileChange, events.QueueSize)
		watchErrors := make(chan error, observerWakeQueueSize)
		processChanged := make(chan struct{}, observerWakeQueueSize)
		workers.Go(func() {
			for {
				change, err := watcher.Next(ctx)
				if err != nil {
					select {
					case watchErrors <- err:
					case <-ctx.Done():
					}
					return
				}
				select {
				case changes <- change:
				case <-ctx.Done():
					return
				}
			}
		})
		// Child QMP workers and filesystem forwarding are joined before return.
		childCtx, cancel := context.WithCancel(ctx)
		defer func() { cancel(); watcher.Close(); workers.Wait() }()
		monitors := map[string]context.CancelFunc{}
		pids := map[string]int{}
		previous := map[string]VMStatus{}
		initialized := false
		healthy := false
		knownDirectories := map[string]bool{}
		watchFailureReported := false
		ticker := time.NewTicker(events.ReconcileInterval)
		defer ticker.Stop()
		reconcile := func() {
			watchFailed, newWatch := false, false
			for _, dir := range []string{filepath.Dir(root), root, filepath.Join(root, "run"), filepath.Join(root, "vms"), filepath.Dir(configPath), p.Config.ImageDir, p.Config.BackupDir} {
				if dir == "" {
					continue
				}
				if err := watcher.Add(dir); err != nil {
					delete(knownDirectories, dir)
					if !os.IsNotExist(err) {
						watchFailed = true
					}
				} else {
					if !knownDirectories[dir] {
						newWatch = true
					}
					knownDirectories[dir] = true
				}
			}
			if watchFailed && !watchFailureReported {
				emit(events.Event{Source: events.Service, Kind: events.Unavailable})
				watchFailureReported = true
			}
			if !watchFailed {
				watchFailureReported = false
			}

			list, err := p.List()
			if err != nil {
				healthy = false
				emit(events.Event{Source: events.Service, Kind: events.Unavailable})
				return
			}
			if !watchFailed && (!healthy || newWatch) {
				healthy = true
				emit(events.Event{Source: events.Service, Kind: events.Resync})
			}
			if watchFailed {
				healthy = false
			}
			next := map[string]VMStatus{}
			for _, v := range list {
				next[v.Name] = v
				old, exists := previous[v.Name]
				if initialized && (!exists || !reflect.DeepEqual(v, old)) {
					kind := events.Changed
					source := events.Filesystem
					if exists && old.State != v.State {
						source = events.Runtime
						if v.State == "running" {
							kind = events.VMRunning
						} else {
							kind = events.VMStopped
						}
					}
					emit(events.Event{Source: source, Kind: kind, VM: v.Name, Resources: events.Resources(events.VMConfigure, v.Name)})
				}
				if v.State == "running" && pids[v.Name] != v.PID {
					if stop := monitors[v.Name]; stop != nil {
						stop()
					}
					monitorCtx, stop := context.WithCancel(childCtx)
					monitors[v.Name] = stop
					pids[v.Name] = v.PID
					workers.Go(func() { p.observeQMP(monitorCtx, v.Name, emit) })
					workers.Go(func() {
						if sysutil.WaitProcessExit(monitorCtx, v.PID) == nil {
							select {
							case processChanged <- struct{}{}:
							case <-monitorCtx.Done():
							}
						}
					})
				}
				if v.State != "running" {
					if stop := monitors[v.Name]; stop != nil {
						stop()
						delete(monitors, v.Name)
						delete(pids, v.Name)
					}
				}
			}
			for name := range previous {
				if _, ok := next[name]; !ok {
					emit(events.Event{Source: events.Filesystem, Kind: events.VMRemoved, VM: name, Resources: events.Resources(events.VMDelete, name)})
					if stop := monitors[name]; stop != nil {
						stop()
						delete(monitors, name)
						delete(pids, name)
					}
				}
			}
			previous = next
			initialized = true

		}
		reconcile()
		for {
			select {
			case <-ctx.Done():
				return
			case <-watchErrors:
				emit(events.Event{Source: events.Service, Kind: events.Unavailable})
				return
			case <-ticker.C:
				reconcile()
			case <-processChanged:
				reconcile()
			case change := <-changes:
				resources := p.changedResources(change, configPath)
				if len(resources) > 0 {
					if change.Path == configPath {
						if err := p.ReloadConfig(configPath); err != nil {
							emit(events.Event{Source: events.Service, Kind: events.Unavailable})
						}
					}
					emit(events.Event{Source: events.Filesystem, Kind: events.Changed, Resources: resources})
					reconcile()
				} else if !change.Content {
					reconcile()
				}
			}
		}
	}
}
func (p *QemuProvider) changedResources(change sysutil.FileChange, configPath string) []string {
	path := filepath.Clean(change.Path)
	if path == filepath.Clean(configPath) {
		return []string{events.Config}
	}
	dir, base := filepath.Dir(path), filepath.Base(path)
	switch dir {
	case filepath.Join(p.RootDir, "run"):
		if strings.HasSuffix(base, ".json") || strings.HasSuffix(base, ".pid") {
			name := strings.TrimSuffix(base, filepath.Ext(base))
			if identifier.Validate(name) == nil {
				return events.Resources(events.VMConfigure, name)
			}
		}
	case filepath.Join(p.RootDir, "vms"):
		if !change.Content && strings.HasSuffix(base, ".qcow2") {
			return []string{events.Fleet}
		}
	case p.Config.BackupDir:
		if !change.Content && strings.HasSuffix(base, ".compact.qcow2") {
			return []string{events.Templates}
		}
	case p.Config.ImageDir:
		if strings.HasSuffix(base, ".yaml") || strings.HasSuffix(base, ".json") {
			return []string{events.Images, events.Blueprints}
		}
		if !change.Content && (strings.HasSuffix(base, ".qcow2") || strings.HasSuffix(base, ".iso")) {
			return []string{events.Cache, events.Images, events.Blueprints}
		}
	}
	return nil
}
