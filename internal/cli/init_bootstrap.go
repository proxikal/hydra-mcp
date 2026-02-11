package cli

import (
	"fmt"

	"github.com/proxikal/hydra/internal/bootstrap"
	"github.com/proxikal/hydra/internal/logger"
)

// bootstrapClient configures a specific client to route through Hydra.
func bootstrapClient(clientType, configPath, registryPath string, dryRun bool, log logger.Logger) error {
	switch clientType {
	case "claude":
		return bootstrap.ConfigureClaude(configPath, registryPath, dryRun, log)
	case "cline":
		return bootstrap.ConfigureCline(configPath, registryPath, dryRun, log)
	case "cursor":
		return bootstrap.ConfigureCursor(configPath, registryPath, dryRun, log)
	case "claude-cli":
		return bootstrap.ConfigureClaudeCLI(configPath, registryPath, dryRun, log)
	case "continue":
		return bootstrap.ConfigureContinue(configPath, registryPath, dryRun, log)
	case "cursor-composer":
		return bootstrap.ConfigureCursorComposer(configPath, registryPath, dryRun, log)
	default:
		return fmt.Errorf("unsupported client: %s (supported: claude, cline, cursor, claude-cli, continue, cursor-composer)", clientType)
	}
}

// supportedClients returns a list of all supported client types.
func supportedClients() []string {
	return []string{"claude", "cline", "cursor", "claude-cli", "continue", "cursor-composer"}
}
