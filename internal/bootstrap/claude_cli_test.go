package bootstrap

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/proxikal/hydra/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestConfigureClaudeCLI(t *testing.T) {
	configPath, registryPath, log := setupTestEnvironment(t)

	cfg := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"cli-server": map[string]interface{}{
				"command": "python",
				"args":    []string{"-m", "server"},
			},
		},
	}
	data, _ := json.Marshal(cfg)
	os.WriteFile(configPath, data, 0644)

	err := ConfigureClaudeCLI(configPath, registryPath, false, log)
	assert.NoError(t, err)
	assert.FileExists(t, configPath+".hydra.bak")

	loader := config.NewLoader(log)
	reg, _ := loader.LoadRegistry(registryPath)
	assert.Contains(t, reg.Servers, "cli-server")
}
