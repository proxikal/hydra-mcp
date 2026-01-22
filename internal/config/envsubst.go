package config

import (
	"os"
	"regexp"
)

var envVarPattern = regexp.MustCompile(`\$\{env:([A-Za-z_][A-Za-z0-9_]*)\}`)

// SubstituteEnvVars replaces ${env:VAR} patterns with environment variable values.
// If a variable is not set, it is replaced with an empty string.
func SubstituteEnvVars(value string) string {
	return envVarPattern.ReplaceAllStringFunc(value, func(match string) string {
		// Extract variable name from ${env:VAR}
		varName := envVarPattern.FindStringSubmatch(match)[1]
		return os.Getenv(varName)
	})
}

// SubstituteEnvVarsInConfig applies environment variable substitution to a ServerConfig.
func SubstituteEnvVarsInConfig(cfg *ServerConfig) {
	// Substitute in command and args
	cfg.Command = SubstituteEnvVars(cfg.Command)
	for i, arg := range cfg.Args {
		cfg.Args[i] = SubstituteEnvVars(arg)
	}

	// Substitute in CWD and EnvFile
	cfg.CWD = SubstituteEnvVars(cfg.CWD)
	cfg.EnvFile = SubstituteEnvVars(cfg.EnvFile)

	// Substitute in environment variables
	for key, value := range cfg.Environment {
		cfg.Environment[key] = SubstituteEnvVars(value)
	}

	// Substitute in watch paths
	for i, path := range cfg.Watch.Paths {
		cfg.Watch.Paths[i] = SubstituteEnvVars(path)
	}

	// Substitute in pre-restart command
	cfg.Behavior.PreRestartCommand = SubstituteEnvVars(cfg.Behavior.PreRestartCommand)
}
