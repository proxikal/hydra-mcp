package watcher

import (
	"log"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watch starts watching the directory and calls onChange when files change
func Watch(root string, onChange func()) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	// Add root directory
	// TODO: Walk directory for recursive watching
	if err := watcher.Add(root); err != nil {
		return err
	}

	var debounceTimer *time.Timer
	debounceDuration := 500 * time.Millisecond

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok { return }

				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
					log.Printf("📂 File changed: %s", filepath.Base(event.Name))

					// Debounce
					if debounceTimer != nil {
						debounceTimer.Stop()
					}
					debounceTimer = time.AfterFunc(debounceDuration, onChange)
				}
			case err, ok := <-watcher.Errors:
				if !ok { return }
				log.Println("error:", err)
			}
		}
	}()

	// Block forever (in this goroutine, but we usually run this in background)
	select {}
}
