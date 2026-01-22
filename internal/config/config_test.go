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

func TestMergeServerConfig(t *testing.T) {
	t.Run("Merge command and args", func(t *testing.T) {
		dst := DefaultServerConfig()
		dst.Command = "old"
		src := &ServerConfig{
			Command: "new",
			Args:    []string{"arg1", "arg2"},
		}
		mergeServerConfig(dst, src)
		assert.Equal(t, "new", dst.Command)
		assert.Equal(t, []string{"arg1", "arg2"}, dst.Args)
	})

	t.Run("Merge CWD and EnvFile", func(t *testing.T) {
		dst := DefaultServerConfig()
		src := &ServerConfig{
			CWD:     "/custom/path",
			EnvFile: ".env.prod",
		}
		mergeServerConfig(dst, src)
		assert.Equal(t, "/custom/path", dst.CWD)
		assert.Equal(t, ".env.prod", dst.EnvFile)
	})

	t.Run("Merge environment variables", func(t *testing.T) {
		dst := DefaultServerConfig()
		dst.Environment = map[string]string{"KEY1": "val1"}
		src := &ServerConfig{
			Environment: map[string]string{"KEY2": "val2"},
		}
		mergeServerConfig(dst, src)
		assert.Equal(t, "val1", dst.Environment["KEY1"])
		assert.Equal(t, "val2", dst.Environment["KEY2"])
	})

	t.Run("Merge watch config", func(t *testing.T) {
		dst := DefaultServerConfig()
		src := &ServerConfig{
			Watch: WatchConfig{
				Enabled:     true,
				Paths:       []string{"/custom/path"},
				Extensions:  []string{".rs"},
				IgnoreFiles: []string{".gitignore"},
				Ignore:      []string{"target"},
			},
		}
		mergeServerConfig(dst, src)
		assert.True(t, dst.Watch.Enabled)
		assert.Equal(t, []string{"/custom/path"}, dst.Watch.Paths)
		assert.Equal(t, []string{".rs"}, dst.Watch.Extensions)
		assert.Equal(t, []string{".gitignore"}, dst.Watch.IgnoreFiles)
		assert.Equal(t, []string{"target"}, dst.Watch.Ignore)
	})

	t.Run("Merge behavior config", func(t *testing.T) {
		dst := DefaultServerConfig()
		src := &ServerConfig{
			Behavior: BehaviorConfig{
				DebounceMS:         1000,
				RestartDelayMS:     100,
				MaxRestarts:        5,
				GracefulShutdownMS: 3000,
				PreRestartCommand:  "cleanup.sh",
			},
		}
		mergeServerConfig(dst, src)
		assert.Equal(t, 1000, dst.Behavior.DebounceMS)
		assert.Equal(t, 100, dst.Behavior.RestartDelayMS)
		assert.Equal(t, 5, dst.Behavior.MaxRestarts)
		assert.Equal(t, 3000, dst.Behavior.GracefulShutdownMS)
		assert.Equal(t, "cleanup.sh", dst.Behavior.PreRestartCommand)
	})
}

func TestSaveRegistry(t *testing.T) {
	t.Run("Save registry successfully", func(t *testing.T) {
		tmpdir := t.TempDir()
		path := filepath.Join(tmpdir, "config.json")

		reg := DefaultRegistry()
		reg.Servers["test"] = &ServerConfig{Command: "echo"}

		err := SaveRegistry(reg, path)
		assert.NoError(t, err)

		// Verify file exists and is valid
		data, err := os.ReadFile(path)
		assert.NoError(t, err)
		assert.Contains(t, string(data), "test")
	})

	t.Run("Create parent directory", func(t *testing.T) {
		tmpdir := t.TempDir()
		path := filepath.Join(tmpdir, "subdir", "config.json")

		reg := DefaultRegistry()
		err := SaveRegistry(reg, path)
		assert.NoError(t, err)

		_, err = os.Stat(path)
		assert.NoError(t, err)
	})
}

func TestRegistryOperations(t *testing.T) {
	t.Run("AddServer", func(t *testing.T) {
		reg := DefaultRegistry()
		cfg := &ServerConfig{Command: "python"}

		AddServer(reg, "my-server", cfg)
		assert.Contains(t, reg.Servers, "my-server")
		assert.Equal(t, "python", reg.Servers["my-server"].Command)
	})

	t.Run("RemoveServer existing", func(t *testing.T) {
		reg := DefaultRegistry()
		reg.Servers["test"] = &ServerConfig{Command: "node"}

		removed := RemoveServer(reg, "test")
		assert.True(t, removed)
		assert.NotContains(t, reg.Servers, "test")
	})

	t.Run("RemoveServer non-existing", func(t *testing.T) {
		reg := DefaultRegistry()
		removed := RemoveServer(reg, "nonexistent")
		assert.False(t, removed)
	})

	t.Run("ListServers", func(t *testing.T) {
		reg := DefaultRegistry()
		reg.Servers["server1"] = &ServerConfig{}
		reg.Servers["server2"] = &ServerConfig{}

		names := ListServers(reg)
		assert.Len(t, names, 2)
		assert.Contains(t, names, "server1")
		assert.Contains(t, names, "server2")
	})
}

func TestSubstituteEnvVars(t *testing.T) {
	_ = os.Setenv("TEST_VAR", "test_value")
	_ = os.Setenv("API_KEY", "secret123")
	defer func() { _ = os.Unsetenv("TEST_VAR") }()
	defer func() { _ = os.Unsetenv("API_KEY") }()

	t.Run("Substitute single variable", func(t *testing.T) {
		result := SubstituteEnvVars("Hello ${env:TEST_VAR}")
		assert.Equal(t, "Hello test_value", result)
	})

	t.Run("Substitute multiple variables", func(t *testing.T) {
		result := SubstituteEnvVars("${env:TEST_VAR} and ${env:API_KEY}")
		assert.Equal(t, "test_value and secret123", result)
	})

	t.Run("No variables", func(t *testing.T) {
		result := SubstituteEnvVars("plain text")
		assert.Equal(t, "plain text", result)
	})

	t.Run("Undefined variable", func(t *testing.T) {
		result := SubstituteEnvVars("${env:UNDEFINED}")
		assert.Equal(t, "", result)
	})
}

func TestSubstituteEnvVarsInConfig(t *testing.T) {
	_ = os.Setenv("CMD", "python")
	_ = os.Setenv("PATH_VAR", "/custom/path")
	defer func() { _ = os.Unsetenv("CMD") }()
	defer func() { _ = os.Unsetenv("PATH_VAR") }()

	cfg := &ServerConfig{
		Command: "${env:CMD}",
		Args:    []string{"${env:PATH_VAR}/script.py"},
		CWD:     "${env:PATH_VAR}",
		EnvFile: "${env:PATH_VAR}/.env",
		Environment: map[string]string{
			"KEY": "${env:CMD}",
		},
		Watch: WatchConfig{
			Paths: []string{"${env:PATH_VAR}/src"},
		},
		Behavior: BehaviorConfig{
			PreRestartCommand: "${env:CMD} cleanup.py",
		},
	}

	SubstituteEnvVarsInConfig(cfg)

	assert.Equal(t, "python", cfg.Command)
	assert.Equal(t, "/custom/path/script.py", cfg.Args[0])
	assert.Equal(t, "/custom/path", cfg.CWD)
	assert.Equal(t, "/custom/path/.env", cfg.EnvFile)
	assert.Equal(t, "python", cfg.Environment["KEY"])
	assert.Equal(t, "/custom/path/src", cfg.Watch.Paths[0])
	assert.Equal(t, "python cleanup.py", cfg.Behavior.PreRestartCommand)
}

func TestDefaultRegistry(t *testing.T) {
	reg := DefaultRegistry()
	assert.Equal(t, "1.0", reg.Version)
	assert.NotNil(t, reg.Servers)
	assert.Equal(t, 500, reg.Defaults.DebounceMS)
	assert.Equal(t, 2000, reg.Defaults.GracefulShutdownMS)
	assert.Equal(t, 50, reg.Defaults.MaxOutputSizeKB)
}

func TestDefaultServerConfig(t *testing.T) {
	cfg := DefaultServerConfig()
	assert.Equal(t, "", cfg.Command)
	assert.Equal(t, 500, cfg.Behavior.DebounceMS)
	assert.Equal(t, 10, cfg.Behavior.MaxRestarts)
	assert.False(t, cfg.Watch.Enabled)
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

func TestSaveRegistry_EdgeCases(t *testing.T) {
	t.Run("Home directory expansion", func(t *testing.T) {
		tmpHome := t.TempDir()
		oldHome := os.Getenv("HOME")
		_ = os.Setenv("HOME", tmpHome)
		defer func() { _ = os.Setenv("HOME", oldHome) }()

		reg := DefaultRegistry()
		path := "~/.hydra/config.json"

		err := SaveRegistry(reg, path)
		assert.NoError(t, err)

		// Check file was created in expanded path
		expandedPath := filepath.Join(tmpHome, ".hydra", "config.json")
		_, err = os.Stat(expandedPath)
		assert.NoError(t, err)
	})
}

func TestAddServer_NilMap(t *testing.T) {
	reg := &Registry{
		Version:  "1.0",
		Defaults: Defaults{},
		Servers:  nil, // nil map
	}

	cfg := &ServerConfig{Command: "test"}
	AddServer(reg, "server1", cfg)

	assert.NotNil(t, reg.Servers)
	assert.Contains(t, reg.Servers, "server1")
}

func TestMerge_EdgeCases(t *testing.T) {
	t.Run("Merge with nil environment in dst", func(t *testing.T) {
		dst := &ServerConfig{Environment: nil}
		src := &ServerConfig{Environment: map[string]string{"KEY": "val"}}

		mergeServerConfig(dst, src)
		assert.NotNil(t, dst.Environment)
		assert.Equal(t, "val", dst.Environment["KEY"])
	})

	t.Run("Merge zero values dont override", func(t *testing.T) {
		dst := DefaultServerConfig()
		dst.Behavior.DebounceMS = 1000

		src := &ServerConfig{
			Behavior: BehaviorConfig{
				DebounceMS: 0, // Zero value should not override
			},
		}

		mergeServerConfig(dst, src)
		// Zero values shouldn't override existing values
		assert.Equal(t, 1000, dst.Behavior.DebounceMS)
	})
}
