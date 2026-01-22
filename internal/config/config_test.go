package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/proxikal/hydra/internal/mocks"
	"github.com/stretchr/testify/assert"
)

func TestLoader_Load(t *testing.T) {
	mockLogger := mocks.NewLogger(t)
	loader := NewLoader(mockLogger)

	t.Run("Defaults when missing", func(t *testing.T) {
		cfg, err := loader.Load("non-existent.json")
		assert.NoError(t, err)
		assert.Equal(t, "echo", cfg.Command)
	})

	t.Run("Valid JSON", func(t *testing.T) {
		tmp := filepath.Join(t.TempDir(), "hydra.json")
		content := `{"command": "python", "args": ["server.py"], "behavior": {"max_restarts": 5}}`
		err := os.WriteFile(tmp, []byte(content), 0644)
		assert.NoError(t, err)

		cfg, err := loader.Load(tmp)
		assert.NoError(t, err)
		assert.Equal(t, "python", cfg.Command)
		assert.Equal(t, 5, cfg.Behavior.MaxRestarts)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		tmp := filepath.Join(t.TempDir(), "bad.json")
		err := os.WriteFile(tmp, []byte("{ bad json "), 0644)
		assert.NoError(t, err)

		_, err = loader.Load(tmp)
		assert.Error(t, err)
	})
	
	t.Run("Load Env File", func(t *testing.T) {
		// Mock logger warning expectation (optional if we trigger it, but here we expect success)
		
		dir := t.TempDir()
		envPath := filepath.Join(dir, ".env")
		err := os.WriteFile(envPath, []byte("TEST_VAR=123"), 0644)
		assert.NoError(t, err)
		
		cfgPath := filepath.Join(dir, "hydra.json")
		content := `{"command": "cmd", "env_file": "` + envPath + `"}`
		err = os.WriteFile(cfgPath, []byte(content), 0644)
		assert.NoError(t, err)

		_, err = loader.Load(cfgPath)
		assert.NoError(t, err)
		
		assert.Equal(t, "123", os.Getenv("TEST_VAR"))
	})
}

func TestLoader_Validate(t *testing.T) {
	loader := NewLoader(nil)

	t.Run("Valid", func(t *testing.T) {
		cfg := DefaultConfig()
		assert.NoError(t, loader.Validate(&cfg))
	})

	t.Run("Missing Command", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Command = ""
		assert.Error(t, loader.Validate(&cfg))
	})
}
