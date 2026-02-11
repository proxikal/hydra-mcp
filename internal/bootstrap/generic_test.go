package bootstrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/proxikal/hydra/internal/config"
	"github.com/proxikal/hydra/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestEnvironment(t *testing.T) (string, string, logger.Logger) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.json")
	registryPath := filepath.Join(tmpDir, "registry.json")

	// Create empty registry
	reg := &config.Registry{Servers: make(map[string]*config.ServerConfig)}
	data, _ := json.Marshal(reg)
	err := os.WriteFile(registryPath, data, 0644)
	require.NoError(t, err)

	log := logger.New("error")
	return configPath, registryPath, log
}

func TestConfigureGenericMCP_ValidConfig(t *testing.T) {
	configPath, registryPath, log := setupTestEnvironment(t)

	// Create valid client config
	cfg := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"test-server": map[string]interface{}{
				"command": "node",
				"args":    []string{"server.js"},
			},
		},
	}
	data, _ := json.Marshal(cfg)
	err := os.WriteFile(configPath, data, 0644)
	require.NoError(t, err)

	// Run configuration
	err = configureGenericMCP(configPath, registryPath, false, log, "test-client")
	require.NoError(t, err)

	// Verify backup created
	backupPath := configPath + ".hydra.bak"
	assert.FileExists(t, backupPath)

	// Verify config updated
	updatedData, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var updatedCfg claudeConfig
	err = json.Unmarshal(updatedData, &updatedCfg)
	require.NoError(t, err)

	server := updatedCfg.MCPServers["test-server"]
	assert.Equal(t, "hydra", server.Command)
	assert.Contains(t, server.Args, "run")
	assert.Contains(t, server.Args, "--name")
	assert.Contains(t, server.Args, "test-server")
}

func TestConfigureGenericMCP_DryRun(t *testing.T) {
	configPath, registryPath, log := setupTestEnvironment(t)

	// Create valid client config
	cfg := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"test-server": map[string]interface{}{
				"command": "node",
				"args":    []string{"server.js"},
			},
		},
	}
	data, _ := json.Marshal(cfg)
	err := os.WriteFile(configPath, data, 0644)
	require.NoError(t, err)

	// Run in dry-run mode
	err = configureGenericMCP(configPath, registryPath, true, log, "test-client")
	require.NoError(t, err)

	// Verify no backup created
	backupPath := configPath + ".hydra.bak"
	assert.NoFileExists(t, backupPath)

	// Verify config not modified
	currentData, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, data, currentData)
}

func TestConfigureGenericMCP_MissingFile(t *testing.T) {
	_, registryPath, log := setupTestEnvironment(t)

	err := configureGenericMCP("/nonexistent/path.json", registryPath, false, log, "test-client")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "read")
}

func TestConfigureGenericMCP_InvalidJSON(t *testing.T) {
	configPath, registryPath, log := setupTestEnvironment(t)

	// Write invalid JSON
	err := os.WriteFile(configPath, []byte("invalid json{"), 0644)
	require.NoError(t, err)

	err = configureGenericMCP(configPath, registryPath, false, log, "test-client")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse")
}

func TestConfigureGenericMCP_NoMCPServers(t *testing.T) {
	configPath, registryPath, log := setupTestEnvironment(t)

	// Create config without mcpServers
	cfg := map[string]interface{}{
		"other": "data",
	}
	data, _ := json.Marshal(cfg)
	err := os.WriteFile(configPath, data, 0644)
	require.NoError(t, err)

	err = configureGenericMCP(configPath, registryPath, false, log, "test-client")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no mcpServers found")
}

func TestConfigureGenericMCP_BackupAlreadyExists(t *testing.T) {
	configPath, registryPath, log := setupTestEnvironment(t)

	// Create valid client config
	cfg := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"test-server": map[string]interface{}{
				"command": "node",
				"args":    []string{"server.js"},
			},
		},
	}
	data, _ := json.Marshal(cfg)
	err := os.WriteFile(configPath, data, 0644)
	require.NoError(t, err)

	// Create backup file
	backupPath := configPath + ".hydra.bak"
	err = os.WriteFile(backupPath, []byte("existing backup"), 0644)
	require.NoError(t, err)

	// Try to configure
	err = configureGenericMCP(configPath, registryPath, false, log, "test-client")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "backup already exists")
}

func TestConfigureGenericMCP_RegistryMerge(t *testing.T) {
	configPath, registryPath, log := setupTestEnvironment(t)

	// Create client config with multiple servers
	cfg := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"server1": map[string]interface{}{
				"command": "node",
				"args":    []string{"server1.js"},
			},
			"server2": map[string]interface{}{
				"command": "python",
				"args":    []string{"server2.py"},
			},
		},
	}
	data, _ := json.Marshal(cfg)
	err := os.WriteFile(configPath, data, 0644)
	require.NoError(t, err)

	// Run configuration
	err = configureGenericMCP(configPath, registryPath, false, log, "test-client")
	require.NoError(t, err)

	// Verify registry updated
	loader := config.NewLoader(log)
	reg, err := loader.LoadRegistry(registryPath)
	require.NoError(t, err)

	assert.Contains(t, reg.Servers, "server1")
	assert.Contains(t, reg.Servers, "server2")
	assert.Equal(t, "node", reg.Servers["server1"].Command)
	assert.Equal(t, "python", reg.Servers["server2"].Command)
}

func TestConfigureGenericMCP_PreservesExistingRegistryEntries(t *testing.T) {
	configPath, registryPath, log := setupTestEnvironment(t)

	// Add existing server to registry
	loader := config.NewLoader(log)
	reg, _ := loader.LoadRegistry(registryPath)
	existing := config.DefaultServerConfig()
	existing.Command = "existing"
	reg.Servers["existing-server"] = existing
	_ = config.SaveRegistry(reg, registryPath)

	// Create client config
	cfg := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"new-server": map[string]interface{}{
				"command": "node",
				"args":    []string{"server.js"},
			},
		},
	}
	data, _ := json.Marshal(cfg)
	err := os.WriteFile(configPath, data, 0644)
	require.NoError(t, err)

	// Run configuration
	err = configureGenericMCP(configPath, registryPath, false, log, "test-client")
	require.NoError(t, err)

	// Verify both servers exist
	reg, err = loader.LoadRegistry(registryPath)
	require.NoError(t, err)

	assert.Contains(t, reg.Servers, "existing-server")
	assert.Contains(t, reg.Servers, "new-server")
	assert.Equal(t, "existing", reg.Servers["existing-server"].Command)
}

func TestConfigureGenericMCP_ExpandsHomePath(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	// Create empty registry
	reg := &config.Registry{Servers: make(map[string]*config.ServerConfig)}
	data, _ := json.Marshal(reg)
	os.WriteFile(registryPath, data, 0644)

	// Use ~ in config path
	configPath := filepath.Join(tmpDir, "config.json")

	// Create valid config
	cfg := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"test": map[string]interface{}{
				"command": "node",
				"args":    []string{"server.js"},
			},
		},
	}
	cfgData, _ := json.Marshal(cfg)
	os.WriteFile(configPath, cfgData, 0644)

	log := logger.New("error")

	// This should work even though we're using absolute paths
	err := configureGenericMCP(configPath, registryPath, false, log, "test-client")
	assert.NoError(t, err)
}
