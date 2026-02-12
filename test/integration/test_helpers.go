package integration

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// getFixturePath returns the absolute path to a fixture file.
func getFixturePath(filename string) string {
	return filepath.Join("..", "fixtures", "mcp_servers", filename)
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// sendMCPInitialize sends an MCP initialize request and waits for response.
func sendMCPInitialize(t *testing.T, stdin *os.File, stdout *os.File, id int) {
	initReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	reqData, _ := json.Marshal(initReq)
	_, err := stdin.Write(append(reqData, '\n'))
	require.NoError(t, err)

	// Read response
	decoder := json.NewDecoder(stdout)
	var resp map[string]interface{}
	err = decoder.Decode(&resp)
	require.NoError(t, err)

	// Send initialized notification
	notif := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	}
	notifData, _ := json.Marshal(notif)
	_, err = stdin.Write(append(notifData, '\n'))
	require.NoError(t, err)
}

// getHydraBinary returns the path to the hydra binary to use for tests.
// Prefers freshly built bin/hydra over system hydra.
func getHydraBinary() string {
	// Try bin/hydra (freshly built)
	binPath := filepath.Join("..", "..", "bin", "hydra")
	if fileExists(binPath) {
		return binPath
	}
	// Fallback to system hydra
	return "hydra"
}

// startHydraServer starts a hydra instance and returns cmd and pipes.
// Also captures and logs stderr output for debugging.
func startHydraServer(t *testing.T, serverName, registryPath, homeDir string) (*exec.Cmd, *os.File, *os.File) {
	cmd := exec.Command(getHydraBinary(), "run",
		"--name", serverName,
		"--config", registryPath,
	)

	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	stderr, err := cmd.StderrPipe()
	require.NoError(t, err)

	cmd.Env = append(os.Environ(), "HOME="+homeDir)
	require.NoError(t, cmd.Start())

	// Capture stderr in background
	go func() {
		buf := make([]byte, 8192)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				t.Logf("[%s stderr] %s", serverName, buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()

	return cmd, stdin.(*os.File), stdout.(*os.File)
}
