package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/proxikal/hydra/internal/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	require.NotNil(t, mgr)
	assert.Equal(t, tmpDir, mgr.stateDir)
}

func TestSaveState(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	collector := metrics.NewCollector()
	collector.RecordRequest(50*time.Millisecond, true)

	err := mgr.SaveState("test-server", os.Getpid(), collector)
	require.NoError(t, err)

	// Verify file exists
	statePath := filepath.Join(tmpDir, "test-server.json")
	assert.FileExists(t, statePath)

	// Verify file content
	data, err := os.ReadFile(statePath)
	require.NoError(t, err)

	var state State
	err = json.Unmarshal(data, &state)
	require.NoError(t, err)

	assert.Equal(t, "test-server", state.ServerName)
	assert.Equal(t, os.Getpid(), state.PID)
	assert.NotNil(t, state.Metrics)
	assert.Equal(t, int64(1), state.Metrics.RequestsTotal)
}

func TestLoadState(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Create a state file
	collector := metrics.NewCollector()
	collector.RecordRequest(100*time.Millisecond, true)
	collector.RecordRestart("test")

	err := mgr.SaveState("load-test", 12345, collector)
	require.NoError(t, err)

	// Load it back
	state, err := mgr.LoadState("load-test")
	require.NoError(t, err)

	assert.Equal(t, "load-test", state.ServerName)
	assert.Equal(t, 12345, state.PID)
	assert.Equal(t, int64(1), state.Metrics.RequestsTotal)
	assert.Equal(t, int64(1), state.Metrics.RestartsTotal)
	assert.NotZero(t, state.LastUpdated)
}

func TestLoadState_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	_, err := mgr.LoadState("nonexistent")
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err), "Error should be not exist")
}

func TestLoadState_Invalid(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Write invalid JSON
	statePath := filepath.Join(tmpDir, "invalid.json")
	err := os.WriteFile(statePath, []byte("invalid json"), 0644)
	require.NoError(t, err)

	_, err = mgr.LoadState("invalid")
	assert.Error(t, err)
}

func TestDeleteState(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Save a state
	err := mgr.SaveState("delete-test", 12345, metrics.NewCollector())
	require.NoError(t, err)

	statePath := filepath.Join(tmpDir, "delete-test.json")
	assert.FileExists(t, statePath)

	// Delete it
	err = mgr.DeleteState("delete-test")
	require.NoError(t, err)

	// Verify it's gone
	_, err = os.Stat(statePath)
	assert.True(t, os.IsNotExist(err))
}

func TestDeleteState_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Deleting nonexistent state should not error
	err := mgr.DeleteState("nonexistent")
	assert.NoError(t, err)
}

func TestListAll(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Create multiple state files
	err := mgr.SaveState("server1", 100, metrics.NewCollector())
	require.NoError(t, err)
	err = mgr.SaveState("server2", 200, metrics.NewCollector())
	require.NoError(t, err)
	err = mgr.SaveState("server3", 300, metrics.NewCollector())
	require.NoError(t, err)

	states, err := mgr.ListAll()
	require.NoError(t, err)
	require.Len(t, states, 3)

	// Verify all servers present
	names := make(map[string]bool)
	for _, state := range states {
		names[state.ServerName] = true
	}
	assert.True(t, names["server1"])
	assert.True(t, names["server2"])
	assert.True(t, names["server3"])
}

func TestListAll_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	states, err := mgr.ListAll()
	require.NoError(t, err)
	assert.Empty(t, states)
}

func TestListAll_SkipsInvalidFiles(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Create valid state
	err := mgr.SaveState("valid", 100, metrics.NewCollector())
	require.NoError(t, err)

	// Create invalid file
	err = os.WriteFile(filepath.Join(tmpDir, "invalid.json"), []byte("bad json"), 0644)
	require.NoError(t, err)

	// Create non-JSON file
	err = os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("hello"), 0644)
	require.NoError(t, err)

	states, err := mgr.ListAll()
	require.NoError(t, err)

	// Should only get valid state
	require.Len(t, states, 1)
	assert.Equal(t, "valid", states[0].ServerName)
}

func TestStateTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	before := time.Now()
	err := mgr.SaveState("timestamp-test", 12345, metrics.NewCollector())
	require.NoError(t, err)
	after := time.Now()

	state, err := mgr.LoadState("timestamp-test")
	require.NoError(t, err)

	assert.True(t, state.LastUpdated.After(before) || state.LastUpdated.Equal(before))
	assert.True(t, state.LastUpdated.Before(after) || state.LastUpdated.Equal(after))
}

func TestConcurrentSave(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Save from multiple goroutines
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			collector := metrics.NewCollector()
			collector.RecordRequest(time.Duration(id)*time.Millisecond, true)
			err := mgr.SaveState("concurrent-test", os.Getpid(), collector)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should be able to load without corruption
	state, err := mgr.LoadState("concurrent-test")
	require.NoError(t, err)
	assert.Equal(t, "concurrent-test", state.ServerName)
}

func TestStateDir_Created(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "nested", "state")

	// Directory doesn't exist yet
	_, err := os.Stat(stateDir)
	assert.True(t, os.IsNotExist(err))

	mgr := NewManager(stateDir)
	err = mgr.SaveState("test", 12345, metrics.NewCollector())
	require.NoError(t, err)

	// Directory should now exist
	info, err := os.Stat(stateDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}
