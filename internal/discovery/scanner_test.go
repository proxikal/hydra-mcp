package discovery

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/proxikal/hydra/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScanner(t *testing.T) {
	log := logger.New("error")
	s := NewScanner(log)
	require.NotNil(t, s)
}

func TestScanClient_ValidConfig(t *testing.T) {
	// Setup temp config with valid MCP servers
	tmpDir := t.TempDir()

	// Get the actual path that would be detected
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	log := logger.New("error")
	s := NewScanner(log)
	discovery, _ := s.ScanClient("claude")
	configPath := discovery.ConfigPath

	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))

	config := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"weather": map[string]interface{}{
				"command": "npx",
				"args":    []string{"@modelcontextprotocol/server-weather"},
			},
			"filesystem": map[string]interface{}{
				"command": "npx",
				"args":    []string{"@modelcontextprotocol/server-filesystem"},
			},
		},
	}
	data, _ := json.Marshal(config)
	require.NoError(t, os.WriteFile(configPath, data, 0644))

	result, err := s.ScanClient("claude")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "claude", result.Client)
	assert.Equal(t, configPath, result.ConfigPath)
	assert.Equal(t, 1.0, result.Confidence) // Full confidence
	assert.Equal(t, 2, result.Servers)
}

func TestScanClient_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	log := logger.New("error")
	s := NewScanner(log)

	result, err := s.ScanClient("claude")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "claude", result.Client)
	assert.Equal(t, 0.0, result.Confidence) // No confidence
	assert.Equal(t, 0, result.Servers)
}

func TestScanClient_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	log := logger.New("error")
	s := NewScanner(log)
	discovery, _ := s.ScanClient("claude")
	configPath := discovery.ConfigPath

	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))
	require.NoError(t, os.WriteFile(configPath, []byte("not json"), 0644))

	result, err := s.ScanClient("claude")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "claude", result.Client)
	assert.Equal(t, 0.5, result.Confidence) // File exists but invalid JSON
	assert.Equal(t, 0, result.Servers)
}

func TestScanClient_EmptyMCPServers(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	log := logger.New("error")
	s := NewScanner(log)
	discovery, _ := s.ScanClient("claude")
	configPath := discovery.ConfigPath

	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))

	config := map[string]interface{}{
		"mcpServers": map[string]interface{}{},
	}
	data, _ := json.Marshal(config)
	require.NoError(t, os.WriteFile(configPath, data, 0644))

	result, err := s.ScanClient("claude")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "claude", result.Client)
	assert.Equal(t, 0.8, result.Confidence) // Has mcpServers but empty
	assert.Equal(t, 0, result.Servers)
}

func TestScanClient_UnknownClient(t *testing.T) {
	log := logger.New("error")
	s := NewScanner(log)

	result, err := s.ScanClient("unknown-client")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unknown client")
}

func TestScanAll_MultipleClients(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	log := logger.New("error")
	s := NewScanner(log)

	// Get actual paths for each client
	claudeDiscovery, _ := s.ScanClient("claude")
	clineDiscovery, _ := s.ScanClient("cline")

	// Create Claude config
	require.NoError(t, os.MkdirAll(filepath.Dir(claudeDiscovery.ConfigPath), 0755))
	claudeConfig := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"weather": map[string]interface{}{"command": "npx"},
		},
	}
	data, _ := json.Marshal(claudeConfig)
	require.NoError(t, os.WriteFile(claudeDiscovery.ConfigPath, data, 0644))

	// Create Cline config
	require.NoError(t, os.MkdirAll(filepath.Dir(clineDiscovery.ConfigPath), 0755))
	clineConfig := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"filesystem": map[string]interface{}{"command": "npx"},
			"git":        map[string]interface{}{"command": "npx"},
		},
	}
	data, _ = json.Marshal(clineConfig)
	require.NoError(t, os.WriteFile(clineDiscovery.ConfigPath, data, 0644))

	results, err := s.ScanAll()
	require.NoError(t, err)
	require.NotEmpty(t, results)

	// Find Claude and Cline in results
	var claudeFound, clineFound bool
	for _, r := range results {
		if r.Client == "claude" && r.Confidence > 0.9 && r.Servers == 1 {
			claudeFound = true
		}
		if r.Client == "cline" && r.Confidence > 0.9 && r.Servers == 2 {
			clineFound = true
		}
	}

	assert.True(t, claudeFound, "Claude config should be discovered")
	assert.True(t, clineFound, "Cline config should be discovered")
}

func TestScanAll_NoClients(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	log := logger.New("error")
	s := NewScanner(log)

	results, err := s.ScanAll()
	require.NoError(t, err)

	// Results may be empty or contain only low-confidence discoveries
	for _, r := range results {
		assert.LessOrEqual(t, r.Confidence, 0.5)
	}
}

func TestCountServers(t *testing.T) {
	tests := []struct {
		name     string
		config   map[string]interface{}
		expected int
	}{
		{
			name: "two servers",
			config: map[string]interface{}{
				"mcpServers": map[string]interface{}{
					"weather": map[string]interface{}{"command": "npx"},
					"git":     map[string]interface{}{"command": "npx"},
				},
			},
			expected: 2,
		},
		{
			name: "no mcpServers key",
			config: map[string]interface{}{
				"other": "value",
			},
			expected: 0,
		},
		{
			name: "empty mcpServers",
			config: map[string]interface{}{
				"mcpServers": map[string]interface{}{},
			},
			expected: 0,
		},
		{
			name: "mcpServers not a map",
			config: map[string]interface{}{
				"mcpServers": "invalid",
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := countServers(tt.config)
			assert.Equal(t, tt.expected, count)
		})
	}
}
