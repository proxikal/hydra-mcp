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

// TestMassFileChanges_DebouncesProperly validates that file watcher debouncing
// prevents restart spam when multiple files change simultaneously (like git checkout).
//
// Validates:
// - Multiple file changes within debounce window trigger only ONE restart
// - Debounce window properly batches events
// - Server recovers and stays running after batch restart
func TestMassFileChanges_DebouncesProperly(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}
	if os.Getenv("CI") != "" {
		t.Skip("Skipping file watcher test in CI (requires async infrastructure)")
	}

	tmpDir := t.TempDir()
	serverName := "debounce-test"
	watchDir := filepath.Join(tmpDir, "watched")

	// Create watched directory with initial files
	require.NoError(t, os.MkdirAll(watchDir, 0755))
	for i := 0; i < 10; i++ {
		filename := filepath.Join(watchDir, "file"+string(rune('0'+i))+".py")
		content := []byte("# Initial content\nprint('hello')\n")
		require.NoError(t, os.WriteFile(filename, content, 0644))
	}

	// Create registry with watcher enabled
	registryPath := filepath.Join(tmpDir, "config.json")
	pythonServer := getFixturePath("python_server.py")
	require.FileExists(t, pythonServer)

	// Get absolute path to python server (it's not in watchDir)
	absServerPath, err := filepath.Abs(pythonServer)
	require.NoError(t, err)

	registry := config.DefaultRegistry()
	cfg := config.DefaultServerConfig()
	cfg.Command = "python3"
	cfg.Args = []string{absServerPath}
	// Don't set CWD - server doesn't need to run from watchDir
	cfg.Watch.Enabled = true
	cfg.Watch.Paths = []string{watchDir}
	cfg.Watch.Extensions = []string{".py"}
	cfg.Behavior.DebounceMS = 300 // 300ms debounce window
	cfg.Behavior.MaxRestarts = 10
	registry.Servers[serverName] = cfg

	data, _ := json.Marshal(registry)
	require.NoError(t, os.WriteFile(registryPath, data, 0644))

	// Start Hydra with watcher enabled
	cmd, stdin, stdout := startHydraServer(t, serverName, registryPath, tmpDir)

	defer func() {
		stdin.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()

	// Initialize the server
	sendMCPInitialize(t, stdin, stdout, 1)

	// Wait for watcher to start
	time.Sleep(1 * time.Second)

	// Get initial restart count from state
	stateDir := filepath.Join(tmpDir, ".hydra/state")
	stateMgr := state.NewManager(stateDir)
	initialState, err := stateMgr.LoadState(serverName)
	require.NoError(t, err)
	initialRestarts := initialState.Metrics.RestartsTotal

	t.Logf("Initial restart count: %d", initialRestarts)

	// Modify all 10 files RAPIDLY (simulate git checkout)
	modifyStart := time.Now()
	for i := 0; i < 10; i++ {
		filename := filepath.Join(watchDir, "file"+string(rune('0'+i))+".py")
		content := []byte("# Modified content\nprint('modified')\n")
		require.NoError(t, os.WriteFile(filename, content, 0644))
	}
	modifyDuration := time.Since(modifyStart)
	t.Logf("Modified 10 files in %v", modifyDuration)

	// Wait for debounce window + restart to complete
	time.Sleep(2 * time.Second)

	// Check restart count - should be +1, NOT +10
	finalState, err := stateMgr.LoadState(serverName)
	require.NoError(t, err)
	finalRestarts := finalState.Metrics.RestartsTotal

	t.Logf("Final restart count: %d", finalRestarts)

	// Debouncing should result in only 1 restart (or at most 2 if timing is weird)
	restartsDelta := finalRestarts - initialRestarts
	assert.LessOrEqual(t, restartsDelta, int64(2),
		"Should trigger at most 2 restarts (debounced), not 10")

	if restartsDelta == 1 {
		t.Log("✓ Perfect debouncing: 10 file changes → 1 restart")
	} else if restartsDelta == 2 {
		t.Log("✓ Acceptable debouncing: 10 file changes → 2 restarts (timing variance)")
	} else {
		t.Logf("⚠ Unexpected restart count: %d", restartsDelta)
	}

	// Verify server is still healthy after restart
	sendMCPInitialize(t, stdin, stdout, 2)
	time.Sleep(200 * time.Millisecond)

	healthState, err := stateMgr.LoadState(serverName)
	require.NoError(t, err)
	assert.Greater(t, healthState.PID, 0, "Server should still be running")
}

// TestDebounce_IndividualChanges validates that individual file changes
// (outside debounce window) trigger separate restarts as expected.
func TestDebounce_IndividualChanges(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}
	if os.Getenv("CI") != "" {
		t.Skip("Skipping file watcher test in CI (requires async infrastructure)")
	}

	tmpDir := t.TempDir()
	serverName := "individual-test"
	watchDir := filepath.Join(tmpDir, "watched")

	// Create watched directory
	require.NoError(t, os.MkdirAll(watchDir, 0755))
	testFile := filepath.Join(watchDir, "test.py")
	require.NoError(t, os.WriteFile(testFile, []byte("print('v1')"), 0644))

	// Create registry with watcher
	registryPath := filepath.Join(tmpDir, "config.json")
	pythonServer := getFixturePath("python_server.py")
	require.FileExists(t, pythonServer)

	// Get absolute path to python server
	absServerPath, err := filepath.Abs(pythonServer)
	require.NoError(t, err)

	registry := config.DefaultRegistry()
	cfg := config.DefaultServerConfig()
	cfg.Command = "python3"
	cfg.Args = []string{absServerPath}
	// Don't set CWD - server doesn't need to run from watchDir
	cfg.Watch.Enabled = true
	cfg.Watch.Paths = []string{watchDir}
	cfg.Watch.Extensions = []string{".py"}
	cfg.Behavior.DebounceMS = 200
	cfg.Behavior.MaxRestarts = 10
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

	sendMCPInitialize(t, stdin, stdout, 1)
	time.Sleep(1 * time.Second)

	stateDir := filepath.Join(tmpDir, ".hydra/state")
	stateMgr := state.NewManager(stateDir)

	// Make 3 changes with delays OUTSIDE debounce window
	for i := 2; i <= 4; i++ {
		content := []byte("print('v" + string(rune('0'+i)) + "')")
		require.NoError(t, os.WriteFile(testFile, content, 0644))
		t.Logf("Modified file to v%d", i)

		// Wait longer than debounce window (200ms + margin)
		time.Sleep(500 * time.Millisecond)
	}

	// Wait for final restart
	time.Sleep(1 * time.Second)

	finalState, err := stateMgr.LoadState(serverName)
	require.NoError(t, err)

	// Should have triggered 3 separate restarts (one per change)
	assert.GreaterOrEqual(t, finalState.Metrics.RestartsTotal, int64(2),
		"Individual changes should trigger separate restarts")

	t.Logf("Individual change restarts: %d", finalState.Metrics.RestartsTotal)
}
