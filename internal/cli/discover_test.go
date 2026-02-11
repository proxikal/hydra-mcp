package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getClaudeConfigPath returns the platform-specific Claude config path
func getClaudeConfigPath(home string) string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library/Application Support/Claude/claude_desktop_config.json")
	case "windows":
		return filepath.Join(home, "AppData/Roaming/Claude/claude_desktop_config.json")
	default: // linux
		return filepath.Join(home, ".config/Claude/claude_desktop_config.json")
	}
}

func TestDiscoverCommand_MultipleClients(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create Claude config (platform-specific path)
	claudePath := getClaudeConfigPath(tmpDir)
	require.NoError(t, os.MkdirAll(filepath.Dir(claudePath), 0755))
	claudeConfig := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"weather": map[string]interface{}{"command": "npx"},
			"git":     map[string]interface{}{"command": "npx"},
		},
	}
	data, _ := json.Marshal(claudeConfig)
	require.NoError(t, os.WriteFile(claudePath, data, 0644))

	// Create Cline config (need extensions dir for detection)
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".vscode/extensions"), 0755))
	clinePath := filepath.Join(tmpDir, ".vscode/mcp_settings.json")
	clineConfig := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"filesystem": map[string]interface{}{"command": "npx"},
		},
	}
	data, _ = json.Marshal(clineConfig)
	require.NoError(t, os.WriteFile(clinePath, data, 0644))

	// Run discover command
	output := new(bytes.Buffer)
	cmd := newDiscoverCommand()
	cmd.SetOut(output)
	cmd.SetErr(output)

	err := cmd.Execute()
	require.NoError(t, err)

	result := output.String()

	// Verify output contains both clients
	assert.Contains(t, result, "claude")
	assert.Contains(t, result, "cline")
	assert.Contains(t, result, "2") // Claude has 2 servers
	assert.Contains(t, result, "1") // Cline has 1 server
	assert.Contains(t, result, "100%")
}

func TestDiscoverCommand_NoClients(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Run discover command with no configs
	output := new(bytes.Buffer)
	cmd := newDiscoverCommand()
	cmd.SetOut(output)
	cmd.SetErr(output)

	err := cmd.Execute()
	require.NoError(t, err)

	result := output.String()
	assert.Contains(t, result, "No MCP clients found")
}

func TestDiscoverCommand_LowConfidence(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create invalid JSON config (confidence = 0.5)
	claudePath := getClaudeConfigPath(tmpDir)
	require.NoError(t, os.MkdirAll(filepath.Dir(claudePath), 0755))
	require.NoError(t, os.WriteFile(claudePath, []byte("invalid json"), 0644))

	// Run discover command
	output := new(bytes.Buffer)
	cmd := newDiscoverCommand()
	cmd.SetOut(output)
	cmd.SetErr(output)

	err := cmd.Execute()
	require.NoError(t, err)

	result := output.String()

	// Should show low confidence discoveries with warning
	assert.Contains(t, strings.ToLower(result), "50%")
}

func TestDiscoverCommand_TableFormatting(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create config (platform-specific path)
	claudePath := getClaudeConfigPath(tmpDir)
	require.NoError(t, os.MkdirAll(filepath.Dir(claudePath), 0755))
	claudeConfig := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"weather": map[string]interface{}{"command": "npx"},
		},
	}
	data, _ := json.Marshal(claudeConfig)
	require.NoError(t, os.WriteFile(claudePath, data, 0644))

	// Run discover command
	output := new(bytes.Buffer)
	cmd := newDiscoverCommand()
	cmd.SetOut(output)

	err := cmd.Execute()
	require.NoError(t, err)

	result := output.String()

	// Verify table headers
	assert.Contains(t, result, "CLIENT")
	assert.Contains(t, result, "PATH")
	assert.Contains(t, result, "SERVERS")
	assert.Contains(t, result, "CONFIDENCE")
}
