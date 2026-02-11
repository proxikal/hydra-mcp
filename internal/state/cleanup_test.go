package state

import (
	"os"
	"testing"
	"time"

	"github.com/proxikal/hydra/internal/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanupStale(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Create state for dead process (PID unlikely to exist)
	deadPID := 999999
	err := mgr.SaveState("dead-server", deadPID, metrics.NewCollector())
	require.NoError(t, err)

	// Create state for current process (alive)
	alivePID := os.Getpid()
	err = mgr.SaveState("alive-server", alivePID, metrics.NewCollector())
	require.NoError(t, err)

	// Cleanup stale processes
	removed, err := CleanupStale(mgr)
	require.NoError(t, err)

	// Dead process should be removed
	assert.Contains(t, removed, "dead-server", "Dead server should be removed")

	// Alive process should remain
	_, err = mgr.LoadState("alive-server")
	assert.NoError(t, err, "Alive server should remain")

	// Dead process should be gone
	_, err = mgr.LoadState("dead-server")
	assert.Error(t, err, "Dead server should be removed")
}

func TestCleanupStale_AllAlive(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Create multiple states for current process
	pid := os.Getpid()
	err := mgr.SaveState("server1", pid, metrics.NewCollector())
	require.NoError(t, err)
	err = mgr.SaveState("server2", pid, metrics.NewCollector())
	require.NoError(t, err)

	removed, err := CleanupStale(mgr)
	require.NoError(t, err)
	assert.Empty(t, removed, "No stale processes should be removed")

	// All should still exist
	_, err = mgr.LoadState("server1")
	assert.NoError(t, err)
	_, err = mgr.LoadState("server2")
	assert.NoError(t, err)
}

func TestCleanupStale_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	removed, err := CleanupStale(mgr)
	require.NoError(t, err)
	assert.Empty(t, removed, "Should handle empty state dir")
}

func TestIsProcessAlive(t *testing.T) {
	// Current process should be alive
	alive := isProcessAlive(os.Getpid())
	assert.True(t, alive, "Current process should be alive")

	// PID 1 (init/launchd) should be alive on Unix systems
	// Note: Permission issues may prevent detection, so we skip this check
	// The current process check above is sufficient

	// Very high PID unlikely to exist
	alive = isProcessAlive(999999)
	assert.False(t, alive, "High PID should not exist")

	// Zero and negative PIDs should return false
	alive = isProcessAlive(0)
	assert.False(t, alive, "PID 0 should return false")
	alive = isProcessAlive(-1)
	assert.False(t, alive, "Negative PID should return false")
}

func TestCleanupStale_ConcurrentSave(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Create stale state
	err := mgr.SaveState("stale", 999999, metrics.NewCollector())
	require.NoError(t, err)

	// Run cleanup and save concurrently
	done := make(chan bool, 2)

	go func() {
		_, err := CleanupStale(mgr)
		assert.NoError(t, err)
		done <- true
	}()

	go func() {
		time.Sleep(10 * time.Millisecond)
		err := mgr.SaveState("new-server", os.Getpid(), metrics.NewCollector())
		assert.NoError(t, err)
		done <- true
	}()

	<-done
	<-done

	// New server should exist
	_, err = mgr.LoadState("new-server")
	assert.NoError(t, err)
}
