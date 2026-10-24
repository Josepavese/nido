package sysutil

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
)

const eventDirectory = "events"
const eventSocket = "bus.sock"
const eventLock = "owner.lock"
const eventDirectoryMode = 0700
const eventFileMode = 0600

func EventsDirectory(root string) string { return filepath.Join(root, eventDirectory) }
func eventPath(root string) string       { return filepath.Join(EventsDirectory(root), eventSocket) }
func DialLocalEvents(ctx context.Context, root string) (net.Conn, error) {
	return (&net.Dialer{}).DialContext(ctx, "unix", eventPath(root))
}

type eventListener struct {
	net.Listener
	release func()
	once    sync.Once
}

func (l *eventListener) Release() { l.once.Do(l.release) }

type LocalEventListener interface {
	net.Listener
	Release()
}

func (l *eventListener) Close() error {
	return l.Listener.Close()
}

// ListenLocalEvents elects one owner using an OS-released lock. Only the lock
// owner may retire a stale socket. AF_UNIX is supported on Linux, macOS and
// Windows 10 1803+. The private directory inherits the user's Windows home ACL.
func ListenLocalEvents(root string) (LocalEventListener, error) {
	dir, err := PrivateEventsDirectory(root)
	if err != nil {
		return nil, err
	}
	release, err := lockEventOwner(filepath.Join(dir, eventLock))
	if err != nil {
		return nil, err
	}
	ok := false
	defer func() {
		if !ok {
			release()
		}
	}()
	path := eventPath(root)
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSocket == 0 {
			return nil, fmt.Errorf("refusing to replace non-socket event endpoint")
		}
		if err := os.Remove(path); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, eventFileMode); err != nil {
		ln.Close()
		return nil, err
	}
	ok = true
	return &eventListener{Listener: ln, release: release}, nil
}

func PrivateEventsDirectory(root string) (string, error) {
	dir := EventsDirectory(root)
	if err := os.MkdirAll(dir, eventDirectoryMode); err != nil {
		return "", err
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return "", err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("event directory must be a private directory")
	}
	if err := privateEventDirectory(info); err != nil {
		return "", err
	}
	return dir, nil
}
