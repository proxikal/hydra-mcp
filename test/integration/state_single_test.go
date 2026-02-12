package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/proxikal/hydra/internal/config"
	"github.com/proxikal/hydra/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStatePersistence_SingleServer validates that state files are created
// and updated correctly for a single hydra instance.
func TestStatePersistence_SingleServer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	serverName := "state-test"
	stateDir := filepath.Join(tmpDir, ".hydra/state")

	// Create registry
	registryPath := filepath.Join(tmpDir, "config.json")
	pythonServer := getFixturePath("python_server.py")
	require.FileExists(t, pythonServer)

	registry := config.DefaultRegistry()
	cfg := config.DefaultServerConfig()
	cfg.Command = "python3"
	cfg.Args = []string{pythonServer}
	registry.Servers[serverName] = cfg

	data, _ := json.Marshal(registry)
	require.NoError(t, os.WriteFile(registryPath, data, 0644))

	// Start hydra (stderr is automatically captured by helper)
	cmd, stdin, stdout := startHydraServer(t, serverName, registryPath, tmpDir)

	defer func() {
		stdin.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()

	// Send MCP initialize
	sendMCPInitialize(t, stdin, stdout, 1)

	// Wait for initial state save
	time.Sleep(1 * time.Second)

	// Verify state file exists
	stateMgr := state.NewManager(stateDir)
	serverState, err := stateMgr.LoadState(serverName)
	require.NoError(t, err, "State file should exist after initialization")
	require.NotNil(t, serverState)

	assert.Equal(t, serverName, serverState.ServerName)
	assert.Greater(t, serverState.PID, 0)
	assert.NotNil(t, serverState.Metrics)

	t.Logf("State saved: PID=%d, StartTime=%v", serverState.PID, serverState.StartTime)

	// Wait for periodic state update (every 5 seconds)
	time.Sleep(6 * time.Second)

	// Verify state was updated
	updatedState, err := stateMgr.LoadState(serverName)
	require.NoError(t, err)
	assert.True(t, updatedState.LastUpdated.After(serverState.LastUpdated),
		"State should be updated periodically")

	t.Logf("State updated: %v -> %v",
		serverState.LastUpdated, updatedState.LastUpdated)
}

// TestStatePersistence_StateContent validates that state file contains
// correct metrics and metadata.
func TestStatePersistence_StateContent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	serverName := "content-test"
	stateDir := filepath.Join(tmpDir, ".hydra/state")

	// Create registry
	registryPath := filepath.Join(tmpDir, "config.json")
	pythonServer := getFixturePath("python_server.py")
	require.FileExists(t, pythonServer)

	registry := config.DefaultRegistry()
	cfg := config.DefaultServerConfig()
	cfg.Command = "python3"
	cfg.Args = []string{pythonServer}
	registry.Servers[serverName] = cfg

	data, _ := json.Marshal(registry)
	require.NoError(t, os.WriteFile(registryPath, data, 0644))

	// Start hydra
	cmd, stdin, stdout := startHydraServer(t, serverName, registryPath, tmpDir)

	defer func() {
		stdin.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()

	// Initialize
	sendMCPInitialize(t, stdin, stdout, 1)
	time.Sleep(1 * time.Second)

	// Load and verify state content
	stateMgr := state.NewManager(stateDir)
	serverState, err := stateMgr.LoadState(serverName)
	require.NoError(t, err)

	// Verify all fields are populated
	assert.NotEmpty(t, serverState.ServerName)
	assert.Greater(t, serverState.PID, 0)
	assert.False(t, serverState.StartTime.IsZero())
	assert.False(t, serverState.LastUpdated.IsZero())
	assert.NotNil(t, serverState.Metrics)

	// Verify JSON file is well-formed
	stateFile := filepath.Join(stateDir, serverName+".json")
	rawData, err := os.ReadFile(stateFile)
	require.NoError(t, err)

	var jsonState map[string]interface{}
	err = json.Unmarshal(rawData, &jsonState)
	require.NoError(t, err, "State file should contain valid JSON")

	// Check required fields exist in JSON
	assert.Contains(t, jsonState, "server_name")
	assert.Contains(t, jsonState, "pid")
	assert.Contains(t, jsonState, "start_time")
	assert.Contains(t, jsonState, "last_updated")
	assert.Contains(t, jsonState, "metrics")

	t.Logf("State file validated: %s", stateFile)
}
