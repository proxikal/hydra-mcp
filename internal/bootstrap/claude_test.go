package bootstrap

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/proxikal/hydra/internal/config"
	"github.com/proxikal/hydra/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigureClaude(t *testing.T) {
	configPath, registryPath, log := setupTestEnvironment(t)

	// Create valid Claude config
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

	err = ConfigureClaude(configPath, registryPath, false, log)
	assert.NoError(t, err)

	// Verify backup created
	assert.FileExists(t, configPath+".hydra.bak")

	// Verify registry updated
	loader := config.NewLoader(log)
	reg, err := loader.LoadRegistry(registryPath)
	require.NoError(t, err)
	assert.Contains(t, reg.Servers, "test-server")
}

func TestConfigureClaude_DefaultPath(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := tmpDir + "/registry.json"

	// Create empty registry
	reg := &config.Registry{Servers: make(map[string]*config.ServerConfig)}
	data, _ := json.Marshal(reg)
	os.WriteFile(registryPath, data, 0644)

	log := logger.New("error")

	// Use empty path to trigger default
	err := ConfigureClaude("", registryPath, true, log)
	// Should error because default path doesn't exist, but that's expected
	assert.Error(t, err)
}
