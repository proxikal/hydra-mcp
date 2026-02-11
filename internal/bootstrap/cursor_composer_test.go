package bootstrap

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/proxikal/hydra/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestConfigureCursorComposer(t *testing.T) {
	configPath, registryPath, log := setupTestEnvironment(t)

	cfg := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"composer-server": map[string]interface{}{
				"command": "composer",
				"args":    []string{"start"},
			},
		},
	}
	data, _ := json.Marshal(cfg)
	_ = os.WriteFile(configPath, data, 0644)

	err := ConfigureCursorComposer(configPath, registryPath, false, log)
	assert.NoError(t, err)
	assert.FileExists(t, configPath+".hydra.bak")

	loader := config.NewLoader(log)
	reg, _ := loader.LoadRegistry(registryPath)
	assert.Contains(t, reg.Servers, "composer-server")
}
