package bootstrap

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/proxikal/hydra/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestConfigureCline(t *testing.T) {
	configPath, registryPath, log := setupTestEnvironment(t)

	cfg := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"cline-server": map[string]interface{}{
				"command": "cline",
				"args":    []string{"serve"},
			},
		},
	}
	data, _ := json.Marshal(cfg)
	_ = os.WriteFile(configPath, data, 0644)

	err := ConfigureCline(configPath, registryPath, false, log)
	assert.NoError(t, err)
	assert.FileExists(t, configPath+".hydra.bak")

	loader := config.NewLoader(log)
	reg, _ := loader.LoadRegistry(registryPath)
	assert.Contains(t, reg.Servers, "cline-server")
}
