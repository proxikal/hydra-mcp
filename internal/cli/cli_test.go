package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/proxikal/hydra/internal/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testRegistry creates a temporary registry file for testing
func testRegistry(t *testing.T, servers map[string]*config.ServerConfig) string {
	t.Helper()
	dir := t.TempDir()
	regPath := filepath.Join(dir, "config.json")

	reg := config.DefaultRegistry()
	if servers != nil {
		reg.Servers = servers
	}

	data, err := json.MarshalIndent(reg, "", "  ")
	require.NoError(t, err)

	err = os.WriteFile(regPath, data, 0644)
	require.NoError(t, err)

	return regPath
}

// resetViper clears viper state between tests
func resetViper() {
	viper.Reset()
}

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "tilde at start",
			input:    "~/test/path",
			expected: filepath.Join(home, "test/path"),
		},
		{
			name:     "just tilde",
			input:    "~",
			expected: home,
		},
		{
			name:     "tilde with slash",
			input:    "~/",
			expected: home,
		},
		{
			name:     "no tilde",
			input:    "/absolute/path",
			expected: "/absolute/path",
		},
		{
			name:     "relative path",
			input:    "relative/path",
			expected: "relative/path",
		},
		{
			name:     "tilde in middle",
			input:    "/path/~/test",
			expected: "/path/~/test",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandHome(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveClientConfigPath(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	t.Run("explicit path provided", func(t *testing.T) {
		result, err := resolveClientConfigPath("claude", "/explicit/path.json")
		assert.NoError(t, err)
		assert.Equal(t, "/explicit/path.json", result)
	})

	t.Run("explicit path with tilde", func(t *testing.T) {
		result, err := resolveClientConfigPath("claude", "~/config.json")
		assert.NoError(t, err)
		assert.Equal(t, filepath.Join(home, "config.json"), result)
	})

	t.Run("claude default path", func(t *testing.T) {
		result, err := resolveClientConfigPath("claude", "")
		assert.NoError(t, err)
		expected := filepath.Join(home, ".config/claude/claude_desktop_config.json")
		assert.Equal(t, expected, result)
	})

	t.Run("cline requires explicit path", func(t *testing.T) {
		result, err := resolveClientConfigPath("cline", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "--config is required")
		assert.Equal(t, "", result)
	})

	t.Run("cursor requires explicit path", func(t *testing.T) {
		result, err := resolveClientConfigPath("cursor", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "--config is required")
		assert.Equal(t, "", result)
	})

	t.Run("unsupported client", func(t *testing.T) {
		result, err := resolveClientConfigPath("unknown", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported client")
		assert.Equal(t, "", result)
	})
}

func TestNewRoot(t *testing.T) {
	root := NewRoot()
	require.NotNil(t, root)

	t.Run("root command properties", func(t *testing.T) {
		assert.Equal(t, "hydra", root.Use)
		assert.Equal(t, "Hydra MCP supervisor", root.Short)
	})

	t.Run("subcommands registered", func(t *testing.T) {
		subcommands := root.Commands()
		assert.NotEmpty(t, subcommands)

		// Build map of command names (first word only)
		cmdMap := make(map[string]bool)
		for _, cmd := range subcommands {
			// Extract first word from Use (e.g., "add <name>" -> "add")
			parts := strings.Fields(cmd.Use)
			if len(parts) > 0 {
				cmdMap[parts[0]] = true
			}
		}

		// Verify all expected commands are present
		expectedCommands := []string{
			"run",
			"init",
			"uninit",
			"add",
			"list",
			"remove",
			"edit",
			"validate",
			"logs",
			"status",
			"restart",
			"recover",
			"export",
		}

		for _, expected := range expectedCommands {
			assert.True(t, cmdMap[expected], "Expected command %s to be registered", expected)
		}
	})

	t.Run("help command available", func(t *testing.T) {
		// Cobra automatically adds help command
		err := root.Help()
		assert.NoError(t, err)
	})
}

func TestExecute(t *testing.T) {
	// This is a smoke test - we can't easily test full execution
	// without complex mocking, but we can verify the function exists
	// and returns error on invalid commands
	t.Run("invalid command returns error", func(t *testing.T) {
		// Save original args
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()

		// Set invalid command
		os.Args = []string{"hydra", "invalid-command-xyz"}

		err := Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown command")
	})

	t.Run("help command succeeds", func(t *testing.T) {
		// Save original args
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()

		// Set help command
		os.Args = []string{"hydra", "help"}

		err := Execute()
		// Help command should succeed
		assert.NoError(t, err)
	})
}

func TestCommandStructure(t *testing.T) {
	root := NewRoot()

	t.Run("run command exists", func(t *testing.T) {
		cmd, _, err := root.Find([]string{"run"})
		require.NoError(t, err)
		assert.Equal(t, "run", cmd.Use)
		assert.NotNil(t, cmd.Flags().Lookup("name"))
	})

	t.Run("list command exists", func(t *testing.T) {
		cmd, _, err := root.Find([]string{"list"})
		require.NoError(t, err)
		assert.Equal(t, "list", cmd.Use)
		assert.NotNil(t, cmd.Flags().Lookup("registry"))
	})

	t.Run("add command exists", func(t *testing.T) {
		cmd, _, err := root.Find([]string{"add"})
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(cmd.Use, "add"))
		assert.NotNil(t, cmd.Flags().Lookup("command"))
		assert.NotNil(t, cmd.Flags().Lookup("args"))
	})

	t.Run("remove command exists", func(t *testing.T) {
		cmd, _, err := root.Find([]string{"remove"})
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(cmd.Use, "remove"))
	})

	t.Run("validate command exists", func(t *testing.T) {
		cmd, _, err := root.Find([]string{"validate"})
		require.NoError(t, err)
		assert.Equal(t, "validate", cmd.Use)
		assert.NotNil(t, cmd.Flags().Lookup("registry"))
	})

	t.Run("init command exists", func(t *testing.T) {
		cmd, _, err := root.Find([]string{"init"})
		require.NoError(t, err)
		assert.Equal(t, "init", cmd.Use)
		assert.NotNil(t, cmd.Flags().Lookup("client"))
	})

	t.Run("uninit command exists", func(t *testing.T) {
		cmd, _, err := root.Find([]string{"uninit"})
		require.NoError(t, err)
		assert.Equal(t, "uninit", cmd.Use)
	})

	t.Run("logs command exists", func(t *testing.T) {
		cmd, _, err := root.Find([]string{"logs"})
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(cmd.Use, "logs"))
	})

	t.Run("status command exists", func(t *testing.T) {
		cmd, _, err := root.Find([]string{"status"})
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(cmd.Use, "status"))
	})

	t.Run("restart command exists", func(t *testing.T) {
		cmd, _, err := root.Find([]string{"restart"})
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(cmd.Use, "restart"))
	})

	t.Run("recover command exists", func(t *testing.T) {
		cmd, _, err := root.Find([]string{"recover"})
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(cmd.Use, "recover"))
	})

	t.Run("export command exists", func(t *testing.T) {
		cmd, _, err := root.Find([]string{"export"})
		require.NoError(t, err)
		assert.Equal(t, "export", cmd.Use)
	})
}

// =============================================================================
// Add Command Tests
// =============================================================================

func TestAddCommand(t *testing.T) {
	t.Run("add server successfully", func(t *testing.T) {
		resetViper()
		regPath := testRegistry(t, nil)

		viper.Set("registry", regPath)
		viper.Set("command", "python")
		viper.Set("args", "-m,mcp_server")
		viper.Set("cwd", "/tmp")

		cmd := newAddCommand()
		err := addCommand(cmd, []string{"test-server"})
		require.NoError(t, err)

		// Verify server was added
		data, err := os.ReadFile(regPath)
		require.NoError(t, err)

		var reg config.Registry
		err = json.Unmarshal(data, &reg)
		require.NoError(t, err)

		assert.Contains(t, reg.Servers, "test-server")
		assert.Equal(t, "python", reg.Servers["test-server"].Command)
		assert.Equal(t, []string{"-m", "mcp_server"}, reg.Servers["test-server"].Args)
	})

	t.Run("add requires command flag", func(t *testing.T) {
		resetViper()
		regPath := testRegistry(t, nil)

		viper.Set("registry", regPath)
		viper.Set("command", "") // Empty command

		cmd := newAddCommand()
		err := addCommand(cmd, []string{"test-server"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "command is required")
	})

	t.Run("add fails with invalid registry path", func(t *testing.T) {
		resetViper()
		// Use a path that can't be written to
		viper.Set("registry", "/nonexistent/path/config.json")
		viper.Set("command", "python")

		cmd := newAddCommand()
		err := addCommand(cmd, []string{"test-server"})
		// The loader creates defaults, but saving to an invalid path fails
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "save registry")
	})
}

// =============================================================================
// List Command Tests
// =============================================================================

func TestListCommand(t *testing.T) {
	t.Run("list empty registry", func(t *testing.T) {
		resetViper()
		regPath := testRegistry(t, nil)
		viper.Set("registry", regPath)

		cmd := newListCommand()
		var buf bytes.Buffer
		cmd.SetOut(&buf)

		err := listCommand(cmd, nil)
		require.NoError(t, err)
		assert.Empty(t, buf.String())
	})

	t.Run("list servers in registry", func(t *testing.T) {
		resetViper()
		servers := map[string]*config.ServerConfig{
			"server1": {Command: "python", Args: []string{"-m", "server1"}},
			"server2": {Command: "node", Args: []string{"server2.js"}},
		}
		regPath := testRegistry(t, servers)
		viper.Set("registry", regPath)

		cmd := newListCommand()
		var buf bytes.Buffer
		cmd.SetOut(&buf)

		err := listCommand(cmd, nil)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "server1")
		assert.Contains(t, output, "server2")
		assert.Contains(t, output, "python")
		assert.Contains(t, output, "node")
	})

	t.Run("list with nonexistent registry returns empty", func(t *testing.T) {
		resetViper()
		// The loader creates defaults with empty servers when registry doesn't exist
		dir := t.TempDir()
		viper.Set("registry", filepath.Join(dir, "nonexistent.json"))

		cmd := newListCommand()
		var buf bytes.Buffer
		cmd.SetOut(&buf)

		err := listCommand(cmd, nil)
		require.NoError(t, err)
		// Empty because loader creates default registry with no servers
		assert.Empty(t, buf.String())
	})
}

// =============================================================================
// Remove Command Tests
// =============================================================================

func TestRemoveCommand(t *testing.T) {
	t.Run("remove server successfully", func(t *testing.T) {
		resetViper()
		servers := map[string]*config.ServerConfig{
			"to-remove": {Command: "python"},
			"to-keep":   {Command: "node"},
		}
		regPath := testRegistry(t, servers)
		viper.Set("registry", regPath)

		cmd := newRemoveCommand()
		err := removeCommand(cmd, []string{"to-remove"})
		require.NoError(t, err)

		// Verify server was removed
		data, err := os.ReadFile(regPath)
		require.NoError(t, err)

		var reg config.Registry
		err = json.Unmarshal(data, &reg)
		require.NoError(t, err)

		assert.NotContains(t, reg.Servers, "to-remove")
		assert.Contains(t, reg.Servers, "to-keep")
	})

	t.Run("remove nonexistent server fails", func(t *testing.T) {
		resetViper()
		regPath := testRegistry(t, nil)
		viper.Set("registry", regPath)

		cmd := newRemoveCommand()
		err := removeCommand(cmd, []string{"nonexistent"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("remove from empty registry fails", func(t *testing.T) {
		resetViper()
		// Use a temp directory so loader creates defaults
		dir := t.TempDir()
		viper.Set("registry", filepath.Join(dir, "config.json"))

		cmd := newRemoveCommand()
		err := removeCommand(cmd, []string{"server"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// =============================================================================
// Validate Command Tests
// =============================================================================

func TestValidateCommand(t *testing.T) {
	t.Run("validate valid registry", func(t *testing.T) {
		resetViper()
		servers := map[string]*config.ServerConfig{
			"valid-server": {Command: "python", Args: []string{"-m", "server"}},
		}
		regPath := testRegistry(t, servers)
		viper.Set("registry", regPath)

		cmd := newValidateCommand()
		err := validateCommand(cmd, nil)
		require.NoError(t, err)
	})

	t.Run("validate empty registry succeeds", func(t *testing.T) {
		resetViper()
		regPath := testRegistry(t, nil)
		viper.Set("registry", regPath)

		cmd := newValidateCommand()
		err := validateCommand(cmd, nil)
		require.NoError(t, err)
	})

	t.Run("validate empty registry succeeds (defaults valid)", func(t *testing.T) {
		resetViper()
		// Use a temp directory so loader creates defaults
		dir := t.TempDir()
		viper.Set("registry", filepath.Join(dir, "config.json"))

		cmd := newValidateCommand()
		err := validateCommand(cmd, nil)
		// Empty registry is valid
		require.NoError(t, err)
	})
}

// =============================================================================
// Status Command Tests
// =============================================================================

func TestStatusCommand(t *testing.T) {
	t.Run("status shows server config", func(t *testing.T) {
		resetViper()
		servers := map[string]*config.ServerConfig{
			"my-server": {
				Command: "python",
				Args:    []string{"-m", "server"},
				CWD:     "/app",
			},
		}
		regPath := testRegistry(t, servers)
		viper.Set("registry", regPath)
		viper.Set("name", "my-server")

		cmd := newStatusCommand()
		var buf bytes.Buffer
		cmd.SetOut(&buf)

		err := statusCommand(cmd, nil)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "my-server")
		assert.Contains(t, output, "python")

		// Verify it's valid JSON
		var status map[string]interface{}
		err = json.Unmarshal([]byte(output), &status)
		require.NoError(t, err)
		assert.Equal(t, "my-server", status["name"])
	})

	t.Run("status fails for nonexistent server", func(t *testing.T) {
		resetViper()
		regPath := testRegistry(t, nil)
		viper.Set("registry", regPath)
		viper.Set("name", "nonexistent")

		cmd := newStatusCommand()
		err := statusCommand(cmd, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("status fails for server not in empty registry", func(t *testing.T) {
		resetViper()
		// Use a temp directory so loader creates defaults
		dir := t.TempDir()
		viper.Set("registry", filepath.Join(dir, "config.json"))
		viper.Set("name", "server")

		cmd := newStatusCommand()
		err := statusCommand(cmd, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// =============================================================================
// Edit Command Tests
// =============================================================================

func TestEditCommand(t *testing.T) {
	t.Run("edit fails when editor exits with error", func(t *testing.T) {
		resetViper()
		regPath := testRegistry(t, nil)
		viper.Set("registry", regPath)

		// Set EDITOR to "false" which immediately exits with error code 1
		t.Setenv("EDITOR", "false")

		cmd := newEditCommand()
		err := editCommand(cmd, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "editor failed")
	})

	t.Run("edit fails with nonexistent registry directory", func(t *testing.T) {
		resetViper()
		viper.Set("registry", "/nonexistent/path/to/config.json")

		cmd := newEditCommand()
		err := editCommand(cmd, nil)
		// Should fail when trying to save to nonexistent directory
		assert.Error(t, err)
	})
}

// =============================================================================
// Export Command Tests
// =============================================================================

func TestExportCommand(t *testing.T) {
	t.Run("export fails for nonexistent server", func(t *testing.T) {
		resetViper()
		regPath := testRegistry(t, nil)
		viper.Set("registry", regPath)
		viper.Set("name", "nonexistent")

		cmd := newExportCommand()
		err := exportCommand(cmd, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("export fails for server not in empty registry", func(t *testing.T) {
		resetViper()
		dir := t.TempDir()
		viper.Set("registry", filepath.Join(dir, "config.json"))
		viper.Set("name", "server")

		cmd := newExportCommand()
		err := exportCommand(cmd, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// =============================================================================
// Logs Command Tests
// =============================================================================

func TestLogsCommand(t *testing.T) {
	t.Run("logs fails for nonexistent server", func(t *testing.T) {
		resetViper()
		regPath := testRegistry(t, nil)
		viper.Set("registry", regPath)
		viper.Set("name", "nonexistent")

		cmd := newLogsCommand()
		err := logsCommand(cmd, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("logs fails for server not in empty registry", func(t *testing.T) {
		resetViper()
		dir := t.TempDir()
		viper.Set("registry", filepath.Join(dir, "config.json"))
		viper.Set("name", "server")

		cmd := newLogsCommand()
		err := logsCommand(cmd, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("logs reads export files when they exist", func(t *testing.T) {
		resetViper()
		dir := t.TempDir()
		exportPath := filepath.Join(dir, "hydra-traffic-{timestamp}.json")

		// Create a mock export file
		mockFile := filepath.Join(dir, "hydra-traffic-20260122-101500.json")
		mockData := `[{"timestamp":"2026-01-22T10:15:00Z","direction":"client","type":"request"}]`
		require.NoError(t, os.WriteFile(mockFile, []byte(mockData), 0644))

		servers := map[string]*config.ServerConfig{
			"my-server": {
				Command: "python",
				Recorder: config.RecorderConfig{
					Enabled:       true,
					ExportOnCrash: true,
					ExportPath:    exportPath,
				},
			},
		}
		regPath := testRegistry(t, servers)
		viper.Set("registry", regPath)
		viper.Set("name", "my-server")

		cmd := newLogsCommand()
		var buf bytes.Buffer
		cmd.SetOut(&buf)

		err := logsCommand(cmd, nil)
		require.NoError(t, err)
		// Should contain log entry info
		assert.Contains(t, buf.String(), "client")
	})

	t.Run("logs fails when no export files exist", func(t *testing.T) {
		resetViper()
		dir := t.TempDir()
		exportPath := filepath.Join(dir, "hydra-traffic-{timestamp}.json")

		servers := map[string]*config.ServerConfig{
			"my-server": {
				Command: "python",
				Recorder: config.RecorderConfig{
					Enabled:       true,
					ExportOnCrash: true,
					ExportPath:    exportPath,
				},
			},
		}
		regPath := testRegistry(t, servers)
		viper.Set("registry", regPath)
		viper.Set("name", "my-server")

		cmd := newLogsCommand()
		err := logsCommand(cmd, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no crash exports found")
	})
}

// =============================================================================
// Restart Command Tests
// =============================================================================

func TestRestartCommand(t *testing.T) {
	t.Run("restart returns error (stub command)", func(t *testing.T) {
		cmd := newRestartCommand()
		err := cmd.RunE(cmd, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "hydra_restart")
	})
}

// =============================================================================
// Recover Command Tests
// =============================================================================

func TestRecoverCommand(t *testing.T) {
	t.Run("recover returns error (stub command)", func(t *testing.T) {
		cmd := newRecoverCommand()
		err := cmd.RunE(cmd, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "hydra_force_restart")
	})
}

// =============================================================================
// Uninit Command Tests
// =============================================================================

func TestUninitCommand(t *testing.T) {
	t.Run("uninit fails when backup not found", func(t *testing.T) {
		resetViper()
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config.json")

		// Create a config file but no backup
		require.NoError(t, os.WriteFile(configPath, []byte("{}"), 0644))

		viper.Set("client", "claude")
		viper.Set("config", configPath)

		cmd := newUninitCommand()
		err := uninitCommand(cmd, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no backup found")
	})

	t.Run("uninit restores from backup successfully", func(t *testing.T) {
		resetViper()
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config.json")
		backupPath := configPath + ".hydra.bak"

		originalContent := `{"original": true}`
		modifiedContent := `{"modified": true}`

		// Create modified config and backup of original
		require.NoError(t, os.WriteFile(configPath, []byte(modifiedContent), 0644))
		require.NoError(t, os.WriteFile(backupPath, []byte(originalContent), 0644))

		viper.Set("client", "claude")
		viper.Set("config", configPath)

		cmd := newUninitCommand()
		var buf bytes.Buffer
		cmd.SetOut(&buf)

		err := uninitCommand(cmd, nil)
		require.NoError(t, err)

		// Verify config was restored
		restored, err := os.ReadFile(configPath)
		require.NoError(t, err)
		assert.Equal(t, originalContent, string(restored))

		// Verify backup was removed
		_, err = os.Stat(backupPath)
		assert.True(t, os.IsNotExist(err))

		// Verify output message
		assert.Contains(t, buf.String(), "Restored")
	})

	t.Run("uninit fails for unsupported client without config", func(t *testing.T) {
		resetViper()
		viper.Set("client", "unknown")
		viper.Set("config", "")

		cmd := newUninitCommand()
		err := uninitCommand(cmd, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported client")
	})

	t.Run("uninit fails for cline without explicit config", func(t *testing.T) {
		resetViper()
		viper.Set("client", "cline")
		viper.Set("config", "")

		cmd := newUninitCommand()
		err := uninitCommand(cmd, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "--config is required")
	})
}

// =============================================================================
// Init Command Tests
// =============================================================================

func TestInitCommand(t *testing.T) {
	t.Run("init fails for unsupported client", func(t *testing.T) {
		resetViper()
		viper.Set("client", "unsupported")
		viper.Set("config", "/tmp/config.json")
		viper.Set("registry", "/tmp/registry.json")
		viper.Set("dry-run", true)

		cmd := newInitCommand()
		err := cmd.RunE(cmd, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported client")
	})
}

// =============================================================================
// Export Command with Output Tests
// =============================================================================

// =============================================================================
// Run Command Tests
// =============================================================================

func TestRunCommand(t *testing.T) {
	t.Run("run fails when server not found", func(t *testing.T) {
		resetViper()
		regPath := testRegistry(t, nil)
		viper.Set("config", regPath)
		viper.Set("name", "nonexistent")
		viper.Set("override", "")

		cmd := newRunCommand()
		err := runCommand(cmd, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("run fails with missing registry path", func(t *testing.T) {
		resetViper()
		viper.Set("config", "/nonexistent/registry.json")
		viper.Set("name", "test-server")
		viper.Set("override", "")

		cmd := newRunCommand()
		err := runCommand(cmd, nil)
		// Config loader creates defaults when registry missing, so this should
		// fail with "server not found" since no servers are defined
		assert.Error(t, err)
	})

	t.Run("run fails with invalid server config", func(t *testing.T) {
		resetViper()
		// Server with empty command should fail validation
		servers := map[string]*config.ServerConfig{
			"invalid-server": {
				Command: "", // Empty command is invalid
			},
		}
		regPath := testRegistry(t, servers)
		viper.Set("config", regPath)
		viper.Set("name", "invalid-server")
		viper.Set("override", "")

		cmd := newRunCommand()
		err := runCommand(cmd, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid server config")
	})
}

func TestExportCommandWithOutput(t *testing.T) {
	t.Run("export writes to output file", func(t *testing.T) {
		resetViper()
		dir := t.TempDir()
		exportPath := filepath.Join(dir, "hydra-traffic-{timestamp}.json")
		outputPath := filepath.Join(dir, "export-output.json")

		// Create a mock export file
		mockFile := filepath.Join(dir, "hydra-traffic-20260122-101500.json")
		mockData := `{"events": ["test"]}`
		require.NoError(t, os.WriteFile(mockFile, []byte(mockData), 0644))

		servers := map[string]*config.ServerConfig{
			"my-server": {
				Command: "python",
				Recorder: config.RecorderConfig{
					Enabled:       true,
					ExportOnCrash: true,
					ExportPath:    exportPath,
				},
			},
		}
		regPath := testRegistry(t, servers)
		viper.Set("registry", regPath)
		viper.Set("name", "my-server")
		viper.Set("output", outputPath)

		cmd := newExportCommand()
		var buf bytes.Buffer
		cmd.SetOut(&buf)

		err := exportCommand(cmd, nil)
		require.NoError(t, err)

		// Verify output file was created
		written, err := os.ReadFile(outputPath)
		require.NoError(t, err)
		assert.Equal(t, mockData, string(written))

		// Verify output message
		assert.Contains(t, buf.String(), "Export written to")
	})

	t.Run("export to stdout when no output path", func(t *testing.T) {
		resetViper()
		dir := t.TempDir()
		exportPath := filepath.Join(dir, "hydra-traffic-{timestamp}.json")

		// Create a mock export file
		mockFile := filepath.Join(dir, "hydra-traffic-20260122-101500.json")
		mockData := `{"events": ["test"]}`
		require.NoError(t, os.WriteFile(mockFile, []byte(mockData), 0644))

		servers := map[string]*config.ServerConfig{
			"my-server": {
				Command: "python",
				Recorder: config.RecorderConfig{
					Enabled:       true,
					ExportOnCrash: true,
					ExportPath:    exportPath,
				},
			},
		}
		regPath := testRegistry(t, servers)
		viper.Set("registry", regPath)
		viper.Set("name", "my-server")
		viper.Set("output", "")

		cmd := newExportCommand()
		var buf bytes.Buffer
		cmd.SetOut(&buf)

		err := exportCommand(cmd, nil)
		require.NoError(t, err)

		// Verify data was written to stdout
		assert.Equal(t, mockData, buf.String())
	})
}
