package state

import (
	"os"
	"runtime"
	"syscall"
)

// CleanupStale removes state files for processes that are no longer running.
// Returns the list of removed server names.
func CleanupStale(mgr *Manager) ([]string, error) {
	states, err := mgr.ListAll()
	if err != nil {
		return nil, err
	}

	var removed []string
	for _, state := range states {
		if !isProcessAlive(state.PID) {
			if err := mgr.DeleteState(state.ServerName); err != nil {
				// Continue cleanup even if one delete fails
				continue
			}
			removed = append(removed, state.ServerName)
		}
	}

	return removed, nil
}

// isProcessAlive checks if a process with the given PID is running.
func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}

	// Platform-specific process check
	if runtime.GOOS == "windows" {
		return isProcessAliveWindows(pid)
	}
	return isProcessAliveUnix(pid)
}

// isProcessAliveUnix checks process existence on Unix-like systems.
func isProcessAliveUnix(pid int) bool {
	// Send signal 0 - doesn't actually send a signal, just checks if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// isProcessAliveWindows checks process existence on Windows.
func isProcessAliveWindows(pid int) bool {
	// On Windows, FindProcess always succeeds even for non-existent PIDs
	// We need to actually try to interact with the process to verify it exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Windows, sending signal 0 doesn't work the same way as Unix
	// Instead, we try Release which will fail if the process doesn't exist
	// This is a best-effort check - false positives are acceptable for cleanup
	err = process.Release()
	if err != nil {
		// If we can't release, assume process might exist (conservative)
		return true
	}

	// If Release succeeded, try to find it again to verify
	_, err = os.FindProcess(pid)
	return err == nil
}
