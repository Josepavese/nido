package sysutil

import (
	"context"
	"github.com/fsnotify/fsnotify"
	"io"
)

// DirectoryWatcher confines native filesystem notification APIs to PAL. It is
// owned by one observer goroutine; Close releases backend handles on all hosts.
type DirectoryWatcher struct{ watcher *fsnotify.Watcher }
type FileChange struct {
	Path    string
	Content bool
}

func NewDirectoryWatcher() (*DirectoryWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &DirectoryWatcher{w}, nil
}
func (w *DirectoryWatcher) Add(path string) error { return w.watcher.Add(path) }
func (w *DirectoryWatcher) Close() error          { return w.watcher.Close() }
func (w *DirectoryWatcher) Next(ctx context.Context) (FileChange, error) {
	select {
	case <-ctx.Done():
		return FileChange{}, ctx.Err()
	case err, ok := <-w.watcher.Errors:
		if !ok {
			return FileChange{}, io.EOF
		}
		return FileChange{}, err
	case e, ok := <-w.watcher.Events:
		if !ok {
			return FileChange{}, io.EOF
		}
		return FileChange{Path: e.Name, Content: e.Op&(fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0}, nil
	}
}
