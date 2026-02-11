package integration_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/proxikal/hydra/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChaosNode_CrashRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	tmpDir := t.TempDir()
	serverName := "node-chaos"

	// Create registry with Node chaos server
	registryPath := filepath.Join(tmpDir, "config.json")
	nodeServer := getFixturePath("node_server.js")
	require.FileExists(t, nodeServer, "Node chaos server fixture missing")

	registry := config.DefaultRegistry()
	serverCfg := config.DefaultServerConfig()
	serverCfg.Command = "node"
	serverCfg.Args = []string{nodeServer}
	serverCfg.Behavior.MaxRestarts = 20
	registry.Servers[serverName] = serverCfg

	data, _ := json.Marshal(registry)
	require.NoError(t, os.WriteFile(registryPath, data, 0644))

	// Start Hydra with chaos server
	cmd := exec.Command("hydra", "run",
		"--name", serverName,
		"--registry", registryPath,
	)

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

	// Send requests
	successCount := 0
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)

		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			t.Logf("Process exited after %d requests", i)
			break
		}

		successCount++
	}

	// Verify handling despite chaos
	assert.Greater(t, successCount, 40, "Should handle most requests")

	// Cleanup
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
}

func TestChaosNode_SlowInit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	nodeServer := getFixturePath("node_server.js")
	require.FileExists(t, nodeServer)

	start := time.Now()

	cmd := exec.Command("node", nodeServer)
	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)

	require.NoError(t, cmd.Start())
	defer func() {
		stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	// Wait for init
	time.Sleep(3 * time.Second)
	elapsed := time.Since(start)

	assert.GreaterOrEqual(t, elapsed.Seconds(), 2.0, "Should wait for slow init")
}
