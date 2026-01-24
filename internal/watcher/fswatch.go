package watcher

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	ignore "github.com/sabhiram/go-gitignore"

	"github.com/proxikal/hydra/internal/logger"
)

type fsWatcher struct {
	paths        []string
	ignoreList   *ignore.GitIgnore
	debounce     time.Duration
	batchWindow  time.Duration
	logger       logger.Logger
	watcher      *fsnotify.Watcher
	events       chan WatchEvent
	done         chan struct{}
	mu           sync.Mutex // Protects pendingEvent and lastEvent
	pendingEvent *WatchEvent
	lastEvent    time.Time
	batchTimer   *time.Timer
}

// New creates a new file system watcher with debouncing
func New(paths []string, ignorePatterns []string, debounce time.Duration, batchWindow time.Duration, log logger.Logger) (Watcher, error) {
	// Compile ignore patterns
	var ignoreList *ignore.GitIgnore
	if len(ignorePatterns) > 0 {
		ignoreList = ignore.CompileIgnoreLines(ignorePatterns...)
	}

	w := &fsWatcher{
		paths:       paths,
		ignoreList:  ignoreList,
		debounce:    debounce,
		batchWindow: batchWindow,
		logger:      log,
		events:      make(chan WatchEvent, 10),
		done:        make(chan struct{}),
	}

	return w, nil
}

func (w *fsWatcher) Start() error {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	w.watcher = fsw

	// Add paths recursively (including all subdirectories)
	for _, path := range w.paths {
		if err := w.addPathRecursive(path); err != nil {
			_ = w.watcher.Close()
			return fmt.Errorf("failed to watch path %s: %w", path, err)
		}
	}

	go w.run()

	return nil
}

func (w *fsWatcher) Stop() error {
	if w.watcher != nil {
		close(w.done)
		if err := w.watcher.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (w *fsWatcher) Events() <-chan WatchEvent {
	return w.events
}

func (w *fsWatcher) run() {
	defer close(w.events)

	for {
		select {
		case <-w.done:
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Check if path should be ignored
			if w.shouldIgnore(event.Name) {
				continue
			}

			// Handle directory creation - add new directories to watcher
			if event.Op&fsnotify.Create == fsnotify.Create {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					if err := w.addPathRecursive(event.Name); err != nil {
						w.logger.Error("failed to watch new directory", err, map[string]interface{}{
							"path": event.Name,
						})
					}
				}
			}

			now := time.Now()

			w.mu.Lock()

			// If no pending event, create one
			if w.pendingEvent == nil {
				w.pendingEvent = &WatchEvent{
					Path:      event.Name,
					Timestamp: now,
				}

				// Start batch timer
				if w.batchTimer != nil {
					w.batchTimer.Stop()
				}
				w.batchTimer = time.AfterFunc(w.debounce, func() {
					w.mu.Lock()
					pending := w.pendingEvent
					w.pendingEvent = nil
					w.mu.Unlock()
					w.flushEvent(pending)
				})
				w.mu.Unlock()
			} else {
				// Update pending event timestamp
				w.pendingEvent.Timestamp = now

				// Reset debounce timer
				if w.batchTimer != nil {
					w.batchTimer.Stop()
				}

				// Check if batch window expired
				if now.Sub(w.lastEvent) > w.batchWindow {
					// Flush immediately
					pending := w.pendingEvent
					w.pendingEvent = nil
					w.mu.Unlock()
					w.flushEvent(pending)
				} else {
					// Continue debouncing
					w.batchTimer = time.AfterFunc(w.debounce, func() {
						w.mu.Lock()
						pending := w.pendingEvent
						w.pendingEvent = nil
						w.mu.Unlock()
						w.flushEvent(pending)
					})
					w.mu.Unlock()
				}
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			w.logger.Error("watcher error", err, nil)
		}
	}
}

func (w *fsWatcher) flushEvent(event *WatchEvent) {
	if event == nil {
		return
	}

	w.mu.Lock()
	w.lastEvent = time.Now()
	w.mu.Unlock()

	select {
	case w.events <- *event:
	case <-w.done:
	}
}
