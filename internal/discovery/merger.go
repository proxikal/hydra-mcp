package discovery

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/proxikal/hydra/internal/config"
	"github.com/proxikal/hydra/internal/logger"
)

// ResolveConflicts handles multiple discoveries for the same client.
// Returns deduplicated list keeping highest confidence discovery per client.
func ResolveConflicts(discoveries []Discovery) []Discovery {
	if len(discoveries) == 0 {
		return []Discovery{}
	}

	// Group by client
	clientMap := make(map[string]Discovery)

	for _, d := range discoveries {
		existing, exists := clientMap[d.Client]
		if !exists {
			clientMap[d.Client] = d
			continue
		}

		// Keep higher confidence
		if d.Confidence > existing.Confidence {
			clientMap[d.Client] = d
		}
	}

	// Convert back to slice
	result := make([]Discovery, 0, len(clientMap))
	for _, d := range clientMap {
		result = append(result, d)
	}

	// Sort by client name for deterministic output
	sort.Slice(result, func(i, j int) bool {
		return result[i].Client < result[j].Client
	})

	return result
}

// MergeIntoRegistry adds discoveries to registry without overwriting existing servers.
func MergeIntoRegistry(discoveries []Discovery, registryPath string, log logger.Logger) error {
	if len(discoveries) == 0 {
		return nil
	}

	// Load registry
	loader := config.NewLoader(log)
	registry, err := loader.LoadRegistry(registryPath)
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Track added servers for logging
	addedCount := 0

	// Process each discovery
	for _, d := range discoveries {
		// Skip low confidence or missing files
		if d.Confidence < 0.5 {
			log.Debug("skipping low confidence discovery", map[string]interface{}{
				"client":     d.Client,
				"confidence": d.Confidence,
			})
			continue
		}

		// Read client config
		configData, err := os.ReadFile(d.ConfigPath)
		if err != nil {
			log.Debug("failed to read client config", map[string]interface{}{
				"client": d.Client,
				"path":   d.ConfigPath,
				"error":  err.Error(),
			})
			continue
		}

		// Parse client config
		var clientConfig map[string]interface{}
		if err := json.Unmarshal(configData, &clientConfig); err != nil {
			log.Debug("failed to parse client config", map[string]interface{}{
				"client": d.Client,
				"error":  err.Error(),
			})
			continue
		}

		// Extract mcpServers
		mcpServers, ok := clientConfig["mcpServers"].(map[string]interface{})
		if !ok {
			continue
		}

		// Add each server if not already in registry
		for serverName, serverConfigInterface := range mcpServers {
			// Skip if already exists
			if _, exists := registry.Servers[serverName]; exists {
				log.Debug("server already in registry, skipping", map[string]interface{}{
					"server": serverName,
					"client": d.Client,
				})
				continue
			}

			// Parse server config
			serverConfigMap, ok := serverConfigInterface.(map[string]interface{})
			if !ok {
				continue
			}

			// Create ServerConfig
			serverConfig := &config.ServerConfig{
				Command: getStringOrEmpty(serverConfigMap, "command"),
				Args:    getStringArrayOrEmpty(serverConfigMap, "args"),
			}

			// Apply defaults
			defaults := config.DefaultServerConfig()
			serverConfig.Watch = defaults.Watch
			serverConfig.Behavior = defaults.Behavior
			serverConfig.InjectableTools = defaults.InjectableTools
			serverConfig.Recorder = defaults.Recorder
			serverConfig.Security = defaults.Security

			// Add to registry
			registry.Servers[serverName] = serverConfig
			addedCount++

			log.Info("discovered new server", map[string]interface{}{
				"server": serverName,
				"client": d.Client,
			})
		}
	}

	// Save registry if we added servers
	if addedCount > 0 {
		if err := config.SaveRegistry(registry, registryPath); err != nil {
			return fmt.Errorf("failed to save registry: %w", err)
		}

		log.Info("merged discoveries into registry", map[string]interface{}{
			"added": addedCount,
		})
	}

	return nil
}

// getStringOrEmpty safely extracts a string from map or returns empty.
func getStringOrEmpty(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// getStringArrayOrEmpty safely extracts a string array from map or returns empty.
func getStringArrayOrEmpty(m map[string]interface{}, key string) []string {
	if val, ok := m[key].([]interface{}); ok {
		result := make([]string, 0, len(val))
		for _, item := range val {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	}
	return []string{}
}
