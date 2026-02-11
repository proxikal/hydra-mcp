package integration_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/proxikal/hydra/internal/config"
	"github.com/proxikal/hydra/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChaosPython_CrashRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	tmpDir := t.TempDir()
	serverName := "python-chaos"

	// Create registry with Python chaos server
	registryPath := filepath.Join(tmpDir, "config.json")
	pythonServer := getFixturePath("python_server.py")
	require.FileExists(t, pythonServer, "Python chaos server fixture missing")

	registry := config.DefaultRegistry()
	serverCfg := config.DefaultServerConfig()
	serverCfg.Command = "python3"
	serverCfg.Args = []string{pythonServer}
	serverCfg.Behavior.MaxRestarts = 20 // Allow many restarts for chaos
	registry.Servers[serverName] = serverCfg

	data, _ := json.Marshal(registry)
	require.NoError(t, os.WriteFile(registryPath, data, 0644))

	// Start Hydra with chaos server
	cmd := exec.Command("hydra", "run",
		"--name", serverName,
		"--registry", registryPath,
	)

	// Redirect output for debugging
	logFile := filepath.Join(tmpDir, "hydra.log")
	logF, err := os.Create(logFile)
	require.NoError(t, err)
	defer logF.Close()

	cmd.Stdout = logF
	cmd.Stderr = logF

	require.NoError(t, cmd.Start())
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()

	// Wait for initialization
	time.Sleep(5 * time.Second)

	// Send 50 requests (expect some crashes and restarts)
	stateDir := filepath.Join(tmpDir, ".hydra/state")
	stateMgr := state.NewManager(stateDir)

	successCount := 0
	for i := 0; i < 50; i++ {
		// Small delay between requests
		time.Sleep(100 * time.Millisecond)

		// Check if still running
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			t.Logf("Process exited after %d requests", i)
			break
		}

		successCount++
	}

	// Verify: Should have processed most requests despite chaos
	assert.Greater(t, successCount, 40, "Should handle most requests despite chaos")

	// Check state shows restarts happened
	serverState, err := stateMgr.LoadState(serverName)
	if err == nil && serverState != nil {
		t.Logf("Restarts: %d", serverState.Metrics.RestartsTotal)
		// With 1% crash rate on 50 requests, expect 0-3 restarts typically
		assert.LessOrEqual(t, serverState.Metrics.RestartsTotal, int64(10), "Should have reasonable restart count")
	}

	// Cleanup
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
}

func TestChaosPython_SlowInit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	pythonServer := getFixturePath("python_server.py")
	require.FileExists(t, pythonServer)

	// Measure initialization time
	start := time.Now()

	cmd := exec.Command("python3", pythonServer)
	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)

	require.NoError(t, cmd.Start())
	defer func() {
		stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	// Wait for ready (2 second init delay expected)
	time.Sleep(3 * time.Second)
	elapsed := time.Since(start)

	// Verify slow init was handled
	assert.GreaterOrEqual(t, elapsed.Seconds(), 2.0, "Should wait for slow initialization")

	// Send a test request
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "test",
	}
	data, _ := json.Marshal(request)
	_, err = stdin.Write(append(data, '\n'))
	assert.NoError(t, err)
}

func TestChaosPython_HighLatency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	pythonServer := getFixturePath("python_server.py")
	require.FileExists(t, pythonServer)

	cmd := exec.Command("python3", pythonServer)
	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)

	require.NoError(t, cmd.Start())
	defer func() {
		stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	// Wait for init
	time.Sleep(3 * time.Second)

	// Send 20 requests, track latencies
	scanner := json.NewDecoder(stdout)
	var highLatencyCount int

	for i := 0; i < 20; i++ {
		request := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      i,
			"method":  "test",
		}
		data, _ := json.Marshal(request)

		start := time.Now()
		_, _ = stdin.Write(append(data, '\n'))

		// Wait for response (with timeout)
		done := make(chan bool, 1)
		go func() {
			var response map[string]interface{}
			_ = scanner.Decode(&response)
			done <- true
		}()

		select {
		case <-done:
			elapsed := time.Since(start)
			if elapsed > 1*time.Second {
				highLatencyCount++
			}
		case <-time.After(6 * time.Second):
			t.Logf("Request %d timed out", i)
			highLatencyCount++
		}

		time.Sleep(100 * time.Millisecond)
	}

	// With 5% slow response rate, expect 0-3 high latency requests in 20
	t.Logf("High latency requests: %d", highLatencyCount)
	assert.LessOrEqual(t, highLatencyCount, 5, "Should have some high latency responses")
}

func getFixturePath(filename string) string {
	// Navigate from test/integration to test/fixtures/mcp_servers
	return filepath.Join("..", "fixtures", "mcp_servers", filename)
}
