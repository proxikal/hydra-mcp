package bootstrap

import "github.com/proxikal/hydra/internal/logger"

// ConfigureContinue wires Continue (VSCode extension) MCP entries through Hydra.
// - Backs up the original config to <path>.hydra.bak (unless dry-run).
// - Adds any discovered MCP servers to the Hydra registry (if missing).
// - Rewrites Continue entries to call "hydra run --name <server>".
func ConfigureContinue(path string, registryPath string, dryRun bool, log logger.Logger) error {
	if path == "" {
		path = detectContinuePath()
	}

	return configureGenericMCP(path, registryPath, dryRun, log, "continue")
}
