package bootstrap

import "github.com/proxikal/hydra/internal/logger"

// ConfigureCursorComposer wires Cursor Composer MCP entries through Hydra.
// - Backs up the original config to <path>.hydra.bak (unless dry-run).
// - Adds any discovered MCP servers to the Hydra registry (if missing).
// - Rewrites Cursor Composer entries to call "hydra run --name <server>".
func ConfigureCursorComposer(path string, registryPath string, dryRun bool, log logger.Logger) error {
	if path == "" {
		path = detectCursorComposerPath()
	}

	return configureGenericMCP(path, registryPath, dryRun, log, "cursor-composer")
}
