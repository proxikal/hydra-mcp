package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/proxikal/hydra/internal/logger"
)

type Loader interface {
	Load(path string) (*Config, error)
	Validate(cfg *Config) error
}

type fileLoader struct {
	logger logger.Logger
}

func NewLoader(l logger.Logger) Loader {
	return &fileLoader{logger: l}
}

func (l *fileLoader) Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	// 1. Read file
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &cfg, nil // Return defaults if no file
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	// 2. Unmarshal
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config json: %w", err)
	}

	// 3. Load .env if specified
	if cfg.EnvFile != "" {
		if err := godotenv.Load(cfg.EnvFile); err != nil {
			l.logger.Warn("Failed to load .env file", map[string]interface{}{
				"path":  cfg.EnvFile,
				"error": err.Error(),
			})
		}
	}

	return &cfg, nil
}

func (l *fileLoader) Validate(cfg *Config) error {
	if cfg.Command == "" {
		return fmt.Errorf("command is required")
	}
	if cfg.Behavior.MaxRestarts < 0 {
		return fmt.Errorf("max_restarts cannot be negative")
	}
	return nil
}
