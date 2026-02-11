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

func TestMultiServer_StateIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping multi-server test in short mode")
	}

	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".hydra/state")

	// Create registry with 3 servers
	registryPath := filepath.Join(tmpDir, "config.json")
	pythonServer := getFixturePath("python_server.py")
	nodeServer := getFixturePath("node_server.js")
	goServer := getFixturePath("go_server")

	require.FileExists(t, pythonServer)
	require.FileExists(t, nodeServer)
	require.FileExists(t, goServer)

	registry := config.DefaultRegistry()

	// Python server
	pythonCfg := config.DefaultServerConfig()
	pythonCfg.Command = "python3"
	pythonCfg.Args = []string{pythonServer}
	registry.Servers["server1"] = pythonCfg

	// Node server
	nodeCfg := config.DefaultServerConfig()
	nodeCfg.Command = "node"
	nodeCfg.Args = []string{nodeServer}
	registry.Servers["server2"] = nodeCfg

	// Go server
	goCfg := config.DefaultServerConfig()
	goCfg.Command = goServer
	goCfg.Args = []string{}
	registry.Servers["server3"] = goCfg

	data, _ := json.Marshal(registry)
	require.NoError(t, os.WriteFile(registryPath, data, 0644))

	// Start all 3 servers
	var cmds []*exec.Cmd
	for _, name := range []string{"server1", "server2", "server3"} {
		cmd := exec.Command("hydra", "run",
			"--name", name,
			"--registry", registryPath,
		)

		logFile := filepath.Join(tmpDir, name+".log")
		logF, err := os.Create(logFile)
		require.NoError(t, err)
		defer logF.Close()

		cmd.Stdout = logF
		cmd.Stderr = logF

		require.NoError(t, cmd.Start())
		cmds = append(cmds, cmd)
	}

	// Cleanup function
	defer func() {
		for _, cmd := range cmds {
			if cmd.Process != nil {
				cmd.Process.Kill()
				cmd.Wait()
			}
		}
	}()

	// Wait for all to initialize
	time.Sleep(6 * time.Second)

	// Verify separate state files exist
	stateMgr := state.NewManager(stateDir)

	for _, name := range []string{"server1", "server2", "server3"} {
		serverState, err := stateMgr.LoadState(name)
		if err == nil && serverState != nil {
			t.Logf("%s: State file exists", name)
			assert.NotNil(t, serverState.Metrics, "Should have metrics")
		}
	}

	// Verify state isolation (no mixing)
	stateFiles, _ := filepath.Glob(filepath.Join(stateDir, "*.json"))
	assert.GreaterOrEqual(t, len(stateFiles), 3, "Should have separate state files")
}

func TestMultiServer_ConcurrentMetrics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent metrics test in short mode")
	}

	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".hydra/state")

	// Create registry with 2 servers
	registryPath := filepath.Join(tmpDir, "config.json")
	pythonServer := getFixturePath("python_server.py")
	nodeServer := getFixturePath("node_server.js")

	require.FileExists(t, pythonServer)
	require.FileExists(t, nodeServer)

	registry := config.DefaultRegistry()

	pythonCfg := config.DefaultServerConfig()
	pythonCfg.Command = "python3"
	pythonCfg.Args = []string{pythonServer}
	registry.Servers["metrics1"] = pythonCfg

	nodeCfg := config.DefaultServerConfig()
	nodeCfg.Command = "node"
	nodeCfg.Args = []string{nodeServer}
	registry.Servers["metrics2"] = nodeCfg

	data, _ := json.Marshal(registry)
	require.NoError(t, os.WriteFile(registryPath, data, 0644))

	// Start both servers
	var cmds []*exec.Cmd
	for _, name := range []string{"metrics1", "metrics2"} {
		cmd := exec.Command("hydra", "run",
			"--name", name,
			"--registry", registryPath,
		)

		logFile := filepath.Join(tmpDir, name+".log")
		logF, err := os.Create(logFile)
		require.NoError(t, err)
		defer logF.Close()

		cmd.Stdout = logF
		cmd.Stderr = logF

		require.NoError(t, cmd.Start())
		cmds = append(cmds, cmd)
	}

	defer func() {
		for _, cmd := range cmds {
			if cmd.Process != nil {
				cmd.Process.Kill()
				cmd.Wait()
			}
		}
	}()

	// Wait for init
	time.Sleep(6 * time.Second)

	// Verify metrics don't mix
	stateMgr := state.NewManager(stateDir)

	state1, _ := stateMgr.LoadState("metrics1")
	state2, _ := stateMgr.LoadState("metrics2")

	if state1 != nil && state2 != nil {
		// Metrics should be independent
		t.Logf("metrics1 restarts: %d", state1.Metrics.RestartsTotal)
		t.Logf("metrics2 restarts: %d", state2.Metrics.RestartsTotal)

		// They shouldn't have identical metrics (different servers)
		assert.NotEqual(t, state1.Metrics, state2.Metrics, "Metrics should be isolated")
	}
}

func TestHydraPS_Accuracy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping ps accuracy test in short mode")
	}

	tmpDir := t.TempDir()

	// Create registry with 2 servers
	registryPath := filepath.Join(tmpDir, "config.json")
	pythonServer := getFixturePath("python_server.py")

	require.FileExists(t, pythonServer)

	registry := config.DefaultRegistry()

	for _, name := range []string{"ps-test-1", "ps-test-2"} {
		cfg := config.DefaultServerConfig()
		cfg.Command = "python3"
		cfg.Args = []string{pythonServer}
		registry.Servers[name] = cfg
	}

	data, _ := json.Marshal(registry)
	require.NoError(t, os.WriteFile(registryPath, data, 0644))

	// Start 2 servers
	var cmds []*exec.Cmd
	for _, name := range []string{"ps-test-1", "ps-test-2"} {
		cmd := exec.Command("hydra", "run",
			"--name", name,
			"--registry", registryPath,
		)

		logFile := filepath.Join(tmpDir, name+".log")
		logF, err := os.Create(logFile)
		require.NoError(t, err)
		defer logF.Close()

		cmd.Stdout = logF
		cmd.Stderr = logF

		require.NoError(t, cmd.Start())
		cmds = append(cmds, cmd)
	}

	defer func() {
		for _, cmd := range cmds {
			if cmd.Process != nil {
				cmd.Process.Kill()
				cmd.Wait()
			}
		}
	}()

	// Wait for startup
	time.Sleep(5 * time.Second)

	// Run hydra ps
	psCmd := exec.Command("hydra", "ps")
	output, err := psCmd.CombinedOutput()
	require.NoError(t, err)

	psOutput := string(output)
	t.Logf("PS Output:\n%s", psOutput)

	// Verify both servers listed
	assert.Contains(t, psOutput, "ps-test-1", "Should list server 1")
	assert.Contains(t, psOutput, "ps-test-2", "Should list server 2")

	// Stop one server
	if cmds[0].Process != nil {
		_ = cmds[0].Process.Kill()
		_ = cmds[0].Wait()
	}

	// Wait for cleanup
	time.Sleep(3 * time.Second)

	// Run ps again
	psCmd2 := exec.Command("hydra", "ps")
	output2, err := psCmd2.CombinedOutput()
	require.NoError(t, err)

	psOutput2 := string(output2)
	t.Logf("PS Output after stopping server 1:\n%s", psOutput2)

	// Should still show server 2
	assert.Contains(t, psOutput2, "ps-test-2", "Should still list running server")
}
