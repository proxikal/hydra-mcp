package bootstrap

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/proxikal/hydra/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestConfigureCursor(t *testing.T) {
	configPath, registryPath, log := setupTestEnvironment(t)

	cfg := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"cursor-server": map[string]interface{}{
				"command": "cursor-mcp",
				"args":    []string{"run"},
			},
		},
	}
	data, _ := json.Marshal(cfg)
	os.WriteFile(configPath, data, 0644)

	err := ConfigureCursor(configPath, registryPath, false, log)
	assert.NoError(t, err)
	assert.FileExists(t, configPath+".hydra.bak")

	loader := config.NewLoader(log)
	reg, _ := loader.LoadRegistry(registryPath)
	assert.Contains(t, reg.Servers, "cursor-server")
}
