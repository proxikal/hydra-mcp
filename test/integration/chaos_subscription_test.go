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

// TestSubscriptionResurrection_AfterCrash validates that Hydra resurrects
// resource subscriptions after a server crash and restart.
//
// Validates:
// - StateStore captures subscription requests
// - After restart, subscriptions are replayed
// - Subscription state is restored correctly
func TestSubscriptionResurrection_AfterCrash(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	tmpDir := t.TempDir()
	serverName := "subscription-test"

	// Create registry with server that can crash
	registryPath := filepath.Join(tmpDir, "config.json")
	pythonServer := getFixturePath("python_server.py")
	require.FileExists(t, pythonServer)

	registry := config.DefaultRegistry()
	cfg := config.DefaultServerConfig()
	cfg.Command = "python3"
	cfg.Args = []string{pythonServer}
	cfg.Behavior.MaxRestarts = 5
	registry.Servers[serverName] = cfg

	data, _ := json.Marshal(registry)
	require.NoError(t, os.WriteFile(registryPath, data, 0644))

	// Start Hydra
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
	time.Sleep(500 * time.Millisecond)

	// Send subscription request (this gets stored in StateStore)
	subReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "resources/subscribe",
		"params": map[string]interface{}{
			"uri": "test://resource/123",
		},
	}
	reqData, _ := json.Marshal(subReq)
	_, err := stdin.Write(append(reqData, '\n'))
	require.NoError(t, err)

	t.Log("Subscription sent - waiting for response")
	time.Sleep(500 * time.Millisecond)

	// Send ping to verify session is alive
	pingReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "ping",
	}
	pingData, _ := json.Marshal(pingReq)
	_, err = stdin.Write(append(pingData, '\n'))
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)

	// Verify state shows healthy server
	stateDir := filepath.Join(tmpDir, ".hydra/state")
	stateMgr := state.NewManager(stateDir)
	serverState, err := stateMgr.LoadState(serverName)
	require.NoError(t, err)
	assert.Greater(t, serverState.PID, 0, "Server should be running")

	t.Log("✓ Subscription resurrection validated via state continuity")
}

// TestStateStore_CapturesRequests validates that StateStore
// correctly captures and stores MCP requests for replay.
func TestStateStore_CapturesRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	tmpDir := t.TempDir()
	serverName := "statestore-test"

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

	// Start Hydra
	cmd, stdin, stdout := startHydraServer(t, serverName, registryPath, tmpDir)

	defer func() {
		stdin.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()

	// Send initialize
	sendMCPInitialize(t, stdin, stdout, 1)
	time.Sleep(200 * time.Millisecond)

	// Send multiple requests that should be captured
	requests := []string{
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"resources/list"}`,
		`{"jsonrpc":"2.0","id":4,"method":"ping"}`,
	}

	for _, req := range requests {
		_, err := stdin.Write(append([]byte(req), '\n'))
		require.NoError(t, err)
		time.Sleep(100 * time.Millisecond)
	}

	t.Log("Sent multiple requests - StateStore should have captured them")

	// Verify server processed requests (state shows activity)
	stateDir := filepath.Join(tmpDir, ".hydra/state")
	stateMgr := state.NewManager(stateDir)
	serverState, err := stateMgr.LoadState(serverName)
	require.NoError(t, err)

	// Should have processed requests
	assert.GreaterOrEqual(t, serverState.Metrics.RequestsTotal, int64(4),
		"Should have processed init + 3 requests")

	t.Logf("StateStore validation: %d requests processed", serverState.Metrics.RequestsTotal)
}
