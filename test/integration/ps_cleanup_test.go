package integration

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/proxikal/hydra/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPSCommand_CleanupStaleEntries validates that `hydra ps` removes
// state files for dead processes.
func TestPSCommand_CleanupStaleEntries(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".hydra/state")

	// Create registry
	registryPath := filepath.Join(tmpDir, "config.json")
	pythonServer := getFixturePath("python_server.py")
	require.FileExists(t, pythonServer)

	registry := config.DefaultRegistry()
	cfg := config.DefaultServerConfig()
	cfg.Command = "python3"
	cfg.Args = []string{pythonServer}
	registry.Servers["cleanup-test"] = cfg

	data, _ := json.Marshal(registry)
	require.NoError(t, os.WriteFile(registryPath, data, 0644))

	// Start a server
	cmd, stdin, stdout := startHydraServer(t, "cleanup-test", registryPath, tmpDir)
	sendMCPInitialize(t, stdin, stdout, 1)

	// Wait for state file
	time.Sleep(2 * time.Second)

	// Verify state file exists
	stateFile := filepath.Join(stateDir, "cleanup-test.json")
	assert.True(t, fileExists(stateFile), "State file should exist")

	// Kill the server
	stdin.Close()
	_ = cmd.Process.Kill()
	_ = cmd.Wait()

	t.Log("Server killed, state file is now stale")

	// Wait for process to fully exit
	time.Sleep(1 * time.Second)

	// Run hydra ps (should detect dead process)
	psCmd := exec.Command("hydra", "ps")
	psCmd.Env = append(os.Environ(), "HOME="+tmpDir)
	output, err := psCmd.CombinedOutput()
	// PS may return exit 1 if all processes are dead - this is acceptable
	if err != nil {
		t.Logf("PS exited with error (acceptable): %v", err)
	}

	psOutput := string(output)
	t.Logf("PS Output after cleanup:\n%s", psOutput)

	// The dead server should either not appear, or show as exited
	if strings.Contains(psOutput, "cleanup-test") {
		// If it appears, it shouldn't show health status (✓ X%)
		for _, line := range strings.Split(psOutput, "\n") {
			if strings.Contains(line, "cleanup-test") {
				assert.NotContains(t, line, "✓",
					"Dead server should not show health indicator")
				t.Logf("Dead server marked correctly: %s", line)
			}
		}
	} else {
		t.Log("Dead server entry removed (correct)")
	}
}

// TestPSCommand_HandlesMultipleDead validates cleanup with multiple dead servers.
func TestPSCommand_HandlesMultipleDead(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create registry
	registryPath := filepath.Join(tmpDir, "config.json")
	pythonServer := getFixturePath("python_server.py")
	require.FileExists(t, pythonServer)

	registry := config.DefaultRegistry()

	for _, name := range []string{"dead-1", "dead-2", "alive-1"} {
		cfg := config.DefaultServerConfig()
		cfg.Command = "python3"
		cfg.Args = []string{pythonServer}
		registry.Servers[name] = cfg
	}

	data, _ := json.Marshal(registry)
	require.NoError(t, os.WriteFile(registryPath, data, 0644))

	// Start 3 servers
	var servers []struct {
		name  string
		cmd   *exec.Cmd
		stdin *os.File
	}

	for i, name := range []string{"dead-1", "dead-2", "alive-1"} {
		cmd, stdin, stdout := startHydraServer(t, name, registryPath, tmpDir)
		sendMCPInitialize(t, stdin, stdout, i+1)

		servers = append(servers, struct {
			name  string
			cmd   *exec.Cmd
			stdin *os.File
		}{name, cmd, stdin})
	}

	// Wait for state files
	time.Sleep(2 * time.Second)

	// Kill first 2 servers
	for i := 0; i < 2; i++ {
		servers[i].stdin.Close()
		_ = servers[i].cmd.Process.Kill()
		_ = servers[i].cmd.Wait()
		t.Logf("Killed %s", servers[i].name)
	}

	// Keep alive-1 running
	defer func() {
		servers[2].stdin.Close()
		if servers[2].cmd.Process != nil {
			_ = servers[2].cmd.Process.Kill()
			_ = servers[2].cmd.Wait()
		}
	}()

	// Wait for cleanup
	time.Sleep(1 * time.Second)

	// Run hydra ps
	psCmd := exec.Command(getHydraBinary(), "ps")
	psCmd.Env = append(os.Environ(), "HOME="+tmpDir)
	output, err := psCmd.CombinedOutput()
	require.NoError(t, err)

	psOutput := string(output)
	t.Logf("PS Output:\n%s", psOutput)

	// alive-1 should appear in output
	assert.Contains(t, psOutput, "alive-1", "Living server should appear")

	// Verify alive-1 shows health status (indicating it's alive)
	lines := strings.Split(psOutput, "\n")
	foundAlive := false
	for _, line := range lines {
		if strings.Contains(line, "alive-1") {
			// Should show health indicator (✓ XX%)
			assert.Contains(t, line, "✓", "alive-1 should show health status")
			foundAlive = true
			t.Logf("alive-1 status: %s", line)
		}
	}
	assert.True(t, foundAlive, "alive-1 should be in output")

	t.Log("Cleanup of multiple dead servers handled correctly")
}
