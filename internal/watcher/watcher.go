package watcher

import "time"

// WatchEvent represents a file system change event
type WatchEvent struct {
	Path      string
	Timestamp time.Time
}

// Watcher monitors file system changes with debouncing
type Watcher interface {
	Start() error
	Stop() error
	Events() <-chan WatchEvent
}
