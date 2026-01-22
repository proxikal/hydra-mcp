package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeServerConfig(t *testing.T) {
	t.Run("Merge command and args", func(t *testing.T) {
		dst := DefaultServerConfig()
		dst.Command = "old"
		src := &ServerConfig{
			Command: "new",
			Args:    []string{"arg1", "arg2"},
		}
		MergeServerConfig(dst, src)
		assert.Equal(t, "new", dst.Command)
		assert.Equal(t, []string{"arg1", "arg2"}, dst.Args)
	})

	t.Run("Merge CWD and EnvFile", func(t *testing.T) {
		dst := DefaultServerConfig()
		src := &ServerConfig{
			CWD:     "/custom/path",
			EnvFile: ".env.prod",
		}
		MergeServerConfig(dst, src)
		assert.Equal(t, "/custom/path", dst.CWD)
		assert.Equal(t, ".env.prod", dst.EnvFile)
	})

	t.Run("Merge environment variables", func(t *testing.T) {
		dst := DefaultServerConfig()
		dst.Environment = map[string]string{"KEY1": "val1"}
		src := &ServerConfig{
			Environment: map[string]string{"KEY2": "val2"},
		}
		MergeServerConfig(dst, src)
		assert.Equal(t, "val1", dst.Environment["KEY1"])
		assert.Equal(t, "val2", dst.Environment["KEY2"])
	})

	t.Run("Merge watch config", func(t *testing.T) {
		dst := DefaultServerConfig()
		src := &ServerConfig{
			Watch: WatchConfig{
				Enabled:     true,
				Paths:       []string{"/custom/path"},
				Extensions:  []string{".rs"},
				IgnoreFiles: []string{".gitignore"},
				Ignore:      []string{"target"},
			},
		}
		MergeServerConfig(dst, src)
		assert.True(t, dst.Watch.Enabled)
		assert.Equal(t, []string{"/custom/path"}, dst.Watch.Paths)
		assert.Equal(t, []string{".rs"}, dst.Watch.Extensions)
		assert.Equal(t, []string{".gitignore"}, dst.Watch.IgnoreFiles)
		assert.Equal(t, []string{"target"}, dst.Watch.Ignore)
	})

	t.Run("Merge behavior config", func(t *testing.T) {
		dst := DefaultServerConfig()
		src := &ServerConfig{
			Behavior: BehaviorConfig{
				DebounceMS:         1000,
				RestartDelayMS:     100,
				MaxRestarts:        5,
				GracefulShutdownMS: 3000,
				PreRestartCommand:  "cleanup.sh",
			},
		}
		MergeServerConfig(dst, src)
		assert.Equal(t, 1000, dst.Behavior.DebounceMS)
		assert.Equal(t, 100, dst.Behavior.RestartDelayMS)
		assert.Equal(t, 5, dst.Behavior.MaxRestarts)
		assert.Equal(t, 3000, dst.Behavior.GracefulShutdownMS)
		assert.Equal(t, "cleanup.sh", dst.Behavior.PreRestartCommand)
	})
}

func TestDefaultServerConfig(t *testing.T) {
	cfg := DefaultServerConfig()
	assert.Equal(t, "", cfg.Command)
	assert.Equal(t, 500, cfg.Behavior.DebounceMS)
	assert.Equal(t, 10, cfg.Behavior.MaxRestarts)
	assert.False(t, cfg.Watch.Enabled)
}

func TestMerge_EdgeCases(t *testing.T) {
	t.Run("Merge with nil environment in dst", func(t *testing.T) {
		dst := &ServerConfig{Environment: nil}
		src := &ServerConfig{Environment: map[string]string{"KEY": "val"}}

		MergeServerConfig(dst, src)
		assert.NotNil(t, dst.Environment)
		assert.Equal(t, "val", dst.Environment["KEY"])
	})

	t.Run("Merge zero values dont override", func(t *testing.T) {
		dst := DefaultServerConfig()
		dst.Behavior.DebounceMS = 1000

		src := &ServerConfig{
			Behavior: BehaviorConfig{
				DebounceMS: 0, // Zero value should not override
			},
		}

		MergeServerConfig(dst, src)
		// Zero values shouldn't override existing values
		assert.Equal(t, 1000, dst.Behavior.DebounceMS)
	})
}
