package watcher

import (
	"fmt"
	"path/filepath"
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

	// Add paths
	for _, path := range w.paths {
		if err := w.watcher.Add(path); err != nil {
			w.watcher.Close()
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

	var pendingEvent *WatchEvent

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

			now := time.Now()

			// If no pending event, create one
			if pendingEvent == nil {
				pendingEvent = &WatchEvent{
					Path:      event.Name,
					Timestamp: now,
				}

				// Start batch timer
				if w.batchTimer != nil {
					w.batchTimer.Stop()
				}
				w.batchTimer = time.AfterFunc(w.debounce, func() {
					w.flushEvent(pendingEvent)
					pendingEvent = nil
				})
			} else {
				// Update pending event timestamp
				pendingEvent.Timestamp = now

				// Reset debounce timer
				if w.batchTimer != nil {
					w.batchTimer.Stop()
				}

				// Check if batch window expired
				if now.Sub(w.lastEvent) > w.batchWindow {
					// Flush immediately
					w.flushEvent(pendingEvent)
					pendingEvent = nil
				} else {
					// Continue debouncing
					w.batchTimer = time.AfterFunc(w.debounce, func() {
						w.flushEvent(pendingEvent)
						pendingEvent = nil
					})
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

func (w *fsWatcher) shouldIgnore(path string) bool {
	if w.ignoreList == nil {
		return false
	}

	// Get relative path for matching
	for _, watchPath := range w.paths {
		if rel, err := filepath.Rel(watchPath, path); err == nil {
			if w.ignoreList.MatchesPath(rel) {
				return true
			}
		}
	}

	// Also check basename
	return w.ignoreList.MatchesPath(filepath.Base(path))
}

func (w *fsWatcher) flushEvent(event *WatchEvent) {
	if event == nil {
		return
	}

	w.lastEvent = time.Now()

	select {
	case w.events <- *event:
	case <-w.done:
	}
}
