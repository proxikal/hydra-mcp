package bootstrap

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "tilde path",
			input:    "~/.config/test",
			expected: filepath.Join(home, ".config/test"),
		},
		{
			name:     "absolute path",
			input:    "/absolute/path",
			expected: "/absolute/path",
		},
		{
			name:     "relative path",
			input:    "relative/path",
			expected: "relative/path",
		},
		{
			name:     "empty path",
			input:    "",
			expected: "",
		},
		{
			name:     "just tilde",
			input:    "~",
			expected: home,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandHome(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectClientConfigPath(t *testing.T) {
	// Test known clients
	tests := []struct {
		client      string
		shouldExist bool
	}{
		{"claude", true},
		{"cline", true},
		{"cursor", true},
		{"claude-cli", true},
		{"continue", true},
		{"cursor-composer", true},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.client, func(t *testing.T) {
			path, err := detectClientConfigPath(tt.client)
			if tt.shouldExist {
				assert.NoError(t, err, "Should detect path for %s", tt.client)
				assert.NotEmpty(t, path, "Path should not be empty for %s", tt.client)
			} else {
				assert.Error(t, err, "Should error for unknown client")
			}
		})
	}
}

func TestDetectClaudeDesktopPath(t *testing.T) {
	path := detectClaudeDesktopPath()
	assert.NotEmpty(t, path)

	// Verify it contains expected patterns
	if runtime.GOOS == "darwin" {
		assert.Contains(t, path, "Library/Application Support/Claude")
	} else if runtime.GOOS == "windows" {
		assert.Contains(t, path, "AppData")
	} else {
		assert.Contains(t, path, ".config")
	}
}

func TestDetectClinePath(t *testing.T) {
	path := detectClinePath()
	assert.NotEmpty(t, path)
	// Should contain either .vscode or .vscodium
	hasVSCode := filepath.Base(filepath.Dir(path)) == ".vscode" || filepath.Base(filepath.Dir(path)) == ".vscodium"
	assert.True(t, hasVSCode, "Path should contain .vscode or .vscodium")
}

func TestDetectCursorPath(t *testing.T) {
	path := detectCursorPath()
	assert.NotEmpty(t, path)

	// Verify OS-specific patterns
	if runtime.GOOS == "darwin" {
		assert.Contains(t, path, "Library/Application Support/Cursor")
	} else if runtime.GOOS == "windows" {
		assert.Contains(t, path, "AppData")
	} else {
		assert.Contains(t, path, ".config")
	}
}

func TestDetectClaudeCLIPath(t *testing.T) {
	path := detectClaudeCLIPath()
	assert.NotEmpty(t, path)
	assert.Contains(t, path, ".config/claude-cli")
}

func TestDetectContinuePath(t *testing.T) {
	path := detectContinuePath()
	assert.NotEmpty(t, path)
	assert.Contains(t, path, ".continue")
}

func TestDetectCursorComposerPath(t *testing.T) {
	path := detectCursorComposerPath()
	assert.NotEmpty(t, path)
	assert.Contains(t, path, "Cursor")
}

func TestConfigFileExists(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.json")

	// File doesn't exist
	assert.False(t, configFileExists(testFile))

	// Create file
	err := os.WriteFile(testFile, []byte("{}"), 0644)
	require.NoError(t, err)

	// File exists
	assert.True(t, configFileExists(testFile))
}
