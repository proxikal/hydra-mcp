package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
