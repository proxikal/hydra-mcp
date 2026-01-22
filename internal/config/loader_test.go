package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/proxikal/hydra/internal/mocks"
	"github.com/stretchr/testify/assert"
)

func TestLoader_LoadRegistry(t *testing.T) {
	mockLogger := mocks.NewLogger(t)
	loader := NewLoader(mockLogger)

	t.Run("Valid registry", func(t *testing.T) {
		tmpfile := filepath.Join(t.TempDir(), "config.json")
		content := `{
			"version": "1.0",
			"defaults": {"debounce_ms": 500},
			"servers": {
				"test-server": {
					"command": "python",
					"args": ["server.py"]
				}
			}
		}`
		err := os.WriteFile(tmpfile, []byte(content), 0644)
		assert.NoError(t, err)

		reg, err := loader.LoadRegistry(tmpfile)
		assert.NoError(t, err)
		assert.Equal(t, "1.0", reg.Version)
		assert.Equal(t, 500, reg.Defaults.DebounceMS)
		assert.Contains(t, reg.Servers, "test-server")
		assert.Equal(t, "python", reg.Servers["test-server"].Command)
	})

	t.Run("Missing file returns defaults", func(t *testing.T) {
		mockLogger.On("Warn", "Registry file not found, using defaults", map[string]interface{}{
			"path": "/nonexistent/path.json",
		}).Return()

		reg, err := loader.LoadRegistry("/nonexistent/path.json")
		assert.NoError(t, err)
		assert.NotNil(t, reg)
		assert.Equal(t, "1.0", reg.Version)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		tmpfile := filepath.Join(t.TempDir(), "invalid.json")
		err := os.WriteFile(tmpfile, []byte("{invalid json"), 0644)
		assert.NoError(t, err)

		_, err = loader.LoadRegistry(tmpfile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid registry JSON")
	})

	t.Run("Empty servers map", func(t *testing.T) {
		tmpfile := filepath.Join(t.TempDir(), "empty.json")
		content := `{"version": "1.0", "defaults": {}}`
		err := os.WriteFile(tmpfile, []byte(content), 0644)
		assert.NoError(t, err)

		reg, err := loader.LoadRegistry(tmpfile)
		assert.NoError(t, err)
		assert.NotNil(t, reg.Servers)
	})
}

func TestLoader_LoadServerConfig(t *testing.T) {
	mockLogger := mocks.NewLogger(t)
	loader := NewLoader(mockLogger)

	t.Run("Valid server config", func(t *testing.T) {
		tmpfile := filepath.Join(t.TempDir(), "server.json")
		content := `{
			"command": "node",
			"args": ["index.js"],
			"watch": {
				"enabled": true,
				"paths": ["./src"]
			}
		}`
		err := os.WriteFile(tmpfile, []byte(content), 0644)
		assert.NoError(t, err)

		cfg, err := loader.LoadServerConfig(tmpfile)
		assert.NoError(t, err)
		assert.Equal(t, "node", cfg.Command)
		assert.True(t, cfg.Watch.Enabled)
	})

	t.Run("Missing file returns nil", func(t *testing.T) {
		cfg, err := loader.LoadServerConfig("/nonexistent/server.json")
		assert.NoError(t, err)
		assert.Nil(t, cfg)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		tmpfile := filepath.Join(t.TempDir(), "invalid.json")
		err := os.WriteFile(tmpfile, []byte("{bad json"), 0644)
		assert.NoError(t, err)

		_, err = loader.LoadServerConfig(tmpfile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid local config JSON")
	})
}

func TestLoader_Validate(t *testing.T) {
	mockLogger := mocks.NewLogger(t)
	loader := NewLoader(mockLogger)

	t.Run("Valid config", func(t *testing.T) {
		cfg := DefaultServerConfig()
		cfg.Command = "python"
		err := loader.Validate(cfg)
		assert.NoError(t, err)
	})

	t.Run("Missing command", func(t *testing.T) {
		cfg := DefaultServerConfig()
		cfg.Command = ""
		err := loader.Validate(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "command is required")
	})

	t.Run("Negative max_restarts", func(t *testing.T) {
		cfg := DefaultServerConfig()
		cfg.Command = "python"
		cfg.Behavior.MaxRestarts = -1
		err := loader.Validate(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max_restarts cannot be negative")
	})

	t.Run("Negative debounce_ms", func(t *testing.T) {
		cfg := DefaultServerConfig()
		cfg.Command = "python"
		cfg.Behavior.DebounceMS = -1
		err := loader.Validate(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "debounce_ms cannot be negative")
	})
}

func TestLoader_ResolveServer_Integration(t *testing.T) {
	mockLogger := mocks.NewLogger(t)

	t.Run("Full flow with registry", func(t *testing.T) {
		// Setup temp home directory
		tmpHome := t.TempDir()
		hydraDir := filepath.Join(tmpHome, ".hydra")
		err := os.MkdirAll(hydraDir, 0755)
		assert.NoError(t, err)

		regPath := filepath.Join(hydraDir, "config.json")
		regContent := `{
			"version": "1.0",
			"defaults": {},
			"servers": {
				"my-server": {
					"command": "python",
					"args": ["server.py"],
					"behavior": {"max_restarts": 5}
				}
			}
		}`
		err = os.WriteFile(regPath, []byte(regContent), 0644)
		assert.NoError(t, err)

		// Set HOME to temp directory
		oldHome := os.Getenv("HOME")
		_ = os.Setenv("HOME", tmpHome)
		defer func() { _ = os.Setenv("HOME", oldHome) }()

		loader := NewLoader(mockLogger)
		cfg, err := loader.ResolveServer("my-server", "")
		assert.NoError(t, err)
		assert.Equal(t, "python", cfg.Command)
		assert.Equal(t, []string{"server.py"}, cfg.Args)
		assert.Equal(t, 5, cfg.Behavior.MaxRestarts)
	})

	t.Run("With local override", func(t *testing.T) {
		tmpHome := t.TempDir()
		hydraDir := filepath.Join(tmpHome, ".hydra")
		err := os.MkdirAll(hydraDir, 0755)
		assert.NoError(t, err)

		regPath := filepath.Join(hydraDir, "config.json")
		regContent := `{
			"version": "1.0",
			"defaults": {},
			"servers": {
				"test": {
					"command": "node",
					"args": ["base.js"]
				}
			}
		}`
		err = os.WriteFile(regPath, []byte(regContent), 0644)
		assert.NoError(t, err)

		// Create local override
		localPath := filepath.Join(tmpHome, "hydra.json")
		localContent := `{"args": ["override.js"]}`
		err = os.WriteFile(localPath, []byte(localContent), 0644)
		assert.NoError(t, err)

		oldHome := os.Getenv("HOME")
		_ = os.Setenv("HOME", tmpHome)
		defer func() { _ = os.Setenv("HOME", oldHome) }()

		loader := NewLoader(mockLogger)
		cfg, err := loader.ResolveServer("test", localPath)
		assert.NoError(t, err)
		assert.Equal(t, "node", cfg.Command)
		assert.Equal(t, []string{"override.js"}, cfg.Args)
	})

	t.Run("Server not found", func(t *testing.T) {
		tmpHome := t.TempDir()
		hydraDir := filepath.Join(tmpHome, ".hydra")
		err := os.MkdirAll(hydraDir, 0755)
		assert.NoError(t, err)

		regPath := filepath.Join(hydraDir, "config.json")
		regContent := `{"version": "1.0", "defaults": {}, "servers": {}}`
		err = os.WriteFile(regPath, []byte(regContent), 0644)
		assert.NoError(t, err)

		oldHome := os.Getenv("HOME")
		_ = os.Setenv("HOME", tmpHome)
		defer func() { _ = os.Setenv("HOME", oldHome) }()

		loader := NewLoader(mockLogger)
		_, err = loader.ResolveServer("missing", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found in registry")
	})

	t.Run("With env substitution", func(t *testing.T) {
		_ = os.Setenv("MY_CMD", "echo")
		defer func() { _ = os.Unsetenv("MY_CMD") }()

		tmpHome := t.TempDir()
		hydraDir := filepath.Join(tmpHome, ".hydra")
		err := os.MkdirAll(hydraDir, 0755)
		assert.NoError(t, err)

		regPath := filepath.Join(hydraDir, "config.json")
		regContent := `{
			"version": "1.0",
			"defaults": {},
			"servers": {
				"env-test": {
					"command": "${env:MY_CMD}",
					"args": ["hello"]
				}
			}
		}`
		err = os.WriteFile(regPath, []byte(regContent), 0644)
		assert.NoError(t, err)

		oldHome := os.Getenv("HOME")
		_ = os.Setenv("HOME", tmpHome)
		defer func() { _ = os.Setenv("HOME", oldHome) }()

		loader := NewLoader(mockLogger)
		cfg, err := loader.ResolveServer("env-test", "")
		assert.NoError(t, err)
		assert.Equal(t, "echo", cfg.Command)
	})
}
