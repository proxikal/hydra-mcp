package integration

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

// TestStatePersistence_MultipleServers validates that multiple hydra instances
// maintain separate, isolated state files.
func TestStatePersistence_MultipleServers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".hydra/state")

	// Create registry with 3 different servers
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
	registry.Servers["server-python"] = pythonCfg

	// Node server
	nodeCfg := config.DefaultServerConfig()
	nodeCfg.Command = "node"
	nodeCfg.Args = []string{nodeServer}
	registry.Servers["server-node"] = nodeCfg

	// Go server
	goCfg := config.DefaultServerConfig()
	goCfg.Command = goServer
	registry.Servers["server-go"] = goCfg

	data, _ := json.Marshal(registry)
	require.NoError(t, os.WriteFile(registryPath, data, 0644))

	// Start all 3 servers
	type serverProc struct {
		name   string
		cmd    *exec.Cmd
		stdin  *os.File
		stdout *os.File
	}

	var servers []serverProc

	for i, name := range []string{"server-python", "server-node", "server-go"} {
		cmd, stdin, stdout := startHydraServer(t, name, registryPath, tmpDir)

		servers = append(servers, serverProc{
			name:   name,
			cmd:    cmd,
			stdin:  stdin,
			stdout: stdout,
		})

		// Wait for hydra to fully initialize before sending MCP messages
		time.Sleep(500 * time.Millisecond)

		// Initialize each server
		sendMCPInitialize(t, stdin, stdout, i+1)
		t.Logf("Started and initialized %s (PID: %d)", name, cmd.Process.Pid)

		// Small delay between server starts to avoid race conditions
		time.Sleep(200 * time.Millisecond)
	}

	defer func() {
		for _, srv := range servers {
			srv.stdin.Close()
			if srv.cmd.Process != nil {
				_ = srv.cmd.Process.Kill()
				_ = srv.cmd.Wait()
			}
		}
	}()

	// Wait for state files to be created
	time.Sleep(2 * time.Second)

	// Verify all state files exist and are isolated
	stateMgr := state.NewManager(stateDir)
	var states []*state.State

	for _, srv := range servers {
		st, err := stateMgr.LoadState(srv.name)
		require.NoError(t, err, "State file for %s should exist", srv.name)
		require.NotNil(t, st)

		assert.Equal(t, srv.name, st.ServerName)
		assert.Greater(t, st.PID, 0)
		assert.NotNil(t, st.Metrics)

		states = append(states, st)
		t.Logf("%s state: PID=%d", srv.name, st.PID)
	}

	// Verify PIDs are different (separate processes)
	assert.NotEqual(t, states[0].PID, states[1].PID, "Python and Node PIDs differ")
	assert.NotEqual(t, states[1].PID, states[2].PID, "Node and Go PIDs differ")
	assert.NotEqual(t, states[0].PID, states[2].PID, "Python and Go PIDs differ")

	// Verify all 3 state files exist
	stateFiles, err := filepath.Glob(filepath.Join(stateDir, "*.json"))
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(stateFiles), 3, "Should have 3 state files")

	t.Logf("State isolation verified: %d state files", len(stateFiles))
}

// TestStatePersistence_ConcurrentUpdates validates that concurrent state
// updates from multiple servers don't interfere with each other.
func TestStatePersistence_ConcurrentUpdates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".hydra/state")

	// Create registry with 2 servers
	registryPath := filepath.Join(tmpDir, "config.json")
	pythonServer := getFixturePath("python_server.py")
	require.FileExists(t, pythonServer)

	registry := config.DefaultRegistry()

	for _, name := range []string{"concurrent-1", "concurrent-2"} {
		cfg := config.DefaultServerConfig()
		cfg.Command = "python3"
		cfg.Args = []string{pythonServer}
		registry.Servers[name] = cfg
	}

	data, _ := json.Marshal(registry)
	require.NoError(t, os.WriteFile(registryPath, data, 0644))

	// Start both servers
	var servers []struct {
		name  string
		cmd   *exec.Cmd
		stdin *os.File
	}

	for i, name := range []string{"concurrent-1", "concurrent-2"} {
		cmd, stdin, stdout := startHydraServer(t, name, registryPath, tmpDir)
		sendMCPInitialize(t, stdin, stdout, i+1)

		servers = append(servers, struct {
			name  string
			cmd   *exec.Cmd
			stdin *os.File
		}{name, cmd, stdin})
	}

	defer func() {
		for _, srv := range servers {
			srv.stdin.Close()
			if srv.cmd.Process != nil {
				_ = srv.cmd.Process.Kill()
				_ = srv.cmd.Wait()
			}
		}
	}()

	// Wait for multiple periodic updates (2 cycles = 12 seconds)
	time.Sleep(12 * time.Second)

	// Verify both state files still exist and are valid
	stateMgr := state.NewManager(stateDir)

	for _, srv := range servers {
		st, err := stateMgr.LoadState(srv.name)
		require.NoError(t, err, "State for %s should exist", srv.name)
		assert.Equal(t, srv.name, st.ServerName)
		assert.Greater(t, st.PID, 0)

		t.Logf("%s: Last updated %v", srv.name, st.LastUpdated)
	}

	t.Log("Concurrent updates completed successfully")
}
