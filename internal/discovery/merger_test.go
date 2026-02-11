package discovery

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/proxikal/hydra/internal/config"
	"github.com/proxikal/hydra/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveConflicts_NoDuplicates(t *testing.T) {
	discoveries := []Discovery{
		{Client: "claude", ConfigPath: "/path/claude", Confidence: 1.0, Servers: 2},
		{Client: "cline", ConfigPath: "/path/cline", Confidence: 0.8, Servers: 1},
	}

	result := ResolveConflicts(discoveries)

	assert.Len(t, result, 2)
	assert.Equal(t, "claude", result[0].Client)
	assert.Equal(t, "cline", result[1].Client)
}

func TestResolveConflicts_SameClientMultipleTimes(t *testing.T) {
	discoveries := []Discovery{
		{Client: "claude", ConfigPath: "/path/claude1", Confidence: 0.8, Servers: 1},
		{Client: "claude", ConfigPath: "/path/claude2", Confidence: 1.0, Servers: 3},
		{Client: "claude", ConfigPath: "/path/claude3", Confidence: 0.5, Servers: 0},
	}

	result := ResolveConflicts(discoveries)

	assert.Len(t, result, 1, "Should keep only one claude entry")
	assert.Equal(t, "claude", result[0].Client)
	assert.Equal(t, 1.0, result[0].Confidence, "Should keep highest confidence")
	assert.Equal(t, "/path/claude2", result[0].ConfigPath)
	assert.Equal(t, 3, result[0].Servers)
}

func TestResolveConflicts_MultipleClientsDuplicates(t *testing.T) {
	discoveries := []Discovery{
		{Client: "claude", ConfigPath: "/path/claude1", Confidence: 0.8, Servers: 1},
		{Client: "cline", ConfigPath: "/path/cline1", Confidence: 0.7, Servers: 2},
		{Client: "claude", ConfigPath: "/path/claude2", Confidence: 1.0, Servers: 3},
		{Client: "cline", ConfigPath: "/path/cline2", Confidence: 0.9, Servers: 4},
		{Client: "cursor", ConfigPath: "/path/cursor", Confidence: 0.6, Servers: 1},
	}

	result := ResolveConflicts(discoveries)

	assert.Len(t, result, 3, "Should have 3 unique clients")

	// Find each client
	var claudeFound, clineFound, cursorFound bool
	for _, d := range result {
		switch d.Client {
		case "claude":
			claudeFound = true
			assert.Equal(t, 1.0, d.Confidence, "Should keep highest confidence claude")
			assert.Equal(t, 3, d.Servers)
		case "cline":
			clineFound = true
			assert.Equal(t, 0.9, d.Confidence, "Should keep highest confidence cline")
			assert.Equal(t, 4, d.Servers)
		case "cursor":
			cursorFound = true
			assert.Equal(t, 0.6, d.Confidence)
		}
	}

	assert.True(t, claudeFound && clineFound && cursorFound, "All clients should be present")
}

func TestResolveConflicts_EmptyInput(t *testing.T) {
	result := ResolveConflicts([]Discovery{})
	assert.Empty(t, result)
}

func TestResolveConflicts_SingleDiscovery(t *testing.T) {
	discoveries := []Discovery{
		{Client: "claude", ConfigPath: "/path/claude", Confidence: 1.0, Servers: 2},
	}

	result := ResolveConflicts(discoveries)

	assert.Len(t, result, 1)
	assert.Equal(t, "claude", result[0].Client)
}

func TestMergeIntoRegistry_NewServers(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	// Create empty registry
	registry := &config.Registry{
		Servers: make(map[string]*config.ServerConfig),
	}
	data, _ := json.Marshal(registry)
	require.NoError(t, os.WriteFile(registryPath, data, 0644))

	// Setup discovered servers
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	claudePath := filepath.Join(tmpDir, "Library/Application Support/Claude/claude_desktop_config.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(claudePath), 0755))
	claudeConfig := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"weather": map[string]interface{}{
				"command": "npx",
				"args":    []string{"@modelcontextprotocol/server-weather"},
			},
		},
	}
	data, _ = json.Marshal(claudeConfig)
	require.NoError(t, os.WriteFile(claudePath, data, 0644))

	discoveries := []Discovery{
		{Client: "claude", ConfigPath: claudePath, Confidence: 1.0, Servers: 1},
	}

	log := logger.New("error")
	err := MergeIntoRegistry(discoveries, registryPath, log)
	require.NoError(t, err)

	// Verify registry was updated
	loader := config.NewLoader(log)
	updatedRegistry, err := loader.LoadRegistry(registryPath)
	require.NoError(t, err)

	assert.Contains(t, updatedRegistry.Servers, "weather")
	assert.Equal(t, "npx", updatedRegistry.Servers["weather"].Command)
}

func TestMergeIntoRegistry_PreservesExisting(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	// Create registry with existing server
	registry := &config.Registry{
		Servers: map[string]*config.ServerConfig{
			"existing": {
				Command: "node",
				Args:    []string{"server.js"},
			},
		},
	}
	data, _ := json.Marshal(registry)
	require.NoError(t, os.WriteFile(registryPath, data, 0644))

	// Setup discovered servers
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	claudePath := filepath.Join(tmpDir, "Library/Application Support/Claude/claude_desktop_config.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(claudePath), 0755))
	claudeConfig := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"weather": map[string]interface{}{
				"command": "npx",
				"args":    []string{"@modelcontextprotocol/server-weather"},
			},
		},
	}
	data, _ = json.Marshal(claudeConfig)
	require.NoError(t, os.WriteFile(claudePath, data, 0644))

	discoveries := []Discovery{
		{Client: "claude", ConfigPath: claudePath, Confidence: 1.0, Servers: 1},
	}

	log := logger.New("error")
	err := MergeIntoRegistry(discoveries, registryPath, log)
	require.NoError(t, err)

	// Verify both servers exist
	loader := config.NewLoader(log)
	updatedRegistry, err := loader.LoadRegistry(registryPath)
	require.NoError(t, err)

	assert.Len(t, updatedRegistry.Servers, 2)
	assert.Contains(t, updatedRegistry.Servers, "existing")
	assert.Contains(t, updatedRegistry.Servers, "weather")
}

func TestMergeIntoRegistry_SkipsDuplicates(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	// Create registry with existing server
	registry := &config.Registry{
		Servers: map[string]*config.ServerConfig{
			"weather": {
				Command: "node",
				Args:    []string{"old-server.js"},
			},
		},
	}
	data, _ := json.Marshal(registry)
	require.NoError(t, os.WriteFile(registryPath, data, 0644))

	// Setup discovered servers with same name
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	claudePath := filepath.Join(tmpDir, "Library/Application Support/Claude/claude_desktop_config.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(claudePath), 0755))
	claudeConfig := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"weather": map[string]interface{}{
				"command": "npx",
				"args":    []string{"@modelcontextprotocol/server-weather"},
			},
		},
	}
	data, _ = json.Marshal(claudeConfig)
	require.NoError(t, os.WriteFile(claudePath, data, 0644))

	discoveries := []Discovery{
		{Client: "claude", ConfigPath: claudePath, Confidence: 1.0, Servers: 1},
	}

	log := logger.New("error")
	err := MergeIntoRegistry(discoveries, registryPath, log)
	require.NoError(t, err)

	// Verify existing server was NOT overwritten
	loader := config.NewLoader(log)
	updatedRegistry, err := loader.LoadRegistry(registryPath)
	require.NoError(t, err)

	assert.Len(t, updatedRegistry.Servers, 1)
	assert.Equal(t, "node", updatedRegistry.Servers["weather"].Command, "Should preserve existing")
	assert.Equal(t, "old-server.js", updatedRegistry.Servers["weather"].Args[0])
}

func TestMergeIntoRegistry_InvalidConfigPath(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	registry := &config.Registry{
		Servers: make(map[string]*config.ServerConfig),
	}
	data, _ := json.Marshal(registry)
	require.NoError(t, os.WriteFile(registryPath, data, 0644))

	discoveries := []Discovery{
		{Client: "claude", ConfigPath: "/nonexistent/path", Confidence: 0.5, Servers: 0},
	}

	log := logger.New("error")
	err := MergeIntoRegistry(discoveries, registryPath, log)

	// Should not error, just skip invalid configs
	require.NoError(t, err)
}

func TestMergeIntoRegistry_EmptyDiscoveries(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	registry := &config.Registry{
		Servers: make(map[string]*config.ServerConfig),
	}
	data, _ := json.Marshal(registry)
	require.NoError(t, os.WriteFile(registryPath, data, 0644))

	log := logger.New("error")
	err := MergeIntoRegistry([]Discovery{}, registryPath, log)

	require.NoError(t, err)
}
