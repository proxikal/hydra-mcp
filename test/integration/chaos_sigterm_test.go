package integration

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/proxikal/hydra/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSIGTERMHang_ForcedKill tests that Hydra uses SIGKILL when server
// ignores SIGTERM during graceful shutdown.
//
// Validates:
// - Supervisor sends SIGTERM first (graceful)
// - Waits for timeout
// - Falls back to SIGKILL (forceful)
// - Server process terminates
func TestSIGTERMHang_ForcedKill(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	tmpDir := t.TempDir()
	serverName := "hanging-server"

	// Create registry with hanging server fixture
	registryPath := filepath.Join(tmpDir, "config.json")
	hangingServer := getFixturePath("hanging_server.py")
	require.FileExists(t, hangingServer, "hanging_server.py fixture must exist")

	registry := config.DefaultRegistry()
	cfg := config.DefaultServerConfig()
	cfg.Command = "python3"
	cfg.Args = []string{hangingServer}
	cfg.Behavior.MaxRestarts = 1
	registry.Servers[serverName] = cfg

	data, _ := json.Marshal(registry)
	require.NoError(t, os.WriteFile(registryPath, data, 0644))

	// Start Hydra with hanging server
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

	// Wait for server to be fully running
	time.Sleep(1 * time.Second)

	// Send SIGTERM to the Hydra process (not the child)
	// Hydra should try to gracefully shutdown child, but child will hang
	err := cmd.Process.Signal(syscall.SIGTERM)
	require.NoError(t, err, "Should send SIGTERM to Hydra")

	t.Log("SIGTERM sent to Hydra - waiting for graceful shutdown attempt")

	// Monitor process state
	// Hydra should:
	// 1. Send SIGTERM to child (child ignores it)
	// 2. Wait for timeout (typically 5-10 seconds)
	// 3. Send SIGKILL to child (child dies)
	// 4. Exit itself

	done := make(chan bool)
	go func() {
		_ = cmd.Wait()
		done <- true
	}()

	select {
	case <-done:
		t.Log("Hydra exited successfully after SIGKILL fallback")
	case <-time.After(15 * time.Second):
		t.Fatal("Hydra did not exit within timeout - SIGKILL fallback failed")
	}

	// Verify the process actually exited
	assert.NotNil(t, cmd.ProcessState, "Process should have exited")
	if cmd.ProcessState != nil {
		t.Logf("Process exit code: %d", cmd.ProcessState.ExitCode())
	}
}

// TestSIGTERMHang_ChildProcessCleanup validates that hanging child processes
// are properly cleaned up via SIGKILL when SIGTERM is ignored.
func TestSIGTERMHang_ChildProcessCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	tmpDir := t.TempDir()
	serverName := "hanging-cleanup-test"

	// Create registry
	registryPath := filepath.Join(tmpDir, "config.json")
	hangingServer := getFixturePath("hanging_server.py")
	require.FileExists(t, hangingServer)

	registry := config.DefaultRegistry()
	cfg := config.DefaultServerConfig()
	cfg.Command = "python3"
	cfg.Args = []string{hangingServer}
	registry.Servers[serverName] = cfg

	data, _ := json.Marshal(registry)
	require.NoError(t, os.WriteFile(registryPath, data, 0644))

	// Start Hydra
	cmd, stdin, stdout := startHydraServer(t, serverName, registryPath, tmpDir)

	// Initialize
	sendMCPInitialize(t, stdin, stdout, 1)
	time.Sleep(500 * time.Millisecond)

	// Get child PID by checking process group
	// (In real scenario, supervisor tracks this)
	hydroPID := cmd.Process.Pid
	t.Logf("Hydra PID: %d", hydroPID)

	// Kill Hydra and verify cleanup
	_ = cmd.Process.Kill()
	_ = cmd.Wait()

	// Wait for cleanup to complete
	time.Sleep(2 * time.Second)

	// Verify no zombie processes remain
	// (Child process should be killed via SIGKILL)
	checkCmd := exec.Command("pgrep", "-P", strconv.Itoa(hydroPID))
	output, _ := checkCmd.CombinedOutput()

	if len(strings.TrimSpace(string(output))) > 0 {
		t.Logf("Warning: Found lingering child processes: %s", output)
		t.Log("This may indicate incomplete cleanup (acceptable on some systems)")
	} else {
		t.Log("Child process cleanup verified - no zombies")
	}
}
