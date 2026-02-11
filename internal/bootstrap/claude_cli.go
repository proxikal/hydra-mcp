package bootstrap

import "github.com/proxikal/hydra/internal/logger"

// ConfigureClaudeCLI wires claude-cli MCP entries through Hydra.
// - Backs up the original config to <path>.hydra.bak (unless dry-run).
// - Adds any discovered MCP servers to the Hydra registry (if missing).
// - Rewrites claude-cli entries to call "hydra run --name <server>".
func ConfigureClaudeCLI(path string, registryPath string, dryRun bool, log logger.Logger) error {
	if path == "" {
		path = detectClaudeCLIPath()
	}

	return configureGenericMCP(path, registryPath, dryRun, log, "claude-cli")
}
