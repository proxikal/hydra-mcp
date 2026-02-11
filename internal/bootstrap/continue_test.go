package bootstrap

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/proxikal/hydra/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestConfigureContinue(t *testing.T) {
	configPath, registryPath, log := setupTestEnvironment(t)

	cfg := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"continue-server": map[string]interface{}{
				"command": "npx",
				"args":    []string{"continue-server"},
			},
		},
	}
	data, _ := json.Marshal(cfg)
	os.WriteFile(configPath, data, 0644)

	err := ConfigureContinue(configPath, registryPath, false, log)
	assert.NoError(t, err)
	assert.FileExists(t, configPath+".hydra.bak")

	loader := config.NewLoader(log)
	reg, _ := loader.LoadRegistry(registryPath)
	assert.Contains(t, reg.Servers, "continue-server")
}
