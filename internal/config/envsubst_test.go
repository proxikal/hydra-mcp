package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubstituteEnvVars(t *testing.T) {
	_ = os.Setenv("TEST_VAR", "test_value")
	_ = os.Setenv("API_KEY", "secret123")
	defer func() { _ = os.Unsetenv("TEST_VAR") }()
	defer func() { _ = os.Unsetenv("API_KEY") }()

	t.Run("Substitute single variable", func(t *testing.T) {
		result := SubstituteEnvVars("Hello ${env:TEST_VAR}")
		assert.Equal(t, "Hello test_value", result)
	})

	t.Run("Substitute multiple variables", func(t *testing.T) {
		result := SubstituteEnvVars("${env:TEST_VAR} and ${env:API_KEY}")
		assert.Equal(t, "test_value and secret123", result)
	})

	t.Run("No variables", func(t *testing.T) {
		result := SubstituteEnvVars("plain text")
		assert.Equal(t, "plain text", result)
	})

	t.Run("Undefined variable", func(t *testing.T) {
		result := SubstituteEnvVars("${env:UNDEFINED}")
		assert.Equal(t, "", result)
	})
}

func TestSubstituteEnvVarsInConfig(t *testing.T) {
	_ = os.Setenv("CMD", "python")
	_ = os.Setenv("PATH_VAR", "/custom/path")
	defer func() { _ = os.Unsetenv("CMD") }()
	defer func() { _ = os.Unsetenv("PATH_VAR") }()

	cfg := &ServerConfig{
		Command: "${env:CMD}",
		Args:    []string{"${env:PATH_VAR}/script.py"},
		CWD:     "${env:PATH_VAR}",
		EnvFile: "${env:PATH_VAR}/.env",
		Environment: map[string]string{
			"KEY": "${env:CMD}",
		},
		Watch: WatchConfig{
			Paths: []string{"${env:PATH_VAR}/src"},
		},
		Behavior: BehaviorConfig{
			PreRestartCommand: "${env:CMD} cleanup.py",
		},
	}

	SubstituteEnvVarsInConfig(cfg)

	assert.Equal(t, "python", cfg.Command)
	assert.Equal(t, "/custom/path/script.py", cfg.Args[0])
	assert.Equal(t, "/custom/path", cfg.CWD)
	assert.Equal(t, "/custom/path/.env", cfg.EnvFile)
	assert.Equal(t, "python", cfg.Environment["KEY"])
	assert.Equal(t, "/custom/path/src", cfg.Watch.Paths[0])
	assert.Equal(t, "python cleanup.py", cfg.Behavior.PreRestartCommand)
}
