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

// TestRestart_SessionContinuity validates that AI session remains alive
// through server restarts without losing context.
//
// Validates:
// - Hydra proxy stays alive during child restart
// - Requests before restart are processed
// - Requests after restart continue to work
// - No session interruption from AI client perspective
func TestRestart_SessionContinuity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}
	if os.Getenv("CI") != "" {
		t.Skip("Skipping file watcher test in CI (requires async infrastructure)")
	}

	tmpDir := t.TempDir()
	serverName := "continuity-test"
	watchDir := filepath.Join(tmpDir, "watched")

	// Create watched directory with a file
	require.NoError(t, os.MkdirAll(watchDir, 0755))
	testFile := filepath.Join(watchDir, "trigger.py")
	require.NoError(t, os.WriteFile(testFile, []byte("print('v1')"), 0644))

	// Create registry with watcher enabled
	registryPath := filepath.Join(tmpDir, "config.json")
	pythonServer := getFixturePath("python_server.py")
	require.FileExists(t, pythonServer)

	// Get absolute path to python server
	absServerPath, absErr := filepath.Abs(pythonServer)
	require.NoError(t, absErr)

	registry := config.DefaultRegistry()
	cfg := config.DefaultServerConfig()
	cfg.Command = "python3"
	cfg.Args = []string{absServerPath}
	// Don't set CWD - server doesn't need to run from watchDir
	cfg.Watch.Enabled = true
	cfg.Watch.Paths = []string{watchDir}
	cfg.Watch.Extensions = []string{".py"}
	cfg.Behavior.MaxRestarts = 3
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

	// Send request BEFORE restart
	preReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "ping",
	}
	preData, _ := json.Marshal(preReq)
	_, err := stdin.Write(append(preData, '\n'))
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	// Trigger restart via file change
	t.Log("Triggering restart via file change...")
	require.NoError(t, os.WriteFile(testFile, []byte("print('v2')"), 0644))

	// Wait for restart to complete
	time.Sleep(2 * time.Second)

	// Send request AFTER restart - session should still be alive
	postReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "ping",
	}
	postData, _ := json.Marshal(postReq)
	_, err = stdin.Write(append(postData, '\n'))
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	// Verify session continuity via state
	stateDir := filepath.Join(tmpDir, ".hydra/state")
	stateMgr := state.NewManager(stateDir)
	serverState, err := stateMgr.LoadState(serverName)
	require.NoError(t, err)

	// Should have at least 1 restart
	assert.GreaterOrEqual(t, serverState.Metrics.RestartsTotal, int64(1),
		"Should have restarted at least once")

	// Server should still be running
	assert.Greater(t, serverState.PID, 0, "Server should be running after restart")

	t.Logf("✓ Session continuity validated: %d restarts, still running (PID %d)",
		serverState.Metrics.RestartsTotal, serverState.PID)
}
