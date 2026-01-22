package bootstrap

import "github.com/proxikal/hydra/internal/logger"

type claudeConfig struct {
	MCPServers map[string]claudeServer `json:"mcpServers"`
}

type claudeServer struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// ConfigureClaude wires Claude Desktop MCP entries through Hydra.
// - Backs up the original config to <path>.hydra.bak (unless dry-run).
// - Adds any discovered MCP servers to the Hydra registry (if missing).
// - Rewrites Claude entries to call "hydra run --name <server>".
func ConfigureClaude(path string, registryPath string, dryRun bool, log logger.Logger) error {
	if path == "" {
		path = "~/.config/claude/claude_desktop_config.json"
	}

	return configureGenericMCP(path, registryPath, dryRun, log, "claude")
}
