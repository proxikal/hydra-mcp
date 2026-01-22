package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SaveRegistry writes a registry to disk.
func SaveRegistry(registry *Registry, path string) error {
	// Expand home directory
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, path[1:])
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}

	return nil
}

// AddServer adds or updates a server in the registry.
func AddServer(registry *Registry, name string, cfg *ServerConfig) {
	if registry.Servers == nil {
		registry.Servers = make(map[string]*ServerConfig)
	}
	registry.Servers[name] = cfg
}

// RemoveServer removes a server from the registry.
func RemoveServer(registry *Registry, name string) bool {
	if registry.Servers == nil {
		return false
	}
	_, exists := registry.Servers[name]
	if exists {
		delete(registry.Servers, name)
	}
	return exists
}

// ListServers returns all server names in the registry.
func ListServers(registry *Registry) []string {
	names := make([]string, 0, len(registry.Servers))
	for name := range registry.Servers {
		names = append(names, name)
	}
	return names
}
