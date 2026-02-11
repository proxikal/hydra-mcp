package discovery

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/proxikal/hydra/internal/bootstrap"
	"github.com/proxikal/hydra/internal/logger"
)

// Discovery represents a detected MCP client configuration.
type Discovery struct {
	Client     string  // "claude", "cline", etc.
	ConfigPath string  // Full path to config file
	Confidence float64 // 0.0-1.0 confidence score
	Servers    int     // Count of mcpServers found
}

// Scanner detects installed MCP clients and their configurations.
type Scanner interface {
	// ScanAll checks all known client paths and returns discoveries.
	ScanAll() ([]Discovery, error)

	// ScanClient checks a specific client's configuration.
	ScanClient(client string) (*Discovery, error)
}

// scanner implements the Scanner interface.
type scanner struct {
	log logger.Logger
}

// NewScanner creates a new discovery scanner.
func NewScanner(log logger.Logger) Scanner {
	return &scanner{
		log: log,
	}
}

// ScanAll checks all known client paths.
func (s *scanner) ScanAll() ([]Discovery, error) {
	knownClients := []string{"claude", "cline", "cursor", "claude-cli", "continue", "cursor-composer"}
	var discoveries []Discovery

	for _, client := range knownClients {
		discovery, err := s.ScanClient(client)
		if err != nil {
			s.log.Debug("failed to scan client", map[string]interface{}{
				"client": client,
				"error":  err.Error(),
			})
			continue
		}

		// Include all discoveries, even with low confidence
		discoveries = append(discoveries, *discovery)
	}

	return discoveries, nil
}

// ScanClient checks a specific client's configuration.
func (s *scanner) ScanClient(client string) (*Discovery, error) {
	// Get path from bootstrap
	configPath, err := bootstrap.DetectClientConfigPath(client)
	if err != nil {
		return nil, fmt.Errorf("unknown client: %s: %w", client, err)
	}

	discovery := &Discovery{
		Client:     client,
		ConfigPath: configPath,
		Confidence: 0.0,
		Servers:    0,
	}

	// Check if file exists
	confidence, servers := s.checkConfigFile(configPath)
	discovery.Confidence = confidence
	discovery.Servers = servers

	return discovery, nil
}

// checkConfigFile validates a config file and returns confidence + server count.
func (s *scanner) checkConfigFile(path string) (confidence float64, servers int) {
	// Check file existence
	data, err := os.ReadFile(path)
	if err != nil {
		return 0.0, 0
	}

	confidence = 0.5 // File exists

	// Try to parse JSON
	var configData map[string]interface{}
	if err := json.Unmarshal(data, &configData); err != nil {
		return confidence, 0
	}

	confidence = 0.8 // Valid JSON

	// Count servers
	servers = countServers(configData)
	if servers > 0 {
		confidence = 1.0 // Has MCP servers
	}

	return confidence, servers
}

// countServers parses config and counts mcpServers entries.
func countServers(configData map[string]interface{}) int {
	mcpServers, ok := configData["mcpServers"]
	if !ok {
		return 0
	}

	serversMap, ok := mcpServers.(map[string]interface{})
	if !ok {
		return 0
	}

	return len(serversMap)
}
