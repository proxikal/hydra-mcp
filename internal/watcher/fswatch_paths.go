package watcher

import (
	"fmt"
	"os"
	"path/filepath"
)

// shouldIgnore checks if a path matches ignore patterns
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

// addPathRecursive recursively adds a directory and all subdirectories to the watcher
func (w *fsWatcher) addPathRecursive(root string) error {
	// Check if root path exists before walking
	if _, err := os.Stat(root); err != nil {
		return fmt.Errorf("cannot access watch path: %w", err)
	}

	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// For root path, return error. For subdirectories, log and skip.
			if path == root {
				return err
			}
			w.logger.Error("error accessing path during watch setup", err, map[string]interface{}{
				"path": path,
			})
			return filepath.SkipDir
		}

		// Only watch directories
		if !info.IsDir() {
			return nil
		}

		// Check if directory should be ignored
		if w.shouldIgnore(path) {
			return filepath.SkipDir
		}

		// Add directory to watcher
		if err := w.watcher.Add(path); err != nil {
			w.logger.Error("failed to watch directory", err, map[string]interface{}{
				"path": path,
			})
			// Don't return error, continue walking
			return nil
		}

		// Only log root directory, not every subdirectory (reduces log spam)
		if path == root {
			w.logger.Info("watching directory tree", map[string]interface{}{
				"root": path,
			})
		}

		return nil
	})
}
