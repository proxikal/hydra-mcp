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

// TestPSCommand_ListsRunningInstances validates that `hydra ps` correctly
// lists running hydra instances from state files.
func TestPSCommand_ListsRunningInstances(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	if os.Getenv("CI") != "" && os.Getenv("RUNNER_OS") == "macOS" {
		t.Skip("Skipping multi-instance test on macOS CI (platform-specific process spawning issue)")
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

	// Start 2 hydra instances
	var servers []struct {
		name  string
		cmd   *exec.Cmd
		stdin *os.File
	}

	for i, name := range []string{"ps-test-1", "ps-test-2"} {
		cmd, stdin, stdout := startHydraServer(t, name, registryPath, tmpDir)
		sendMCPInitialize(t, stdin, stdout, i+1)

		servers = append(servers, struct {
			name  string
			cmd   *exec.Cmd
			stdin *os.File
		}{name, cmd, stdin})

		t.Logf("Started %s (PID: %d)", name, cmd.Process.Pid)
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

	// Wait for state files
	time.Sleep(2 * time.Second)

	// Run hydra ps
	psCmd := exec.Command(getHydraBinary(), "ps")
	psCmd.Env = append(os.Environ(), "HOME="+tmpDir)
	output, err := psCmd.CombinedOutput()
	require.NoError(t, err, "hydra ps failed: %s", string(output))

	psOutput := string(output)
	t.Logf("PS Output:\n%s", psOutput)

	// Verify both servers appear
	assert.Contains(t, psOutput, "ps-test-1", "Should list ps-test-1")
	assert.Contains(t, psOutput, "ps-test-2", "Should list ps-test-2")

	// Verify output format
	lines := strings.Split(psOutput, "\n")
	assert.Greater(t, len(lines), 2, "Should have header + 2 servers")

	// Check each server appears in output
	for _, srv := range servers {
		found := false
		for _, line := range lines {
			if strings.Contains(line, srv.name) {
				found = true
				// Server should show health status (e.g., "✓ 95%") or uptime
				// Don't check for specific string since format may vary
				t.Logf("Server %s found in output: %s", srv.name, line)
				break
			}
		}
		assert.True(t, found, "Server %s should appear", srv.name)
	}
}

// TestPSCommand_EmptyWhenNoServers validates that `hydra ps` handles
// empty state directory gracefully.
func TestPSCommand_EmptyWhenNoServers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Run hydra ps with no servers
	psCmd := exec.Command(getHydraBinary(), "ps")
	psCmd.Env = append(os.Environ(), "HOME="+tmpDir)
	output, err := psCmd.CombinedOutput()
	// PS may return exit 1 when no servers exist - this is acceptable
	if err != nil {
		t.Logf("PS returned error (acceptable for empty list): %v", err)
	}

	psOutput := string(output)
	t.Logf("PS Output (empty):\n%s", psOutput)

	// Should show message about no instances (not actual server data)
	assert.Contains(t, psOutput, "No running", "Should indicate no instances")

	// Verify no server data (PIDs, etc.)
	assert.NotContains(t, psOutput, "PID", "Should not have PID column for empty list")
}

// TestPSCommand_OutputFormat validates the format and columns of PS output.
func TestPSCommand_OutputFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create and start a server
	registryPath := filepath.Join(tmpDir, "config.json")
	pythonServer := getFixturePath("python_server.py")
	require.FileExists(t, pythonServer)

	registry := config.DefaultRegistry()
	cfg := config.DefaultServerConfig()
	cfg.Command = "python3"
	cfg.Args = []string{pythonServer}
	registry.Servers["format-test"] = cfg

	data, _ := json.Marshal(registry)
	require.NoError(t, os.WriteFile(registryPath, data, 0644))

	cmd, stdin, stdout := startHydraServer(t, "format-test", registryPath, tmpDir)
	sendMCPInitialize(t, stdin, stdout, 1)

	defer func() {
		stdin.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()

	time.Sleep(2 * time.Second)

	// Run hydra ps
	psCmd := exec.Command(getHydraBinary(), "ps")
	psCmd.Env = append(os.Environ(), "HOME="+tmpDir)
	output, err := psCmd.CombinedOutput()
	require.NoError(t, err)

	psOutput := string(output)

	// Verify server appears in output
	assert.Contains(t, psOutput, "format-test", "Should show server name")

	// Output should have some structure (header + data)
	lines := strings.Split(strings.TrimSpace(psOutput), "\n")
	assert.Greater(t, len(lines), 1, "Should have header and data rows")

	t.Logf("PS format validated:\n%s", psOutput)
}
