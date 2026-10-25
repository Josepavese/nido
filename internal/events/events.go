// Package events defines Nido's bounded, advisory event stream. Persistent VM
// state remains authoritative: events invalidate projections, never command VMs.
package events

import (
	"context"
	"fmt"
	"github.com/Josepavese/nido/v5/internal/pkg/identifier"
	"net/url"
	"slices"
	"strings"
	"time"
)

const (
	Version           = 1
	QueueSize         = 128
	MaxClients        = 32
	MaxFrameBytes     = 8192
	IOTimeout         = 2 * time.Second
	ReconnectDelay    = 250 * time.Millisecond
	CoalesceWindow    = 150 * time.Millisecond
	ReconcileInterval = 2 * time.Second
	Fleet             = "nido://fleet/vms"
	Templates         = "nido://fleet/templates"
	Images            = "nido://catalog/images"
	Blueprints        = "nido://catalog/blueprints"
	Cache             = "nido://storage/cache"
	Config            = "nido://system/config"
	Stream            = "nido://system/events"
	Operation         = "nido"
	Runtime           = "runtime"
	Filesystem        = "filesystem"
	Service           = "service"
	Started           = "operation.started"
	Completed         = "operation.completed"
	Failed            = "operation.failed"
	Changed           = "resources.changed"
	Resync            = "stream.resync"
	Unavailable       = "stream.unavailable"
	VMRunning         = "vm.running"
	VMStopped         = "vm.stopped"
	VMPaused          = "vm.paused"
	VMResumed         = "vm.resumed"
	VMReset           = "vm.reset"
	VMShutdown        = "vm.shutdown"
	VMPanic           = "vm.panic"
	VMIOError         = "vm.io_error"
	VMWatchdog        = "vm.watchdog"
	VMSuspended       = "vm.suspended"
	VMWakeup          = "vm.wakeup"
	VMCreate          = "vm.create"
	VMStart           = "vm.start"
	VMStop            = "vm.stop"
	VMDelete          = "vm.delete"
	VMConfigure       = "vm.configure"
	VMPortForward     = "vm.port_forward"
	VMPortUnforward   = "vm.port_unforward"
	DiskCreate        = "disk.create"
	TemplateCreate    = "template.create"
	TemplateDelete    = "template.delete"
	CacheRemove       = "cache.remove"
	CachePrune        = "cache.prune"
	ImageAcquire      = "image.acquire"
	BlueprintBuild    = "blueprint.build"
	ConfigSet         = "config.set"
	CatalogRefresh    = "catalog.refresh"
	SystemUpdate      = "system.update"
	SystemUninstall   = "system.uninstall"
	VersionResource   = "nido://system/version"
	VMRemoved         = "vm.removed"
	VMResourcePrefix  = "nido://vm/"
)

type Event struct {
	Version     int       `json:"version"`
	Epoch       string    `json:"epoch"`
	Sequence    uint64    `json:"sequence"`
	Time        time.Time `json:"time"`
	Source      string    `json:"source"`
	Kind        string    `json:"kind"`
	Action      string    `json:"action,omitempty"`
	OperationID string    `json:"operation_id,omitempty"`
	VM          string    `json:"vm,omitempty"`
	Resources   []string  `json:"resources,omitempty"`
	Snapshot    bool      `json:"snapshot,omitempty"`
}

// Publisher is synchronous and thread-safe. Delivery is advisory and bounded;
// an unavailable observer must never roll back a completed domain operation.
type Publisher func(Event)
type Observer func(context.Context, Publisher)
type Source interface {
	Watch(context.Context) <-chan Event
}

func VMResource(name string) string { return VMResourcePrefix + url.PathEscape(name) }
func Resources(action, name string) []string {
	switch action {
	case VMCreate, VMStart, VMStop, VMDelete, VMConfigure, VMPortForward, VMPortUnforward, DiskCreate:
		return []string{Fleet, VMResource(name)}
	case SystemUpdate:
		return []string{VersionResource, Images, Blueprints}
	case SystemUninstall:
		return []string{Fleet, Templates, Images, Blueprints, Cache, Config, VersionResource}
	case CatalogRefresh:
		return []string{Images, Blueprints}
	case ConfigSet:
		return []string{Config}
	case TemplateCreate, TemplateDelete:
		return []string{Templates}
	case CacheRemove, CachePrune, ImageAcquire, BlueprintBuild:
		return []string{Cache, Images, Blueprints}
	}
	return nil
}
func Subscribable(uri string) bool {
	if slices.Contains([]string{Fleet, Templates, Images, Blueprints, Cache, Config, Stream, VersionResource}, uri) {
		return true
	}
	_, err := ParseVMResource(uri)
	return err == nil
}

// ParseVMResource is shared by subscription validation and resource reads.
func ParseVMResource(uri string) (string, error) {
	if !strings.HasPrefix(uri, VMResourcePrefix) {
		return "", fmt.Errorf("not a VM resource URI")
	}
	name, err := url.PathUnescape(strings.TrimPrefix(uri, VMResourcePrefix))
	if err != nil {
		return "", err
	}
	if err := identifier.Validate(name); err != nil {
		return "", err
	}
	if VMResource(name) != uri {
		return "", fmt.Errorf("non-canonical VM resource URI; use the URI advertised by Nido")
	}
	return name, nil
}

func RuntimeKinds() []string {
	return []string{VMRunning, VMStopped, VMPaused, VMResumed, VMReset, VMShutdown, VMPanic, VMIOError, VMWatchdog, VMSuspended, VMWakeup}
}

// Begin records intent and its terminal outcome without arguments or errors.
func Begin(publish Publisher, action, name string) func(error) {
	if publish == nil {
		return func(error) {}
	}
	id := Identity()
	emit := func(kind string) {
		publish(Event{Source: Operation, Kind: kind, Action: action, OperationID: id, VM: name, Resources: Resources(action, name)})
	}
	emit(Started)
	return func(err error) {
		if err != nil {
			emit(Failed)
		} else {
			emit(Completed)
		}
	}
}

func Validate(e Event) error {
	if e.VM != "" && identifier.Validate(e.VM) != nil {
		return fmt.Errorf("invalid VM identity")
	}
	switch e.Source {
	case Operation:
		if e.Kind != Started && e.Kind != Completed && e.Kind != Failed {
			return fmt.Errorf("invalid operation event")
		}
		if len(Resources(e.Action, e.VM)) == 0 {
			return fmt.Errorf("unknown action")
		}
		if e.OperationID == "" {
			return fmt.Errorf("missing operation identity")
		}
	case Runtime:
		if e.VM == "" || !slices.Contains(RuntimeKinds(), e.Kind) {
			return fmt.Errorf("invalid runtime event")
		}
	case Filesystem:
		if e.Kind != Changed && e.Kind != VMRemoved {
			return fmt.Errorf("invalid filesystem event")
		}
	case Service:
		if e.Kind != Resync && e.Kind != Unavailable {
			return fmt.Errorf("invalid service event")
		}
	default:
		return fmt.Errorf("unknown source")
	}
	for _, uri := range e.Resources {
		if !Subscribable(uri) {
			return fmt.Errorf("unknown resource")
		}
	}
	return nil
}
