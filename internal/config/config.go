package config

// Config represents the full Hydra configuration.
type Config struct {
	Command     string      `json:"command"`
	Args        []string    `json:"args"`
	EnvFile     string      `json:"env_file,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	Watch       WatchConfig `json:"watch"`
	Behavior    BehaviorConfig `json:"behavior"`
}

type WatchConfig struct {
	Paths       []string `json:"paths"`
	Extensions  []string `json:"extensions"`
	IgnoreFiles []string `json:"ignore_files"`
	Ignore      []string `json:"ignore"`
}

type BehaviorConfig struct {
	DebounceMS         int    `json:"debounce_ms"`
	RestartDelayMS     int    `json:"restart_delay_ms"`
	MaxRestarts        int    `json:"max_restarts"`
	GracefulShutdownMS int    `json:"graceful_shutdown_ms"`
	PreRestartCommand  string `json:"pre_restart_command,omitempty"`
}

// DefaultConfig returns the safe defaults.
func DefaultConfig() Config {
	return Config{
		Command: "echo",
		Args:    []string{"server"},
		Watch: WatchConfig{
			Paths:       []string{"."},
			Extensions:  []string{".py", ".js", ".ts", ".go"},
			IgnoreFiles: []string{".gitignore"},
			Ignore:      []string{".git", "node_modules", "__pycache__", ".hydra"},
		},
		Behavior: BehaviorConfig{
			DebounceMS:         500,
			RestartDelayMS:     0,
			MaxRestarts:        10,
			GracefulShutdownMS: 2000,
		},
	}
}
