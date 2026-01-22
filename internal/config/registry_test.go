package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSaveRegistry(t *testing.T) {
	t.Run("Save registry successfully", func(t *testing.T) {
		tmpdir := t.TempDir()
		path := filepath.Join(tmpdir, "config.json")

		reg := DefaultRegistry()
		reg.Servers["test"] = &ServerConfig{Command: "echo"}

		err := SaveRegistry(reg, path)
		assert.NoError(t, err)

		// Verify file exists and is valid
		data, err := os.ReadFile(path)
		assert.NoError(t, err)
		assert.Contains(t, string(data), "test")
	})

	t.Run("Create parent directory", func(t *testing.T) {
		tmpdir := t.TempDir()
		path := filepath.Join(tmpdir, "subdir", "config.json")

		reg := DefaultRegistry()
		err := SaveRegistry(reg, path)
		assert.NoError(t, err)

		_, err = os.Stat(path)
		assert.NoError(t, err)
	})
}

func TestRegistryOperations(t *testing.T) {
	t.Run("AddServer", func(t *testing.T) {
		reg := DefaultRegistry()
		cfg := &ServerConfig{Command: "python"}

		AddServer(reg, "my-server", cfg)
		assert.Contains(t, reg.Servers, "my-server")
		assert.Equal(t, "python", reg.Servers["my-server"].Command)
	})

	t.Run("RemoveServer existing", func(t *testing.T) {
		reg := DefaultRegistry()
		reg.Servers["test"] = &ServerConfig{Command: "node"}

		removed := RemoveServer(reg, "test")
		assert.True(t, removed)
		assert.NotContains(t, reg.Servers, "test")
	})

	t.Run("RemoveServer non-existing", func(t *testing.T) {
		reg := DefaultRegistry()
		removed := RemoveServer(reg, "nonexistent")
		assert.False(t, removed)
	})

	t.Run("ListServers", func(t *testing.T) {
		reg := DefaultRegistry()
		reg.Servers["server1"] = &ServerConfig{}
		reg.Servers["server2"] = &ServerConfig{}

		names := ListServers(reg)
		assert.Len(t, names, 2)
		assert.Contains(t, names, "server1")
		assert.Contains(t, names, "server2")
	})
}

func TestDefaultRegistry(t *testing.T) {
	reg := DefaultRegistry()
	assert.Equal(t, "1.0", reg.Version)
	assert.NotNil(t, reg.Servers)
	assert.Equal(t, 500, reg.Defaults.DebounceMS)
	assert.Equal(t, 2000, reg.Defaults.GracefulShutdownMS)
	assert.Equal(t, 50, reg.Defaults.MaxOutputSizeKB)
}

func TestSaveRegistry_EdgeCases(t *testing.T) {
	t.Run("Home directory expansion", func(t *testing.T) {
		tmpHome := t.TempDir()
		oldHome := os.Getenv("HOME")
		_ = os.Setenv("HOME", tmpHome)
		defer func() { _ = os.Setenv("HOME", oldHome) }()

		reg := DefaultRegistry()
		path := "~/.hydra/config.json"

		err := SaveRegistry(reg, path)
		assert.NoError(t, err)

		// Check file was created in expanded path
		expandedPath := filepath.Join(tmpHome, ".hydra", "config.json")
		_, err = os.Stat(expandedPath)
		assert.NoError(t, err)
	})
}

func TestAddServer_NilMap(t *testing.T) {
	reg := &Registry{
		Version:  "1.0",
		Defaults: Defaults{},
		Servers:  nil, // nil map
	}

	cfg := &ServerConfig{Command: "test"}
	AddServer(reg, "server1", cfg)

	assert.NotNil(t, reg.Servers)
	assert.Contains(t, reg.Servers, "server1")
}
