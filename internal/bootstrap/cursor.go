package bootstrap

import (
	"fmt"

	"github.com/proxikal/hydra/internal/logger"
)

// ConfigureCursor wires Cursor MCP config through Hydra.
// Requires --config path because Cursor locations vary.
func ConfigureCursor(path string, registryPath string, dryRun bool, log logger.Logger) error {
	if path == "" {
		return fmt.Errorf("--config is required for client cursor")
	}
	return configureGenericMCP(path, registryPath, dryRun, log, "cursor")
}
