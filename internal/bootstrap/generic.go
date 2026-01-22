package bootstrap

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/proxikal/hydra/internal/config"
	"github.com/proxikal/hydra/internal/logger"
)

func configureGenericMCP(path string, registryPath string, dryRun bool, log logger.Logger, client string) error {
	path = expandHome(path)
	registryPath = expandHome(registryPath)

	loader := config.NewLoader(log)
	reg, err := loader.LoadRegistry(registryPath)
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s config: %w", client, err)
	}

	var cfg claudeConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parse %s config: %w", client, err)
	}

	if len(cfg.MCPServers) == 0 {
		return fmt.Errorf("no mcpServers found in %s config", client)
	}

	changed := false
	for name, srv := range cfg.MCPServers {
		if _, ok := reg.Servers[name]; !ok {
			serverCfg := config.DefaultServerConfig()
			serverCfg.Command = srv.Command
			serverCfg.Args = srv.Args
			reg.Servers[name] = serverCfg
			changed = true
		}
	}

	if changed && !dryRun {
		if err := config.SaveRegistry(reg, registryPath); err != nil {
			return fmt.Errorf("save registry: %w", err)
		}
	}

	for name := range cfg.MCPServers {
		cfg.MCPServers[name] = claudeServer{
			Command: "hydra",
			Args: []string{
				"run",
				"--name", name,
				"--config", registryPath,
			},
		}
	}

	if dryRun {
		return nil
	}

	backup := path + ".hydra.bak"
	if _, err := os.Stat(backup); err == nil {
		return fmt.Errorf("backup already exists at %s (run uninit first)", backup)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("ensure config dir: %w", err)
	}

	if err := os.WriteFile(backup, data, 0o644); err != nil {
		return fmt.Errorf("write backup: %w", err)
	}

	updated, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s config: %w", client, err)
	}

	if err := os.WriteFile(path, updated, 0o644); err != nil {
		return fmt.Errorf("write %s config: %w", client, err)
	}

	return nil
}
