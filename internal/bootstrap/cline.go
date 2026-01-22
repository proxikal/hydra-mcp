package bootstrap

import (
	"fmt"

	"github.com/proxikal/hydra/internal/logger"
)

// ConfigureCline wires Cline MCP config through Hydra.
// Requires --config path because Cline locations vary.
func ConfigureCline(path string, registryPath string, dryRun bool, log logger.Logger) error {
	if path == "" {
		return fmt.Errorf("--config is required for client cline")
	}
	return configureGenericMCP(path, registryPath, dryRun, log, "cline")
}
