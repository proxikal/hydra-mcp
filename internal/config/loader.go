package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/proxikal/hydra/internal/logger"
)

// Loader handles configuration loading, merging, and validation.
type Loader interface {
	// LoadRegistry loads the global registry from ~/.hydra/config.json.
	LoadRegistry(path string) (*Registry, error)

	// LoadServerConfig loads a local override from path (e.g., ./hydra.json).
	LoadServerConfig(path string) (*ServerConfig, error)

	// ResolveServer resolves final config for a named server.
	// 1. Load registry
	// 2. Find server entry
	// 3. Merge with local override if present
	// 4. Apply defaults
	ResolveServer(serverName, localOverridePath string) (*ServerConfig, error)

	// Validate checks if a server config is valid.
	Validate(cfg *ServerConfig) error
}

type fileLoader struct {
	logger logger.Logger
}

// NewLoader creates a new configuration loader.
func NewLoader(l logger.Logger) Loader {
	return &fileLoader{logger: l}
}

func (l *fileLoader) LoadRegistry(path string) (*Registry, error) {
	// Expand home directory
	if path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, path[1:])
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No registry file, return defaults
			l.logger.Warn("Registry file not found, using defaults", map[string]interface{}{
				"path": path,
			})
			return DefaultRegistry(), nil
		}
		return nil, fmt.Errorf("failed to read registry: %w", err)
	}

	var reg Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("invalid registry JSON: %w", err)
	}

	// Ensure servers map is initialized
	if reg.Servers == nil {
		reg.Servers = make(map[string]*ServerConfig)
	}

	return &reg, nil
}

func (l *fileLoader) LoadServerConfig(path string) (*ServerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No local override, return nil (not an error)
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read local config: %w", err)
	}

	var cfg ServerConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid local config JSON: %w", err)
	}

	return &cfg, nil
}

func (l *fileLoader) ResolveServer(serverName, localOverridePath string) (*ServerConfig, error) {
	// 1. Load global registry
	registryPath := "~/.hydra/config.json"
	registry, err := l.LoadRegistry(registryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	// 2. Find server entry
	serverCfg, exists := registry.Servers[serverName]
	if !exists {
		return nil, fmt.Errorf("server '%s' not found in registry", serverName)
	}

	// 3. Start with defaults
	final := DefaultServerConfig()

	// 4. Apply registry server config
	MergeServerConfig(final, serverCfg)

	// 5. Load and merge local override if present
	if localOverridePath != "" {
		localCfg, err := l.LoadServerConfig(localOverridePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load local override: %w", err)
		}
		if localCfg != nil {
			MergeServerConfig(final, localCfg)
		}
	}

	// 6. Apply environment variable substitution
	SubstituteEnvVarsInConfig(final)

	// 7. Load .env file if specified
	if final.EnvFile != "" {
		if err := godotenv.Load(final.EnvFile); err != nil {
			l.logger.Warn("Failed to load .env file", map[string]interface{}{
				"path":  final.EnvFile,
				"error": err.Error(),
			})
		}
	}

	return final, nil
}

func (l *fileLoader) Validate(cfg *ServerConfig) error {
	if cfg.Command == "" {
		return fmt.Errorf("command is required")
	}
	if cfg.Behavior.MaxRestarts < 0 {
		return fmt.Errorf("max_restarts cannot be negative")
	}
	if cfg.Behavior.DebounceMS < 0 {
		return fmt.Errorf("debounce_ms cannot be negative")
	}
	if cfg.Behavior.RestartWindowSeconds < 0 {
		return fmt.Errorf("restart_window_seconds cannot be negative")
	}
	switch cfg.InjectableTools.OnCollision {
	case "error", "warn", "disable_hydra_tool", "":
	default:
		return fmt.Errorf("invalid injectable_tools.on_collision: %s", cfg.InjectableTools.OnCollision)
	}
	if cfg.Recorder.BufferSize < 0 {
		return fmt.Errorf("recorder.buffer_size cannot be negative")
	}
	if cfg.Security.RedactPatterns == nil {
		cfg.Security.RedactPatterns = []string{}
	}
	return nil
}

// MergeServerConfig merges src into dst (src wins on conflicts).
func MergeServerConfig(dst, src *ServerConfig) {
	if src.Command != "" {
		dst.Command = src.Command
	}
	if len(src.Args) > 0 {
		dst.Args = src.Args
	}
	if src.CWD != "" {
		dst.CWD = src.CWD
	}
	if src.EnvFile != "" {
		dst.EnvFile = src.EnvFile
	}
	if src.Environment != nil {
		if dst.Environment == nil {
			dst.Environment = make(map[string]string)
		}
		for k, v := range src.Environment {
			dst.Environment[k] = v
		}
	}

	// Merge Watch config
	if src.Watch.Enabled {
		dst.Watch.Enabled = src.Watch.Enabled
	}
	if len(src.Watch.Paths) > 0 {
		dst.Watch.Paths = src.Watch.Paths
	}
	if len(src.Watch.Extensions) > 0 {
		dst.Watch.Extensions = src.Watch.Extensions
	}
	if len(src.Watch.IgnoreFiles) > 0 {
		dst.Watch.IgnoreFiles = src.Watch.IgnoreFiles
	}
	if len(src.Watch.Ignore) > 0 {
		dst.Watch.Ignore = src.Watch.Ignore
	}

	// Merge Behavior config
	if src.Behavior.DebounceMS != 0 {
		dst.Behavior.DebounceMS = src.Behavior.DebounceMS
	}
	if src.Behavior.RestartDelayMS != 0 {
		dst.Behavior.RestartDelayMS = src.Behavior.RestartDelayMS
	}
	if src.Behavior.MaxRestarts != 0 {
		dst.Behavior.MaxRestarts = src.Behavior.MaxRestarts
	}
	if src.Behavior.GracefulShutdownMS != 0 {
		dst.Behavior.GracefulShutdownMS = src.Behavior.GracefulShutdownMS
	}
	if src.Behavior.RestartWindowSeconds != 0 {
		dst.Behavior.RestartWindowSeconds = src.Behavior.RestartWindowSeconds
	}
	if src.Behavior.PreRestartCommand != "" {
		dst.Behavior.PreRestartCommand = src.Behavior.PreRestartCommand
	}

	// Merge InjectableTools
	if src.InjectableTools.Enabled {
		dst.InjectableTools.Enabled = src.InjectableTools.Enabled
	}
	if len(src.InjectableTools.Tools) > 0 {
		dst.InjectableTools.Tools = src.InjectableTools.Tools
	}
	if src.InjectableTools.OnCollision != "" {
		dst.InjectableTools.OnCollision = src.InjectableTools.OnCollision
	}

	// Merge Recorder
	if src.Recorder.Enabled {
		dst.Recorder.Enabled = src.Recorder.Enabled
	}
	if src.Recorder.BufferSize != 0 {
		dst.Recorder.BufferSize = src.Recorder.BufferSize
	}
	if src.Recorder.IncludeRequestBodies {
		dst.Recorder.IncludeRequestBodies = src.Recorder.IncludeRequestBodies
	}
	if src.Recorder.IncludeResponseBodies {
		dst.Recorder.IncludeResponseBodies = src.Recorder.IncludeResponseBodies
	}
	if src.Recorder.ExportOnCrash {
		dst.Recorder.ExportOnCrash = src.Recorder.ExportOnCrash
	}
	if src.Recorder.ExportPath != "" {
		dst.Recorder.ExportPath = src.Recorder.ExportPath
	}

	// Merge Security
	if len(src.Security.RedactPatterns) > 0 {
		dst.Security.RedactPatterns = src.Security.RedactPatterns
	}
	if src.Security.RedactReplacement != "" {
		dst.Security.RedactReplacement = src.Security.RedactReplacement
	}
}
